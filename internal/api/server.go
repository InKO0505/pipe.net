package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"clinet/internal/db"
)

type Config struct {
	Port int
}

type Server struct {
	httpServer *http.Server
	db         *db.DB
}

func Start(database *db.DB, cfg Config) *Server {
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	s := &Server{db: database}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/mobile/login", s.handleLogin)
	mux.HandleFunc("/api/mobile/me", s.withAuth(s.handleMe))
	mux.HandleFunc("/api/mobile/channels", s.withAuth(s.handleChannels))
	mux.HandleFunc("/api/mobile/channels/", s.withAuth(s.handleChannelRoutes))
	mux.HandleFunc("/api/mobile/dm", s.withAuth(s.handleDM))
	mux.HandleFunc("/api/mobile/modlog", s.withAuth(s.handleModLog))
	mux.HandleFunc("/api/mobile/mentions", s.withAuth(s.handleMentions))
	mux.HandleFunc("/api/mobile/users/", s.withAuth(s.handleUserLookup))
	mux.HandleFunc("/api/mobile/messages/", s.withAuth(s.handleMessageRoutes))

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           withJSONHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("Starting mobile API on port :%d", cfg.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Mobile API error: %v", err)
		}
	}()

	return s
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func withJSONHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withAuth(next func(http.ResponseWriter, *http.Request, *db.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" || token == auth && !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		user, err := s.db.GetUserByMobileToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		if user.IsBanned {
			writeError(w, http.StatusForbidden, "user is banned")
			return
		}
		_ = s.db.TouchUserActivity(user.ID)
		next(w, r, user)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

type loginRequest struct {
	Username string `json:"username"`
}

type userPayload struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Color    string `json:"color"`
	Bio      string `json:"bio"`
}

type channelPayload struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Topic        string `json:"topic"`
	Kind         string `json:"kind"`
	IsPrivate    bool   `json:"is_private"`
	UnreadCount  int    `json:"unread_count"`
	MentionCount int    `json:"mention_count"`
	MessageCount int    `json:"message_count"`
}

type messagePayload struct {
	ID              string   `json:"id"`
	ChannelID       string   `json:"channel_id"`
	AuthorID        string   `json:"author_id"`
	AuthorName      string   `json:"author_name"`
	AuthorColor     string   `json:"author_color"`
	AuthorRole      string   `json:"author_role"`
	Body            string   `json:"body"`
	CreatedAt       string   `json:"created_at"`
	EditedAt        string   `json:"edited_at,omitempty"`
	IsEdited        bool     `json:"is_edited"`
	ReplyToID       string   `json:"reply_to_id,omitempty"`
	ReplyToUsername string   `json:"reply_to_username,omitempty"`
	ReplyToContent  string   `json:"reply_to_content,omitempty"`
	Mentions        []string `json:"mentions,omitempty"`
}

func toUserPayload(user *db.User) userPayload {
	return userPayload{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
		Color:    user.Color,
		Bio:      user.Bio,
	}
}

func toChannelPayload(channel db.Channel, unread db.UnreadInfo) channelPayload {
	return channelPayload{
		ID:           channel.ID,
		Name:         channel.Name,
		Topic:        channel.Topic,
		Kind:         channel.Kind,
		IsPrivate:    channel.IsPrivate,
		UnreadCount:  unread.Count,
		MentionCount: unread.MentionCount,
		MessageCount: channel.MessageCount,
	}
}

func toMessagePayload(message db.Message) messagePayload {
	payload := messagePayload{
		ID:              message.ID,
		ChannelID:       message.ChannelID,
		AuthorID:        message.UserID,
		AuthorName:      message.Username,
		AuthorColor:     message.UserColor,
		AuthorRole:      message.UserRole,
		Body:            message.Content,
		CreatedAt:       message.CreatedAt.Format(time.RFC3339),
		IsEdited:        message.IsEdited,
		ReplyToID:       message.ReplyToID,
		ReplyToUsername: message.ReplyToUsername,
		ReplyToContent:  message.ReplyToContent,
		Mentions:        message.Mentions,
	}
	if message.IsEdited {
		payload.EditedAt = message.EditedAt.Format(time.RFC3339)
	}
	return payload
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "unknown username")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if user.IsBanned {
		writeError(w, http.StatusForbidden, "user is banned")
		return
	}

	session, err := s.db.CreateMobileSession(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": session.Token,
		"user":  toUserPayload(user),
	})
}

func (s *Server) handleMe(w http.ResponseWriter, _ *http.Request, user *db.User) {
	writeJSON(w, http.StatusOK, map[string]any{"user": toUserPayload(user)})
}

