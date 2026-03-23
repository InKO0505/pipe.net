package pubsub

import (
	"sync"

	"clinet/internal/db"
)

type Broker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan db.Message]struct{} // channelID -> subset of active listening clients
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string]map[chan db.Message]struct{}),
	}
}

func (b *Broker) Subscribe(channelID string) chan db.Message {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.subscribers[channelID] == nil {
		b.subscribers[channelID] = make(map[chan db.Message]struct{})
	}

	ch := make(chan db.Message, 100)
	b.subscribers[channelID][ch] = struct{}{}
	return ch
}

func (b *Broker) Unsubscribe(channelID string, ch chan db.Message) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.subscribers[channelID]; ok {
		if _, exists := subs[ch]; exists {
			delete(subs, ch)
			close(ch)
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
