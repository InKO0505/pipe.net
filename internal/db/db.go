package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultUserColor   = "#6366F1"
	defaultMessagePage = 60
	maxUsernameLen     = 24
	maxChannelNameLen  = 32
	maxTopicLen        = 160
	maxBioLen          = 160
)

var (
	ErrUsernameTaken        = errors.New("username already taken")
	ErrInvalidUsername      = errors.New("invalid username")
	ErrInvalidColor         = errors.New("invalid color")
	ErrChannelExists        = errors.New("channel already exists")
	ErrInvalidChannelName   = errors.New("invalid channel name")
	ErrChannelNotAccessible = errors.New("channel not accessible")
	ErrEmptyMessage         = errors.New("message is empty")

	ansiRegexp     = regexp.MustCompile(`\x1b(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~]|\].*?(?:\x07|\x1b\\))`)
	controlRegexp  = regexp.MustCompile(`[\x00-\x08\x0b-\x1f\x7f]`)
	usernameRegexp = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,23}$`)
	channelRegexp  = regexp.MustCompile(`^#[a-z0-9][a-z0-9_-]{1,31}$`)
	hexColorRegexp = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	mentionRegexp  = regexp.MustCompile(`@([A-Za-z0-9][A-Za-z0-9_-]{1,23})`)
	namedColorMap  = map[string]string{
		"red":    "#ef4444",
		"orange": "#f97316",
		"yellow": "#eab308",
		"green":  "#22c55e",
		"teal":   "#14b8a6",
		"blue":   "#3b82f6",
		"indigo": "#6366f1",
		"pink":   "#ec4899",
		"gray":   "#9ca3af",
		"white":  "#f8fafc",
	}
)

type DB struct {
	*sql.DB
}

type User struct {
	ID         string
	SSHPubKey  string
	Username   string
	IsVerified bool
	CreatedAt  time.Time
	Role       string
	Color      string
	IsBanned   bool
	Bio        string
	LastSeenAt time.Time
}

type Channel struct {
	ID           string
	Name         string
	Topic        string
	CreatedAt    time.Time
	IsPrivate    bool
	Kind         string
	CreatedBy    string
	DMUserID     string
	MessageCount int
}

type Message struct {
	ID              string
	ChannelID       string
	UserID          string
	Content         string
	CreatedAt       time.Time
	Username        string
	UserColor       string
	UserRole        string
	ReplyToID       string
	ReplyToUsername string
	ReplyToContent  string
	EditedAt        time.Time
	IsEdited        bool
	Mentions        []string
}

type ModerationLog struct {
	ID             string
	ActorUserID    string
	ActorUsername  string
	TargetUserID   string
	TargetUsername string
	ChannelID      string
	ChannelName    string
	Action         string
	Details        string
	CreatedAt      time.Time
}

type UnreadInfo struct {
	Count         int
	LastMessageID string
	MentionCount  int
}

type MobileSession struct {
	ID         string
	UserID     string
	Token      string
	CreatedAt  time.Time
	LastUsedAt time.Time
}

