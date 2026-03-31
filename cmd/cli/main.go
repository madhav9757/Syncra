package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"syncra/internal/client/storage"
	clientWS "syncra/internal/client/websocket"
	"syncra/internal/config"
	"syncra/internal/discovery"
	"syncra/internal/models"
	"syncra/internal/ui"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func initialModel(isLocal bool) model {
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
		isLocal:       isLocal,
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
		if m.isLocal {
			startLocalNode(&m)
		} else {
			// Auto-connect
			conn, err := clientWS.Connect("localhost:8080")
			if err == nil {
				m.conn = conn
				go m.conn.WritePump()
			}
		}
	}

	return m
}
func startLocalNode(m *model) {
	if m.localNode == nil {
		rand.Seed(time.Now().UnixNano())
		// random port for the local TCP/HTTP server
		port := fmt.Sprintf("%d", 8000+rand.Intn(1000))
		m.localNode = discovery.NewNode(m.cfg.Username, m.cfg.FullName, port)
		m.localNode.Start()
		go m.localNode.StartServer(func(data []byte) {
			// Actually we need to send this data to the model.
			// Since StartServer runs in background, we let the standard listen routine or Bubbletea command handle it,
			// but we can't easily send tea.Msg without the program reference.
			// For simplicity, we can do it by saving to local DB and updating chat messages.
			var packet models.Packet
			json.Unmarshal(data, &packet)
			if packet.Type == models.TypeChat {
				var chat models.ChatPayload
				json.Unmarshal(packet.Payload, &chat)
				localMsg := models.LocalChatMessage{
					From:      packet.From,
					Content:   chat.Message,
					Timestamp: packet.Timestamp,
					IsMe:      false,
				}
				storage.AppendMessage(packet.From, localMsg)
			}
		})
	}
}
func main() {
	var isLocal bool
	if len(os.Args) > 1 && os.Args[1] == "local" {
		isLocal = true
	}

	p := tea.NewProgram(initialModel(isLocal))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running Syncra: %v\n", err)
		os.Exit(1)
	}
}
