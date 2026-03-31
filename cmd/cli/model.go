package main

import (
	clientWS "syncra/internal/client/websocket"
	"syncra/internal/config"
	"syncra/internal/discovery"
	"syncra/internal/models"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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
	stateLanNetwork
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
	isLocal      bool
	localNode    *discovery.Node

	// Friends list
	chats              []string
	chatSelectionIndex int

	// LAN Network list
	lanPeers           []discovery.Peer
	lanSelectionIndex  int
}

type reconnectMsg struct{}
type wsMessage struct {
	packet models.Packet
}

type wsErrorMsg struct {
	err error
}
type setupResult struct {
	err error
	cfg *config.Config
}
type searchResult struct {
	users []*models.User
	err   error
}
type usernameCheckResult struct {
	exists bool
	err    error
}
