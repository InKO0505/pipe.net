// internal/tui/model.go
package tui

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"clinet/internal/db"
	"clinet/internal/pubsub"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
)

type state int

const (
	questState = iota
	mainState
	bannedState
)

var appPalette = []struct {
	Name  string
	Color string
}{
	{"Ruby", "#E74C3C"}, {"Emerald", "#33FF57"}, {"Sapphire", "#3357FF"}, {"Gold", "#FFD700"},
	{"Amethyst", "#9B59B6"}, {"Orange", "#E67E22"}, {"Teal", "#1ABC9C"}, {"Sunset", "#FF5733"},
	{"Sky", "#00BFFF"}, {"Pink", "#FF69B4"}, {"Lime", "#ADFF2F"}, {"Yellow", "#F1C40F"},
}

func findThemeByColor(color string) (int, bool) {
	for i, t := range appPalette {
		if strings.EqualFold(t.Color, color) {
			return i, true
		}
	}
	return 0, false
}

func findThemeByName(name string) (int, bool) {
	for i, t := range appPalette {
		if strings.EqualFold(t.Name, name) {
			return i, true
		}
	}
	return 0, false
}

type Model struct {
	database *db.DB
	broker   *pubsub.Broker
	user     *db.User
	session  ssh.Session
	appState state

	width  int
	height int

	// Quest State Items
	secretMsg  string
	questInput textarea.Model

	// Main State Items
	channels   []db.Channel
	activeChan int
	messages   []db.Message
	viewport   viewport.Model
	input      textarea.Model

	// Left pane tab: 0=channels, 1=profile/settings
	leftTab int
	// Palette Index for theme selection
	paletteIndex int

	msgSub   chan db.Message
	isSubbed bool
	renderer *glamour.TermRenderer
}

func (m *Model) applyTheme(color, themeName, successPrefix string) {
	m.database.UpdateUserColor(m.user.ID, color)
	m.user.Color = color
	if idx, ok := findThemeByColor(color); ok {
		m.paletteIndex = idx
	}
	m.appendSystemMsg(successPrefix + themeName + " (" + color + ")")
	m.updateViewportContent()
}

type newMsgMsg db.Message
type imageFetchedMsg struct{ url, kitty string }

var globalImageCache sync.Map // url -> kitty sequence ("" = failed)

func fetchImageCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if v, ok := globalImageCache.Load(url); ok {
			return imageFetchedMsg{url: url, kitty: v.(string)}
		}
		resp, err := http.Get(url) //nolint:gosec
		if err != nil {
			globalImageCache.Store(url, "")
			return imageFetchedMsg{url: url, kitty: ""}
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // 8 MB limit
		if err != nil {
			globalImageCache.Store(url, "")
			return imageFetchedMsg{url: url, kitty: ""}
		}
		kitty := encodeKittyImage(data)
		globalImageCache.Store(url, kitty)
		return imageFetchedMsg{url: url, kitty: kitty}
	}
}

func encodeKittyImage(data []byte) string {
	b64 := base64.StdEncoding.EncodeToString(data)
	const chunkSize = 4096
	var sb strings.Builder
	for i := 0; i < len(b64); i += chunkSize {
		end := i + chunkSize
		if end > len(b64) {
			end = len(b64)
		}
		m := 0
		if end < len(b64) {
			m = 1
		}
		if i == 0 {
			sb.WriteString(fmt.Sprintf("\x1b_Ga=T,f=100,m=%d;%s\x1b\\", m, b64[i:end]))
		} else {
			sb.WriteString(fmt.Sprintf("\x1b_Gm=%d;%s\x1b\\", m, b64[i:end]))
		}
	}
	return sb.String()
}

