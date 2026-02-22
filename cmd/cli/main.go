package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syncra/internal/client/storage"
	clientWS "syncra/internal/client/websocket"
	"syncra/internal/config"
	"syncra/internal/crypto"
	"syncra/internal/models"
	"syncra/internal/server/database"
	"syncra/internal/ui"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateSetupWorkspace state = iota
	stateSetupUsername
	stateSetupFullName
	stateSetupProcessing
	stateSuccess
	stateMain
	stateSettings
	stateSearch
	stateChat
	stateConfirmPurge
)

type model struct {
	state         state
	textInput     textinput.Model
	nameInput     textinput.Model
	settingsIndex int
	cfg           *config.Config
	err           error
	startTime     time.Time
	successMsg    string
	quitting      bool
	suspending    bool

	// Temp setup data
	tempWorkspace string
	tempUsername  string
	tempFullName  string

	// Search data
	searchInput   textinput.Model
	searchResults []*models.User

	// Chat data
	chatInput    textinput.Model
	chatTarget   string
	chatMessages []models.LocalChatMessage
	conn         *clientWS.Connection

	// App state
	reconnecting bool
	spinner      spinner.Model

	// Friends list
	chats              []string
	chatSelectionIndex int
}

type reconnectMsg struct{}

func initialModel() model {
	// Try to load existing config
	cfg, _ := config.LoadConfig()

	ti := textinput.New()
	ti.Placeholder = "workspace path..."
	ti.CharLimit = 156
	ti.Width = 50
	ti.TextStyle = ui.InputStyle

	home, _ := os.UserHomeDir()
	if home != "" {
		ti.SetValue(home)
	}

	ni := textinput.New()
	ni.Placeholder = "full name..."
	ni.CharLimit = 156
	ni.Width = 50
	ni.TextStyle = ui.InputStyle

	si := textinput.New()
	si.Placeholder = "search username..."
	si.CharLimit = 156
	si.Width = 50
	si.TextStyle = ui.InputStyle

	ci := textinput.New()
	ci.Placeholder = "type a message..."
	ci.CharLimit = 1000
	ci.Width = 60
	ci.TextStyle = ui.InputStyle

	m := model{
		startTime:     time.Now(),
		cfg:           cfg,
		textInput:     ti,
		nameInput:     ni,
		searchInput:   si,
		chatInput:     ci,
		searchResults: []*models.User{},
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.Primary)
	m.spinner = s

	chats, _ := storage.ListChats()
	m.chats = chats

	if cfg == nil || cfg.Username == "" {
		m.state = stateSetupWorkspace
		m.textInput.Focus()
	} else {
		m.state = stateMain
		// Auto-connect
		conn, err := clientWS.Connect("localhost:8080")
		if err == nil {
			m.conn = conn
			go m.conn.WritePump()
		}
	}

	return m
}

type wsMessage struct {
	packet models.Packet
}

type wsErrorMsg struct {
	err error
}

func (m model) listenWS() tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return wsErrorMsg{err: fmt.Errorf("no connection")}
		}
		for {
			_, data, err := m.conn.Conn.ReadMessage()
			if err != nil {
				return wsErrorMsg{err: err}
			}
			var p models.Packet
			if err := json.Unmarshal(data, &p); err != nil {
				continue
			}
			return wsMessage{packet: p}
		}
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, tea.EnterAltScreen)
	if m.state == stateSetupWorkspace || m.state == stateSetupUsername || m.state == stateSetupFullName {
		cmds = append(cmds, textinput.Blink)
	}
	if m.conn != nil {
		cmds = append(cmds, m.listenWS())
	} else if m.state == stateMain {
		// Not connected at start, trigger reconnect loop
		cmds = append(cmds, func() tea.Msg { return reconnectMsg{} })
	}
	if m.state == stateSetupProcessing {
		cmds = append(cmds, m.spinner.Tick)
	}
	return tea.Batch(cmds...)
}

type setupResult struct {
	err error
	cfg *config.Config
}

