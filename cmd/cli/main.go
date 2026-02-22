package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syncra/internal/config"
	"syncra/internal/crypto"
	"syncra/internal/models"
	"syncra/internal/server/database"
	"syncra/internal/ui"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	altscreen     bool
	quitting      bool
	suspending    bool

	// Temp setup data
	tempWorkspace string
	tempUsername  string
	tempFullName  string
}

func initialModel() model {
	// Try to load existing config
	cfg, _ := config.LoadConfig()

	ti := textinput.New()
	ti.Placeholder = "Enter workspace path..."
	ti.CharLimit = 156
	ti.Width = 50

	home, _ := os.UserHomeDir()
	if home != "" {
		ti.SetValue(home)
	}

	ni := textinput.New()
	ni.Placeholder = "Enter full name..."
	ni.CharLimit = 156
	ni.Width = 50

	m := model{
		startTime: time.Now(),
		cfg:       cfg,
		altscreen: true,
		textInput: ti,
		nameInput: ni,
	}

	if cfg == nil || cfg.Username == "" {
		m.state = stateSetupWorkspace
		m.textInput.Focus()
	} else {
		m.state = stateMain
	}

	return m
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, tea.EnterAltScreen)
	if m.state == stateSetupWorkspace || m.state == stateSetupUsername || m.state == stateSetupFullName {
		cmds = append(cmds, textinput.Blink)
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
		user := &models.User{
			Username:      m.tempUsername,
			FullName:      m.tempFullName,
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
		m.textInput.Placeholder = "Enter your full name..."
		m.textInput.Focus()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "ctrl+z":
			m.suspending = true
			return m, tea.Suspend
		case " ":
			if m.state == stateSetupWorkspace || m.state == stateSetupUsername || m.state == stateSetupFullName || m.state == stateSettings {
				break // Let text input handle space during setup or settings
			}
			var cmd tea.Cmd
			if m.altscreen {
				cmd = tea.ExitAltScreen
			} else {
				cmd = tea.EnterAltScreen
			}
			m.altscreen = !m.altscreen
			return m, cmd
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
		case "esc":
			if m.state == stateSettings {
				m.state = stateMain
				return m, nil
			}
		}

		switch m.state {
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
				m.textInput.Placeholder = "Choose a unique username..."
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
				return m, m.performSetup()
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

		case stateSuccess:
			if msg.Type == tea.KeyEnter {
				m.state = stateMain
				m.startTime = time.Now()
				return m, nil
			}
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

	header = ui.HeaderStyle.Render(" SYNCRA CLI v1.0.0 ") + "\n"

	switch m.state {
	case stateSetupWorkspace:
		subHeader = ui.SubHeaderStyle.Render("Step 1/3: Workspace Path") + "\n"
		content := ui.InfoKeyStyle.Render("Specify the path where your Syncra data will be stored.") + "\n\n"
		content += ui.InfoKeyStyle.Render("Base Path:") + "\n" + m.textInput.View()
		if m.err != nil {
			content += "\n\n" + ui.ErrorTextStyle.Render("✘ Error: "+m.err.Error())
		}
		body = ui.CardStyle.Render(content)
		footer = ui.FooterStyle.Render("▸ Enter: Next • Space: Toggle AltScreen • Ctrl+C: Exit")

	case stateSetupUsername:
		subHeader = ui.SubHeaderStyle.Render("Step 2/3: Identity Setup") + "\n"
		content := ui.InfoKeyStyle.Render("Choose a globally unique username for your identity.") + "\n\n"
		content += ui.InfoKeyStyle.Render("Username:") + "\n" + m.textInput.View()
		if m.err != nil {
			content += "\n\n" + ui.ErrorTextStyle.Render("✘ Error: "+m.err.Error())
		}
		body = ui.CardStyle.Render(content)
		footer = ui.FooterStyle.Render("▸ Enter: Validate • Space: Toggle AltScreen • Ctrl+C: Exit")

	case stateSetupFullName:
		subHeader = ui.SubHeaderStyle.Render("Step 3/3: Profile Details") + "\n"
		content := ui.InfoKeyStyle.Render("Enter your full name to complete your profile.") + "\n\n"
		content += ui.InfoKeyStyle.Render("Full Name:") + "\n" + m.textInput.View()
		if m.err != nil {
			content += "\n\n" + ui.ErrorTextStyle.Render("✘ Error: "+m.err.Error())
		}
		body = ui.CardStyle.Render(content)
		footer = ui.FooterStyle.Render("▸ Enter: Create Identity • Space: Toggle AltScreen • Ctrl+C: Exit")

	case stateSetupProcessing:
		subHeader = ui.SubHeaderStyle.Render("Finalizing Setup") + "\n"
		content := ui.InfoKeyStyle.Render("Generating secure keys and registering your identity...") + "\n\n"
		content += ui.MutedStyle.Render("This may take a moment.")
		body = ui.CardStyle.Render(content)
		footer = ui.FooterStyle.Render("● PROCESSING")

	case stateSuccess:
		subHeader = ui.SubHeaderStyle.Render("Account Created Successfully") + "\n"
		content := ui.SuccessStyle().Render("✓ IDENTITY VERIFIED") + "\n\n"
		content += ui.InfoKeyStyle.Render("Username: ") + ui.InfoValueStyle.Render(m.cfg.Username) + "\n"
		content += ui.InfoKeyStyle.Render("Path:     ") + ui.InfoValueStyle.Render(filepath.Join(m.cfg.WorkspacePath, "syncra")) + "\n\n"
		content += ui.MutedStyle.Render("Your private key is stored securely in your workspace.")
		body = ui.CardStyle.Render(content)
		footer = ui.FooterStyle.Render("▸ Enter: Launch Syncra")

	case stateMain:
		subHeader = ui.SubHeaderStyle.Render("Terminal Suite • "+m.cfg.Username) + "\n"
		statusContent := fmt.Sprintf("%s\n\n%s %s\n%s %s",
			ui.StatusLabelStyle.Foreground(ui.Success).Render("● ACTIVE"),
			ui.InfoKeyStyle.Render("Identity"), ui.InfoValueStyle.Render(m.cfg.FullName),
			ui.InfoKeyStyle.Render("Session"), ui.InfoValueStyle.Foreground(ui.Secondary).Render(time.Since(m.startTime).Truncate(time.Second).String()),
		)
		body = ui.CardStyle.Render(statusContent)
		footer = ui.FooterStyle.Render("▸ 's': settings • Space: toggle mode • Ctrl+Z: suspend • 'q': exit")

	case stateSettings:
		subHeader = ui.SubHeaderStyle.Render("Settings & Configuration") + "\n"

		wsLabel := ui.InfoKeyStyle.Render("Workspace Path:")
		if m.settingsIndex == 0 {
			wsLabel = ui.InfoValueStyle.Render("> Workspace Path:")
		}
		content := wsLabel + "\n" + m.textInput.View() + "\n\n"

		nameLabel := ui.InfoKeyStyle.Render("Full Name:")
		if m.settingsIndex == 1 {
			nameLabel = ui.InfoValueStyle.Render("> Full Name:")
		}
		content += nameLabel + "\n" + m.nameInput.View() + "\n\n"

		content += ui.MutedStyle.Render("Tab/Arrows to navigate. Updating full name will sync with server.")

		if m.err != nil {
			content += "\n\n" + ui.ErrorTextStyle.Render("✘ Error: "+m.err.Error())
		}
		body = ui.CardStyle.Render(content)
		footer = ui.FooterStyle.Render("▸ Enter: save • Esc: back • Tab: switch field")
	}

	return fmt.Sprintf("%s%s%s\n%s", header, subHeader, body, footer)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running Syncra: %v\n", err)
		os.Exit(1)
	}
}
