package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
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
}

type Channel struct {
	ID        string
	Name      string
	Topic     string
	CreatedAt time.Time
}

type Message struct {
	ID        string
	ChannelID string
	UserID    string
	Content   string
	CreatedAt time.Time
	Username  string // Used for Joins
	UserColor string // Used for Joins
	UserRole  string // Used for Joins
}

func InitDB(filepath string) (*DB, error) {
	db, err := sql.Open("sqlite3", filepath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	// Schema Migrations
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			ssh_pub_key TEXT UNIQUE,
			username TEXT,
			is_verified BOOLEAN,
			created_at DATETIME
		);
		CREATE TABLE IF NOT EXISTS channels (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE,
			created_at DATETIME
		);
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			channel_id TEXT,
			user_id TEXT,
			content TEXT,
			created_at DATETIME
		);
	`)
	if err != nil {
		return nil, err
	}

	// Migrations for existing DBs
	db.Exec(`ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'user'`)
	db.Exec(`ALTER TABLE users ADD COLUMN color TEXT DEFAULT ''`)
	db.Exec(`ALTER TABLE users ADD COLUMN is_banned BOOLEAN DEFAULT 0`)
	db.Exec(`ALTER TABLE channels ADD COLUMN topic TEXT DEFAULT ''`)

	// Pre-seed default channels
	channels := []string{"#general", "#linux", "#bash-magic"}
	for _, ch := range channels {
		_, err = db.Exec(`INSERT OR IGNORE INTO channels (id, name, created_at) VALUES (?, ?, ?)`,
			uuid.New().String(), ch, time.Now())
		if err != nil {
			return nil, err
		}
	}

	return &DB{db}, nil
}

func (db *DB) GetUserByPubKey(pubKey string) (*User, error) {
	row := db.QueryRow("SELECT id, ssh_pub_key, username, is_verified, created_at, role, color, is_banned FROM users WHERE ssh_pub_key = ?", pubKey)
	var u User
	err := row.Scan(&u.ID, &u.SSHPubKey, &u.Username, &u.IsVerified, &u.CreatedAt, &u.Role, &u.Color, &u.IsBanned)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) GetUserByUsername(username string) (*User, error) {
	row := db.QueryRow("SELECT id, ssh_pub_key, username, is_verified, created_at, role, color, is_banned FROM users WHERE username = ?", username)
	var u User
	err := row.Scan(&u.ID, &u.SSHPubKey, &u.Username, &u.IsVerified, &u.CreatedAt, &u.Role, &u.Color, &u.IsBanned)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) CreateUser(pubKey string) *User {
	color := "#6366F1" // Default Indigo color

	var count int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	role := "user"
	if count == 0 {
		role = "owner"
	}

	u := &User{
		ID:         uuid.New().String(),
		SSHPubKey:  pubKey,
		Username:   fmt.Sprintf("anon_%s", uuid.New().String()[:4]),
		IsVerified: false,
		CreatedAt:  time.Now(),
		Role:       role,
		Color:      color,
		IsBanned:   false,
	}
	_, err := db.Exec("INSERT INTO users (id, ssh_pub_key, username, is_verified, created_at, role, color, is_banned) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		u.ID, u.SSHPubKey, u.Username, u.IsVerified, u.CreatedAt, u.Role, u.Color, u.IsBanned)
	if err != nil {
		log.Printf("Error creating user: %v\n", err)
	}
	return u
}

func (db *DB) SetVerified(userID string) {
	_, err := db.Exec("UPDATE users SET is_verified = 1 WHERE id = ?", userID)
	if err != nil {
		log.Printf("Error updating verification: %v\n", err)
	}
}

func (db *DB) UpdateUsername(userID, newName string) {
	_, err := db.Exec("UPDATE users SET username = ? WHERE id = ?", newName, userID)
	if err != nil {
		log.Printf("Error updating username: %v\n", err)
	}
}


func (db *DB) SetUserRole(userID, role string) {
	_, err := db.Exec("UPDATE users SET role = ? WHERE id = ?", role, userID)
	if err != nil {
		log.Printf("Error updating role: %v\n", err)
	}
}

func (db *DB) SetBanned(userID string, banned bool) {
	val := 0
	if banned {
		val = 1
	}
	_, err := db.Exec("UPDATE users SET is_banned = ? WHERE id = ?", val, userID)
	if err != nil {
		log.Printf("Error updating ban status: %v\n", err)
	}
}

func (db *DB) GetChannels() []Channel {
	rows, err := db.Query(`
		SELECT id, name, topic, created_at
		FROM channels
		ORDER BY CASE name
			WHEN '#general' THEN 0
			WHEN '#linux' THEN 1
			WHEN '#bash-magic' THEN 2
			ELSE 3
		END, name
	`)
	if err != nil {
		log.Printf("Error fetching channels: %v\n", err)
		return nil
	}
	defer rows.Close()

	var chs []Channel
	for rows.Next() {
		var c Channel
		if err := rows.Scan(&c.ID, &c.Name, &c.Topic, &c.CreatedAt); err != nil {
			log.Printf("Error scanning channel: %v\n", err)
			return nil
		}
		chs = append(chs, c)
	}
	return chs
}

func (db *DB) CreateChannel(name string) (*Channel, error) {
	name = "#" + strings.TrimPrefix(name, "#")
	ch := &Channel{
		ID:        uuid.New().String(),
		Name:      name,
		Topic:     "",
		CreatedAt: time.Now(),
	}
	_, err := db.Exec("INSERT OR IGNORE INTO channels (id, name, topic, created_at) VALUES (?, ?, ?, ?)",
		ch.ID, ch.Name, ch.Topic, ch.CreatedAt)
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func (db *DB) SetChannelTopic(channelID, topic string) {
	_, err := db.Exec("UPDATE channels SET topic = ? WHERE id = ?", topic, channelID)
	if err != nil {
		log.Printf("Error setting topic: %v\n", err)
	}
}

func (db *DB) GetMessages(channelID string) []Message {
	rows, err := db.Query(`
        SELECT m.id, m.channel_id, m.user_id, m.content, m.created_at, u.username, u.color, u.role
        FROM messages m
        JOIN users u ON m.user_id = u.id
        WHERE m.channel_id = ?
        ORDER BY m.created_at ASC
    `, channelID)
	if err != nil {
		log.Printf("Error fetching messages: %v\n", err)
		return nil
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.UserID, &m.Content, &m.CreatedAt, &m.Username, &m.UserColor, &m.UserRole); err != nil {
			log.Printf("Error scanning message: %v\n", err)
			return nil
		}
		msgs = append(msgs, m)
	}
	return msgs
}

func (db *DB) CreateMessage(channelID, userID, content string) Message {
	m := Message{
		ID:        uuid.New().String(),
		ChannelID: channelID,
		UserID:    userID,
		Content:   content,
		CreatedAt: time.Now(),
	}
	_, err := db.Exec("INSERT INTO messages (id, channel_id, user_id, content, created_at) VALUES (?, ?, ?, ?, ?)",
		m.ID, m.ChannelID, m.UserID, m.Content, m.CreatedAt)
	if err != nil {
		log.Printf("Error creating message: %v\n", err)
	}

	row := db.QueryRow("SELECT username, color, role FROM users WHERE id = ?", userID)
	_ = row.Scan(&m.Username, &m.UserColor, &m.UserRole)
	return m
}
func (db *DB) GetChannelByName(name string) (*Channel, error) {
	name = "#" + strings.TrimPrefix(name, "#")
	row := db.QueryRow("SELECT id, name, topic, created_at FROM channels WHERE name = ?", name)
	var c Channel
	err := row.Scan(&c.ID, &c.Name, &c.Topic, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *DB) DeleteChannel(id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM messages WHERE channel_id = ?", id)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM channels WHERE id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) ClearChannelMessages(channelID string) error {
	_, err := db.Exec("DELETE FROM messages WHERE channel_id = ?", channelID)
	return err
}

func (db *DB) DeleteLastMessage(channelID string) error {
	_, err := db.Exec(`
		DELETE FROM messages 
		WHERE id = (
			SELECT id FROM messages 
			WHERE channel_id = ? 
			ORDER BY created_at DESC 
			LIMIT 1
		)
	`, channelID)
	return err
}
