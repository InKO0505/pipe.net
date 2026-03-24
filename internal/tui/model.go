// internal/tui/model.go
package tui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strconv"
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
	msgSub     chan db.Message
	isSubbed   bool
	renderer *glamour.TermRenderer

	commands    []string
	tabPrefix   string
	tabMatchIdx int
}


type newMsgMsg db.Message
type imageFetchedMsg struct {
	url    string
	kitty  string
	ansi   string // FALLBACK
	status string // DEBUG
}

type imageAsset struct {
	kitty string
	ansi  string
}

var globalImageCache sync.Map // url -> imageAsset ("" = failed)

func fetchImageCmd(url string, targetWidth int) tea.Cmd {
	return func() tea.Msg {
		if v, ok := globalImageCache.Load(url); ok {
			asset := v.(imageAsset)
			return imageFetchedMsg{url: url, kitty: asset.kitty, ansi: asset.ansi}
		}

		var data []byte
		var err error

		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			resp, httpErr := http.Get(url) //nolint:gosec
			if httpErr != nil {
				globalImageCache.Store(url, imageAsset{})
				return imageFetchedMsg{url: url}
			}
			defer resp.Body.Close()
			data, err = io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB limit
		} else {
			f, openErr := os.Open(url)
			if openErr != nil {
				globalImageCache.Store(url, imageAsset{})
				return imageFetchedMsg{url: url}
			}
			defer f.Close()
			data, err = io.ReadAll(io.LimitReader(f, 10<<20)) // 10 MB limit
		}

		if err != nil {
			globalImageCache.Store(url, imageAsset{})
			return imageFetchedMsg{url: url}
		}

		// Decode image for ANSI fallback
		img, _, decodeErr := image.Decode(bytes.NewReader(data))
		ansi := ""
		if decodeErr == nil {
			ansi = renderHalfBlocks(img, targetWidth)
		}

		kitty := encodeKittyImage(data)
		globalImageCache.Store(url, imageAsset{kitty: kitty, ansi: ansi})
		status := fmt.Sprintf("Image loaded. Kitty size: %d, ANSI size: %d", len(kitty), len(ansi))
		return imageFetchedMsg{url: url, kitty: kitty, ansi: ansi, status: status}
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

func (m *Model) currentWidth() int {
	return int(float64(m.width) * 0.6)
}

func renderHalfBlocks(img image.Image, targetWidth int) string {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if targetWidth > w {
		targetWidth = w
	}
	if targetWidth < 10 {
		targetWidth = 10
	}
	scale := float64(w) / float64(targetWidth)
	targetHeight := int(float64(h) / scale)
	if targetHeight%2 != 0 {
		targetHeight--
	}

	var sb strings.Builder
	for y := 0; y < targetHeight; y += 2 {
		for x := 0; x < targetWidth; x++ {
			srcX := int(float64(x) * scale)
			srcY1 := int(float64(y) * scale)
			srcY2 := int(float64(y+1) * scale)

			c1 := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY1)
			c2 := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY2)

			r1, g1, b1, _ := c1.RGBA()
			r2, g2, b2, _ := c2.RGBA()

			fg := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r1>>8, g1>>8, b1>>8))
			bg := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r2>>8, g2>>8, b2>>8))

			sb.WriteString(lipgloss.NewStyle().Foreground(fg).Background(bg).Render("▀"))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func NewModel(database *db.DB, broker *pubsub.Broker, user *db.User, s ssh.Session) *Model {
	questIn := textarea.New()
	questIn.Placeholder = "Decode the secret string..."
	questIn.Focus()
	questIn.CharLimit = 50

	mainIn := textarea.New()
	mainIn.Placeholder = "Message..."
	mainIn.Focus()
	mainIn.CharLimit = 2000

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

	if m.user.Color == "" {
		m.user.Color = "#6366F1"
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

	m.commands = []string{
		"/nick", "/clear", "/help", "/img", "/op", "/deop",
		"/kick", "/ban", "/unban", "/newchan", "/delchan",
		"/topic", "/del", "/setowner",
	}

	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.waitForMessages(),
	)
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
				asset := v.(imageAsset)
				if asset.kitty == "" && asset.ansi == "" {
					b.WriteString(fmt.Sprintf("[%s] %s: ❌ Image failed to load (%s)\n", timeStr, userStr, url))
				} else {
					// Prefer ANSI for universal support as requested
					imgStr := asset.ansi
					if imgStr == "" {
						imgStr = asset.kitty
					}
					b.WriteString(fmt.Sprintf("[%s] %s: 🖼️\n%s\n", timeStr, userStr, imgStr))
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

		case tea.KeyEnter:
			if m.appState == bannedState {
				return m, tea.Quit
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
					case "/nick":
						if arg != "" {
							m.database.UpdateUsername(m.user.ID, arg)
							m.user.Username = arg
							m.appendSystemMsg("Nickname changed to " + arg)
						}
					case "/clear":
						if isAdmin {
							chID := m.channels[m.activeChan].ID
							_ = m.database.ClearChannelMessages(chID)
							m.broker.Broadcast(chID, db.Message{ID: "CMD_CLEAR", ChannelID: chID})
						} else {
							m.messages = []db.Message{}
							m.updateViewportContent()
						}
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
							help += border("│ ") + cmd("/clear") + "         clear chat for all" + "\n"
							help += border("│ ") + cmd("/del [count]") + "   delete last N messages" + "\n"
							help += border("│ ") + cmd("/topic <text>") + "          set channel topic" + "\n"
						}
						if isOwner {
							help += border("├── ") + title("OWNER") + border(" ───────────────────────────────────┤") + "\n"
							help += border("│ ") + cmd("/op /deop <name>") + "        manage admins" + "\n"
							help += border("│ ") + cmd("/newchan <name>") + "       create channel" + "\n"
							help += border("│ ") + cmd("/delchan <name>") + "       delete channel" + "\n"
							help += border("│ ") + cmd("/setowner <name>") + "      transfer ownership" + "\n"
						}
						help += border("├── ") + title("KEYS") + border(" ─────────────────────────────────────┤") + "\n"
						help += border("│ ") + cmd("Tab") + " switch-chan  " + cmd("Ctrl+C") + " quit  " + cmd("Ctrl+Y") + " copy" + "\n"
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
					case "/delchan":
						if isOwner {
							if arg == "" {
								m.appendSystemMsg("Usage: /delchan <name>")
							} else {
								targetChan, err := m.database.GetChannelByName(arg)
								if err == nil {
									if len(m.channels) <= 1 {
										m.appendSystemMsg("Cannot delete the last remaining channel.")
									} else {
										isDeletingActive := m.channels[m.activeChan].ID == targetChan.ID
										oldID := m.channels[m.activeChan].ID

										err := m.database.DeleteChannel(targetChan.ID)
										if err != nil {
											m.appendSystemMsg("Error deleting channel: " + err.Error())
										} else {
											m.appendSystemMsg("Channel " + arg + " deleted.")
											m.channels = m.database.GetChannels()

											if isDeletingActive {
												if m.isSubbed && m.msgSub != nil {
													m.broker.Unsubscribe(targetChan.ID, m.msgSub)
													m.isSubbed = false
												}
												m.activeChan = 0
												m.loadChannelMessages()
												m.msgSub = m.broker.Subscribe(m.channels[m.activeChan].ID, m.user)
												m.isSubbed = true
												cmds = append(cmds, m.waitForMessages())
											} else {
												// Find the new index of our old channel
												for i, c := range m.channels {
													if c.ID == oldID {
														m.activeChan = i
														break
													}
												}
											}
										}
									}
								} else {
									m.appendSystemMsg("Channel not found.")
								}
							}
						} else {
							m.appendSystemMsg("Owner privileges required.")
						}
					case "/del":
						if isAdmin {
							count := 1
							if arg != "" {
								if c, err := strconv.Atoi(arg); err == nil && c > 0 {
									count = c
								}
							}
							chID := m.channels[m.activeChan].ID
							for i := 0; i < count; i++ {
								_ = m.database.DeleteLastMessage(chID)
							}
							m.broker.Broadcast(chID, db.Message{ID: "CMD_DEL_LAST", ChannelID: chID, Content: strconv.Itoa(count)})
						} else {
							m.appendSystemMsg("Admin privileges required.")
						}
					case "/topic":
						if isAdmin {
							if arg == "" {
								m.appendSystemMsg("Usage: /topic <text>")
							} else {
								ch := m.channels[m.activeChan]
								m.database.SetChannelTopic(ch.ID, arg)
								m.channels[m.activeChan].Topic = arg
								m.broker.Broadcast(ch.ID, db.Message{ID: "CMD_TOPIC", ChannelID: ch.ID, Content: arg})
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
								cmds = append(cmds, fetchImageCmd(arg, m.currentWidth()))
							}
						}
						handled = true
					default:
						// If not a recognized command, treat as regular message
						handled = false
					}
				}

				if !handled {
					chID := m.channels[m.activeChan].ID
					newMsg := m.database.CreateMessage(chID, m.user.ID, content)
					m.broker.Broadcast(chID, newMsg)
				}
				m.input.Reset()
			}
			return m, tea.Batch(cmds...)

		case tea.KeyTab:
			if m.appState == mainState {
				val := m.input.Value()
				if strings.HasPrefix(val, "/") && !strings.Contains(val, " ") {
					// Autocomplete mode
					if m.tabPrefix == "" {
						m.tabPrefix = val
						m.tabMatchIdx = 0
					}

					var matches []string
					for _, cmd := range m.commands {
						if strings.HasPrefix(cmd, m.tabPrefix) {
							matches = append(matches, cmd)
						}
					}

					if len(matches) > 0 {
						m.input.SetValue(matches[m.tabMatchIdx%len(matches)] + " ")
						m.input.CursorEnd()
						m.tabMatchIdx++
					}
					return m, nil
				}

				// Reset autocomplete if not in it
				m.tabPrefix = ""

				// Default channel switching
				if len(m.channels) > 0 {
					if m.isSubbed && m.msgSub != nil {
						m.broker.Unsubscribe(m.channels[m.activeChan].ID, m.msgSub)
						m.isSubbed = false
					}
					m.activeChan = (m.activeChan + 1) % len(m.channels)
					m.loadChannelMessages()
					m.msgSub = m.broker.Subscribe(m.channels[m.activeChan].ID, m.user)
					m.isSubbed = true
					return m, m.waitForMessages()
				}
			}

		case tea.KeyRunes:
			// Reset autocomplete when typing
			m.tabPrefix = ""


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
		if msg.ID == "CMD_DEL_LAST" {
			count, _ := strconv.Atoi(msg.Content)
			if count <= 0 {
				count = 1
			}
			for i := 0; i < count; i++ {
				if len(m.messages) > 0 {
					m.messages = m.messages[:len(m.messages)-1]
				}
			}
			m.updateViewportContent()
			cmds = append(cmds, m.waitForMessages())
			return m, tea.Batch(cmds...)
		}
		if msg.ID == "CMD_CLEAR" {
			m.messages = []db.Message{}
			m.updateViewportContent()
			cmds = append(cmds, m.waitForMessages())
			return m, tea.Batch(cmds...)
		}
		if msg.ID == "CMD_TOPIC" {
			for i, c := range m.channels {
				if c.ID == msg.ChannelID {
					m.channels[i].Topic = msg.Content
					break
				}
			}
			m.updateViewportContent()
			cmds = append(cmds, m.waitForMessages())
			return m, tea.Batch(cmds...)
		}
		newMsg := db.Message(msg)
		m.messages = append(m.messages, newMsg)
		// If it's an image, launch async fetch
		if strings.HasPrefix(newMsg.Content, "🖼️ ") {
			url := strings.TrimPrefix(newMsg.Content, "🖼️ ")
			if _, ok := globalImageCache.Load(url); !ok {
				cmds = append(cmds, fetchImageCmd(url, m.currentWidth()))
			}
		}
		m.updateViewportContent()
		cmds = append(cmds, m.waitForMessages())
		return m, tea.Batch(cmds...)

	case imageFetchedMsg:
		if msg.status != "" {
			m.appendSystemMsg(msg.status)
		}
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
	
	// Smart Scaling for Input
	inputH := 1
	if m.height >= 40 {
		inputH = 5
	} else if m.height >= 25 {
		inputH = 3
	}
	
	// Guard: Ensure chat feed (midH) always has at least 8 lines
	footerH := inputH + 2 // 2 for borders
	if m.height-headerH-footerH < 8 {
		inputH = 1
		footerH = 3
	}
	
	midH := m.height - headerH - footerH
	if midH < 5 {
		midH = 5
	}

	// Header Panel
	ch := m.channels[m.activeChan]
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color(m.user.Color)).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center)

	topicStr := "No topic set"
	if ch.Topic != "" {
		topicStr = ch.Topic
	}

	headerText := fmt.Sprintf(" %s  •  Topic: %s ", ch.Name, topicStr)
	header := headerStyle.Render(headerText)

	// Left Panel (Channels)
	var leftPaneContent string
	leftPaneContent += lipgloss.NewStyle().Foreground(lipgloss.Color(m.user.Color)).Bold(true).Render("  CHANNELS") + "\n\n"

	for i, ch := range m.channels {
		if i == m.activeChan {
			leftPaneContent += lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("  > "+ch.Name) + "\n"
		} else {
			leftPaneContent += "    " + ch.Name + "\n"
		}
	}

	border := lipgloss.RoundedBorder()
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
	m.input.SetHeight(inputH)
	bottomPane := dynamicBorder.Width(m.width - 2).Height(inputH).Render(m.input.View())

	midSection := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, centerPane, rightPane)
	return lipgloss.JoinVertical(lipgloss.Left, header, midSection, bottomPane)
}
