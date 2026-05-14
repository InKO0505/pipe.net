// Package design exposes the pipe.net design tokens for the Go TUI client.
// Use with lipgloss: lipgloss.Color(design.Accent) etc.
//
// Single source of truth — keep in sync with handoff/tokens.css and the
// Android Compose theme in handoff/tokens.kt.
package design

// ─────────────────────────────────────────────────────────────
// Palette — monochrome with ONE accent (acid lime).
//
// Strict accent usage:
//   ✓ live cursor / active channel rail / send action / owner crown / unread badge / mention pill
//   ✗ body text / borders (except focus) / decorative fills / hover states
// ─────────────────────────────────────────────────────────────

const (
	// Surfaces (darkest → lightest)
	BGDeep    = "#07070a"
	BG        = "#0a0a0c"
	Surface1  = "#101015"
	Surface2  = "#15151c"
	Surface3  = "#1b1b24"

	// Borders
	BorderSubtle = "#1a1a22"
	Border       = "#22222b"
	BorderStrong = "#2f2f3a"

	// Text
	Text          = "#f1f1ec"
	TextSecondary = "#a3a3a0"
	TextMuted     = "#6f6f6d"
	TextDim       = "#4a4a4d"

	// The accent. Use sparingly.
	Accent    = "#c8f24a"
	AccentInk = "#0a0a0c" // foreground when accent is the background

	// Role tints (still mono, distinguishable)
	RoleOwner = Accent // owner = accent
	RoleAdmin = Text   // admin = bright text
	RoleUser  = TextSecondary

	// Status
	Online = Accent
	Away   = TextMuted
	Danger = "#f24a4a"
)

// ─────────────────────────────────────────────────────────────
// Glyphs — single source of truth for role markers, channel sigils, etc.
// ─────────────────────────────────────────────────────────────

const (
	GlyphOwner   = "👑"
	GlyphAdmin   = "★"
	GlyphChannel = "#"
	GlyphPrivate = "⊡"
	GlyphDM      = "@"
	GlyphChev    = ">"
	GlyphCheck   = "✓"
	GlyphDot     = "●"
	GlyphActive  = "│" // 2px-equivalent left rail marker
	GlyphTyping  = "…"
)

// ─────────────────────────────────────────────────────────────
// Layout — fixed cells (terminal-cell math; not px).
// 4-panel main: [channels 22ch] [feed flex] [online 22ch]
// ─────────────────────────────────────────────────────────────

const (
	PanelChannelsCols = 22
	PanelOnlineCols   = 22
	PanelHeadHeight   = 1 // rows
	PanelFootHeight   = 1
	InputHeight       = 3 // 1 padding + 1 content + 1 padding
)
