package tui

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"clinet/internal/db"
	"clinet/internal/pubsub"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
)

type state int

const (
	onboardingState = iota
	mainState
	bannedState
)

type Model struct {
	database *db.DB
	broker   *pubsub.Broker
	user     *db.User
	session  ssh.Session
	appState state

	width  int
	height int

	onboarding viewport.Model
	input      textarea.Model
	viewport   viewport.Model
	renderer   *glamour.TermRenderer

	channels        []db.Channel
	unread          map[string]db.UnreadInfo
	activeChan      int
	messages        []db.Message
	loadedMessages  int
	messagePageSize int
	msgSub          chan db.Message
	isSubbed        bool
	commandPalette  []string
	tabPrefix       string
	tabMatchIdx     int
	userCache       map[string]*db.User
	exportDir       string
	backupDir       string
	lastStatus      string
}

type newMsgMsg db.Message

type imageFetchedMsg struct {
	url   string
	kitty string
	ansi  string
}

type imageAsset struct {
	kitty string
	ansi  string
}

var (
	globalImageCache sync.Map
	imageHTTPClient  = &http.Client{Timeout: 10 * time.Second}
)

func fetchImageCmd(rawURL string, targetWidth int) tea.Cmd {
	return func() tea.Msg {
		cleanURL, err := sanitizeImageURL(rawURL)
		if err != nil {
			globalImageCache.Store(rawURL, imageAsset{})
			return imageFetchedMsg{url: rawURL}
		}

		if v, ok := globalImageCache.Load(cleanURL); ok {
			asset := v.(imageAsset)
			return imageFetchedMsg{url: cleanURL, kitty: asset.kitty, ansi: asset.ansi}
		}

		req, err := http.NewRequest(http.MethodGet, cleanURL, nil)
		if err != nil {
			globalImageCache.Store(cleanURL, imageAsset{})
			return imageFetchedMsg{url: cleanURL}
		}

		resp, err := imageHTTPClient.Do(req) //nolint:bodyclose
		if err != nil {
			globalImageCache.Store(cleanURL, imageAsset{})
			return imageFetchedMsg{url: cleanURL}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			globalImageCache.Store(cleanURL, imageAsset{})
			return imageFetchedMsg{url: cleanURL}
		}
		if contentType := strings.ToLower(resp.Header.Get("Content-Type")); !strings.HasPrefix(contentType, "image/") {
			globalImageCache.Store(cleanURL, imageAsset{})
			return imageFetchedMsg{url: cleanURL}
		}

		data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		if err != nil || len(data) == 0 {
			globalImageCache.Store(cleanURL, imageAsset{})
			return imageFetchedMsg{url: cleanURL}
		}

		img, _, decodeErr := image.Decode(bytes.NewReader(data))
		ansi := ""
		if decodeErr == nil {
			ansi = renderHalfBlocks(img, targetWidth)
		}
		kitty := encodeKittyImage(data)
		globalImageCache.Store(cleanURL, imageAsset{kitty: kitty, ansi: ansi})
		return imageFetchedMsg{url: cleanURL, kitty: kitty, ansi: ansi}
	}
}

func sanitizeImageURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("invalid image URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("only http/https image URLs are allowed")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("image URL must include a host")
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".local") {
		return "", fmt.Errorf("local image hosts are not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsMulticast() {
			return "", fmt.Errorf("private image hosts are not allowed")
		}
	}

	return parsed.String(), nil
}

func encodeKittyImage(data []byte) string {
	b64 := base64.StdEncoding.EncodeToString(data)
	const chunkSize = 4096
	var sb strings.Builder
	for i := 0; i < len(b64); i += chunkSize {
		end := i + chunkSize
		if end > len(b64) {
			end = len(b64)
		}
		more := 0
		if end < len(b64) {
			more = 1
		}
		if i == 0 {
			sb.WriteString(fmt.Sprintf("\x1b_Ga=T,f=100,m=%d;%s\x1b\\", more, b64[i:end]))
			continue
		}
		sb.WriteString(fmt.Sprintf("\x1b_Gm=%d;%s\x1b\\", more, b64[i:end]))
	}
	return sb.String()
}

func renderHalfBlocks(img image.Image, targetWidth int) string {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return ""
	}
	if targetWidth > w {
		targetWidth = w
	}
	if targetWidth < 10 {
		targetWidth = 10
	}

	scale := float64(w) / float64(targetWidth)
	if scale <= 0 {
		return ""
	}
	targetHeight := int(float64(h) / scale)
	if targetHeight < 2 {
		targetHeight = 2
	}
	if targetHeight%2 != 0 {
		targetHeight--
	}

	var sb strings.Builder
	for y := 0; y < targetHeight; y += 2 {
		for x := 0; x < targetWidth; x++ {
			srcX := int(float64(x) * scale)
			srcY1 := int(float64(y) * scale)
			srcY2 := int(float64(y+1) * scale)
			if srcX >= w {
				srcX = w - 1
			}
			if srcY1 >= h {
				srcY1 = h - 1
			}
			if srcY2 >= h {
				srcY2 = h - 1
			}

			c1 := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY1)
			c2 := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY2)

			r1, g1, b1, _ := c1.RGBA()
			r2, g2, b2, _ := c2.RGBA()
			fg := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r1>>8, g1>>8, b1>>8))
			bg := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r2>>8, g2>>8, b2>>8))
			sb.WriteString(lipgloss.NewStyle().Foreground(fg).Background(bg).Render("▀"))
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func NewModel(database *db.DB, broker *pubsub.Broker, user *db.User, s ssh.Session) *Model {
	input := textarea.New()
	input.Placeholder = "Message... (/help for commands)"
	input.Focus()
	input.CharLimit = 2000

	vp := viewport.New(0, 0)
	onboarding := viewport.New(0, 0)
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(80),
	)

	exportDir := os.Getenv("CLINET_EXPORT_DIR")
	if exportDir == "" {
		exportDir = "exports"
	}
	backupDir := os.Getenv("CLINET_BACKUP_DIR")
	if backupDir == "" {
		backupDir = "backups"
	}

	m := &Model{
		database:        database,
		broker:          broker,
		user:            user,
		session:         s,
		input:           input,
		viewport:        vp,
		onboarding:      onboarding,
		renderer:        renderer,
		unread:          make(map[string]db.UnreadInfo),
		messagePageSize: 60,
		userCache:       make(map[string]*db.User),
		exportDir:       exportDir,
		backupDir:       backupDir,
		commandPalette: []string{
			"/help", "/nick", "/color", "/bio", "/clear", "/img", "/code", "/reply",
			"/search", "/older", "/edit", "/rm", "/mentions", "/members", "/whois",
			"/dm", "/invite", "/remove", "/modlog", "/export", "/backup",
			"/op", "/deop", "/kick", "/ban", "/unban", "/newchan", "/delchan",
			"/topic", "/del", "/setowner",
		},
	}

	if m.user.Color == "" {
		m.user.Color = "#6366F1"
	}
	_ = m.database.TouchUserActivity(m.user.ID)

	switch {
	case m.user.IsBanned:
		m.appState = bannedState
	case m.user.IsVerified:
		m.appState = mainState
	default:
		m.appState = onboardingState
	}

	m.refreshOnboarding()
	if m.appState == mainState {
		m.reloadAccessibleChannels("")
	}
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.waitForMessages())
}

