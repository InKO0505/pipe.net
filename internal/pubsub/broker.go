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
	if b.subscribers[channelID] == nil {
		b.subscribers[channelID] = make(map[chan db.Message]*db.User)
	}

	ch := make(chan db.Message, 100)
	b.subscribers[channelID][ch] = user
	shouldAnnounce := user != nil && user.Username != "" && len(b.subscribers[channelID]) > 1
	b.mu.Unlock()

	if shouldAnnounce {
		go b.broadcastExcept(channelID, ch, db.Message{
			ChannelID: channelID,
			UserID:    "system",
			Content:   user.Username + " joined the channel",
			CreatedAt: time.Now(),
			Username:  "Server",
			UserColor: "#808080",
		})
	}
	return ch
}

func (b *Broker) Unsubscribe(channelID string, ch chan db.Message) {
	b.mu.Lock()
	var (
		user           *db.User
		shouldAnnounce bool
	)
	if subs, ok := b.subscribers[channelID]; ok {
		if existingUser, exists := subs[ch]; exists {
			user = existingUser
			delete(subs, ch)
			close(ch)
			shouldAnnounce = user != nil && user.Username != "" && len(subs) > 0
		}
		if len(subs) == 0 {
			delete(b.subscribers, channelID)
		}
	}
	b.mu.Unlock()

	if shouldAnnounce {
		go b.Broadcast(channelID, db.Message{
			ChannelID: channelID,
			UserID:    "system",
			Content:   user.Username + " left the channel",
			CreatedAt: time.Now(),
			Username:  "Server",
			UserColor: "#808080",
		})
	}
}

func (b *Broker) Broadcast(channelID string, msg db.Message) {
	b.broadcastExcept(channelID, nil, msg)
}

func (b *Broker) broadcastExcept(channelID string, exclude chan db.Message, msg db.Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subs, ok := b.subscribers[channelID]; ok {
		for ch := range subs {
			if exclude != nil && ch == exclude {
				continue
			}
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

func (b *Broker) NotifyUser(username string, msg db.Message) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	notified := false
	for _, subs := range b.subscribers {
		for ch, u := range subs {
			if u != nil && u.Username == username {
				select {
				case ch <- msg:
				default:
				}
				notified = true
			}
		}
	}
	return notified
}
