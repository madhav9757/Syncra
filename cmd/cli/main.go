package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syncra/internal/config"
	"syncra/internal/ui"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type state int

const (
	stateSetup state = iota
	stateSuccess
	stateMain
	stateSettings
)

type model struct {
	state      state
	textInput  textinput.Model
	cfg        *config.Config
	err        error
	startTime  time.Time
	successMsg string
	altscreen  bool
	quitting   bool
	suspending bool
}

func initialModel() model {
	// Try to load existing config
	cfg, _ := config.LoadConfig()

	ti := textinput.New()
	ti.Placeholder = "C:\\Users\\example\\Documents"
	ti.CharLimit = 156
	ti.Width = 50

	home, _ := os.UserHomeDir()
	if home != "" {
		ti.SetValue(home)
	}

	m := model{
		startTime: time.Now(),
		cfg:       cfg,
		altscreen: true,
		textInput: ti,
	}

	if cfg == nil {
		m.state = stateSetup
		m.textInput.Focus()
	} else {
		m.state = stateMain
	}

	return m
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, tea.EnterAltScreen)
	if m.state == stateSetup {
		cmds = append(cmds, textinput.Blink)
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.ResumeMsg:
		m.suspending = false
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
				m.textInput.SetValue(m.cfg.WorkspacePath)
				m.textInput.Focus()
				return m, textinput.Blink
			}
		case "esc":
			if m.state == stateSettings {
				m.state = stateMain
				return m, nil
			}
		}

		if m.state == stateSetup || m.state == stateSettings {
			switch msg.Type {
			case tea.KeyEnter:
				path := m.textInput.Value()
				if path == "" {
					m.err = fmt.Errorf("path cannot be empty")
					return m, nil
				}

				// Initialize structure
				err := config.InitializeStructure(path)
				if err != nil {
					m.err = err
					return m, nil
				}

				// Save config
				cfg := &config.Config{WorkspacePath: path}
				err = config.SaveConfig(cfg)
				if err != nil {
					m.err = err
					return m, nil
				}

				m.cfg = cfg
				if m.state == stateSetup {
					m.successMsg = fmt.Sprintf("Syncra folder structure created at:\n%s", filepath.Join(path, "syncra"))
					m.state = stateSuccess
				} else {
					m.state = stateMain
				}
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		if m.state == stateSuccess {
			if msg.Type == tea.KeyEnter {
				m.state = stateMain
				m.startTime = time.Now()
				return m, nil
			}
		}

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
	case stateSetup:
		subHeader = ui.SubHeaderStyle.Render("First-time Setup Required") + "\n"

		content := ui.InfoKeyStyle.Render("Welcome to Syncra.") + " To get started, please specify the path\nwhere you want to initialize the Syncra folder structure.\n\n"
		content += ui.InfoKeyStyle.Render("Base Path:") + "\n" + m.textInput.View()

		if m.err != nil {
			content += "\n\n" + ui.ErrorTextStyle.Render("✘ Error: "+m.err.Error())
		}

		body = ui.CardStyle.Render(content)
		footer = ui.FooterStyle.Render("▸ Enter: init • Space: toggle mode • Ctrl+Z: suspend • Ctrl+C: exit")

	case stateSuccess:
		subHeader = ui.SubHeaderStyle.Render("Initialization Complete") + "\n"

		content := ui.SuccessStyle().Render("✓ SUCCESS!") + "\n\n"
		content += ui.MutedStyle.Render("Syncra folder structure created at:") + "\n"
		content += ui.InfoValueStyle.Render(filepath.Join(m.cfg.WorkspacePath, "syncra"))

		body = ui.CardStyle.Render(content)
		footer = ui.FooterStyle.Render("▸ Enter: start • Space: toggle mode • Ctrl+Z: suspend")

	case stateMain:
		subHeader = ui.SubHeaderStyle.Render("End-to-End Encrypted Terminal Suite") + "\n"

		statusContent := fmt.Sprintf("%s\n\n%s %s\n%s %s",
			ui.StatusLabelStyle.Foreground(ui.Success).Render("● ACTIVE"),
			ui.InfoKeyStyle.Render("Workspace"), ui.InfoValueStyle.Render(m.cfg.WorkspacePath),
			ui.InfoKeyStyle.Render("Session"), ui.InfoValueStyle.Foreground(ui.Secondary).Render(time.Since(m.startTime).Truncate(time.Second).String()),
		)

		body = ui.CardStyle.Render(statusContent)
		footer = ui.FooterStyle.Render("▸ 's': settings • Space: toggle mode • Ctrl+Z: suspend • 'q': exit")

	case stateSettings:
		subHeader = ui.SubHeaderStyle.Render("Settings & Configuration") + "\n"

		content := ui.InfoKeyStyle.Render("Workspace Path:") + "\n" + m.textInput.View() + "\n\n"
		content += ui.MutedStyle.Render("Updating this will re-initialize the folder structure\nat the new location.")

		if m.err != nil {
			content += "\n\n" + ui.ErrorTextStyle.Render("✘ Error: "+m.err.Error())
		}

		body = ui.CardStyle.Render(content)
		footer = ui.FooterStyle.Render("▸ Enter: save • Esc: back • Space: toggle mode")
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
