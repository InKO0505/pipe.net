package pubsub

import (
	"sort"
	"sync"
	"time"

	"clinet/internal/db"
)

type Broker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan db.Message]*db.User // channelID -> mapped by channel to user
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string]map[chan db.Message]*db.User),
	}
}

func (b *Broker) Subscribe(channelID string, user *db.User) chan db.Message {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.subscribers[channelID] == nil {
		b.subscribers[channelID] = make(map[chan db.Message]*db.User)
	}

	ch := make(chan db.Message, 100)
	b.subscribers[channelID][ch] = user

	go b.Broadcast(channelID, db.Message{
		ID:        "",
		ChannelID: channelID,
		UserID:    "system",
		Content:   user.Username + " joined the channel",
		CreatedAt: time.Now(),
		Username:  "Server",
		UserColor: "#808080",
	})
	return ch
}

func (b *Broker) Unsubscribe(channelID string, ch chan db.Message) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.subscribers[channelID]; ok {
		if user, exists := subs[ch]; exists { // Retrieve user before deleting
			delete(subs, ch)
			close(ch)

			// Broadcast "left" message asynchronously
			if user != nil {
				go b.Broadcast(channelID, db.Message{
					ID:        "",
					ChannelID: channelID,
					UserID:    "system",
					Content:   user.Username + " left the channel",
					CreatedAt: time.Now(),
					Username:  "Server",
					UserColor: "#808080",
				})
			}
		}
		if len(subs) == 0 {
			delete(b.subscribers, channelID)
		}
	}
}

func (b *Broker) Broadcast(channelID string, msg db.Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subs, ok := b.subscribers[channelID]; ok {
		for ch := range subs {
			select {
			case ch <- msg:
			default:
				// Channel buffer full; drop to prevent blocking the broker
			}
		}
	}
}

func (b *Broker) GetOnlineUsers(channelID string) []*db.User {
	b.mu.RLock()
	defer b.mu.RUnlock()

	usersMap := make(map[string]*db.User)
	if subs, ok := b.subscribers[channelID]; ok {
		for _, u := range subs {
			if u != nil {
				usersMap[u.ID] = u
			}
		}
	}

	var users []*db.User
	for _, u := range usersMap {
		users = append(users, u)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})

	return users
}

func (b *Broker) KickUser(username string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	kicked := false
	for _, subs := range b.subscribers {
		for ch, u := range subs {
			if u != nil && u.Username == username {
				select {
				case ch <- db.Message{ID: "CMD_KICK"}:
				default:
				}
				kicked = true
			}
		}
	}
	return kicked
}
