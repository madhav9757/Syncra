package websocket

import (
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[string]*Client

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Authentication updates
	authenticate chan *Client

	// Rooms map: RoomID -> Set of Usernames
	rooms map[string]map[string]bool

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		authenticate: make(chan *Client),
		clients:      make(map[string]*Client),
		rooms:        make(map[string]map[string]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.register:
			log.Printf("New connection pending authentication")

		case client := <-h.authenticate:
			h.mu.Lock()
			if client.Username != "" {
				h.clients[client.Username] = client
				log.Printf("Client authenticated and registered: %s", client.Username)
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if client.Username != "" {
				if _, ok := h.clients[client.Username]; ok {
					delete(h.clients, client.Username)
					// Clean up rooms this client was in
					for roomID, users := range h.rooms {
						if users[client.Username] {
							delete(users, client.Username)
							if len(users) == 0 {
								delete(h.rooms, roomID)
							}
						}
					}
					log.Printf("Client unregistered: %s", client.Username)
				}
			}
			h.mu.Unlock()
			close(client.send)
		}
	}
}

// JoinRoom adds a user to a deterministic room
func (h *Hub) JoinRoom(roomID string, username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[roomID]; !ok {
		h.rooms[roomID] = make(map[string]bool)
	}
	h.rooms[roomID][username] = true
}

// GetRoomUsers returns usernames in a room
func (h *Hub) GetRoomUsers(roomID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	users := []string{}
	for u := range h.rooms[roomID] {
		users = append(users, u)
	}
	return users
}

// GetClient returns a client by username
func (h *Hub) GetClient(username string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, ok := h.clients[username]
	return client, ok
}
