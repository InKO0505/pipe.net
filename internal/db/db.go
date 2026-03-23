// internal/db/db.go
package db

import (
	"database/sql"
	"fmt"
	"log"
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
}

type Channel struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

type Message struct {
	ID        string
	ChannelID string
	UserID    string
	Content   string
	CreatedAt time.Time
	Username  string // Used for Joins
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

	// Pre-seed default channels
	channels := []string{"#general", "#linux", "#bash-magic"}
	for _, ch := range channels {
		_, err = db.Exec(`INSERT OR IGNORE INTO channels (id, name, created_at) VALUES (?, ?, ?)`,
			uuid.New().String(), ch, time.Now())
	}

	return &DB{db}, nil
}

func (db *DB) GetUserByPubKey(pubKey string) (*User, error) {
	row := db.QueryRow("SELECT id, ssh_pub_key, username, is_verified, created_at FROM users WHERE ssh_pub_key = ?", pubKey)
	var u User
	err := row.Scan(&u.ID, &u.SSHPubKey, &u.Username, &u.IsVerified, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) CreateUser(pubKey string) *User {
	u := &User{
		ID:         uuid.New().String(),
		SSHPubKey:  pubKey,
		Username:   fmt.Sprintf("anon_%s", uuid.New().String()[:4]),
		IsVerified: false,
		CreatedAt:  time.Now(),
	}
	_, err := db.Exec("INSERT INTO users (id, ssh_pub_key, username, is_verified, created_at) VALUES (?, ?, ?, ?, ?)",
		u.ID, u.SSHPubKey, u.Username, u.IsVerified, u.CreatedAt)
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

func (db *DB) GetChannels() []Channel {
	rows, err := db.Query("SELECT id, name, created_at FROM channels ORDER BY name")
	if err != nil {
		log.Printf("Error fetching channels: %v\n", err)
		return nil
	}
	defer rows.Close()

	var chs []Channel
	for rows.Next() {
		var c Channel
		rows.Scan(&c.ID, &c.Name, &c.CreatedAt)
		chs = append(chs, c)
	}
	return chs
}

func (db *DB) GetMessages(channelID string) []Message {
	rows, err := db.Query(`
        SELECT m.id, m.channel_id, m.user_id, m.content, m.created_at, u.username
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
		rows.Scan(&m.ID, &m.ChannelID, &m.UserID, &m.Content, &m.CreatedAt, &m.Username)
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

	row := db.QueryRow("SELECT username FROM users WHERE id = ?", userID)
	row.Scan(&m.Username)
	return m
}