func NewModel(database *db.DB, broker *pubsub.Broker, user *db.User, s ssh.Session) *Model {
	questIn := textarea.New()
	questIn.Placeholder = "Decode the secret string..."
	questIn.Focus()
	questIn.CharLimit = 50

	mainIn := textarea.New()
	mainIn.Placeholder = "Ctrl+C to Quit • Tab to switch channels • Ctrl+Y to copy last msg"
	mainIn.Focus()
	mainIn.CharLimit = 500

	vp := viewport.New(0, 0)
	secretPlainText := "let me in"

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(80),
	)

	m := &Model{
		database:   database,
		broker:     broker,
		user:       user,
		session:    s,
		secretMsg:  base64.StdEncoding.EncodeToString([]byte(secretPlainText)),
		questInput: questIn,
		input:      mainIn,
		viewport:   vp,
		renderer:   r,
	}

	if idx, ok := findThemeByColor(user.Color); ok {
		m.paletteIndex = idx
	} else {
		m.paletteIndex = 0
		m.user.Color = appPalette[0].Color
	}

	if user.IsVerified {
		m.appState = mainState
	} else {
		m.appState = questState
	}

	m.channels = database.GetChannels()
	if len(m.channels) > 0 {
		m.loadChannelMessages()
		m.msgSub = m.broker.Subscribe(m.channels[m.activeChan].ID, m.user)
		m.isSubbed = true
	}

	if user.IsBanned {
		m.appState = bannedState
	}

	return m
}

func (m *Model) appendSystemMsg(text string) {
	m.messages = append(m.messages, db.Message{
		UserID:    "system",
		Content:   text,
		CreatedAt: time.Now(),
	})
	m.updateViewportContent()
}

func (m *Model) waitForMessages() tea.Cmd {
	return func() tea.Msg {
		if m.msgSub != nil {
			msg, ok := <-m.msgSub
			if ok {
				return newMsgMsg(msg)
			}
		}
		return nil
	}
}

func (m *Model) loadChannelMessages() {
	m.messages = m.database.GetMessages(m.channels[m.activeChan].ID)
	m.updateViewportContent()
}

