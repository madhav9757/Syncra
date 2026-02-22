package websocket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"syncra/internal/models"

	"github.com/gorilla/websocket"
)

type Connection struct {
	Conn *websocket.Conn
	Send chan models.Packet
}

func Connect(serverAddr string) (*Connection, error) {
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: "/ws"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}

	conn := &Connection{
		Conn: c,
		Send: make(chan models.Packet, 256),
	}

	return conn, nil
}

func (c *Connection) WritePump() {
	for packet := range c.Send {
		data, _ := json.Marshal(packet)
		c.Conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (c *Connection) Close() {
	c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	c.Conn.Close()
}