func (m model) performSetup() tea.Cmd {
	return func() tea.Msg {
		// 1. Connect to DB
		db, err := database.Connect()
		if err != nil {
			return setupResult{err: fmt.Errorf("failed to connect to server: %v", err)}
		}
		defer db.Close()

		ctx := context.Background()

		// 2. Validate Username uniqueness (again, just to be sure)
		taken, err := db.IsUsernameTaken(ctx, m.tempUsername)
		if err != nil {
			return setupResult{err: fmt.Errorf("failed to validate username: %v", err)}
		}
		if taken {
			return setupResult{err: fmt.Errorf("username already taken")}
		}

		// 3. Generate Key Pair
		pub, priv, err := crypto.GenerateKeyPair()
		if err != nil {
			return setupResult{err: fmt.Errorf("failed to generate keys: %v", err)}
		}

		// 4. Initialize local structure
		if err := config.InitializeStructure(m.tempWorkspace); err != nil {
			return setupResult{err: fmt.Errorf("failed to create folders: %v", err)}
		}

		// 5. Save Private Key locally
		keyPath := filepath.Join(m.tempWorkspace, "syncra", "identities", "id_ed25519")
		if err := crypto.SavePrivateKey(keyPath, priv); err != nil {
			return setupResult{err: fmt.Errorf("failed to save private key: %v", err)}
		}

		// 6. Register on Server
		pubHash := crypto.HashPublicKey(pub)
		pubHex := hex.EncodeToString(pub)
		user := &models.User{
			Username:      m.tempUsername,
			FullName:      m.tempFullName,
			PublicKey:     pubHex,
			PublicKeyHash: pubHash,
		}

		if err := db.CreateUser(ctx, user); err != nil {
			// Cleanup: delete private key if registration fails?
			// For now, just return error
			return setupResult{err: fmt.Errorf("failed to register user on server: %v", err)}
		}

		// 7. Save Config
		cfg := &config.Config{
			WorkspacePath: m.tempWorkspace,
			Username:      m.tempUsername,
			FullName:      m.tempFullName,
		}
		if err := config.SaveConfig(cfg); err != nil {
			return setupResult{err: fmt.Errorf("failed to save config: %v", err)}
		}

		return setupResult{cfg: cfg}
	}
}

type searchResult struct {
	users []*models.User
	err   error
}

func (m model) performSearch(query string) tea.Cmd {
	return func() tea.Msg {
		db, err := database.Connect()
		if err != nil {
			return searchResult{err: fmt.Errorf("failed to connect to server: %v", err)}
		}
		defer db.Close()

		users, err := db.SearchUsers(context.Background(), query)
		if err != nil {
			return searchResult{err: fmt.Errorf("search failed: %v", err)}
		}

		return searchResult{users: users}
	}
}

type usernameCheckResult struct {
	exists bool
	err    error
}

