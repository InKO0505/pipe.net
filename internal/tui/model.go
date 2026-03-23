// internal/tui/model.go
package tui

import (
	"encoding/base64"
	"fmt"
	"strings"

	"clinet/internal/db"
	"clinet/internal/pubsub"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
)

type state int

const (
	questState state = iota
	mainState
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

	msgSub   chan db.Message
	isSubbed bool
}

type newMsgMsg db.Message

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

	m := &Model{
		database:   database,
		broker:     broker,
		user:       user,
		session:    s,
		secretMsg:  base64.StdEncoding.EncodeToString([]byte(secretPlainText)),
		questInput: questIn,
		input:      mainIn,
		viewport:   vp,
	}

	if user.IsVerified {
		m.appState = mainState
	} else {
		m.appState = questState
	}

	m.channels = database.GetChannels()
	if len(m.channels) > 0 {
		m.loadChannelMessages()
		m.msgSub = m.broker.Subscribe(m.channels[m.activeChan].ID)
		m.isSubbed = true
	}

	return m
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
		userStr := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true).Render(msg.Username)
		b.WriteString(fmt.Sprintf("[%s] %s: %s\n", timeStr, userStr, msg.Content))
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

		case tea.KeyEnter:
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

			// In MainState: Broadcast Message
			content := strings.TrimSpace(m.input.Value())
			if content != "" && len(m.channels) > 0 {
				chID := m.channels[m.activeChan].ID
				newMsg := m.database.CreateMessage(chID, m.user.ID, content)
				m.broker.Broadcast(chID, newMsg)
				m.input.Reset()
			}
			return m, nil

		case tea.KeyTab:
			if m.appState == mainState && len(m.channels) > 0 {
				if m.isSubbed && m.msgSub != nil {
					m.broker.Unsubscribe(m.channels[m.activeChan].ID, m.msgSub)
					m.isSubbed = false
				}
				m.activeChan = (m.activeChan + 1) % len(m.channels)
				m.loadChannelMessages()
				m.msgSub = m.broker.Subscribe(m.channels[m.activeChan].ID)
				m.isSubbed = true
				return m, m.waitForMessages()
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
		m.messages = append(m.messages, db.Message(msg))
		m.updateViewportContent()
		return m, m.waitForMessages()
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

var (
	leftPaneStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238"))
	centerPaneStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238"))
	bottomPaneStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238"))
)

func (m *Model) View() string {
	if m.width == 0 {
		return "Initializing...\n"
	}

	if m.appState == questState {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fmt.Sprintf(
			"Welcome to CLI-Net! Prove you are worthy.\n\nDecode this base64 string: %s\n\n%s\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("211")).Render(m.secretMsg),
			m.questInput.View(),
		))
	}

	leftW := int(float64(m.width) * 0.2)
	rightW := m.width - leftW - 4
	topH := int(float64(m.height) * 0.8)
	bottomH := m.height - topH - 4

	var channelsStr string
	for i, ch := range m.channels {
		if i == m.activeChan {
			channelsStr += lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("> "+ch.Name) + "\n"
		} else {
			channelsStr += "  " + ch.Name + "\n"
		}
	}

	// Layout Rendering
	leftPane := leftPaneStyle.Width(leftW).Height(m.height - 2).Render("\n" + channelsStr)

	m.viewport.Width = rightW
	m.viewport.Height = topH
	centerPane := centerPaneStyle.Width(rightW).Height(topH).Render(m.viewport.View())

	m.input.SetWidth(rightW)
	m.input.SetHeight(bottomH)
	bottomPane := bottomPaneStyle.Width(rightW).Height(bottomH).Render(m.input.View())

	rightPane := lipgloss.JoinVertical(lipgloss.Left, centerPane, bottomPane)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}