func (s *Server) handleChannels(w http.ResponseWriter, r *http.Request, user *db.User) {
	switch r.Method {
	case http.MethodGet:
		channels := s.db.GetAccessibleChannels(user)
		unread := s.db.GetUnreadInfo(user, channels)
		payload := make([]channelPayload, 0, len(channels))
		for _, channel := range channels {
			payload = append(payload, toChannelPayload(channel, unread[channel.ID]))
		}
		writeJSON(w, http.StatusOK, map[string]any{"channels": payload})
	case http.MethodPost:
		if user.Role != "owner" {
			writeError(w, http.StatusForbidden, "owner privileges required")
			return
		}
		var req struct {
			Name      string `json:"name"`
			IsPrivate bool   `json:"is_private"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		channel, err := s.db.CreateChannel(req.Name, req.IsPrivate, user.ID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"channel": toChannelPayload(*channel, db.UnreadInfo{})})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleChannelRoutes(w http.ResponseWriter, r *http.Request, user *db.User) {
	path := strings.TrimPrefix(r.URL.Path, "/api/mobile/channels/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	channelID := parts[0]
	channel, err := s.db.GetChannelByID(channelID, user)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, db.ErrChannelNotAccessible) {
			status = http.StatusForbidden
		}
		writeError(w, status, err.Error())
		return
	}

	if len(parts) == 1 {
		writeJSON(w, http.StatusOK, map[string]any{"channel": toChannelPayload(*channel, db.UnreadInfo{})})
		return
	}

	switch parts[1] {
	case "messages":
		s.handleChannelMessages(w, r, user, channel)
	case "topic":
		s.handleChannelTopic(w, r, user, channel)
	case "invite":
		s.handleChannelInvite(w, r, user, channel, true)
	case "remove":
		s.handleChannelInvite(w, r, user, channel, false)
	case "members":
		s.handleChannelMembers(w, r, channel)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) handleChannelMessages(w http.ResponseWriter, r *http.Request, user *db.User, channel *db.Channel) {
	switch r.Method {
	case http.MethodGet:
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		messages := s.db.GetMessagesPage(channel.ID, limit, offset)
		payload := make([]messagePayload, 0, len(messages))
		for _, message := range messages {
			payload = append(payload, toMessagePayload(message))
		}
		writeJSON(w, http.StatusOK, map[string]any{"messages": payload})
	case http.MethodPost:
		var req struct {
			Content   string `json:"content"`
			ReplyToID string `json:"reply_to_id"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		message, err := s.db.CreateMessage(channel.ID, user.ID, req.Content, req.ReplyToID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"message": toMessagePayload(message)})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleChannelTopic(w http.ResponseWriter, r *http.Request, user *db.User, channel *db.Channel) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if user.Role != "owner" && user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin privileges required")
		return
	}
	var req struct {
		Topic string `json:"topic"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.db.SetChannelTopic(channel.ID, req.Topic); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleChannelInvite(w http.ResponseWriter, r *http.Request, user *db.User, channel *db.Channel, add bool) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if user.Role != "owner" && user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin privileges required")
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	target, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user not found")
		return
	}
	if add {
		err = s.db.AddChannelMember(channel.ID, target.ID)
	} else {
		err = s.db.RemoveChannelMember(channel.ID, target.ID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleChannelMembers(w http.ResponseWriter, r *http.Request, channel *db.Channel) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	members := s.db.GetChannelMembers(channel.ID)
	payload := make([]userPayload, 0, len(members))
	for _, member := range members {
		memberCopy := member
		payload = append(payload, toUserPayload(&memberCopy))
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": payload})
}

func (s *Server) handleDM(w http.ResponseWriter, r *http.Request, user *db.User) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	target, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user not found")
		return
	}
	channel, err := s.db.GetOrCreateDirectChannel(user, target)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channel": toChannelPayload(*channel, db.UnreadInfo{})})
}

func (s *Server) handleModLog(w http.ResponseWriter, r *http.Request, user *db.User) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if user.Role != "owner" && user.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin privileges required")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, map[string]any{"logs": s.db.GetModerationLogs(limit)})
}

func (s *Server) handleMentions(w http.ResponseWriter, r *http.Request, user *db.User) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	messages := s.db.GetMentionMessages(user, 20)
	payload := make([]messagePayload, 0, len(messages))
	for _, message := range messages {
		payload = append(payload, toMessagePayload(message))
	}
	writeJSON(w, http.StatusOK, map[string]any{"mentions": payload})
}

func (s *Server) handleUserLookup(w http.ResponseWriter, r *http.Request, _ *db.User) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	username := strings.TrimPrefix(r.URL.Path, "/api/mobile/users/")
	target, err := s.db.GetUserByUsername(username)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": toUserPayload(target)})
}

func (s *Server) handleMessageRoutes(w http.ResponseWriter, r *http.Request, user *db.User) {
	path := strings.TrimPrefix(r.URL.Path, "/api/mobile/messages/")
	messageID := strings.Trim(path, "/")
	if messageID == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		writeError(w, http.StatusBadRequest, "missing channel_id")
		return
	}

	allowModeration := user.Role == "owner" || user.Role == "admin"
	switch r.Method {
	case http.MethodPatch:
		var req struct {
			Content string `json:"content"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		message, err := s.db.UpdateMessage(channelID, messageID, user.ID, req.Content, allowModeration)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": toMessagePayload(*message)})
	case http.MethodDelete:
		if err := s.db.DeleteMessage(channelID, messageID, user.ID, allowModeration); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