func (m *Model) refreshOnboarding() {
	text := strings.Join([]string{
		"CLI-Net",
		"",
		"Welcome. This server now starts with a normal onboarding flow instead of the base64 gate.",
		"",
		"Available now:",
		"- public and private channels",
		"- direct messages via /dm <user>",
		"- replies, search, mentions and unread counters",
		"- edit/delete own messages",
		"- moderation log, export and backup commands",
		"",
		"Press Enter to continue.",
	}, "\n")
	m.onboarding.SetContent(text)
}

func (m *Model) currentWidth() int {
	if m.width <= 0 {
		return 80
	}
	width := int(float64(m.width) * 0.6)
	if width < 20 {
		return 20
	}
	return width
}

func (m *Model) activeChannel() *db.Channel {
	if len(m.channels) == 0 || m.activeChan < 0 || m.activeChan >= len(m.channels) {
		return nil
	}
	return &m.channels[m.activeChan]
}

func (m *Model) refreshUnread() {
	m.unread = m.database.GetUnreadInfo(m.user, m.channels)
}

func (m *Model) waitForMessages() tea.Cmd {
	return func() tea.Msg {
		if m.msgSub != nil {
			if msg, ok := <-m.msgSub; ok {
				return newMsgMsg(msg)
			}
		}
		return nil
	}
}

func (m *Model) appendSystemMsg(text string) {
	m.lastStatus = text
	m.messages = append(m.messages, db.Message{
		UserID:    "system",
		Username:  "Server",
		Content:   text,
		CreatedAt: time.Now(),
	})
	m.updateViewportContent()
}

func (m *Model) unsubscribeChannel(channelID string) {
	if !m.isSubbed || m.msgSub == nil || channelID == "" {
		m.msgSub = nil
		m.isSubbed = false
		return
	}
	m.broker.Unsubscribe(channelID, m.msgSub)
	m.msgSub = nil
	m.isSubbed = false
}

func (m *Model) subscribeActive() tea.Cmd {
	channel := m.activeChannel()
	if channel == nil {
		return nil
	}
	m.msgSub = m.broker.Subscribe(channel.ID, m.user)
	m.isSubbed = true
	return m.waitForMessages()
}

func (m *Model) markActiveChannelRead(lastMessageID string) {
	channel := m.activeChannel()
	if channel == nil {
		return
	}
	if lastMessageID == "" && len(m.messages) > 0 {
		lastMessageID = m.messages[len(m.messages)-1].ID
	}
	_ = m.database.MarkChannelRead(channel.ID, m.user.ID, lastMessageID, time.Now())
	m.refreshUnread()
}

func (m *Model) loadChannelMessages() {
	channel := m.activeChannel()
	if channel == nil {
		m.messages = nil
		m.loadedMessages = 0
		m.updateViewportContent()
		return
	}
	m.messages = m.database.GetMessagesPage(channel.ID, m.messagePageSize, 0)
	m.loadedMessages = len(m.messages)
	m.updateViewportContent()
	m.markActiveChannelRead("")
}

func (m *Model) loadOlderMessages() {
	channel := m.activeChannel()
	if channel == nil {
		return
	}
	older := m.database.GetMessagesPage(channel.ID, m.messagePageSize, m.loadedMessages)
	if len(older) == 0 {
		m.appendSystemMsg("No older messages.")
		return
	}
	m.messages = append(older, m.messages...)
	m.loadedMessages += len(older)
	m.updateViewportContent()
}

func (m *Model) resolveUser(userID string) *db.User {
	if userID == "" {
		return nil
	}
	if user, ok := m.userCache[userID]; ok {
		return user
	}
	user, err := m.database.GetUserByID(userID)
	if err != nil {
		return nil
	}
	m.userCache[userID] = user
	return user
}

func (m *Model) syncCurrentUser() {
	updated, err := m.database.GetUserByPubKey(m.user.SSHPubKey)
	if err == nil {
		*m.user = *updated
		m.userCache[m.user.ID] = updated
	}
}

func (m *Model) reloadAccessibleChannels(preferredID string) tea.Cmd {
	oldID := ""
	if current := m.activeChannel(); current != nil {
		oldID = current.ID
	}
	if preferredID == "" {
		preferredID = oldID
	}

	channels := m.database.GetAccessibleChannels(m.user)
	if len(channels) == 0 {
		m.unsubscribeChannel(oldID)
		m.channels = nil
		m.unread = map[string]db.UnreadInfo{}
		m.messages = nil
		m.loadedMessages = 0
		m.updateViewportContent()
		return nil
	}

	nextIndex := 0
	for i, channel := range channels {
		if channel.ID == preferredID {
			nextIndex = i
			break
		}
	}

	nextID := channels[nextIndex].ID
	shouldResubscribe := !m.isSubbed || oldID == "" || oldID != nextID
	if shouldResubscribe {
		m.unsubscribeChannel(oldID)
	}

	m.channels = channels
	m.activeChan = nextIndex
	m.loadChannelMessages()
	if shouldResubscribe {
		return m.subscribeActive()
	}
	return nil
}