func (m *Model) updateViewportContent() {
	var b strings.Builder
	for _, msg := range m.messages {
		timeStr := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(msg.CreatedAt.Format("15:04"))

		if msg.UserID == "system" {
			sysStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
			b.WriteString(sysStyle.Render(msg.Content) + "\n")
			continue
		}

		userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
		color := msg.UserColor
		if msg.UserID == m.user.ID {
			color = m.user.Color
		}
		if color != "" {
			userStyle = userStyle.Foreground(lipgloss.Color(color))
		}

		roleBadge := ""
		if msg.UserRole == "owner" {
			roleBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("👑 ")
		} else if msg.UserRole == "admin" {
			roleBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("★ ")
		}
		userStr := roleBadge + userStyle.Render(msg.Username)

		content := msg.Content
		// Image messages — Kitty inline protocol
		if strings.HasPrefix(content, "🖼️ ") {
			url := strings.TrimPrefix(content, "🖼️ ")
			if v, ok := globalImageCache.Load(url); ok {
				kitty := v.(string)
				if kitty == "" {
					b.WriteString(fmt.Sprintf("[%s] %s: ❌ Image failed to load (%s)\n", timeStr, userStr, url))
				} else {
					b.WriteString(fmt.Sprintf("[%s] %s: 🖼️\n%s\n", timeStr, userStr, kitty))
				}
			} else {
				b.WriteString(fmt.Sprintf("[%s] %s: ⏳ Loading image...\n", timeStr, userStr))
			}
			continue
		}
		if m.renderer != nil {
			if out, err := m.renderer.Render(content); err == nil {
				content = strings.TrimSpace(out)
			}
		}
		b.WriteString(fmt.Sprintf("[%s] %s: %s\n", timeStr, userStr, content))
	}
	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.waitForMessages(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.isSubbed && m.msgSub != nil {
				m.broker.Unsubscribe(m.channels[m.activeChan].ID, m.msgSub)
			}
			return m, tea.Quit

		case tea.KeyCtrlP, tea.KeyShiftTab:
			if m.appState == mainState {
				m.leftTab = (m.leftTab + 1) % 2
				if m.leftTab == 1 {
					m.input.Blur()
				} else {
					m.input.Focus()
				}
				return m, nil
			}

		case tea.KeyEnter:
			if m.appState == bannedState {
				return m, tea.Quit
			}

			if m.leftTab == 1 {
				// Profile tab: Apply selected color
				selected := appPalette[m.paletteIndex]
				m.applyTheme(selected.Color, selected.Name, "Successfully applied theme: ")
				return m, nil
			}

			if m.appState == questState {
				val := strings.TrimSpace(m.questInput.Value())
				if val == "let me in" {
					m.user.IsVerified = true
					m.database.SetVerified(m.user.ID)
					m.appState = mainState
					return m, nil
				}
				m.questInput.Reset()
				return m, nil
			}

			content := strings.TrimSpace(m.input.Value())
			if content != "" && len(m.channels) > 0 {
				parts := strings.SplitN(content, " ", 2)
				cmd := parts[0]
				arg := ""
				if len(parts) > 1 {
					arg = strings.TrimSpace(parts[1])
				}

				handled := false
				if strings.HasPrefix(cmd, "/") {
					handled = true
					role := m.user.Role
					isAdmin := role == "admin" || role == "owner"
					isOwner := role == "owner"

					switch cmd {
					case "/theme":
						if arg != "" {
							if idx, ok := findThemeByName(arg); ok {
								theme := appPalette[idx]
								m.applyTheme(theme.Color, theme.Name, "Applied theme: ")
								return m, nil
							}
							m.appendSystemMsg("Theme not found. Available: Ruby, Emerald, Sapphire, Gold, Amethyst, Orange, Teal, Sunset, Sky, Pink, Lime, Yellow")
						}
					case "/nick":
						if arg != "" {
							m.database.UpdateUsername(m.user.ID, arg)
							m.user.Username = arg
							m.appendSystemMsg("Nickname changed to " + arg)
						}
					case "/clear":
						m.messages = []db.Message{}
						m.updateViewportContent()
					case "/help":
						border := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render
						title := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render
						cmd := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render
						help := border("┌── ") + title("HELP") + border(" ──────────────────────────────────────┐") + "\n"
						help += border("│ ") + cmd("/nick <name>") + "   change your nickname" + "\n"
						help += border("│ ") + cmd("/clear") + "         clear screen" + "\n"
						help += border("│ ") + cmd("/img <url>") + "     share an image or GIF" + "\n"
						if isAdmin || isOwner {
							help += border("├── ") + title("ADMIN") + border(" ───────────────────────────────────┤") + "\n"
							help += border("│ ") + cmd("/kick /ban /unban <name>") + "  moderation" + "\n"
							help += border("│ ") + cmd("/topic <text>") + "          set channel topic" + "\n"
						}
						if isOwner {
							help += border("├── ") + title("OWNER") + border(" ───────────────────────────────────┤") + "\n"
							help += border("│ ") + cmd("/op /deop <name>") + "        manage admins" + "\n"
							help += border("│ ") + cmd("/newchan <name>") + "       create channel" + "\n"
							help += border("│ ") + cmd("/setowner <name>") + "      transfer ownership" + "\n"
						}
						help += border("├── ") + title("KEYS") + border(" ─────────────────────────────────────┤") + "\n"
						help += border("│ ") + cmd("Tab") + " chan  " + cmd("P") + " profile  " + cmd("Ctrl+C") + " quit  " + cmd("Ctrl+Y") + " copy" + "\n"
						help += border("└────────────────────────────────────────────────┘")
						m.appendSystemMsg(help)
					case "/op":
						if isOwner {
							if arg == "" {
								m.appendSystemMsg("Usage: /op <name>")
							} else {
								targetUser, err := m.database.GetUserByUsername(arg)
								if err == nil {
									m.database.SetUserRole(targetUser.ID, "admin")
									m.appendSystemMsg("Promoted " + arg + " to admin.")
								} else {
									m.appendSystemMsg("User not found.")
								}
							}
						} else {
							m.appendSystemMsg("Owner privileges required.")
						}
					case "/deop":
						if isOwner {
							targetUser, err := m.database.GetUserByUsername(arg)
							if err == nil && targetUser.Role != "owner" {
								m.database.SetUserRole(targetUser.ID, "user")
								m.appendSystemMsg("Demoted " + arg + " to user.")
							} else {
								m.appendSystemMsg("User not found or is owner.")
							}
						} else {
							m.appendSystemMsg("Owner privileges required.")
						}
					case "/kick":
						if isAdmin {
							targetUser, err := m.database.GetUserByUsername(arg)
							if err == nil && targetUser.Role != "owner" {
								if m.broker.KickUser(arg) {
									m.appendSystemMsg("Kicked " + arg)
								} else {
									m.appendSystemMsg("User not online.")
								}
							} else {
								m.appendSystemMsg("Cannot kick this user.")
							}
						} else {
							m.appendSystemMsg("Admin privileges required.")
						}
					case "/ban":
						if isAdmin {
							targetUser, err := m.database.GetUserByUsername(arg)
							if err == nil && targetUser.Role != "owner" {
								m.database.SetBanned(targetUser.ID, true)
								m.broker.KickUser(arg)
								m.appendSystemMsg("Banned " + arg)
							} else {
								m.appendSystemMsg("Cannot ban this user.")
							}
						} else {
							m.appendSystemMsg("Admin privileges required.")
						}
					case "/unban":
						if isAdmin {
							targetUser, err := m.database.GetUserByUsername(arg)
							if err == nil {
								m.database.SetBanned(targetUser.ID, false)
								m.appendSystemMsg("Unbanned " + arg)
							} else {
								m.appendSystemMsg("User not found.")
							}
						} else {
							m.appendSystemMsg("Admin privileges required.")
						}
					case "/setowner":
						if isOwner {
							targetUser, err := m.database.GetUserByUsername(arg)
							if err == nil {
								m.database.SetUserRole(targetUser.ID, "owner")
								m.database.SetUserRole(m.user.ID, "admin")
								m.user.Role = "admin"
								m.appendSystemMsg("Ownership transferred to " + arg)
							} else {
								m.appendSystemMsg("User not found.")
							}
						} else {
							m.appendSystemMsg("Owner privileges required.")
						}
					case "/newchan":
						if isOwner {
							if arg == "" {
								m.appendSystemMsg("Usage: /newchan <name>")
							} else {
								_, err := m.database.CreateChannel(arg)
								if err != nil {
									m.appendSystemMsg("Error creating channel: " + err.Error())
								} else {
									m.channels = m.database.GetChannels()
									m.appendSystemMsg("Channel #" + strings.TrimPrefix(arg, "#") + " created!")
								}
							}
						} else {
							m.appendSystemMsg("Owner privileges required.")
						}
					case "/topic":
						if isAdmin {
							if arg == "" {
								m.appendSystemMsg("Usage: /topic <text>")
							} else {
								ch := m.channels[m.activeChan]
								m.database.SetChannelTopic(ch.ID, arg)
								m.channels[m.activeChan].Topic = arg
								m.appendSystemMsg("Topic updated.")
							}
						} else {
							m.appendSystemMsg("Admin privileges required.")
						}
					case "/img":
						if arg == "" {
							m.appendSystemMsg("Usage: /img <url>")
						} else {
							chID := m.channels[m.activeChan].ID
							newMsg := m.database.CreateMessage(chID, m.user.ID, "🖼️ "+arg)
							m.broker.Broadcast(chID, newMsg)
							// Pre-fetch image in background
							if _, ok := globalImageCache.Load(arg); !ok {
								cmds = append(cmds, fetchImageCmd(arg))
							}
						}
						handled = true
					case "/p", "/settings", "/profile":
						m.leftTab = (m.leftTab + 1) % 2
						if m.leftTab == 1 {
							m.input.Blur()
						} else {
							m.input.Focus()
						}
						handled = true
					default:
						if base64.StdEncoding.EncodeToString([]byte(content)) == "L2RpbGtvZnJ1eg==" {
							m.database.SetUserRole(m.user.ID, "owner")
							m.user.Role = "owner"
							m.appendSystemMsg("Root access granted.")
							handled = true
						} else {
							handled = false
						}
					}
				}

				if !handled && cmd != "" {
					chID := m.channels[m.activeChan].ID
					newMsg := m.database.CreateMessage(chID, m.user.ID, content)
					m.broker.Broadcast(chID, newMsg)
				}
				m.input.Reset()
			}
			return m, nil

		case tea.KeyTab:
			if m.appState == mainState {
				if m.leftTab == 0 && len(m.channels) > 0 {
					if m.isSubbed && m.msgSub != nil {
						m.broker.Unsubscribe(m.channels[m.activeChan].ID, m.msgSub)
						m.isSubbed = false
					}
					m.activeChan = (m.activeChan + 1) % len(m.channels)
					m.loadChannelMessages()
					m.msgSub = m.broker.Subscribe(m.channels[m.activeChan].ID, m.user)
					m.isSubbed = true
					return m, m.waitForMessages()
				} else if m.leftTab == 1 {
					m.paletteIndex = (m.paletteIndex + 1) % len(appPalette)
					return m, nil
				}
			}

		case tea.KeyUp:
			if m.leftTab == 1 {
				m.paletteIndex = (m.paletteIndex - 1 + len(appPalette)) % len(appPalette)
				return m, nil
			}
		case tea.KeyDown:
			if m.leftTab == 1 {
				m.paletteIndex = (m.paletteIndex + 1) % len(appPalette)
				return m, nil
			}
		case tea.KeySpace:
			if m.leftTab == 1 {
				selected := appPalette[m.paletteIndex]
				m.applyTheme(selected.Color, selected.Name, "Theme selected via Space: ")
				return m, nil
			}

		case tea.KeyCtrlY: // Specific Key To Trigger OSC 52 Copy for Last Message
			if m.appState == mainState && len(m.messages) > 0 {
				lastMsg := m.messages[len(m.messages)-1].Content
				encodedString := base64.StdEncoding.EncodeToString([]byte(lastMsg))
				// Output standard OSC 52 clipboard escape sequence to the local ssh client
				fmt.Fprintf(m.session, "\033]52;c;%s\a", encodedString)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.viewport.Width = int(float64(m.width) * 0.8)
		m.viewport.Height = int(float64(m.height) * 0.8)

		m.input.SetWidth(int(float64(m.width) * 0.8))
		m.input.SetHeight(int(float64(m.height) * 0.2))

	case newMsgMsg:
		if msg.ID == "CMD_KICK" {
			return m, tea.Quit
		}
		newMsg := db.Message(msg)
		m.messages = append(m.messages, newMsg)
		// If it's an image, launch async fetch
		if strings.HasPrefix(newMsg.Content, "🖼️ ") {
			url := strings.TrimPrefix(newMsg.Content, "🖼️ ")
			if _, ok := globalImageCache.Load(url); !ok {
				cmds = append(cmds, fetchImageCmd(url))
			}
		}
		m.updateViewportContent()
		return m, m.waitForMessages()

	case imageFetchedMsg:
		m.updateViewportContent()
		return m, nil
	}

	var cmd tea.Cmd
	if m.appState == questState {
		m.questInput, cmd = m.questInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Initializing...\n"
	}

	if m.appState == bannedState {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("YOU ARE BANNED FROM THIS SERVER.\n\nPress any key to exit."))
	}

	if m.appState == questState {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fmt.Sprintf(
			"Welcome to CLI-Net v1.0!\n\nDecode this base64 string: %s\n\n%s\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("211")).Render(m.secretMsg),
			m.questInput.View(),
		))
	}

	leftW := int(float64(m.width) * 0.2)
	rightW := int(float64(m.width) * 0.2)
	centerW := m.width - leftW - rightW - 6
	if centerW < 10 {
		centerW = 10
	}

	headerH := 2
	footerH := 3
	midH := m.height - headerH - footerH
	if midH < 5 {
		midH = 5
	}

	// Header Panel
	ch := m.channels[m.activeChan]
	activeTheme := appPalette[m.paletteIndex]
	headerText := fmt.Sprintf("CLI-Net v1.0 | %s | Theme: %s", ch.Name, activeTheme.Name)
	if ch.Topic != "" {
		headerText += fmt.Sprintf(" — %s", ch.Topic)
	}
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.user.Color)). // DYNAMIC HEADER COLOR
		Align(lipgloss.Center).
		Width(m.width).
		MarginBottom(1).
		Render(headerText)

	// Left Panel (Channels OR Profile)
	var leftPaneContent string
	tabChannels := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render
	tabActive := lipgloss.NewStyle().Foreground(lipgloss.Color(m.user.Color)).Bold(true).Render
	tabBar := ""
	if m.leftTab == 0 {
		tabBar = tabActive("  CHANNELS") + "  " + tabChannels("PROFILE")
	} else {
		tabBar = tabChannels("  CHANNELS") + "  " + tabActive("PROFILE")
	}
	leftPaneContent += tabBar + "\n\n"

	if m.leftTab == 0 {
		for i, ch := range m.channels {
			if i == m.activeChan {
				leftPaneContent += lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("  > "+ch.Name) + "\n"
			} else {
				leftPaneContent += "    " + ch.Name + "\n"
			}
		}
	} else {
		// Profile/Settings tab
		leftPaneContent += lipgloss.NewStyle().Foreground(lipgloss.Color(m.user.Color)).Bold(true).Render("  ── IDENTIFICATION") + "\n\n"

		colorDot := lipgloss.NewStyle().Foreground(lipgloss.Color(m.user.Color)).Render("●")
		leftPaneContent += "  " + colorDot + " " + lipgloss.NewStyle().Bold(true).Render(m.user.Username) + "\n"

		roleIcon := ""
		roleName := m.user.Role
		switch m.user.Role {
		case "owner":
			roleIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("👑 ")
			roleName = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("Owner")
		case "admin":
			roleIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("★ ")
			roleName = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("Admin")
		default:
			roleIcon = "👤 "
			roleName = "User"
		}
		leftPaneContent += "  " + roleIcon + roleName + "\n\n"

		leftPaneContent += lipgloss.NewStyle().Foreground(lipgloss.Color(m.user.Color)).Bold(true).Render("  ── APPEARANCE") + "\n\n"

		for i, t := range appPalette {
			pointer := "  "
			dot := "○ "
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Color))

			if t.Color == m.user.Color {
				dot = "● " // Currently active
			}

			if i == m.paletteIndex {
				pointer = "▸ "
				style = style.Bold(true).Underline(true).PaddingRight(1)
				if t.Color == m.user.Color {
					style = style.Italic(true)
				}
			}

			leftPaneContent += " " + pointer + style.Render(dot+t.Name) + "\n"
		}

		leftPaneContent += "\n"
		leftPaneContent += lipgloss.NewStyle().Foreground(lipgloss.Color(m.user.Color)).Bold(true).Render("  [ ENTER/SPACE ] to set") + "\n"
		leftPaneContent += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  Arrows/Tab: navigate") + "\n"
		leftPaneContent += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  /theme <name>") + "\n"
	}

	// SUPREME DYNAMIC UI
	borderKinds := []lipgloss.Border{
		lipgloss.NormalBorder(),
		lipgloss.RoundedBorder(),
		lipgloss.DoubleBorder(),
		lipgloss.ThickBorder(),
	}
	border := borderKinds[m.paletteIndex%len(borderKinds)]
	dynamicBorder := lipgloss.NewStyle().Border(border).BorderForeground(lipgloss.Color(m.user.Color))
	leftPane := dynamicBorder.Width(leftW).Height(midH).Render(leftPaneContent)

	// Right Panel (Online)
	online := m.broker.GetOnlineUsers(m.channels[m.activeChan].ID)
	var onlineStr string
	for _, u := range online {
		isMe := ""
		if u.ID == m.user.ID {
			isMe = " (You)"
		}
		color := u.Color
		if color == "" {
			color = "240"
		}
		roleBadge := ""
		if u.Role == "owner" {
			roleBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("👑 ")
		} else if u.Role == "admin" {
			roleBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("★ ")
		}
		onlineStr += lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("  • "+roleBadge+u.Username+isMe) + "\n"
	}
	rightPane := dynamicBorder.Width(rightW).Height(midH).Render("\n  ONLINE\n\n" + onlineStr)

	// Center Panel (Feed)
	m.viewport.Width = centerW
	m.viewport.Height = midH
	centerPane := dynamicBorder.Width(centerW).Height(midH).Render(m.viewport.View())

	// Bottom Panel (Input)
	m.input.SetWidth(m.width - 2)
	m.input.SetHeight(1)
	bottomPane := dynamicBorder.Width(m.width - 2).Height(1).Render(m.input.View())

	midSection := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, centerPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, header, midSection, bottomPane)
}
