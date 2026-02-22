package models

import (
	"time"
)

// User represents a user in the Syncra system.
type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	FullName      string    `json:"full_name"`
	PublicKey     string    `json:"public_key"`      // Hex encoded Ed25519 Public Key
	PublicKeyHash string    `json:"public_key_hash"` // Still keeping hash for lookup?
	CreatedAt     time.Time `json:"created_at"`
}