func InitDB(filepath string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite3", filepath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&_loc=auto")
	if err != nil {
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	if err := initSchema(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	if err := migrateUsers(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	if err := seedChannels(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return &DB{sqlDB}, nil
}

func initSchema(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			ssh_pub_key TEXT UNIQUE,
			username TEXT,
			username_normalized TEXT DEFAULT '',
			is_verified BOOLEAN,
			created_at DATETIME,
			role TEXT DEFAULT 'user',
			color TEXT DEFAULT '',
			is_banned BOOLEAN DEFAULT 0,
			bio TEXT DEFAULT '',
			last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS channels (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE,
			topic TEXT DEFAULT '',
			created_at DATETIME,
			is_private BOOLEAN DEFAULT 0,
			kind TEXT DEFAULT 'channel',
			created_by TEXT DEFAULT '',
			dm_user_1 TEXT DEFAULT '',
			dm_user_2 TEXT DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			channel_id TEXT,
			user_id TEXT,
			content TEXT,
			reply_to_id TEXT DEFAULT '',
			edited_at DATETIME DEFAULT NULL,
			is_deleted BOOLEAN DEFAULT 0,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS channel_members (
			channel_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (channel_id, user_id)
		);`,
		`CREATE TABLE IF NOT EXISTS channel_reads (
			channel_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			last_read_at DATETIME NOT NULL,
			last_read_message_id TEXT DEFAULT '',
			PRIMARY KEY (channel_id, user_id)
		);`,
		`CREATE TABLE IF NOT EXISTS moderation_logs (
			id TEXT PRIMARY KEY,
			actor_user_id TEXT NOT NULL,
			target_user_id TEXT DEFAULT '',
			channel_id TEXT DEFAULT '',
			action TEXT NOT NULL,
			details TEXT DEFAULT '',
			created_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS mobile_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL,
			last_used_at DATETIME NOT NULL
		);`,
	}

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	migrations := []struct {
		table  string
		column string
		def    string
	}{
		{"users", "role", "TEXT DEFAULT 'user'"},
		{"users", "color", "TEXT DEFAULT ''"},
		{"users", "is_banned", "BOOLEAN DEFAULT 0"},
		{"users", "username_normalized", "TEXT DEFAULT ''"},
		{"users", "bio", "TEXT DEFAULT ''"},
		{"users", "last_seen_at", "DATETIME DEFAULT ''"},
		{"channels", "topic", "TEXT DEFAULT ''"},
		{"channels", "is_private", "BOOLEAN DEFAULT 0"},
		{"channels", "kind", "TEXT DEFAULT 'channel'"},
		{"channels", "created_by", "TEXT DEFAULT ''"},
		{"channels", "dm_user_1", "TEXT DEFAULT ''"},
		{"channels", "dm_user_2", "TEXT DEFAULT ''"},
		{"messages", "reply_to_id", "TEXT DEFAULT ''"},
		{"messages", "edited_at", "DATETIME DEFAULT NULL"},
		{"messages", "is_deleted", "BOOLEAN DEFAULT 0"},
	}

	for _, migration := range migrations {
		if err := ensureColumn(db, migration.table, migration.column, migration.def); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_normalized_unique
		ON users(username_normalized)
		WHERE username_normalized != ''`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		UPDATE users
		SET last_seen_at = COALESCE(NULLIF(last_seen_at, ''), created_at)
		WHERE COALESCE(last_seen_at, '') = ''
	`); err != nil {
		return err
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_messages_channel_created_at ON messages(channel_id, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_reply_to_id ON messages(reply_to_id);`,
		`CREATE INDEX IF NOT EXISTS idx_channel_members_user_id ON channel_members(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_moderation_logs_created_at ON moderation_logs(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_users_last_seen_at ON users(last_seen_at);`,
		`CREATE INDEX IF NOT EXISTS idx_channels_kind ON channels(kind);`,
		`CREATE INDEX IF NOT EXISTS idx_mobile_sessions_user_id ON mobile_sessions(user_id);`,
	}

	for _, stmt := range indexes {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func ensureColumn(db *sql.DB, table, column, def string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			typ        string
			notNull    int
			defaultV   sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultV, &primaryKey); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, def))
	return err
}

func migrateUsers(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, username, COALESCE(username_normalized, ''), COALESCE(color, ''), COALESCE(role, ''), COALESCE(is_banned, 0), COALESCE(bio, '')
		FROM users
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type userRow struct {
		id         string
		username   string
		normalized string
		color      string
		role       string
		isBanned   bool
		bio        string
	}

	var users []userRow
	for rows.Next() {
		var item userRow
		if err := rows.Scan(&item.id, &item.username, &item.normalized, &item.color, &item.role, &item.isBanned, &item.bio); err != nil {
			return err
		}
		users = append(users, item)
	}

	taken := make(map[string]struct{}, len(users))
	for _, item := range users {
		username := sanitizeSingleLine(item.username, maxUsernameLen)
		if !usernameRegexp.MatchString(username) {
			username = fmt.Sprintf("anon_%s", item.id[:4])
		}
		username = makeUniqueUsername(username, taken)
		normalized := normalizeUsername(username)
		taken[normalized] = struct{}{}

		role := item.role
		if role == "" {
			role = "user"
		}

		color := item.color
		if _, err := normalizeColor(color); err != nil {
			color = defaultUserColor
		}

		if _, err := db.Exec(`
			UPDATE users
			SET username = ?, username_normalized = ?, role = ?, color = ?, is_banned = ?, bio = ?
			WHERE id = ?
		`, username, normalized, role, color, item.isBanned, sanitizeSingleLine(item.bio, maxBioLen), item.id); err != nil {
			return err
		}
	}

	return nil
}

func seedChannels(db *sql.DB) error {
	type defaultChannel struct {
		name  string
		topic string
	}

	channels := []defaultChannel{
		{name: "#general", topic: "Welcome to the main channel! Be polite."},
		{name: "#linux", topic: "All about the penguin OS"},
		{name: "#bash-magic", topic: "Shell scripting, grep, awk and other wizardry"},
	}

	for _, ch := range channels {
		if _, err := db.Exec(`
			INSERT OR IGNORE INTO channels (id, name, topic, created_at, is_private)
			VALUES (?, ?, ?, ?, 0)
		`, uuid.New().String(), ch.name, ch.topic, time.Now()); err != nil {
			return err
		}
		if _, err := db.Exec(`
			UPDATE channels
			SET topic = ?
			WHERE name = ? AND (topic = '' OR topic IS NULL)
		`, ch.topic, ch.name); err != nil {
			return err
		}
	}

	return nil
}

func normalizeUsername(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func sanitizeSingleLine(input string, limit int) string {
	clean := stripTerminalSequences(input)
	clean = strings.ReplaceAll(clean, "\n", " ")
	clean = strings.ReplaceAll(clean, "\r", " ")
	clean = strings.ReplaceAll(clean, "\t", " ")
	clean = strings.Join(strings.Fields(clean), " ")
	if limit > 0 && len(clean) > limit {
		clean = clean[:limit]
	}
	return clean
}

func SanitizeMessageContent(input string) string {
	clean := stripTerminalSequences(input)
	lines := strings.Split(clean, "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		lines[i] = line
	}
	clean = strings.TrimSpace(strings.Join(lines, "\n"))
	if len(clean) > 2000 {
		clean = clean[:2000]
	}
	return clean
}

func extractMentions(input string) []string {
	matches := mentionRegexp.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	var mentions []string
	for _, match := range matches {
		name := normalizeUsername(match[1])
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		mentions = append(mentions, name)
	}
	return mentions
}

func stripTerminalSequences(input string) string {
	clean := ansiRegexp.ReplaceAllString(input, "")
	clean = controlRegexp.ReplaceAllStringFunc(clean, func(_ string) string {
		return ""
	})
	return clean
}

func makeUniqueUsername(candidate string, taken map[string]struct{}) string {
	base := sanitizeSingleLine(candidate, maxUsernameLen)
	if !usernameRegexp.MatchString(base) {
		base = fmt.Sprintf("anon_%s", uuid.New().String()[:4])
	}

	current := base
	for idx := 2; ; idx++ {
		key := normalizeUsername(current)
		if _, exists := taken[key]; !exists {
			return current
		}

		suffix := "_" + strconv.Itoa(idx)
		trimLen := maxUsernameLen - len(suffix)
		if trimLen < 2 {
			trimLen = 2
		}
		if len(base) > trimLen {
			current = base[:trimLen] + suffix
		} else {
			current = base + suffix
		}
	}
}

func normalizeChannelName(name string) (string, error) {
	clean := sanitizeSingleLine(strings.ToLower(name), maxChannelNameLen)
	if clean == "" {
		return "", ErrInvalidChannelName
	}
	if !strings.HasPrefix(clean, "#") {
		clean = "#" + clean
	}
	if !channelRegexp.MatchString(clean) {
		return "", ErrInvalidChannelName
	}
	return clean, nil
}

func normalizeColor(input string) (string, error) {
	color := strings.ToLower(strings.TrimSpace(input))
	if color == "" {
		return defaultUserColor, nil
	}
	if hexColorRegexp.MatchString(color) {
		return strings.ToUpper(color), nil
	}
	if value, ok := namedColorMap[color]; ok {
		return strings.ToUpper(value), nil
	}
	return "", ErrInvalidColor
}

func normalizePubKey(pubKey string) string {
	return strings.TrimSpace(pubKey)
}

type userScanner interface {
	Scan(dest ...any) error
}

var sqliteTimeFormats = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	time.RFC3339Nano,
	time.RFC3339,
}

func parseSQLiteTime(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		for _, layout := range sqliteTimeFormats {
			if parsed, err := time.Parse(layout, v); err == nil {
				return parsed, nil
			}
		}
		return time.Time{}, fmt.Errorf("unsupported sqlite time format %q", v)
	case []byte:
		return parseSQLiteTime(string(v))
	default:
		return time.Time{}, fmt.Errorf("unsupported sqlite time type %T", value)
	}
}

func scanUser(scanner userScanner) (*User, error) {
	var (
		user       User
		createdAt  any
		lastSeenAt any
	)

	if err := scanner.Scan(&user.ID, &user.SSHPubKey, &user.Username, &user.IsVerified, &createdAt, &user.Role, &user.Color, &user.IsBanned, &user.Bio, &lastSeenAt); err != nil {
		return nil, err
	}

	var err error
	user.CreatedAt, err = parseSQLiteTime(createdAt)
	if err != nil {
		return nil, err
	}
	user.LastSeenAt, err = parseSQLiteTime(lastSeenAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (db *DB) GetUserByPubKey(pubKey string) (*User, error) {
	pubKey = normalizePubKey(pubKey)
	row := db.QueryRow(`
		SELECT id, ssh_pub_key, username, is_verified, created_at, role, color, is_banned, COALESCE(bio, ''), COALESCE(last_seen_at, created_at)
		FROM users
		WHERE TRIM(ssh_pub_key) = ?
	`, pubKey)

	return scanUser(row)
}

func (db *DB) GetUserByUsername(username string) (*User, error) {
	row := db.QueryRow(`
		SELECT id, ssh_pub_key, username, is_verified, created_at, role, color, is_banned, COALESCE(bio, ''), COALESCE(last_seen_at, created_at)
		FROM users
		WHERE username_normalized = ?
	`, normalizeUsername(username))

	return scanUser(row)
}

func (db *DB) GetUserByID(userID string) (*User, error) {
	row := db.QueryRow(`
		SELECT id, ssh_pub_key, username, is_verified, created_at, role, color, is_banned, COALESCE(bio, ''), COALESCE(last_seen_at, created_at)
		FROM users
		WHERE id = ?
	`, userID)

	return scanUser(row)
}

func (db *DB) CreateUser(pubKey string) *User {
	pubKey = normalizePubKey(pubKey)
	if existing, err := db.GetUserByPubKey(pubKey); err == nil {
		return existing
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)

	role := "user"
	if count == 0 {
		role = "owner"
	}

	for attempt := 0; attempt < 8; attempt++ {
		user := &User{
			ID:         uuid.New().String(),
			SSHPubKey:  pubKey,
			Username:   fmt.Sprintf("anon_%s", uuid.New().String()[:8]),
			IsVerified: false,
			CreatedAt:  time.Now(),
			Role:       role,
			Color:      defaultUserColor,
			IsBanned:   false,
			Bio:        "",
			LastSeenAt: time.Now(),
		}

		_, err := db.Exec(`
			INSERT INTO users (id, ssh_pub_key, username, username_normalized, is_verified, created_at, role, color, is_banned, bio, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, user.ID, user.SSHPubKey, user.Username, normalizeUsername(user.Username), user.IsVerified, user.CreatedAt, user.Role, user.Color, user.IsBanned, user.Bio, user.LastSeenAt)
		if err == nil {
			if joinErr := db.EnsurePublicChannelMemberships(user.ID); joinErr != nil {
				log.Printf("error joining public channels for %q: %v", user.Username, joinErr)
			}
			return user
		}

		errText := strings.ToLower(err.Error())
		if strings.Contains(errText, "ssh_pub_key") {
			existing, lookupErr := db.GetUserByPubKey(pubKey)
			if lookupErr == nil {
				return existing
			}
			log.Printf("error resolving existing user after ssh key conflict: %v", lookupErr)
			return user
		}

		if strings.Contains(errText, "username") || strings.Contains(errText, "username_normalized") {
			continue
		}

		log.Printf("error creating user: %v", err)
		return user
	}

	user := &User{
		ID:         uuid.New().String(),
		SSHPubKey:  pubKey,
		Username:   fmt.Sprintf("anon_%s", uuid.New().String()[:12]),
		IsVerified: false,
		CreatedAt:  time.Now(),
		Role:       role,
		Color:      defaultUserColor,
		IsBanned:   false,
		Bio:        "",
		LastSeenAt: time.Now(),
	}
	log.Printf("error creating user: exhausted username retries for %q", pubKey)
	return user
}

func (db *DB) EnsurePublicChannelMemberships(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}

	_, err := db.Exec(`
		INSERT OR IGNORE INTO channel_members (channel_id, user_id, created_at)
		SELECT id, ?, ?
		FROM channels
		WHERE is_private = 0 AND COALESCE(kind, 'channel') != 'dm'
	`, userID, time.Now())
	return err
}

func (db *DB) CreateMobileUser(username string) (*User, error) {
	username = sanitizeSingleLine(username, maxUsernameLen)
	if !usernameRegexp.MatchString(username) {
		return nil, ErrInvalidUsername
	}

	if existing, err := db.GetUserByUsername(username); err == nil {
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return nil, err
	}

	role := "user"
	if count == 0 {
		role = "owner"
	}

	user := &User{
		ID:         uuid.New().String(),
		SSHPubKey:  fmt.Sprintf("mobile:%s:%s", normalizeUsername(username), uuid.NewString()),
		Username:   username,
		IsVerified: false,
		CreatedAt:  time.Now(),
		Role:       role,
		Color:      defaultUserColor,
		IsBanned:   false,
		Bio:        "",
		LastSeenAt: time.Now(),
	}

	_, err := db.Exec(`
		INSERT INTO users (id, ssh_pub_key, username, username_normalized, is_verified, created_at, role, color, is_banned, bio, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.SSHPubKey, user.Username, normalizeUsername(user.Username), user.IsVerified, user.CreatedAt, user.Role, user.Color, user.IsBanned, user.Bio, user.LastSeenAt)
	if err == nil {
		if err := db.EnsurePublicChannelMemberships(user.ID); err != nil {
			return nil, err
		}
		return user, nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "username") {
		return db.GetUserByUsername(username)
	}

	return nil, err
}

func (db *DB) SetVerified(userID string) error {
	_, err := db.Exec("UPDATE users SET is_verified = 1 WHERE id = ?", userID)
	return err
}

func (db *DB) UpdateUsername(userID, newName string) error {
	username := sanitizeSingleLine(newName, maxUsernameLen)
	if !usernameRegexp.MatchString(username) {
		return ErrInvalidUsername
	}

	existing, err := db.GetUserByUsername(username)
	if err == nil && existing.ID != userID {
		return ErrUsernameTaken
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err = db.Exec(`
		UPDATE users
		SET username = ?, username_normalized = ?
		WHERE id = ?
	`, username, normalizeUsername(username), userID)
	return err
}

func (db *DB) UpdateUserColor(userID, color string) (string, error) {
	normalized, err := normalizeColor(color)
	if err != nil {
		return "", err
	}
	_, err = db.Exec("UPDATE users SET color = ? WHERE id = ?", normalized, userID)
	return normalized, err
}

func (db *DB) UpdateUserBio(userID, bio string) error {
	_, err := db.Exec("UPDATE users SET bio = ? WHERE id = ?", sanitizeSingleLine(bio, maxBioLen), userID)
	return err
}

func (db *DB) TouchUserActivity(userID string) error {
	if userID == "" {
		return nil
	}
	_, err := db.Exec("UPDATE users SET last_seen_at = ? WHERE id = ?", time.Now(), userID)
	return err
}

func (db *DB) GetUsers() []User {
	rows, err := db.Query(`
		SELECT id, ssh_pub_key, username, is_verified, created_at, role, color, is_banned, COALESCE(bio, ''), COALESCE(last_seen_at, created_at)
		FROM users
		ORDER BY username_normalized ASC
	`)
	if err != nil {
		log.Printf("error fetching users: %v", err)
		return nil
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			log.Printf("error scanning user: %v", err)
			return nil
		}
		users = append(users, *user)
	}
	return users
}

func (db *DB) CreateMobileSession(userID string) (*MobileSession, error) {
	session := &MobileSession{
		ID:         uuid.New().String(),
		UserID:     userID,
		Token:      uuid.NewString() + uuid.NewString(),
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	if _, err := db.Exec(`
		INSERT INTO mobile_sessions (id, user_id, token, created_at, last_used_at)
		VALUES (?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.Token, session.CreatedAt, session.LastUsedAt); err != nil {
		return nil, err
	}
	return session, nil
}

func (db *DB) GetUserByMobileToken(token string) (*User, error) {
	row := db.QueryRow(`
		SELECT u.id, u.ssh_pub_key, u.username, u.is_verified, u.created_at, u.role, u.color, u.is_banned, COALESCE(u.bio, ''), COALESCE(u.last_seen_at, u.created_at)
		FROM mobile_sessions ms
		JOIN users u ON u.id = ms.user_id
		WHERE ms.token = ?
	`, strings.TrimSpace(token))

	user, err := scanUser(row)
	if err != nil {
		return nil, err
	}
	_, _ = db.Exec("UPDATE mobile_sessions SET last_used_at = ? WHERE token = ?", time.Now(), strings.TrimSpace(token))
	return user, nil
}

func (db *DB) SetUserRole(userID, role string) error {
	_, err := db.Exec("UPDATE users SET role = ? WHERE id = ?", role, userID)
	return err
}

func (db *DB) SetBanned(userID string, banned bool) error {
	_, err := db.Exec("UPDATE users SET is_banned = ? WHERE id = ?", banned, userID)
	return err
}

func (db *DB) GetChannels() []Channel {
	rows, err := db.Query(`
		SELECT id, name, topic, created_at, is_private, COALESCE(kind, 'channel'), COALESCE(created_by, ''), '', 
			(SELECT COUNT(*) FROM messages m WHERE m.channel_id = channels.id AND COALESCE(m.is_deleted, 0) = 0)
		FROM channels
		ORDER BY CASE name
			WHEN '#general' THEN 0
			WHEN '#linux' THEN 1
			WHEN '#bash-magic' THEN 2
			ELSE 3
		END, name
	`)
	if err != nil {
		log.Printf("error fetching channels: %v", err)
		return nil
	}
	defer rows.Close()

	return scanChannels(rows)
}

func (db *DB) GetAccessibleChannels(user *User) []Channel {
	if user == nil {
		return nil
	}
	if user.Role == "owner" || user.Role == "admin" {
		return db.GetChannels()
	}

	rows, err := db.Query(`
		SELECT
			c.id,
			c.name,
			c.topic,
			c.created_at,
			c.is_private,
			COALESCE(c.kind, 'channel'),
			COALESCE(c.created_by, ''),
			CASE
				WHEN COALESCE(c.kind, 'channel') = 'dm' AND c.dm_user_1 = ? THEN c.dm_user_2
				WHEN COALESCE(c.kind, 'channel') = 'dm' THEN c.dm_user_1
				ELSE ''
			END AS dm_user_id,
			(SELECT COUNT(*) FROM messages m WHERE m.channel_id = c.id AND COALESCE(m.is_deleted, 0) = 0) AS message_count
		FROM channels c
		LEFT JOIN channel_members cm
			ON cm.channel_id = c.id AND cm.user_id = ?
		WHERE c.is_private = 0 OR cm.user_id IS NOT NULL
		ORDER BY CASE c.name
			WHEN '#general' THEN 0
			WHEN '#linux' THEN 1
			WHEN '#bash-magic' THEN 2
			ELSE 3
		END, c.name
	`, user.ID, user.ID)
	if err != nil {
		log.Printf("error fetching accessible channels: %v", err)
		return nil
	}
	defer rows.Close()

	return scanChannels(rows)
}

func scanChannels(rows *sql.Rows) []Channel {
	var channels []Channel
	for rows.Next() {
		var c Channel
		if err := rows.Scan(&c.ID, &c.Name, &c.Topic, &c.CreatedAt, &c.IsPrivate, &c.Kind, &c.CreatedBy, &c.DMUserID, &c.MessageCount); err != nil {
			log.Printf("error scanning channel: %v", err)
			return nil
		}
		channels = append(channels, c)
	}
	return channels
}

func (db *DB) CreateChannel(name string, isPrivate bool, creatorID string) (*Channel, error) {
	normalizedName, err := normalizeChannelName(name)
	if err != nil {
		return nil, err
	}

	channel := &Channel{
		ID:        uuid.New().String(),
		Name:      normalizedName,
		Topic:     "",
		CreatedAt: time.Now(),
		IsPrivate: isPrivate,
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
			INSERT INTO channels (id, name, topic, created_at, is_private, kind, created_by)
			VALUES (?, ?, ?, ?, ?, 'channel', ?)
	`, channel.ID, channel.Name, channel.Topic, channel.CreatedAt, channel.IsPrivate, creatorID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrChannelExists
		}
		return nil, err
	}

	if creatorID != "" {
		if _, err := tx.Exec(`
			INSERT OR IGNORE INTO channel_members (channel_id, user_id, created_at)
			VALUES (?, ?, ?)
		`, channel.ID, creatorID, time.Now()); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return channel, nil
}

func (db *DB) AddChannelMember(channelID, userID string) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO channel_members (channel_id, user_id, created_at)
		VALUES (?, ?, ?)
	`, channelID, userID, time.Now())
	return err
}

func (db *DB) RemoveChannelMember(channelID, userID string) error {
	_, err := db.Exec(`
		DELETE FROM channel_members
		WHERE channel_id = ? AND user_id = ?
	`, channelID, userID)
	return err
}

func makeDMName(userA, userB string) string {
	names := []string{sanitizeSingleLine(userA, maxUsernameLen), sanitizeSingleLine(userB, maxUsernameLen)}
	if strings.ToLower(names[0]) > strings.ToLower(names[1]) {
		names[0], names[1] = names[1], names[0]
	}
	return "@" + names[0] + "+" + names[1]
}

func (db *DB) GetOrCreateDirectChannel(userA, userB *User) (*Channel, error) {
	if userA == nil || userB == nil {
		return nil, ErrChannelNotAccessible
	}
	if userA.ID == userB.ID {
		return nil, fmt.Errorf("cannot create DM with yourself")
	}

	row := db.QueryRow(`
		SELECT id, name, topic, created_at, is_private, COALESCE(kind, 'channel'), COALESCE(created_by, ''),
			CASE WHEN dm_user_1 = ? THEN dm_user_2 ELSE dm_user_1 END,
			(SELECT COUNT(*) FROM messages m WHERE m.channel_id = channels.id AND COALESCE(m.is_deleted, 0) = 0)
		FROM channels
		WHERE COALESCE(kind, 'channel') = 'dm'
			AND ((dm_user_1 = ? AND dm_user_2 = ?) OR (dm_user_1 = ? AND dm_user_2 = ?))
		LIMIT 1
	`, userA.ID, userA.ID, userB.ID, userB.ID, userA.ID)

	var channel Channel
	if err := row.Scan(&channel.ID, &channel.Name, &channel.Topic, &channel.CreatedAt, &channel.IsPrivate, &channel.Kind, &channel.CreatedBy, &channel.DMUserID, &channel.MessageCount); err == nil {
		return &channel, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	channel = Channel{
		ID:        uuid.New().String(),
		Name:      makeDMName(userA.Username, userB.Username),
		Topic:     "Direct messages",
		CreatedAt: time.Now(),
		IsPrivate: true,
		Kind:      "dm",
		CreatedBy: userA.ID,
		DMUserID:  userB.ID,
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO channels (id, name, topic, created_at, is_private, kind, created_by, dm_user_1, dm_user_2)
		VALUES (?, ?, ?, ?, 1, 'dm', ?, ?, ?)
	`, channel.ID, channel.Name, channel.Topic, channel.CreatedAt, userA.ID, userA.ID, userB.ID); err != nil {
		return nil, err
	}

	for _, userID := range []string{userA.ID, userB.ID} {
		if _, err := tx.Exec(`
			INSERT OR IGNORE INTO channel_members (channel_id, user_id, created_at)
			VALUES (?, ?, ?)
		`, channel.ID, userID, time.Now()); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &channel, nil
}

func (db *DB) GetChannelMembers(channelID string) []User {
	rows, err := db.Query(`
		SELECT u.id, u.ssh_pub_key, u.username, u.is_verified, u.created_at, u.role, u.color, u.is_banned, COALESCE(u.bio, ''), COALESCE(u.last_seen_at, u.created_at)
		FROM channel_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.channel_id = ?
		ORDER BY u.username_normalized ASC
	`, channelID)
	if err != nil {
		log.Printf("error fetching channel members: %v", err)
		return nil
	}
	defer rows.Close()

	var members []User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			log.Printf("error scanning channel member: %v", err)
			return nil
		}
		members = append(members, *user)
	}
	return members
}

func (db *DB) CanAccessChannel(channelID string, user *User) bool {
	if user == nil {
		return false
	}
	if user.Role == "owner" || user.Role == "admin" {
		return true
	}

	var isPrivate bool
	err := db.QueryRow("SELECT is_private FROM channels WHERE id = ?", channelID).Scan(&isPrivate)
	if err != nil {
		return false
	}
	if !isPrivate {
		return true
	}

	var count int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM channel_members
		WHERE channel_id = ? AND user_id = ?
	`, channelID, user.ID).Scan(&count)
	return err == nil && count > 0
}

func (db *DB) SetChannelTopic(channelID, topic string) error {
	clean := sanitizeSingleLine(topic, maxTopicLen)
	_, err := db.Exec("UPDATE channels SET topic = ? WHERE id = ?", clean, channelID)
	return err
}

func (db *DB) GetChannelByName(name string) (*Channel, error) {
	normalizedName, err := normalizeChannelName(name)
	if err != nil {
		return nil, err
	}

	row := db.QueryRow(`
		SELECT id, name, topic, created_at, is_private, COALESCE(kind, 'channel'), COALESCE(created_by, ''), '',
			(SELECT COUNT(*) FROM messages m WHERE m.channel_id = channels.id AND COALESCE(m.is_deleted, 0) = 0)
		FROM channels
		WHERE name = ?
	`, normalizedName)
	var channel Channel
	if err := row.Scan(&channel.ID, &channel.Name, &channel.Topic, &channel.CreatedAt, &channel.IsPrivate, &channel.Kind, &channel.CreatedBy, &channel.DMUserID, &channel.MessageCount); err != nil {
		return nil, err
	}
	return &channel, nil
}

func (db *DB) GetChannelForUser(name string, user *User) (*Channel, error) {
	channel, err := db.GetChannelByName(name)
	if err != nil {
		return nil, err
	}
	if !db.CanAccessChannel(channel.ID, user) {
		return nil, ErrChannelNotAccessible
	}
	return channel, nil
}

func (db *DB) GetChannelByID(channelID string, user *User) (*Channel, error) {
	row := db.QueryRow(`
		SELECT id, name, topic, created_at, is_private, COALESCE(kind, 'channel'), COALESCE(created_by, ''),
			CASE
				WHEN COALESCE(kind, 'channel') = 'dm' AND dm_user_1 = ? THEN dm_user_2
				WHEN COALESCE(kind, 'channel') = 'dm' THEN dm_user_1
				ELSE ''
			END,
			(SELECT COUNT(*) FROM messages m WHERE m.channel_id = channels.id AND COALESCE(m.is_deleted, 0) = 0)
		FROM channels
		WHERE id = ?
	`, user.ID, channelID)
	var channel Channel
	if err := row.Scan(&channel.ID, &channel.Name, &channel.Topic, &channel.CreatedAt, &channel.IsPrivate, &channel.Kind, &channel.CreatedBy, &channel.DMUserID, &channel.MessageCount); err != nil {
		return nil, err
	}
	if !db.CanAccessChannel(channel.ID, user) {
		return nil, ErrChannelNotAccessible
	}
	return &channel, nil
}

func (db *DB) GetMessages(channelID string) []Message {
	return db.GetMessagesPage(channelID, 200, 0)
}

func (db *DB) GetMessagesPage(channelID string, limit, offset int) []Message {
	if limit <= 0 {
		limit = defaultMessagePage
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := db.Query(`
		SELECT
			id,
			channel_id,
			user_id,
			content,
			created_at,
			COALESCE(edited_at, created_at) AS edited_at,
			COALESCE(is_deleted, 0) AS is_deleted,
			username,
			user_color,
			user_role,
			reply_to_id,
			reply_to_username,
			reply_to_content
		FROM (
			SELECT
				m.id,
				m.channel_id,
				m.user_id,
				m.content,
				CASE WHEN COALESCE(m.is_deleted, 0) = 1 THEN '[deleted]' ELSE m.content END AS content,
				m.created_at,
				COALESCE(m.edited_at, m.created_at) AS edited_at,
				COALESCE(m.is_deleted, 0) AS is_deleted,
				u.username,
				u.color AS user_color,
				u.role AS user_role,
				COALESCE(m.reply_to_id, '') AS reply_to_id,
				COALESCE(ru.username, '') AS reply_to_username,
				COALESCE(rm.content, '') AS reply_to_content
			FROM messages m
			JOIN users u ON m.user_id = u.id
			LEFT JOIN messages rm ON m.reply_to_id = rm.id
			LEFT JOIN users ru ON rm.user_id = ru.id
			WHERE m.channel_id = ?
			ORDER BY m.created_at DESC
			LIMIT ? OFFSET ?
		)
		ORDER BY created_at ASC
	`, channelID, limit, offset)
	if err != nil {
		log.Printf("error fetching messages: %v", err)
		return nil
	}
	defer rows.Close()

	return scanMessages(rows)
}

func (db *DB) SearchMessages(channelID, term string, limit int) []Message {
	search := "%" + sanitizeSingleLine(term, 120) + "%"
	if strings.Trim(search, "%") == "" {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}

	rows, err := db.Query(`
		SELECT
			m.id,
			m.channel_id,
			m.user_id,
			CASE WHEN COALESCE(m.is_deleted, 0) = 1 THEN '[deleted]' ELSE m.content END,
			m.created_at,
			COALESCE(m.edited_at, m.created_at),
			COALESCE(m.is_deleted, 0),
			u.username,
			u.color,
			u.role,
			COALESCE(m.reply_to_id, ''),
			COALESCE(ru.username, ''),
			COALESCE(rm.content, '')
		FROM messages m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN messages rm ON m.reply_to_id = rm.id
		LEFT JOIN users ru ON rm.user_id = ru.id
		WHERE m.channel_id = ?
			AND ((CASE WHEN COALESCE(m.is_deleted, 0) = 1 THEN '' ELSE m.content END) LIKE ? OR u.username LIKE ?)
		ORDER BY m.created_at DESC
		LIMIT ?
	`, channelID, search, search, limit)
	if err != nil {
		log.Printf("error searching messages: %v", err)
		return nil
	}
	defer rows.Close()

	return scanMessages(rows)
}

func scanMessages(rows *sql.Rows) []Message {
	var messages []Message
	for rows.Next() {
		var message Message
		var editedAtRaw sql.NullString
		var isDeleted bool
		if err := rows.Scan(
			&message.ID,
			&message.ChannelID,
			&message.UserID,
			&message.Content,
			&message.CreatedAt,
			&editedAtRaw,
			&isDeleted,
			&message.Username,
			&message.UserColor,
			&message.UserRole,
			&message.ReplyToID,
			&message.ReplyToUsername,
			&message.ReplyToContent,
		); err != nil {
			log.Printf("error scanning message: %v", err)
			return nil
		}
		message.EditedAt = message.CreatedAt
		if editedAtRaw.Valid {
			if parsed, err := parseSQLiteTime(editedAtRaw.String); err == nil {
				message.EditedAt = parsed
			}
		}
		message.IsEdited = !isDeleted && !message.EditedAt.Equal(message.CreatedAt)
		message.Mentions = extractMentions(message.Content)
		messages = append(messages, message)
	}
	return messages
}

func (db *DB) GetMessageByID(channelID, messageID string) (*Message, error) {
	rows, err := db.Query(`
		SELECT
			m.id,
			m.channel_id,
			m.user_id,
			CASE WHEN COALESCE(m.is_deleted, 0) = 1 THEN '[deleted]' ELSE m.content END,
			m.created_at,
			COALESCE(m.edited_at, m.created_at),
			COALESCE(m.is_deleted, 0),
			u.username,
			u.color,
			u.role,
			COALESCE(m.reply_to_id, ''),
			COALESCE(ru.username, ''),
			COALESCE(rm.content, '')
		FROM messages m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN messages rm ON m.reply_to_id = rm.id
		LEFT JOIN users ru ON rm.user_id = ru.id
		WHERE m.channel_id = ? AND m.id = ?
	`, channelID, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := scanMessages(rows)
	if len(messages) == 0 {
		return nil, sql.ErrNoRows
	}
	return &messages[0], nil
}

func (db *DB) FindMessageByPrefix(channelID, prefix string) (*Message, error) {
	clean := sanitizeSingleLine(prefix, 12)
	rows, err := db.Query(`
		SELECT
			m.id,
			m.channel_id,
			m.user_id,
			CASE WHEN COALESCE(m.is_deleted, 0) = 1 THEN '[deleted]' ELSE m.content END,
			m.created_at,
			COALESCE(m.edited_at, m.created_at),
			COALESCE(m.is_deleted, 0),
			u.username,
			u.color,
			u.role,
			COALESCE(m.reply_to_id, ''),
			COALESCE(ru.username, ''),
			COALESCE(rm.content, '')
		FROM messages m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN messages rm ON m.reply_to_id = rm.id
		LEFT JOIN users ru ON rm.user_id = ru.id
		WHERE m.channel_id = ? AND m.id LIKE ?
		ORDER BY m.created_at DESC
		LIMIT 2
	`, channelID, clean+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := scanMessages(rows)
	if len(messages) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(messages) > 1 {
		return nil, fmt.Errorf("message prefix %q is ambiguous", clean)
	}
	return &messages[0], nil
}

func (db *DB) CreateMessage(channelID, userID, content, replyToID string) (Message, error) {
	clean := SanitizeMessageContent(content)
	if clean == "" {
		return Message{}, ErrEmptyMessage
	}

	if replyToID != "" {
		if _, err := db.GetMessageByID(channelID, replyToID); err != nil {
			replyToID = ""
		}
	}

	message := Message{
		ID:        uuid.New().String(),
		ChannelID: channelID,
		UserID:    userID,
		Content:   clean,
		ReplyToID: replyToID,
		CreatedAt: time.Now(),
	}

	if _, err := db.Exec(`
		INSERT INTO messages (id, channel_id, user_id, content, reply_to_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, message.ID, message.ChannelID, message.UserID, message.Content, message.ReplyToID, message.CreatedAt); err != nil {
		return Message{}, err
	}
	_ = db.TouchUserActivity(userID)

	created, err := db.GetMessageByID(channelID, message.ID)
	if err != nil {
		return Message{}, err
	}
	return *created, nil
}

func (db *DB) UpdateMessage(channelID, messageID, editorID, content string, allowModeration bool) (*Message, error) {
	current, err := db.GetMessageByID(channelID, messageID)
	if err != nil {
		return nil, err
	}
	if current.UserID != editorID && !allowModeration {
		return nil, fmt.Errorf("cannot edit another user's message")
	}

	clean := SanitizeMessageContent(content)
	if clean == "" {
		return nil, ErrEmptyMessage
	}

	if _, err := db.Exec(`
		UPDATE messages
		SET content = ?, edited_at = ?, is_deleted = 0
		WHERE id = ? AND channel_id = ?
	`, clean, time.Now(), messageID, channelID); err != nil {
		return nil, err
	}
	_ = db.TouchUserActivity(editorID)
	return db.GetMessageByID(channelID, messageID)
}

func (db *DB) DeleteMessage(channelID, messageID, actorID string, allowModeration bool) error {
	current, err := db.GetMessageByID(channelID, messageID)
	if err != nil {
		return err
	}
	if current.UserID != actorID && !allowModeration {
		return fmt.Errorf("cannot delete another user's message")
	}
	_, err = db.Exec(`
		UPDATE messages
		SET content = '[deleted]', is_deleted = 1, edited_at = ?
		WHERE id = ? AND channel_id = ?
	`, time.Now(), messageID, channelID)
	if err == nil {
		_ = db.TouchUserActivity(actorID)
	}
	return err
}

func (db *DB) DeleteChannel(id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []struct {
		query string
		args  []any
	}{
		{query: "DELETE FROM messages WHERE channel_id = ?", args: []any{id}},
		{query: "DELETE FROM channel_members WHERE channel_id = ?", args: []any{id}},
		{query: "DELETE FROM channel_reads WHERE channel_id = ?", args: []any{id}},
		{query: "DELETE FROM channels WHERE id = ?", args: []any{id}},
	}

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt.query, stmt.args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) ClearChannelMessages(channelID string) error {
	_, err := db.Exec("DELETE FROM messages WHERE channel_id = ?", channelID)
	return err
}

func (db *DB) DeleteLastMessages(channelID string, count int) error {
	if count <= 0 {
		count = 1
	}
	_, err := db.Exec(`
		DELETE FROM messages
		WHERE id IN (
			SELECT id
			FROM messages
			WHERE channel_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		)
	`, channelID, count)
	return err
}

func (db *DB) DeleteLastMessage(channelID string) error {
	return db.DeleteLastMessages(channelID, 1)
}

func (db *DB) MarkChannelRead(channelID, userID, lastMessageID string, readAt time.Time) error {
	if channelID == "" || userID == "" {
		return nil
	}

	_, err := db.Exec(`
		INSERT INTO channel_reads (channel_id, user_id, last_read_at, last_read_message_id)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(channel_id, user_id)
		DO UPDATE SET
			last_read_at = excluded.last_read_at,
			last_read_message_id = excluded.last_read_message_id
	`, channelID, userID, readAt, lastMessageID)
	return err
}

func (db *DB) GetUnreadInfo(user *User, channels []Channel) map[string]UnreadInfo {
	info := make(map[string]UnreadInfo, len(channels))
	if user == nil {
		return info
	}

	for _, channel := range channels {
		var count int
		if err := db.QueryRow(`
			SELECT COUNT(*)
			FROM messages m
			LEFT JOIN channel_reads cr
				ON cr.channel_id = m.channel_id AND cr.user_id = ?
			WHERE m.channel_id = ?
				AND m.user_id != ?
				AND (cr.last_read_at IS NULL OR m.created_at > cr.last_read_at)
		`, user.ID, channel.ID, user.ID).Scan(&count); err != nil {
			log.Printf("error loading unread count for %s: %v", channel.ID, err)
			continue
		}

		var lastMessageID string
		_ = db.QueryRow(`
			SELECT m.id
			FROM messages m
			LEFT JOIN channel_reads cr
				ON cr.channel_id = m.channel_id AND cr.user_id = ?
			WHERE m.channel_id = ?
				AND m.user_id != ?
				AND (cr.last_read_at IS NULL OR m.created_at > cr.last_read_at)
			ORDER BY m.created_at DESC
			LIMIT 1
		`, user.ID, channel.ID, user.ID).Scan(&lastMessageID)

		var mentionCount int
		_ = db.QueryRow(`
			SELECT COUNT(*)
			FROM messages m
			LEFT JOIN channel_reads cr
				ON cr.channel_id = m.channel_id AND cr.user_id = ?
			WHERE m.channel_id = ?
				AND m.user_id != ?
				AND COALESCE(m.is_deleted, 0) = 0
				AND (cr.last_read_at IS NULL OR m.created_at > cr.last_read_at)
				AND LOWER(m.content) LIKE ?
		`, user.ID, channel.ID, user.ID, "%@"+normalizeUsername(user.Username)+"%").Scan(&mentionCount)

		info[channel.ID] = UnreadInfo{
			Count:         count,
			LastMessageID: lastMessageID,
			MentionCount:  mentionCount,
		}
	}

	return info
}

func (db *DB) CreateModerationLog(actorID, targetID, channelID, action, details string) error {
	_, err := db.Exec(`
		INSERT INTO moderation_logs (id, actor_user_id, target_user_id, channel_id, action, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, uuid.New().String(), actorID, targetID, channelID, sanitizeSingleLine(action, 40), sanitizeSingleLine(details, 240), time.Now())
	return err
}

func (db *DB) GetModerationLogs(limit int) []ModerationLog {
	if limit <= 0 {
		limit = 30
	}

	rows, err := db.Query(`
		SELECT
			ml.id,
			ml.actor_user_id,
			COALESCE(actor.username, ''),
			COALESCE(ml.target_user_id, ''),
			COALESCE(target.username, ''),
			COALESCE(ml.channel_id, ''),
			COALESCE(c.name, ''),
			ml.action,
			ml.details,
			ml.created_at
		FROM moderation_logs ml
		LEFT JOIN users actor ON actor.id = ml.actor_user_id
		LEFT JOIN users target ON target.id = ml.target_user_id
		LEFT JOIN channels c ON c.id = ml.channel_id
		ORDER BY ml.created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		log.Printf("error loading moderation logs: %v", err)
		return nil
	}
	defer rows.Close()

	var logs []ModerationLog
	for rows.Next() {
		var entry ModerationLog
		if err := rows.Scan(
			&entry.ID,
			&entry.ActorUserID,
			&entry.ActorUsername,
			&entry.TargetUserID,
			&entry.TargetUsername,
			&entry.ChannelID,
			&entry.ChannelName,
			&entry.Action,
			&entry.Details,
			&entry.CreatedAt,
		); err != nil {
			log.Printf("error scanning moderation log: %v", err)
			return nil
		}
		logs = append(logs, entry)
	}

	return logs
}

func (db *DB) GetMentionMessages(user *User, limit int) []Message {
	if user == nil {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}

	rows, err := db.Query(`
		SELECT
			m.id,
			m.channel_id,
			m.user_id,
			CASE WHEN COALESCE(m.is_deleted, 0) = 1 THEN '[deleted]' ELSE m.content END,
			m.created_at,
			COALESCE(m.edited_at, m.created_at),
			COALESCE(m.is_deleted, 0),
			u.username,
			u.color,
			u.role,
			COALESCE(m.reply_to_id, ''),
			COALESCE(ru.username, ''),
			COALESCE(rm.content, '')
		FROM messages m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN messages rm ON m.reply_to_id = rm.id
		LEFT JOIN users ru ON rm.user_id = ru.id
		WHERE m.user_id != ?
			AND COALESCE(m.is_deleted, 0) = 0
			AND LOWER(m.content) LIKE ?
		ORDER BY m.created_at DESC
		LIMIT ?
	`, user.ID, "%@"+normalizeUsername(user.Username)+"%", limit)
	if err != nil {
		log.Printf("error fetching mentions: %v", err)
		return nil
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (db *DB) ExportChannelTranscript(channelID string) ([]Message, error) {
	rows, err := db.Query(`
		SELECT
			m.id,
			m.channel_id,
			m.user_id,
			CASE WHEN COALESCE(m.is_deleted, 0) = 1 THEN '[deleted]' ELSE m.content END,
			m.created_at,
			COALESCE(m.edited_at, m.created_at),
			COALESCE(m.is_deleted, 0),
			u.username,
			u.color,
			u.role,
			COALESCE(m.reply_to_id, ''),
			COALESCE(ru.username, ''),
			COALESCE(rm.content, '')
		FROM messages m
		JOIN users u ON m.user_id = u.id
		LEFT JOIN messages rm ON m.reply_to_id = rm.id
		LEFT JOIN users ru ON rm.user_id = ru.id
		WHERE m.channel_id = ?
		ORDER BY m.created_at ASC
	`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows), nil
}

func (db *DB) CreateBackup(path string) error {
	if path == "" {
		return fmt.Errorf("backup path is empty")
	}
	_ = os.Remove(path)
	escaped := strings.ReplaceAll(path, "'", "''")
	_, err := db.Exec("VACUUM INTO '" + escaped + "'")
	return err
}
