package tui

import "testing"

func TestSanitizeImageURLRejectsLocalAndPrivateTargets(t *testing.T) {
	t.Parallel()

	cases := []string{
		"/etc/passwd",
		"file:///etc/passwd",
		"http://localhost/image.png",
		"http://127.0.0.1/image.png",
		"http://10.0.0.5/image.png",
		"http://192.168.1.15/image.png",
	}

	for _, raw := range cases {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			if _, err := sanitizeImageURL(raw); err == nil {
				t.Fatalf("sanitizeImageURL(%q) unexpectedly succeeded", raw)
			}
		})
	}
}

func TestSanitizeImageURLAllowsPublicRemoteImages(t *testing.T) {
	t.Parallel()

	got, err := sanitizeImageURL("https://example.com/assets/pic.png")
	if err != nil {
		t.Fatalf("sanitizeImageURL() error = %v", err)
	}
	if got != "https://example.com/assets/pic.png" {
		t.Fatalf("sanitizeImageURL() = %q, want %q", got, "https://example.com/assets/pic.png")
	}
}