func (m *Model) switchToChannel(index int) tea.Cmd {
	if len(m.channels) == 0 {
		return nil
	}
	if index < 0 || index >= len(m.channels) {
		index = 0
	}

	currentID := ""
	if current := m.activeChannel(); current != nil {
		currentID = current.ID
	}
	nextID := m.channels[index].ID
	if currentID != "" && currentID != nextID {
		m.unsubscribeChannel(currentID)
	}
	m.activeChan = index
	m.loadChannelMessages()
	if !m.isSubbed || currentID != nextID {
		return m.subscribeActive()
	}
	return nil
}

func (m *Model) publishMessage(content, replyToID string) (*db.Message, error) {
	channel := m.activeChannel()
	if channel == nil {
		return nil, fmt.Errorf("no accessible channel selected")
	}
	created, err := m.database.CreateMessage(channel.ID, m.user.ID, content, replyToID)
	if err != nil {
		return nil, err
	}
	_ = m.database.TouchUserActivity(m.user.ID)
	m.markActiveChannelRead(created.ID)
	m.broker.Broadcast(channel.ID, created)
	return &created, nil
}

func humanizeError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, db.ErrUsernameTaken):
		return "Nickname is already taken."
	case errors.Is(err, db.ErrInvalidUsername):
		return "Nickname must be 2-24 chars and use letters, numbers, _ or -."
	case errors.Is(err, db.ErrInvalidColor):
		return "Color must be a named color or #RRGGBB."
	case errors.Is(err, db.ErrChannelExists):
		return "Channel already exists."
	case errors.Is(err, db.ErrInvalidChannelName):
		return "Channel names must look like #general."
	case errors.Is(err, db.ErrChannelNotAccessible):
		return "You do not have access to that channel."
	case errors.Is(err, db.ErrEmptyMessage):
		return "Message is empty after sanitization."
	default:
		return err.Error()
	}
}

func shortMessageID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func snippet(text string) string {
	flat := strings.Join(strings.Fields(strings.ReplaceAll(text, "\n", " ")), " ")
	if len(flat) <= 72 {
		return flat
	}
	return flat[:69] + "..."
}

