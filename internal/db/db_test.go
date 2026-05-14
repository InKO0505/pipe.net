package db

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestInitDBSeedsChannelsInReadmeOrder(t *testing.T) {
	t.Parallel()

	database, err := InitDB(filepath.Join(t.TempDir(), "clinet.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	defer database.Close()

	channels := database.GetChannels()
	if len(channels) < 3 {
		t.Fatalf("expected at least 3 channels, got %d", len(channels))
	}

	got := []string{channels[0].Name, channels[1].Name, channels[2].Name}
	want := []string{"#general", "#linux", "#bash-magic"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("channel %d = %q, want %q (got %v)", i, got[i], want[i], got)
		}
	}
}

func TestGetAccessibleChannelsRespectsPrivateMembership(t *testing.T) {
	t.Parallel()

	database, err := InitDB(filepath.Join(t.TempDir(), "clinet.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	defer database.Close()

	owner := database.CreateUser("owner-pubkey")
	member := database.CreateUser("member-pubkey")

	privateChannel, err := database.CreateChannel("staff", true, owner.ID)
	if err != nil {
		t.Fatalf("CreateChannel() error = %v", err)
	}

	for _, channel := range database.GetAccessibleChannels(member) {
		if channel.ID == privateChannel.ID {
			t.Fatalf("private channel %q should not be visible before membership", privateChannel.Name)
		}
	}

	if err := database.AddChannelMember(privateChannel.ID, member.ID); err != nil {
		t.Fatalf("AddChannelMember() error = %v", err)
	}

	found := false
	for _, channel := range database.GetAccessibleChannels(member) {
		if channel.ID == privateChannel.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("private channel %q should be visible after membership", privateChannel.Name)
	}
}

func TestCreateMessageSanitizesReplyMetadata(t *testing.T) {
	t.Parallel()

	database, err := InitDB(filepath.Join(t.TempDir(), "clinet.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	defer database.Close()

	user := database.CreateUser("reply-pubkey")
	channel, err := database.GetChannelByName("#general")
	if err != nil {
		t.Fatalf("GetChannelByName() error = %v", err)
	}

	root, err := database.CreateMessage(channel.ID, user.ID, "hello there", "")
	if err != nil {
		t.Fatalf("CreateMessage(root) error = %v", err)
	}

	reply, err := database.CreateMessage(channel.ID, user.ID, "\x1b[31mreply\x1b[0m", root.ID)
	if err != nil {
		t.Fatalf("CreateMessage(reply) error = %v", err)
	}

	if reply.Content != "reply" {
		t.Fatalf("reply content = %q, want %q", reply.Content, "reply")
	}
	if reply.ReplyToID != root.ID {
		t.Fatalf("ReplyToID = %q, want %q", reply.ReplyToID, root.ID)
	}
	if reply.ReplyToUsername != user.Username {
		t.Fatalf("ReplyToUsername = %q, want %q", reply.ReplyToUsername, user.Username)
	}
	if reply.ReplyToContent != root.Content {
		t.Fatalf("ReplyToContent = %q, want %q", reply.ReplyToContent, root.Content)
	}

	_, err = database.CreateMessage(channel.ID, user.ID, "\x1b[31m\x1b[0m", "")
	if !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("CreateMessage(empty sanitized) error = %v, want %v", err, ErrEmptyMessage)
	}
}

func TestCreateMobileUserCreatesAndReusesUsername(t *testing.T) {
	t.Parallel()

	database, err := InitDB(filepath.Join(t.TempDir(), "clinet.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	defer database.Close()

	user, err := database.CreateMobileUser("inko_mobile")
	if err != nil {
		t.Fatalf("CreateMobileUser() error = %v", err)
	}
	if user.Username != "inko_mobile" {
		t.Fatalf("username = %q, want %q", user.Username, "inko_mobile")
	}
	if user.SSHPubKey == "" || user.SSHPubKey[:7] != "mobile:" {
		t.Fatalf("ssh pub key = %q, want mobile sentinel", user.SSHPubKey)
	}
	for _, publicChannel := range []string{"#general", "#linux", "#bash-magic"} {
		channel, err := database.GetChannelByName(publicChannel)
		if err != nil {
			t.Fatalf("GetChannelByName(%q) error = %v", publicChannel, err)
		}

		found := false
		for _, member := range database.GetChannelMembers(channel.ID) {
			if member.ID == user.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("mobile user %q should be joined to public channel %q", user.Username, publicChannel)
		}
	}

	same, err := database.CreateMobileUser("inko_mobile")
	if err != nil {
		t.Fatalf("CreateMobileUser(existing) error = %v", err)
	}
	if same.ID != user.ID {
		t.Fatalf("existing user ID = %q, want %q", same.ID, user.ID)
	}

	_, err = database.CreateMobileUser("x")
	if !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("CreateMobileUser(invalid) error = %v, want %v", err, ErrInvalidUsername)
	}
}

func TestEnsurePublicChannelMembershipsBackfillsExistingUser(t *testing.T) {
	t.Parallel()

	database, err := InitDB(filepath.Join(t.TempDir(), "clinet.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	defer database.Close()

	user := database.CreateUser("backfill-pubkey")
	if _, err := database.Exec("DELETE FROM channel_members WHERE user_id = ?", user.ID); err != nil {
		t.Fatalf("DELETE channel_members error = %v", err)
	}

	if err := database.EnsurePublicChannelMemberships(user.ID); err != nil {
		t.Fatalf("EnsurePublicChannelMemberships() error = %v", err)
	}

	for _, publicChannel := range []string{"#general", "#linux", "#bash-magic"} {
		channel, err := database.GetChannelByName(publicChannel)
		if err != nil {
			t.Fatalf("GetChannelByName(%q) error = %v", publicChannel, err)
		}

		found := false
		for _, member := range database.GetChannelMembers(channel.ID) {
			if member.ID == user.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("user %q should be backfilled into public channel %q", user.Username, publicChannel)
		}
	}
}

func TestGetOrCreateDirectChannelReturnsStableChannel(t *testing.T) {
	t.Parallel()

	database, err := InitDB(filepath.Join(t.TempDir(), "clinet.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	defer database.Close()

	alice := database.CreateUser("alice-pubkey")
	bob := database.CreateUser("bob-pubkey")

	first, err := database.GetOrCreateDirectChannel(alice, bob)
	if err != nil {
		t.Fatalf("GetOrCreateDirectChannel(first) error = %v", err)
	}
	second, err := database.GetOrCreateDirectChannel(alice, bob)
	if err != nil {
		t.Fatalf("GetOrCreateDirectChannel(second) error = %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("DM channel IDs differ: %q vs %q", first.ID, second.ID)
	}
	if first.Kind != "dm" || !first.IsPrivate {
		t.Fatalf("DM channel = %#v, want private dm channel", first)
	}
}

func TestUpdateDeleteMessageAndMentionUnread(t *testing.T) {
	t.Parallel()

	database, err := InitDB(filepath.Join(t.TempDir(), "clinet.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	defer database.Close()

	alice := database.CreateUser("alice-pubkey")
	bob := database.CreateUser("bob-pubkey")
	channel, err := database.GetChannelByName("#general")
	if err != nil {
		t.Fatalf("GetChannelByName() error = %v", err)
	}

	msg, err := database.CreateMessage(channel.ID, bob.ID, "hello @"+alice.Username, "")
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}

	unread := database.GetUnreadInfo(alice, []Channel{*channel})
	info := unread[channel.ID]
	if info.Count != 1 || info.MentionCount != 1 {
		t.Fatalf("UnreadInfo = %#v, want Count=1 MentionCount=1", info)
	}

	updated, err := database.UpdateMessage(channel.ID, msg.ID, bob.ID, "updated @"+alice.Username, false)
	if err != nil {
		t.Fatalf("UpdateMessage() error = %v", err)
	}
	if !updated.IsEdited {
		t.Fatalf("updated message should be marked edited")
	}

	if err := database.DeleteMessage(channel.ID, msg.ID, bob.ID, false); err != nil {
		t.Fatalf("DeleteMessage() error = %v", err)
	}
	deleted, err := database.GetMessageByID(channel.ID, msg.ID)
	if err != nil {
		t.Fatalf("GetMessageByID(deleted) error = %v", err)
	}
	if deleted.Content != "[deleted]" {
		t.Fatalf("deleted content = %q, want %q", deleted.Content, "[deleted]")
	}

	if err := database.MarkChannelRead(channel.ID, alice.ID, msg.ID, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("MarkChannelRead() error = %v", err)
	}
	unread = database.GetUnreadInfo(alice, []Channel{*channel})
	if unread[channel.ID].Count != 0 {
		t.Fatalf("Unread after mark read = %#v, want zero", unread[channel.ID])
	}
}
