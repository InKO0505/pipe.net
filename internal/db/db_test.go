package db

import (
	"path/filepath"
	"testing"
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
