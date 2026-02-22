package main

import (
	"fmt"
	"os"
	"time"

	"log"
	"net/http"
	"syncra/internal/server/database"
	"syncra/internal/server/websocket"
	"syncra/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type errMsg error

type dbConnectedMsg struct {
	db *database.DB
}

type serverModel struct {
	db        *database.DB
	hub       *websocket.Hub
	err       error
	loading   bool
	startTime time.Time
	tick      int
	port      string
}

func initialModel() serverModel {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return serverModel{
		loading:   true,
		startTime: time.Now(),
		port:      port,
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

func (m serverModel) Init() tea.Cmd {
	return tea.Batch(connectToDB, tick())
}

func startRelay(hub *websocket.Hub, port string) tea.Cmd {
	return func() tea.Msg {
		go hub.Run()

		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			websocket.ServeWs(hub, w, r)
		})

		log.Printf("Starting relay server on :%s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			return errMsg(fmt.Errorf("server failed: %v", err))
		}
		return nil
	}
}

func (m serverModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.hub = websocket.NewHub()
		return m, startRelay(m.hub, m.port)

	case errMsg:
		m.loading = false
		m.err = msg
		return m, nil
	}
	return m, nil
}

func (m serverModel) View() string {
	// 1. Header Section
	header := ui.HeaderStyle.Render(" SYNCRA NODE v1.0.0 ") + "\n"
	subHeader := ui.SubHeaderStyle.Render("Secure Relay Infrastructure • Zero Knowledge") + "\n"

	// 2. Status Content
	var statusContent string
	if m.loading {
		statusContent = fmt.Sprintf("\n  %s\n  %s",
			ui.StatusLabelStyle.Background(ui.Warning).Foreground(lipgloss.Color("#000000")).Render(" CONNECTING "),
			ui.MutedStyle.Render("Orchestrating database handshake..."))
	} else if m.err != nil {
		statusContent = fmt.Sprintf("\n  %s\n  %s",
			ui.StatusLabelStyle.Background(ui.ErrorCol).Foreground(lipgloss.Color("#FFFFFF")).Render(" FATAL ERROR "),
			ui.ErrorTextStyle.Render(m.err.Error()))
	} else {
		onlineTag := " ONLINE "
		if m.tick%2 == 0 {
			onlineTag = " • ONLINE "
		}

		statusContent = fmt.Sprintf("%s\n\n%s %s\n%s %s\n%s %s\n%s %s",
			ui.StatusLabelStyle.Background(ui.Success).Foreground(lipgloss.Color("#FFFFFF")).Render(onlineTag),
			ui.InfoKeyStyle.Render("Database"), ui.InfoValueStyle.Render("Neon PostreSQL (Cloud)"),
			ui.InfoKeyStyle.Render("Endpoint"), ui.InfoValueStyle.Render(":"+m.port+"/ws"),
			ui.InfoKeyStyle.Render("Protocol"), ui.InfoValueStyle.Render("WSS / AES-256-GCM"),
			ui.InfoKeyStyle.Render("Uptime"), ui.InfoValueStyle.Foreground(ui.Secondary).Render(time.Since(m.startTime).Truncate(time.Second).String()),
		)
	}

	body := ui.CardStyle.Render(statusContent)

	// 3. Footer Section
	footer := ui.FooterStyle.Render("▸ Press 'q' to gracefully shutdown")

	return fmt.Sprintf("%s%s%s\n%s", header, subHeader, body, footer)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