func indentBlock(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func parseNewChannelArgs(arg string) (string, bool) {
	clean := strings.TrimSpace(arg)
	switch {
	case strings.HasPrefix(clean, "private "):
		return strings.TrimSpace(strings.TrimPrefix(clean, "private ")), true
	case strings.HasPrefix(clean, "--private "):
		return strings.TrimSpace(strings.TrimPrefix(clean, "--private ")), true
	default:
		return clean, false
	}
}

func channelLabel(ch db.Channel) string {
	label := ch.Name
	if ch.Kind == "dm" {
		label = "DM " + strings.TrimPrefix(ch.Name, "@")
	}
	if ch.IsPrivate && ch.Kind != "dm" {
		label += " [private]"
	}
	return label
}

func formatLastSeen(ts time.Time) string {
	if ts.IsZero() {
		return "unknown"
	}
	return ts.Local().Format("2006-01-02 15:04")
}

func (m *Model) exportCurrentChannel() error {
	channel := m.activeChannel()
	if channel == nil {
		return fmt.Errorf("no active channel")
	}
	if err := os.MkdirAll(m.exportDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(m.exportDir, strings.TrimLeft(strings.ReplaceAll(channel.Name, "/", "_"), "#@")+".md")
	messages, err := m.database.ExportChannelTranscript(channel.ID)
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", channel.Name)
	fmt.Fprintf(&b, "Topic: %s\n\n", channel.Topic)
	for _, msg := range messages {
		fmt.Fprintf(&b, "[%s] %s (%s)\n", msg.CreatedAt.Format(time.RFC3339), msg.Username, shortMessageID(msg.ID))
		if msg.ReplyToID != "" {
			fmt.Fprintf(&b, "> Reply to %s: %s\n", msg.ReplyToUsername, snippet(msg.ReplyToContent))
		}
		fmt.Fprintf(&b, "%s\n\n", msg.Content)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func (m *Model) exportAllChannels() error {
	for _, channel := range m.channels {
		currentID := ""
		if current := m.activeChannel(); current != nil {
			currentID = current.ID
		}
		m.activeChan = 0
		for i := range m.channels {
			if m.channels[i].ID == channel.ID {
				m.activeChan = i
				break
			}
		}
		if err := m.exportCurrentChannel(); err != nil {
			return err
		}
		for i := range m.channels {
			if m.channels[i].ID == currentID {
				m.activeChan = i
				break
			}
		}
	}
	return nil
}

func (m *Model) renderHelp(isAdmin, isOwner bool) string {
	var lines []string
	lines = append(lines, "Local help")
	lines = append(lines, "/help /nick /color /bio /img /code /reply /search /older")
	lines = append(lines, "/edit /rm /mentions /members /whois /dm /export /backup")
	if isAdmin || isOwner {
		lines = append(lines, "Admin: /clear /kick /ban /unban /topic /invite /remove /modlog /del")
	}
	if isOwner {
		lines = append(lines, "Owner: /newchan [private] <name> /delchan <name> /op /deop /setowner")
	}
	lines = append(lines, "Keys: Tab switch channel, PgUp older, Ctrl+Y copy, Ctrl+C quit")
	return strings.Join(lines, "\n")
}

func (m *Model) renderModLog(limit int) string {
	logs := m.database.GetModerationLogs(limit)
	if len(logs) == 0 {
		return "Moderation log is empty."
	}
	var b strings.Builder
	for _, entry := range logs {
		fmt.Fprintf(&b, "[%s] %s -> %s on %s: %s (%s)\n",
			entry.CreatedAt.Format("01-02 15:04"),
			entry.ActorUsername,
			entry.TargetUsername,
			entry.ChannelName,
			entry.Action,
			entry.Details,
		)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *Model) renderSearchResults(term string, matches []db.Message) string {
	if len(matches) == 0 {
		return fmt.Sprintf("No matches for %q.", term)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Search results for %q:\n", term)
	for _, match := range matches {
		fmt.Fprintf(&b, "[%s] %s %s: %s\n", shortMessageID(match.ID), match.CreatedAt.Format("15:04"), match.Username, snippet(match.Content))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *Model) renderMembers() string {
	channel := m.activeChannel()
	if channel == nil {
		return "No active channel."
	}
	members := m.database.GetChannelMembers(channel.ID)
	if len(members) == 0 {
		return "No members."
	}
	var b strings.Builder
	for _, member := range members {
		status := "offline"
		for _, online := range m.broker.GetOnlineUsers(channel.ID) {
			if online.ID == member.ID {
				status = "online"
				break
			}
		}
		fmt.Fprintf(&b, "%s [%s] last seen %s\n", member.Username, status, formatLastSeen(member.LastSeenAt))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *Model) renderWhois(user *db.User) string {
	if user == nil {
		return "User not found."
	}
	return fmt.Sprintf("%s\nrole: %s\ncolor: %s\nbio: %s\nlast seen: %s", user.Username, user.Role, user.Color, user.Bio, formatLastSeen(user.LastSeenAt))
}

func (m *Model) renderMentions() string {
	matches := m.database.GetMentionMessages(m.user, 20)
	if len(matches) == 0 {
		return "No recent mentions."
	}
	var b strings.Builder
	for _, match := range matches {
		channel, err := m.database.GetChannelByID(match.ChannelID, m.user)
		channelName := match.ChannelID
		if err == nil {
			channelName = channel.Name
		}
		fmt.Fprintf(&b, "[%s] %s %s: %s\n", channelName, match.Username, shortMessageID(match.ID), snippet(match.Content))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *Model) notifyUserRefresh(user *db.User, details string) {
	if user == nil {
		return
	}
	m.broker.NotifyUser(user.Username, db.Message{
		ID:        "CMD_CHANNELS",
		UserID:    "system",
		Username:  "Server",
		Content:   details,
		CreatedAt: time.Now(),
	})
}

func (m *Model) handleSubmit(content string) []tea.Cmd {
	var cmds []tea.Cmd
	channel := m.activeChannel()
	if channel == nil {
		m.appendSystemMsg("No accessible channels. Ask an admin to invite you or create a public channel.")
		return nil
	}

	parts := strings.SplitN(content, " ", 2)
	name := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	role := m.user.Role
	isAdmin := role == "admin" || role == "owner"
	isOwner := role == "owner"

	if !strings.HasPrefix(name, "/") {
		if _, err := m.publishMessage(content, ""); err != nil {
			m.appendSystemMsg("Cannot send message: " + humanizeError(err))
		}
		return nil
	}

	switch name {
	case "/help":
		m.appendSystemMsg(m.renderHelp(isAdmin, isOwner))
	case "/nick":
		if arg == "" {
			m.appendSystemMsg("Usage: /nick <name>")
			return nil
		}
		if err := m.database.UpdateUsername(m.user.ID, arg); err != nil {
			m.appendSystemMsg(humanizeError(err))
			return nil
		}
		oldUsername := m.user.Username
		m.syncCurrentUser()
		delete(m.userCache, m.user.ID)
		m.appendSystemMsg("Nickname updated.")
		m.broker.NotifyUser(oldUsername, db.Message{ID: "CMD_REFRESH_USER", Content: "profile changed"})
	case "/color":
		if arg == "" {
			m.appendSystemMsg("Usage: /color <name|#RRGGBB>")
			return nil
		}
		color, err := m.database.UpdateUserColor(m.user.ID, arg)
		if err != nil {
			m.appendSystemMsg(humanizeError(err))
			return nil
		}
		m.user.Color = color
		m.appendSystemMsg("Nickname color updated to " + color)
	case "/bio":
		if err := m.database.UpdateUserBio(m.user.ID, arg); err != nil {
			m.appendSystemMsg("Bio update failed: " + err.Error())
			return nil
		}
		m.syncCurrentUser()
		m.appendSystemMsg("Bio updated.")
	case "/clear":
		if isAdmin {
			if err := m.database.ClearChannelMessages(channel.ID); err != nil {
				m.appendSystemMsg("Clear failed: " + err.Error())
				return nil
			}
			m.broker.Broadcast(channel.ID, db.Message{ID: "CMD_CLEAR", ChannelID: channel.ID})
			return nil
		}
		m.messages = nil
		m.updateViewportContent()
	case "/img":
		if arg == "" {
			m.appendSystemMsg("Usage: /img <url>")
			return nil
		}
		cleanURL, err := sanitizeImageURL(arg)
		if err != nil {
			m.appendSystemMsg("Image rejected: " + err.Error())
			return nil
		}
		if _, err := m.publishMessage("🖼️ "+cleanURL, ""); err != nil {
			m.appendSystemMsg("Cannot send image: " + humanizeError(err))
			return nil
		}
		if _, ok := globalImageCache.Load(cleanURL); !ok {
			cmds = append(cmds, fetchImageCmd(cleanURL, m.currentWidth()))
		}
	case "/code":
		if arg == "" {
			m.appendSystemMsg("Usage: /code <text>")
			return nil
		}
		if _, err := m.publishMessage("```\n"+arg+"\n```", ""); err != nil {
			m.appendSystemMsg("Cannot send code block: " + humanizeError(err))
		}
	case "/reply":
		replyParts := strings.SplitN(arg, " ", 2)
		if len(replyParts) != 2 || strings.TrimSpace(replyParts[0]) == "" || strings.TrimSpace(replyParts[1]) == "" {
			m.appendSystemMsg("Usage: /reply <message-id-prefix> <text>")
			return nil
		}
		target, err := m.database.FindMessageByPrefix(channel.ID, replyParts[0])
		if err != nil {
			m.appendSystemMsg("Reply failed: " + humanizeError(err))
			return nil
		}
		if _, err := m.publishMessage(replyParts[1], target.ID); err != nil {
			m.appendSystemMsg("Reply failed: " + humanizeError(err))
		}
	case "/search":
		if arg == "" {
			m.appendSystemMsg("Usage: /search <term>")
			return nil
		}
		m.appendSystemMsg(m.renderSearchResults(arg, m.database.SearchMessages(channel.ID, arg, 8)))
	case "/older":
		m.loadOlderMessages()
	case "/edit":
		editParts := strings.SplitN(arg, " ", 2)
		if len(editParts) != 2 {
			m.appendSystemMsg("Usage: /edit <message-id-prefix> <text>")
			return nil
		}
		target, err := m.database.FindMessageByPrefix(channel.ID, editParts[0])
		if err != nil {
			m.appendSystemMsg("Edit failed: " + err.Error())
			return nil
		}
		_, err = m.database.UpdateMessage(channel.ID, target.ID, m.user.ID, editParts[1], isAdmin)
		if err != nil {
			m.appendSystemMsg("Edit failed: " + humanizeError(err))
			return nil
		}
		m.loadChannelMessages()
		m.broker.Broadcast(channel.ID, db.Message{ID: "CMD_REFRESH", ChannelID: channel.ID})
	case "/rm":
		if arg == "" {
			m.appendSystemMsg("Usage: /rm <message-id-prefix>")
			return nil
		}
		target, err := m.database.FindMessageByPrefix(channel.ID, arg)
		if err != nil {
			m.appendSystemMsg("Delete failed: " + err.Error())
			return nil
		}
		if err := m.database.DeleteMessage(channel.ID, target.ID, m.user.ID, isAdmin); err != nil {
			m.appendSystemMsg("Delete failed: " + err.Error())
			return nil
		}
		m.loadChannelMessages()
		m.broker.Broadcast(channel.ID, db.Message{ID: "CMD_REFRESH", ChannelID: channel.ID})
	case "/mentions":
		m.appendSystemMsg(m.renderMentions())
	case "/members":
		m.appendSystemMsg(m.renderMembers())
	case "/whois":
		if arg == "" {
			m.appendSystemMsg("Usage: /whois <user>")
			return nil
		}
		target, err := m.database.GetUserByUsername(arg)
		if err != nil {
			m.appendSystemMsg("User not found.")
			return nil
		}
		m.appendSystemMsg(m.renderWhois(target))
	case "/dm":
		if arg == "" {
			m.appendSystemMsg("Usage: /dm <user>")
			return nil
		}
		target, err := m.database.GetUserByUsername(arg)
		if err != nil {
			m.appendSystemMsg("User not found.")
			return nil
		}
		dm, err := m.database.GetOrCreateDirectChannel(m.user, target)
		if err != nil {
			m.appendSystemMsg("DM failed: " + err.Error())
			return nil
		}
		m.notifyUserRefresh(target, "direct message ready")
		cmds = append(cmds, m.reloadAccessibleChannels(dm.ID))
	case "/invite":
		if !isAdmin {
			m.appendSystemMsg("Admin privileges required.")
			return nil
		}
		if !channel.IsPrivate || channel.Kind == "dm" {
			m.appendSystemMsg("Invites only apply to private non-DM channels.")
			return nil
		}
		target, err := m.database.GetUserByUsername(arg)
		if err != nil {
			m.appendSystemMsg("User not found.")
			return nil
		}
		if err := m.database.AddChannelMember(channel.ID, target.ID); err != nil {
			m.appendSystemMsg("Invite failed: " + err.Error())
			return nil
		}
		_ = m.database.CreateModerationLog(m.user.ID, target.ID, channel.ID, "invite", "added to private channel")
		m.notifyUserRefresh(target, "you were invited to "+channel.Name)
		m.appendSystemMsg("Invited " + target.Username)
	case "/remove":
		if !isAdmin {
			m.appendSystemMsg("Admin privileges required.")
			return nil
		}
		if !channel.IsPrivate || channel.Kind == "dm" {
			m.appendSystemMsg("Removal only applies to private non-DM channels.")
			return nil
		}
		target, err := m.database.GetUserByUsername(arg)
		if err != nil {
			m.appendSystemMsg("User not found.")
			return nil
		}
		if err := m.database.RemoveChannelMember(channel.ID, target.ID); err != nil {
			m.appendSystemMsg("Remove failed: " + err.Error())
			return nil
		}
		_ = m.database.CreateModerationLog(m.user.ID, target.ID, channel.ID, "remove", "removed from private channel")
		m.notifyUserRefresh(target, "removed from "+channel.Name)
		m.appendSystemMsg("Removed " + target.Username)
	case "/modlog":
		if !isAdmin {
			m.appendSystemMsg("Admin privileges required.")
			return nil
		}
		limit := 20
		if arg != "" {
			if parsed, err := strconv.Atoi(arg); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		m.appendSystemMsg(m.renderModLog(limit))
	case "/export":
		target := strings.ToLower(strings.TrimSpace(arg))
		var err error
		if target == "" || target == "current" {
			err = m.exportCurrentChannel()
		} else if target == "all" {
			err = m.exportAllChannels()
		} else {
			err = fmt.Errorf("usage: /export [current|all]")
		}
		if err != nil {
			m.appendSystemMsg("Export failed: " + err.Error())
			return nil
		}
		m.appendSystemMsg("Export completed into " + m.exportDir)
	case "/backup":
		if err := os.MkdirAll(m.backupDir, 0o755); err != nil {
			m.appendSystemMsg("Backup failed: " + err.Error())
			return nil
		}
		path := filepath.Join(m.backupDir, "clinet-"+time.Now().Format("20060102-150405")+".db")
		if err := m.database.CreateBackup(path); err != nil {
			m.appendSystemMsg("Backup failed: " + err.Error())
			return nil
		}
		m.appendSystemMsg("Backup saved to " + path)
	case "/op":
		if !isOwner {
			m.appendSystemMsg("Owner privileges required.")
			return nil
		}
		target, err := m.database.GetUserByUsername(arg)
		if err != nil {
			m.appendSystemMsg("User not found.")
			return nil
		}
		if err := m.database.SetUserRole(target.ID, "admin"); err != nil {
			m.appendSystemMsg("Promotion failed: " + err.Error())
			return nil
		}
		_ = m.database.CreateModerationLog(m.user.ID, target.ID, channel.ID, "op", "promoted to admin")
		m.notifyUserRefresh(target, "you were promoted to admin")
		m.appendSystemMsg("Promoted " + target.Username + " to admin.")
	case "/deop":
		if !isOwner {
			m.appendSystemMsg("Owner privileges required.")
			return nil
		}
		target, err := m.database.GetUserByUsername(arg)
		if err != nil || target.Role == "owner" {
			m.appendSystemMsg("User not found or is owner.")
			return nil
		}
		if err := m.database.SetUserRole(target.ID, "user"); err != nil {
			m.appendSystemMsg("Demotion failed: " + err.Error())
			return nil
		}
		_ = m.database.CreateModerationLog(m.user.ID, target.ID, channel.ID, "deop", "demoted to user")
		m.notifyUserRefresh(target, "your role changed to user")
		m.appendSystemMsg("Demoted " + target.Username + " to user.")
	case "/kick":
		if !isAdmin {
			m.appendSystemMsg("Admin privileges required.")
			return nil
		}
		target, err := m.database.GetUserByUsername(arg)
		if err != nil || target.Role == "owner" {
			m.appendSystemMsg("Cannot kick this user.")
			return nil
		}
		if !m.broker.KickUser(target.Username) {
			m.appendSystemMsg("User is not online.")
			return nil
		}
		_ = m.database.CreateModerationLog(m.user.ID, target.ID, channel.ID, "kick", "kicked from live session")
		m.appendSystemMsg("Kicked " + target.Username)
	case "/ban":
		if !isAdmin {
			m.appendSystemMsg("Admin privileges required.")
			return nil
		}
		target, err := m.database.GetUserByUsername(arg)
		if err != nil || target.Role == "owner" {
			m.appendSystemMsg("Cannot ban this user.")
			return nil
		}
		if err := m.database.SetBanned(target.ID, true); err != nil {
			m.appendSystemMsg("Ban failed: " + err.Error())
			return nil
		}
		m.broker.KickUser(target.Username)
		_ = m.database.CreateModerationLog(m.user.ID, target.ID, channel.ID, "ban", "user banned")
		m.appendSystemMsg("Banned " + target.Username)
	case "/unban":
		if !isAdmin {
			m.appendSystemMsg("Admin privileges required.")
			return nil
		}
		target, err := m.database.GetUserByUsername(arg)
		if err != nil {
			m.appendSystemMsg("User not found.")
			return nil
		}
		if err := m.database.SetBanned(target.ID, false); err != nil {
			m.appendSystemMsg("Unban failed: " + err.Error())
			return nil
		}
		_ = m.database.CreateModerationLog(m.user.ID, target.ID, channel.ID, "unban", "user unbanned")
		m.appendSystemMsg("Unbanned " + target.Username)
	case "/newchan":
		if !isOwner {
			m.appendSystemMsg("Owner privileges required.")
			return nil
		}
		channelName, isPrivate := parseNewChannelArgs(arg)
		if channelName == "" {
			m.appendSystemMsg("Usage: /newchan [private] <name>")
			return nil
		}
		created, err := m.database.CreateChannel(channelName, isPrivate, m.user.ID)
		if err != nil {
			m.appendSystemMsg("Create channel failed: " + humanizeError(err))
			return nil
		}
		_ = m.database.CreateModerationLog(m.user.ID, "", created.ID, "newchan", created.Name)
		cmds = append(cmds, m.reloadAccessibleChannels(created.ID))
	case "/delchan":
		if !isOwner {
			m.appendSystemMsg("Owner privileges required.")
			return nil
		}
		target, err := m.database.GetChannelByName(arg)
		if err != nil {
			m.appendSystemMsg("Channel not found.")
			return nil
		}
		if len(m.database.GetChannels()) <= 1 {
			m.appendSystemMsg("Cannot delete the last remaining channel.")
			return nil
		}
		if err := m.database.DeleteChannel(target.ID); err != nil {
			m.appendSystemMsg("Delete channel failed: " + err.Error())
			return nil
		}
		_ = m.database.CreateModerationLog(m.user.ID, "", target.ID, "delchan", target.Name)
		cmds = append(cmds, m.reloadAccessibleChannels(""))
	case "/topic":
		if !isAdmin {
			m.appendSystemMsg("Admin privileges required.")
			return nil
		}
		if err := m.database.SetChannelTopic(channel.ID, arg); err != nil {
			m.appendSystemMsg("Topic update failed: " + err.Error())
			return nil
		}
		m.broker.Broadcast(channel.ID, db.Message{ID: "CMD_TOPIC", ChannelID: channel.ID, Content: arg})
	case "/del":
		if !isAdmin {
			m.appendSystemMsg("Admin privileges required.")
			return nil
		}
		count := 1
		if arg != "" {
			if parsed, err := strconv.Atoi(arg); err == nil && parsed > 0 {
				count = parsed
			}
		}
		if err := m.database.DeleteLastMessages(channel.ID, count); err != nil {
			m.appendSystemMsg("Delete failed: " + err.Error())
			return nil
		}
		m.broker.Broadcast(channel.ID, db.Message{ID: "CMD_DEL_LAST", ChannelID: channel.ID, Content: strconv.Itoa(count)})
	case "/setowner":
		if !isOwner {
			m.appendSystemMsg("Owner privileges required.")
			return nil
		}
		target, err := m.database.GetUserByUsername(arg)
		if err != nil {
			m.appendSystemMsg("User not found.")
			return nil
		}
		if err := m.database.SetUserRole(target.ID, "owner"); err != nil {
			m.appendSystemMsg("Ownership transfer failed: " + err.Error())
			return nil
		}
		if err := m.database.SetUserRole(m.user.ID, "admin"); err != nil {
			m.appendSystemMsg("Ownership transfer partially failed: " + err.Error())
			return nil
		}
		m.user.Role = "admin"
		_ = m.database.CreateModerationLog(m.user.ID, target.ID, channel.ID, "setowner", "ownership transferred")
		m.notifyUserRefresh(target, "you are now owner")
	default:
		if _, err := m.publishMessage(content, ""); err != nil {
			m.appendSystemMsg("Cannot send message: " + humanizeError(err))
		}
	}

	return cmds
}

func messageMentionsUser(msg db.Message, user *db.User) bool {
	if user == nil {
		return false
	}
	for _, mention := range msg.Mentions {
		if mention == strings.ToLower(user.Username) {
			return true
		}
	}
	return false
}

func (m *Model) updateViewportContent() {
	var b strings.Builder
	systemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	replyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
	mentionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("52")).Bold(true)

	for _, msg := range m.messages {
		timeStr := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(msg.CreatedAt.Format("15:04"))
		if msg.UserID == "system" {
			b.WriteString(systemStyle.Render(msg.Content) + "\n")
			continue
		}

		userStyle := lipgloss.NewStyle().Bold(true)
		color := msg.UserColor
		if msg.UserID == m.user.ID && m.user.Color != "" {
			color = m.user.Color
		}
		if color == "" {
			color = "#7C7C7C"
		}
		userStyle = userStyle.Foreground(lipgloss.Color(color))

		roleBadge := ""
		if msg.UserRole == "owner" {
			roleBadge = "👑 "
		} else if msg.UserRole == "admin" {
			roleBadge = "★ "
		}
		header := fmt.Sprintf("[%s] %s%s", timeStr, roleBadge, userStyle.Render(msg.Username))
		if msg.IsEdited {
			header += lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(" (edited)")
		}
		if messageMentionsUser(msg, m.user) {
			header = mentionStyle.Render(header)
		}

		content := msg.Content
		if m.renderer != nil && !strings.HasPrefix(content, "🖼️ ") {
			if rendered, err := m.renderer.Render(content); err == nil {
				content = strings.TrimSpace(rendered)
			}
		}

		if strings.HasPrefix(msg.Content, "🖼️ ") {
			url := strings.TrimPrefix(msg.Content, "🖼️ ")
			b.WriteString(header + "\n")
			if msg.ReplyToID != "" {
				b.WriteString(replyStyle.Render("  ↪ "+msg.ReplyToUsername+": "+snippet(msg.ReplyToContent)) + "\n")
			}
			if v, ok := globalImageCache.Load(url); ok {
				asset := v.(imageAsset)
				rendered := asset.ansi
				if rendered == "" {
					rendered = asset.kitty
				}
				if rendered == "" {
					b.WriteString("  ❌ Image failed to load\n")
				} else {
					b.WriteString(indentBlock(rendered, "  ") + "\n")
				}
			} else {
				b.WriteString("  ⏳ Loading image...\n")
			}
			continue
		}

		if msg.ReplyToID != "" || strings.Contains(content, "\n") {
			b.WriteString(header + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(shortMessageID(msg.ID)) + "\n")
			if msg.ReplyToID != "" {
				b.WriteString(replyStyle.Render("  ↪ "+msg.ReplyToUsername+": "+snippet(msg.ReplyToContent)) + "\n")
			}
			b.WriteString(indentBlock(content, "  ") + "\n")
			continue
		}
		b.WriteString(fmt.Sprintf("%s %s: %s\n", header, lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(shortMessageID(msg.ID)), content))
	}

	m.viewport.SetContent(strings.TrimRight(b.String(), "\n"))
	m.viewport.GotoBottom()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if current := m.activeChannel(); current != nil {
				m.unsubscribeChannel(current.ID)
			}
			return m, tea.Quit
		case tea.KeyEnter:
			if m.appState == bannedState {
				return m, tea.Quit
			}
			if m.appState == onboardingState {
				if err := m.database.SetVerified(m.user.ID); err == nil {
					m.user.IsVerified = true
					m.appState = mainState
					cmds = append(cmds, m.reloadAccessibleChannels(""))
				}
				return m, tea.Batch(cmds...)
			}

			content := strings.TrimSpace(m.input.Value())
			if content == "" {
				return m, nil
			}
			cmds = append(cmds, m.handleSubmit(content)...)
			m.input.Reset()
			return m, tea.Batch(cmds...)
		case tea.KeyTab:
			if m.appState != mainState {
				break
			}
			value := m.input.Value()
			if strings.HasPrefix(value, "/") && !strings.Contains(value, " ") {
				if m.tabPrefix == "" {
					m.tabPrefix = value
					m.tabMatchIdx = 0
				}
				var matches []string
				for _, command := range m.commandPalette {
					if strings.HasPrefix(command, m.tabPrefix) {
						matches = append(matches, command)
					}
				}
				if len(matches) > 0 {
					m.input.SetValue(matches[m.tabMatchIdx%len(matches)] + " ")
					m.input.CursorEnd()
					m.tabMatchIdx++
				}
				return m, nil
			}
			m.tabPrefix = ""
			if len(m.channels) > 0 {
				return m, m.switchToChannel((m.activeChan + 1) % len(m.channels))
			}
		case tea.KeyPgUp:
			if m.appState == mainState {
				m.loadOlderMessages()
			}
		case tea.KeyRunes:
			m.tabPrefix = ""
		case tea.KeyCtrlY:
			if m.appState == mainState && len(m.messages) > 0 {
				lastMsg := m.messages[len(m.messages)-1].Content
				encoded := base64.StdEncoding.EncodeToString([]byte(lastMsg))
				fmt.Fprintf(m.session, "\033]52;c;%s\a", encoded)
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = int(float64(m.width) * 0.8)
		m.viewport.Height = int(float64(m.height) * 0.8)
		m.onboarding.Width = int(float64(m.width) * 0.8)
		m.onboarding.Height = int(float64(m.height) * 0.8)
		m.input.SetWidth(int(float64(m.width) * 0.8))
		m.input.SetHeight(int(float64(m.height) * 0.2))
	case newMsgMsg:
		switch msg.ID {
		case "CMD_KICK":
			return m, tea.Quit
		case "CMD_CHANNELS":
			m.appendSystemMsg(msg.Content)
			return m, tea.Batch(m.reloadAccessibleChannels(""), m.waitForMessages())
		case "CMD_REFRESH_USER":
			m.syncCurrentUser()
			return m, tea.Batch(m.reloadAccessibleChannels(""), m.waitForMessages())
		case "CMD_DEL_LAST":
			count, _ := strconv.Atoi(msg.Content)
			if count <= 0 {
				count = 1
			}
			if count > len(m.messages) {
				count = len(m.messages)
			}
			if count > 0 {
				m.messages = m.messages[:len(m.messages)-count]
				m.loadedMessages = len(m.messages)
			}
			m.updateViewportContent()
			m.markActiveChannelRead("")
			return m, m.waitForMessages()
		case "CMD_CLEAR":
			m.messages = nil
			m.loadedMessages = 0
			m.updateViewportContent()
			return m, m.waitForMessages()
		case "CMD_TOPIC":
			for i, channel := range m.channels {
				if channel.ID == msg.ChannelID {
					m.channels[i].Topic = msg.Content
					break
				}
			}
			m.updateViewportContent()
			return m, m.waitForMessages()
		case "CMD_REFRESH":
			m.loadChannelMessages()
			return m, m.waitForMessages()
		}

		received := db.Message(msg)
		m.messages = append(m.messages, received)
		m.loadedMessages = len(m.messages)
		_ = m.database.TouchUserActivity(m.user.ID)
		if strings.HasPrefix(received.Content, "🖼️ ") {
			url := strings.TrimPrefix(received.Content, "🖼️ ")
			if _, ok := globalImageCache.Load(url); !ok {
				cmds = append(cmds, fetchImageCmd(url, m.currentWidth()))
			}
		}
		m.updateViewportContent()
		m.markActiveChannelRead(received.ID)
		cmds = append(cmds, m.waitForMessages())
		return m, tea.Batch(cmds...)
	case imageFetchedMsg:
		m.updateViewportContent()
		return m, nil
	}

	var cmd tea.Cmd
	if m.appState == onboardingState {
		m.onboarding, cmd = m.onboarding.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Initializing...\n"
	}
	if m.appState == bannedState {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("YOU ARE BANNED FROM THIS SERVER.\n\nPress any key to exit."))
	}
	if m.appState == onboardingState {
		card := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(m.user.Color)).
			Padding(1, 2).
			Width(min(90, m.width-4)).
			Render(m.onboarding.View())
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
	}
	if len(m.channels) == 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("No accessible channels.\nAsk an admin to invite you or create a public channel."))
	}

	leftW := int(float64(m.width) * 0.22)
	rightW := int(float64(m.width) * 0.22)
	centerW := m.width - leftW - rightW - 6
	if centerW < 16 {
		centerW = 16
	}

	headerH := 2
	inputH := 1
	if m.height >= 40 {
		inputH = 5
	} else if m.height >= 25 {
		inputH = 3
	}
	footerH := inputH + 2
	if m.height-headerH-footerH < 8 {
		inputH = 1
		footerH = 3
	}
	midH := m.height - headerH - footerH
	if midH < 5 {
		midH = 5
	}

	current := m.channels[m.activeChan]
	topic := current.Topic
	if topic == "" {
		topic = "No topic set"
	}
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color(m.user.Color)).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center).
		Render(fmt.Sprintf(" %s • %s • Topic: %s ", channelLabel(current), current.Kind, topic))

	var left strings.Builder
	left.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(m.user.Color)).Bold(true).Render("  CHANNELS"))
	left.WriteString("\n\n")
	for i, ch := range m.channels {
		label := channelLabel(ch)
		if unread := m.unread[ch.ID]; i != m.activeChan && unread.Count > 0 {
			label += fmt.Sprintf(" (%d", unread.Count)
			if unread.MentionCount > 0 {
				label += fmt.Sprintf(" @%d", unread.MentionCount)
			}
			label += ")"
		}
		if i == m.activeChan {
			left.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("  > " + label))
		} else {
			left.WriteString("    " + label)
		}
		left.WriteString("\n")
	}

	border := lipgloss.RoundedBorder()
	panel := lipgloss.NewStyle().Border(border).BorderForeground(lipgloss.Color(m.user.Color))
	leftPane := panel.Width(leftW).Height(midH).Render(left.String())

	var right strings.Builder
	right.WriteString("  ONLINE\n\n")
	for _, u := range m.broker.GetOnlineUsers(current.ID) {
		role := ""
		if u.Role == "owner" {
			role = "👑 "
		} else if u.Role == "admin" {
			role = "★ "
		}
		me := ""
		if u.ID == m.user.ID {
			me = " (you)"
		}
		right.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(u.Color)).Render("  • " + role + u.Username + me))
		right.WriteString("\n")
	}
	right.WriteString("\n  YOU\n\n")
	right.WriteString(fmt.Sprintf("  role: %s\n  bio: %s\n  seen: %s\n", m.user.Role, snippet(m.user.Bio), formatLastSeen(m.user.LastSeenAt)))
	if m.lastStatus != "" {
		right.WriteString("\n  STATUS\n\n")
		right.WriteString("  " + strings.ReplaceAll(snippet(m.lastStatus), "\n", " "))
	}
	rightPane := panel.Width(rightW).Height(midH).Render(right.String())

	m.viewport.Width = centerW
	m.viewport.Height = midH
	centerPane := panel.Width(centerW).Height(midH).Render(m.viewport.View())

	m.input.SetWidth(m.width - 2)
	m.input.SetHeight(inputH)
	bottomPane := panel.Width(m.width - 2).Height(inputH).Render(m.input.View())

	midSection := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, centerPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, header, midSection, bottomPane)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
