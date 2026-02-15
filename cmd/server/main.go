package main

import (
	"fmt"
	"os"
	"time"

	"syncra/internal/server/database"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Premium UI Styling
var (
	primary   = lipgloss.Color("#7D56F4") // Vibrant Purple
	secondary = lipgloss.Color("#00E5FF") // Electric Cyan
	accent    = lipgloss.Color("#FF4081") // Hot Pink
	success   = lipgloss.Color("#00C853") // Emerald Green
	warning   = lipgloss.Color("#FFD600") // Neon Gold
	errorCol  = lipgloss.Color("#FF1744") // Radical Red
	bg        = lipgloss.Color("#1A1B26") // Deep Tokyo Night Background
	text      = lipgloss.Color("#C0CAF5") // Soft White/Blue Text
	muted     = lipgloss.Color("#565F89") // Muted Blue Gray

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primary).
			Bold(true).
			Padding(0, 2).
			MarginLeft(2).
			MarginTop(1)

	subHeaderStyle = lipgloss.NewStyle().
			Foreground(secondary).
			Italic(true).
			MarginLeft(2).
			MarginBottom(1)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(primary).
			Padding(1, 4).
			Background(bg).
			Margin(1, 2).
			Width(64)

	statusLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1).
				MarginRight(2)

	infoKeyStyle = lipgloss.NewStyle().
			Foreground(muted).
			Width(15)

	infoValueStyle = lipgloss.NewStyle().
			Foreground(text).
			Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(muted)

	errorTextStyle = lipgloss.NewStyle().
			Foreground(errorCol)

	footerStyle = lipgloss.NewStyle().
			Foreground(muted).
			MarginLeft(3).
			MarginTop(1).
			Italic(true)
)

type errMsg error

type dbConnectedMsg struct {
	db *database.DB
}

type model struct {
	db        *database.DB
	err       error
	loading   bool
	startTime time.Time
	tick      int
}

func initialModel() model {
	return model{
		loading:   true,
		startTime: time.Now(),
	}
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func connectToDB() tea.Msg {
	db, err := database.Connect()
	if err != nil {
		return errMsg(err)
	}
	return dbConnectedMsg{db: db}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(connectToDB, tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			if m.db != nil {
				m.db.Close()
			}
			return m, tea.Quit
		}

	case tickMsg:
		m.tick++
		return m, tick()

	case dbConnectedMsg:
		m.loading = false
		m.db = msg.db
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	var body string

	// 1. Header Section
	header := headerStyle.Render(" SYNCRA NODE v1.0.0 ") + "\n"
	subHeader := subHeaderStyle.Render("Secure Relay Infrastructure • Zero Knowledge") + "\n"

	// 2. Status Content
	var statusContent string
	if m.loading {
		statusContent = fmt.Sprintf("\n  %s\n  %s",
			statusLabelStyle.Background(warning).Foreground(lipgloss.Color("#000000")).Render(" CONNECTING "),
			mutedStyle.Render("Orchestrating database handshake..."))
	} else if m.err != nil {
		statusContent = fmt.Sprintf("\n  %s\n  %s",
			statusLabelStyle.Background(errorCol).Foreground(lipgloss.Color("#FFFFFF")).Render(" FATAL ERROR "),
			errorTextStyle.Render(m.err.Error()))
	} else {
		onlineTag := " ONLINE "
		if m.tick%2 == 0 {
			onlineTag = " • ONLINE "
		}

		statusContent = fmt.Sprintf("%s\n\n%s %s\n%s %s\n%s %s",
			statusLabelStyle.Background(success).Foreground(lipgloss.Color("#FFFFFF")).Render(onlineTag),
			infoKeyStyle.Render("Database"), infoValueStyle.Render("Neon PostreSQL (Cloud)"),
			infoKeyStyle.Render("Protocol"), infoValueStyle.Render("WSS / AES-256-GCM"),
			infoKeyStyle.Render("Uptime"), infoValueStyle.Foreground(secondary).Render(time.Since(m.startTime).Truncate(time.Second).String()),
		)
	}

	body = cardStyle.Render(statusContent)

	// 3. Footer Section
	footer := footerStyle.Render("▸ Press 'q' to gracefully shutdown")

	return fmt.Sprintf("%s%s%s\n%s", header, subHeader, body, footer)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
