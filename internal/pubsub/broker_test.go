package pubsub

import (
	"testing"
	"time"

	"clinet/internal/db"
)

func TestUnsubscribeClosesSubscriptionChannel(t *testing.T) {
	t.Parallel()

	broker := NewBroker()
	sub := broker.Subscribe("general")
	broker.Unsubscribe("general", sub)

	select {
	case _, ok := <-sub:
		if ok {
			t.Fatal("expected subscription channel to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for closed subscription channel")
	}
}

func TestBroadcastOnlyReachesActiveSubscribers(t *testing.T) {
	t.Parallel()

	broker := NewBroker()
	active := broker.Subscribe("general")
	inactive := broker.Subscribe("general")
	broker.Unsubscribe("general", inactive)

	want := db.Message{Content: "hello"}
	broker.Broadcast("general", want)

	select {
	case got := <-active:
		if got.Content != want.Content {
			t.Fatalf("broadcast content = %q, want %q", got.Content, want.Content)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for active subscriber")
	}

	select {
	case _, ok := <-inactive:
		if ok {
			t.Fatal("inactive subscriber unexpectedly received a message")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for inactive subscriber closure")
	}
}
