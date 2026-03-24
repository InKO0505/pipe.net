package tui

import "testing"

func TestFindThemeByColor(t *testing.T) {
	idx, ok := findThemeByColor("#e74c3c")
	if !ok {
		t.Fatalf("expected to find theme by color")
	}
	if appPalette[idx].Name != "Ruby" {
		t.Fatalf("expected Ruby, got %s", appPalette[idx].Name)
	}
}

func TestFindThemeByName(t *testing.T) {
	idx, ok := findThemeByName("sKy")
	if !ok {
		t.Fatalf("expected to find theme by name")
	}
	if appPalette[idx].Color != "#00BFFF" {
		t.Fatalf("expected #00BFFF, got %s", appPalette[idx].Color)
	}
}
