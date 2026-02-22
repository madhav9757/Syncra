package websocket

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"syncra/internal/models"
	"syncra/internal/server/database"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub *Hub

	// The websocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Username of the connected identity
	Username string

	// Is authenticated via challenge-response
	Authenticated bool

	// Challenge sent to this client
	Challenge string
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, msgData, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var packet models.Packet
		if err := json.Unmarshal(msgData, &packet); err != nil {
			continue
		}

		switch packet.Type {
		case models.TypeAuth:
			var auth models.AuthPayload
			if err := json.Unmarshal(packet.Payload, &auth); err != nil {
				c.sendError("Invalid auth payload")
				continue
			}
			c.handleAuth(auth)

		case models.TypeChat:
			if !c.Authenticated {
				c.sendError("Unauthorized")
				continue
			}
			c.handleChat(packet)
		}
	}
}

func (c *Client) handleAuth(auth models.AuthPayload) {
	db, err := database.Connect()
	if err != nil {
		c.sendError("Internal server error")
		return
	}
	defer db.Close()

	user, err := db.GetUserByUsername(context.Background(), auth.Username)
	if err != nil {
		c.sendError("User not found")
		return
	}

	sig, err := hex.DecodeString(auth.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		c.sendError("Invalid signature format or size")
		return
	}

	pubKey, err := hex.DecodeString(user.PublicKey)
	if err != nil || len(pubKey) != ed25519.PublicKeySize {
		c.sendError("Server error: invalid public key stored (may be legacy user)")
		return
	}

	if !ed25519.Verify(pubKey, []byte(c.Challenge), sig) {
		c.sendError("Invalid signature")
		return
	}

	c.Authenticated = true
	c.Username = auth.Username
	c.Hub.authenticate <- c
	c.sendSystem("Authenticated")
}

func (c *Client) handleChat(packet models.Packet) {
	target, ok := c.Hub.GetClient(packet.To)
	if !ok {
		c.sendError("Recipient offline")
		return
	}

	// Deterministic Room ID
	users := []string{c.Username, packet.To}
	sort.Strings(users)
	roomID := strings.Join(users, ":")

	c.Hub.JoinRoom(roomID, c.Username)
	c.Hub.JoinRoom(roomID, packet.To)

	// Relay the packet
	packet.From = c.Username
	packet.Timestamp = time.Now()
	data, _ := json.Marshal(packet)
	target.send <- data
}

func (c *Client) sendError(msg string) {
	p := models.Packet{
		Type:      models.TypeError,
		Payload:   json.RawMessage(`"` + msg + `"`),
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(p)
	c.send <- data
}

func (c *Client) sendSystem(msg string) {
	p := models.Packet{
		Type:      models.TypeSystem,
		Payload:   json.RawMessage(`"` + msg + `"`),
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(p)
	c.send <- data
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Generate challenge
	challenge := make([]byte, 32)
	rand.Read(challenge)
	challengeHex := hex.EncodeToString(challenge)

	client := &Client{
		Hub:       hub,
		Conn:      conn,
		send:      make(chan []byte, 256),
		Challenge: challengeHex,
	}
	client.Hub.register <- client

	// Send challenge immediately
	challengePkg := models.Packet{
		Type:      models.TypeChallenge,
		Payload:   json.RawMessage(`"` + challengeHex + `"`),
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(challengePkg)
	client.send <- data

	go client.WritePump()
	go client.ReadPump()
}