func checkUsername(username string) tea.Cmd {
	return func() tea.Msg {
		db, err := database.Connect()
		if err != nil {
			return usernameCheckResult{err: err}
		}
		defer db.Close()

		taken, err := db.IsUsernameTaken(context.Background(), username)
		return usernameCheckResult{exists: taken, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.ResumeMsg:
		m.suspending = false
		return m, nil

	case setupResult:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateSetupFullName // Fallback to last valid state or error state
			return m, nil
		}
		m.cfg = msg.cfg
		m.state = stateSuccess
		return m, nil

	case wsMessage:
		p := msg.packet
		switch p.Type {
		case models.TypeChallenge:
			var challenge string
			json.Unmarshal(p.Payload, &challenge)
			// Sign challenge
			keyPath := filepath.Join(m.cfg.WorkspacePath, "syncra", "identities", "id_ed25519")
			priv, _ := crypto.LoadPrivateKey(keyPath)
			sig := crypto.Sign(priv, []byte(challenge))
			// Send Auth
			auth := models.AuthPayload{Username: m.cfg.Username, Signature: sig}
			authData, _ := json.Marshal(auth)
			m.conn.Send <- models.Packet{Type: models.TypeAuth, Payload: authData}
		case models.TypeChat:
			var chat models.ChatPayload
			json.Unmarshal(p.Payload, &chat)
			localMsg := models.LocalChatMessage{
				From:      p.From,
				Content:   chat.Message,
				Timestamp: p.Timestamp,
				IsMe:      false,
			}
			storage.AppendMessage(p.From, localMsg)
			if m.state == stateChat && m.chatTarget == p.From {
				m.chatMessages = append(m.chatMessages, localMsg)
			}
			// Refresh chats list
			m.chats, _ = storage.ListChats()
		case models.TypeError:
			var errMsg string
			json.Unmarshal(p.Payload, &errMsg)
			m.err = fmt.Errorf("%s", errMsg)
		}
		return m, m.listenWS()

	case wsErrorMsg:
		m.conn = nil
		// Try to reconnect after 2 seconds
		return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
			return reconnectMsg{}
		})

	case reconnectMsg:
		if m.cfg == nil || m.cfg.Username == "" {
			return m, nil
		}
		conn, err := clientWS.Connect("localhost:8080")
		if err == nil {
			m.conn = conn
			go m.conn.WritePump()
			return m, m.listenWS()
		}
		// If failed, wait and try again
		return m, tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
			return reconnectMsg{}
		})

	case usernameCheckResult:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if msg.exists {
			m.err = fmt.Errorf("username '%s' is already taken", m.tempUsername)
			m.state = stateSetupUsername
			m.textInput.Focus()
			return m, nil
		}
		// Username ok, proceed to Full Name
		m.state = stateSetupFullName
		m.textInput.Reset()
		m.textInput.Placeholder = "full name..."
		m.textInput.Focus()
		return m, nil

	case searchResult:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.searchResults = msg.users
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "ctrl+z":
			m.suspending = true
			return m, tea.Suspend
		case "s":
			if m.state == stateMain {
				m.state = stateSettings
				m.settingsIndex = 0
				m.textInput.SetValue(m.cfg.WorkspacePath)
				m.nameInput.SetValue(m.cfg.FullName)
				m.textInput.Focus()
				m.nameInput.Blur()
				return m, textinput.Blink
			}
		case "f":
			if m.state == stateMain {
				m.state = stateSearch
				m.searchInput.Focus()
				m.searchInput.Reset()
				m.searchResults = []*models.User{}
				m.err = nil
				return m, textinput.Blink
			}
		case "esc":
			if m.state == stateSettings || m.state == stateSearch || m.state == stateChat {
				m.state = stateMain
				return m, nil
			}
		case "x":
			if m.state == stateSettings {
				m.state = stateConfirmPurge
				return m, nil
			}
		}

		switch m.state {
		case stateMain:
			if msg.Type == tea.KeyUp {
				if m.chatSelectionIndex > 0 {
					m.chatSelectionIndex--
				}
			} else if msg.Type == tea.KeyDown {
				if m.chatSelectionIndex < len(m.chats)-1 {
					m.chatSelectionIndex++
				}
			} else if msg.Type == tea.KeyEnter && len(m.chats) > 0 {
				m.chatTarget = m.chats[m.chatSelectionIndex]
				m.state = stateChat
				m.chatMessages, _ = storage.LoadMessages(m.chatTarget)
				m.chatInput.Focus()
				return m, nil
			}
		case stateSetupWorkspace:
			if msg.Type == tea.KeyEnter {
				path := m.textInput.Value()
				if path == "" {
					m.err = fmt.Errorf("path cannot be empty")
					return m, nil
				}
				m.tempWorkspace = path
				m.err = nil
				m.state = stateSetupUsername
				m.textInput.Reset()
				m.textInput.Placeholder = "choose username..."
				m.textInput.Focus()
				return m, nil
			}

		case stateSetupUsername:
			if msg.Type == tea.KeyEnter {
				username := m.textInput.Value()
				if username == "" {
					m.err = fmt.Errorf("username cannot be empty")
					return m, nil
				}
				m.tempUsername = username
				m.err = nil
				return m, checkUsername(username)
			}

		case stateSetupFullName:
			if msg.Type == tea.KeyEnter {
				fullName := m.textInput.Value()
				if fullName == "" {
					m.err = fmt.Errorf("full name cannot be empty")
					return m, nil
				}
				m.tempFullName = fullName
				m.err = nil
				m.state = stateSetupProcessing
				return m, tea.Batch(m.performSetup(), m.spinner.Tick)
			}

		case stateSettings:
			if msg.Type == tea.KeyTab || msg.Type == tea.KeyShiftTab || msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
				if m.settingsIndex == 0 {
					m.settingsIndex = 1
					m.textInput.Blur()
					m.nameInput.Focus()
				} else {
					m.settingsIndex = 0
					m.nameInput.Blur()
					m.textInput.Focus()
				}
				return m, textinput.Blink
			}

			if msg.Type == tea.KeyEnter {
				path := m.textInput.Value()
				fullName := m.nameInput.Value()

				if path == "" || fullName == "" {
					m.err = fmt.Errorf("fields cannot be empty")
					return m, nil
				}

				// 1. Update Workspace (Local Only)
				if path != m.cfg.WorkspacePath {
					err := config.InitializeStructure(path)
					if err != nil {
						m.err = err
						return m, nil
					}
					m.cfg.WorkspacePath = path
				}

				// 2. Update Full Name (Local & Server)
				if fullName != m.cfg.FullName {
					db, err := database.Connect()
					if err != nil {
						m.err = fmt.Errorf("failed to connect to server: %v", err)
						return m, nil
					}
					defer db.Close()

					err = db.UpdateFullName(context.Background(), m.cfg.Username, fullName)
					if err != nil {
						m.err = fmt.Errorf("failed to update server: %v", err)
						return m, nil
					}
					m.cfg.FullName = fullName
				}

				config.SaveConfig(m.cfg)
				m.state = stateMain
				m.err = nil
				return m, nil
			}

			if m.settingsIndex == 0 {
				m.textInput, cmd = m.textInput.Update(msg)
			} else {
				m.nameInput, cmd = m.nameInput.Update(msg)
			}
			return m, cmd

		case stateSearch:
			if msg.Type == tea.KeyEnter {
				query := m.searchInput.Value()
				// If we have search results, enter starts a chat with the first one for now
				if len(m.searchResults) > 0 {
					m.chatTarget = m.searchResults[0].Username
					m.state = stateChat
					m.chatMessages, _ = storage.LoadMessages(m.chatTarget)
					m.chatInput.Focus()
					return m, nil
				}
				if query != "" {
					return m, m.performSearch(query)
				}
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd

		case stateChat:
			if msg.Type == tea.KeyEnter {
				content := m.chatInput.Value()
				if content != "" {
					// 1. Send via WebSocket
					pkg := models.Packet{
						Type:    models.TypeChat,
						To:      m.chatTarget,
						Payload: json.RawMessage(`{"message": "` + content + `"}`),
					}
					if m.conn != nil {
						m.conn.Send <- pkg
					}

					// 2. Storage Locally
					localMsg := models.LocalChatMessage{
						From:      m.cfg.Username,
						Content:   content,
						Timestamp: time.Now(),
						IsMe:      true,
					}
					storage.AppendMessage(m.chatTarget, localMsg)
					m.chatMessages = append(m.chatMessages, localMsg)
					m.chatInput.Reset()
					// Refresh chats list
					m.chats, _ = storage.ListChats()
				}
			}
			m.chatInput, cmd = m.chatInput.Update(msg)
			return m, cmd

		case stateConfirmPurge:
			if msg.String() == "y" {
				return m, m.performSelfDestruct()
			}
			if msg.String() == "n" || msg.String() == "esc" {
				m.state = stateSettings
				return m, nil
			}
			return m, nil

		case stateSuccess:
			if msg.Type == tea.KeyEnter {
				m.state = stateMain
				m.startTime = time.Now()
				// Connect on launch
				conn, _ := clientWS.Connect("localhost:8080")
				if conn != nil {
					m.conn = conn
					go m.conn.WritePump()
					return m, m.listenWS()
				}
				return m, nil
			}
		}

		if m.state == stateSetupProcessing {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	case error:
		m.err = msg
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.suspending {
		return ""
	}

	if m.quitting {
		return "Bye!\n"
	}

	var header, subHeader, body, footer string

	header = ui.HeaderStyle.Render("SYNCRA") + "\n"

	var content string
	switch m.state {
	case stateSetupWorkspace:
		subHeader = ui.SubHeaderStyle.Render("setup / workspace") + "\n"
		inner := ui.MutedStyle.Render("Where should Syncra live?") + "\n\n"
		inner += ui.InfoKeyStyle.Render("path") + "\n" + m.textInput.View()
		if m.err != nil {
			inner += "\n\n" + ui.ErrorTextStyle.Render("! "+m.err.Error())
		}
		content = inner
		footer = ui.FooterStyle.Render("enter: next • esc: quit")

	case stateSetupUsername:
		subHeader = ui.SubHeaderStyle.Render("setup / identity") + "\n"
		inner := ui.MutedStyle.Render("Choose your unique handle.") + "\n\n"
		inner += ui.InfoKeyStyle.Render("username") + "\n" + m.textInput.View()
		if m.err != nil {
			inner += "\n\n" + ui.ErrorTextStyle.Render("! "+m.err.Error())
		}
		content = inner
		footer = ui.FooterStyle.Render("enter: check • esc: quit")

	case stateSetupFullName:
		subHeader = ui.SubHeaderStyle.Render("setup / profile") + "\n"
		inner := ui.MutedStyle.Render("Whatever you like to be called.") + "\n\n"
		inner += ui.InfoKeyStyle.Render("name") + "\n" + m.textInput.View()
		if m.err != nil {
			inner += "\n\n" + ui.ErrorTextStyle.Render("! "+m.err.Error())
		}
		content = inner
		footer = ui.FooterStyle.Render("enter: finish • esc: quit")

	case stateSetupProcessing:
		subHeader = ui.SubHeaderStyle.Render("creating identity...") + "\n"
		content = fmt.Sprintf("\n  %s %s\n", m.spinner.View(), ui.MutedStyle.Render("Generating cryptographic keys & registering..."))
		footer = ""

	case stateSuccess:
		subHeader = ui.SubHeaderStyle.Render("ready") + "\n"
		inner := ui.SuccessStyle().Render("Identity verified.") + "\n\n"
		inner += ui.InfoKeyStyle.Render("user") + ui.InfoValueStyle.Render(m.cfg.Username) + "\n"
		content = inner
		footer = ui.FooterStyle.Render("enter: launch")

	case stateMain:
		subHeader = ui.SubHeaderStyle.Render("active / "+m.cfg.Username) + "\n"

		statusLabel := ui.StatusLabelStyle.Copy().
			Foreground(ui.Success).
			Render("● ONLINE")

		// Status Section
		statusContent := fmt.Sprintf("%s\n\n%s %s\n%s %s",
			statusLabel,
			ui.InfoKeyStyle.Render("identity"), ui.InfoValueStyle.Render(m.cfg.FullName),
			ui.InfoKeyStyle.Render("session"), ui.InfoValueStyle.Render(time.Since(m.startTime).Truncate(time.Second).String()),
		)

		// Chats Section
		friendsList := ""
		if len(m.chats) > 0 {
			friendsList = "\n" + ui.SectionTitleStyle.Render(fmt.Sprintf("RECENT CHATS (%d)", len(m.chats))) + "\n"
			for i, friend := range m.chats {
				cursor := "  "
				style := ui.InfoValueStyle
				if i == m.chatSelectionIndex {
					cursor = lipgloss.NewStyle().Foreground(ui.Primary).Render("» ")
					style = ui.SelectedStyle
				}
				friendsList += fmt.Sprintf("%s %s\n", cursor, style.Render(friend))
			}
		} else {
			friendsList = "\n" + ui.MutedStyle.Render("No recent conversations.")
		}

		content = statusContent + "\n" + friendsList
		footer = ui.FooterStyle.Render("↑/↓: select chat • s: settings • f: find • q: quit")

	case stateSearch:
		subHeader = ui.SubHeaderStyle.Render("find") + "\n"
		inner := ui.InfoKeyStyle.Render("query") + "\n" + m.searchInput.View() + "\n\n"

		if len(m.searchResults) > 0 {
			inner += ui.SectionTitleStyle.Render("RESULTS") + "\n"
			for _, user := range m.searchResults {
				inner += fmt.Sprintf("  %s @%s\n",
					ui.InfoValueStyle.Render(user.FullName),
					ui.MutedStyle.Render(user.Username),
				)
			}
		}

		if m.err != nil {
			inner += "\n\n" + ui.ErrorTextStyle.Render("! "+m.err.Error())
		}
		content = inner
		footer = ui.FooterStyle.Render("enter: search/chat • esc: back")

	case stateChat:
		statusStr := ui.StatusLabelStyle.Foreground(ui.Success).Render("● online")
		if m.conn == nil {
			statusStr = ui.StatusLabelStyle.Foreground(ui.ErrorCol).Render("○ offline")
		}
		subHeader = ui.SubHeaderStyle.Render("chat / "+m.chatTarget+"  "+statusStr) + "\n"

		var chatContent string
		if len(m.chatMessages) == 0 {
			chatContent = ui.MutedStyle.Render("No messages yet. Say hello!")
		} else {
			// Show last 12 messages for better vertical space
			start := 0
			if len(m.chatMessages) > 12 {
				start = len(m.chatMessages) - 12
			}
			for i := start; i < len(m.chatMessages); i++ {
				msg := m.chatMessages[i]
				prefix := lipgloss.NewStyle().Foreground(ui.Secondary).Render("@" + msg.From + ":")
				if msg.IsMe {
					prefix = ui.SelectedStyle.Render("You:")
				}
				chatContent += fmt.Sprintf("%s %s\n", prefix, msg.Content)
			}
		}

		content = chatContent + "\n" + m.chatInput.View()
		footer = ui.FooterStyle.Render("enter: send • esc: back")

	case stateSettings:
		subHeader = ui.SubHeaderStyle.Render("settings") + "\n"

		inner := ui.SectionTitleStyle.Render("WORKSPACE") + "\n" + m.textInput.View() + "\n\n"
		inner += ui.SectionTitleStyle.Render("FULL NAME") + "\n" + m.nameInput.View() + "\n\n"

		if m.err != nil {
			inner += "\n\n" + ui.ErrorTextStyle.Render("! "+m.err.Error())
		}
		content = inner
		footer = ui.FooterStyle.Render("enter: save • x: self-destruct • esc: back")

	case stateConfirmPurge:
		subHeader = ui.SubHeaderStyle.Render("danger / self-destruct") + "\n"
		inner := ui.ErrorTextStyle.Render("ARE YOU ABSOLUTELY SURE?") + "\n\n"
		inner += "This will permanently:\n"
		inner += " - Delete your identity from the server\n"
		inner += " - Delete your local chat history\n"
		inner += " - Delete your cryptographic keys\n"
		inner += " - Reset this application\n\n"
		inner += ui.MutedStyle.Render("Press 'y' to confirm or 'n' to cancel")

		content = inner
		footer = ui.FooterStyle.Render("y/n")
	}

	body = ui.MainContainerStyle.Render(subHeader + content)

	return fmt.Sprintf("%s%s\n%s", header, body, footer)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running Syncra: %v\n", err)
		os.Exit(1)
	}
}
func (m model) performSelfDestruct() tea.Cmd {
	return func() tea.Msg {
		// 1. Delete from Server
		db, err := database.Connect()
		if err == nil {
			db.DeleteUser(context.Background(), m.cfg.Username)
			db.Close()
		}

		// 2. Delete local folder
		syncraPath := filepath.Join(m.cfg.WorkspacePath, "syncra")
		os.RemoveAll(syncraPath)

		// 3. Purge config
		config.PurgeConfig()

		return tea.Quit()
	}
}
