package models

import (
	"time"
)

// User represents a user in the Syncra system.
type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	FullName      string    `json:"full_name"`
	PublicKeyHash string    `json:"public_key_hash"`
	CreatedAt     time.Time `json:"created_at"`
}
