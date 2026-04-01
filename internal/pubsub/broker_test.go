package pubsub

import (
	"testing"
	"time"

	"clinet/internal/db"
)

func TestUnsubscribeClosesSubscriptionChannel(t *testing.T) {
	t.Parallel()

	broker := NewBroker()
	dummyUser := &db.User{ID: "test_user_id"}
	sub := broker.Subscribe("general", dummyUser)
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
	dummyUser1 := &db.User{ID: "test_user_1"}
	dummyUser2 := &db.User{ID: "test_user_2"}
	active := broker.Subscribe("general", dummyUser1)
	inactive := broker.Subscribe("general", dummyUser2)
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

func TestSubscribeAnnouncesJoinToOtherSubscribersOnly(t *testing.T) {
	t.Parallel()

	broker := NewBroker()
	first := broker.Subscribe("general", &db.User{ID: "first", Username: "alice"})
	second := broker.Subscribe("general", &db.User{ID: "second", Username: "bob"})
	defer broker.Unsubscribe("general", second)
	defer broker.Unsubscribe("general", first)

	select {
	case msg := <-first:
		if msg.Content != "bob joined the channel" {
			t.Fatalf("join message = %q, want %q", msg.Content, "bob joined the channel")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for join announcement")
	}

	select {
	case msg := <-second:
		t.Fatalf("new subscriber unexpectedly received its own join announcement: %#v", msg)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestNotifyUserTargetsMatchingSubscriber(t *testing.T) {
	t.Parallel()

	broker := NewBroker()
	alice := broker.Subscribe("general", &db.User{ID: "1", Username: "alice"})
	bob := broker.Subscribe("general", &db.User{ID: "2", Username: "bob"})
	defer broker.Unsubscribe("general", alice)
	defer broker.Unsubscribe("general", bob)

	select {
	case msg := <-alice:
		if msg.Content != "bob joined the channel" {
			t.Fatalf("alice join message = %q, want %q", msg.Content, "bob joined the channel")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out draining alice join announcement")
	}

	if !broker.NotifyUser("bob", db.Message{ID: "CMD_CHANNELS", Content: "refresh"}) {
		t.Fatal("NotifyUser() = false, want true")
	}

	select {
	case msg := <-bob:
		if msg.ID != "CMD_CHANNELS" {
			t.Fatalf("bob message ID = %q, want %q", msg.ID, "CMD_CHANNELS")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bob notification")
	}

	select {
	case msg := <-alice:
		t.Fatalf("alice unexpectedly received notification: %#v", msg)
	case <-time.After(150 * time.Millisecond):
	}
}
