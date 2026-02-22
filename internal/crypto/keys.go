package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// GenerateKeyPair generates a new Ed25519 key pair.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ed25519 key pair: %v", err)
	}
	return pub, priv, nil
}

// HashPublicKey returns the SHA-256 hash of the public key as a hex string.
func HashPublicKey(pub ed25519.PublicKey) string {
	hash := sha256.Sum256(pub)
	return hex.EncodeToString(hash[:])
}

// SavePrivateKey saves the private key to a file with secure permissions (0600).
func SavePrivateKey(path string, priv ed25519.PrivateKey) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for private key: %v", err)
	}

	// In Go, os.WriteFile uses the provided mode.
	// 0600 means read/write for owner only.
	return os.WriteFile(path, priv, 0600)
}

// LoadPrivateKey loads the private key from a file.
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size")
	}
	return ed25519.PrivateKey(data), nil
}
