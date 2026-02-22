package models

import (
	"encoding/json"
	"time"
)

type MessageType string

const (
	TypeChallenge MessageType = "challenge"
	TypeAuth      MessageType = "auth"
	TypeChat      MessageType = "chat"
	TypeSystem    MessageType = "system"
	TypeError     MessageType = "error"
)

// Packet is the base structure for all WebSocket communication
type Packet struct {
	Type      MessageType     `json:"type"`
	From      string          `json:"from,omitempty"`
	To        string          `json:"to,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
	Signature string          `json:"signature,omitempty"`
}

// ChallengePayload sent by server
type ChallengePayload struct {
	Nonce string `json:"nonce"`
}

// AuthPayload sent by client
type AuthPayload struct {
	Username  string `json:"username"`
	Signature string `json:"signature"` // Hex encoded signature of the nonce
}

// ChatPayload for E2EE messages
type ChatPayload struct {
	Message   string `json:"message"`   // Usually encrypted ciphertext
	Ephemeral string `json:"ephemeral"` // Ephemeral public key for DH (optional)
}

// LocalChatMessage for storage in syncra/chats/
type LocalChatMessage struct {
	From      string    `json:"from"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	IsMe      bool      `json:"is_me"`
}
