package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syncra/internal/config"
	"syncra/internal/crypto"
	"syncra/internal/models"
	"syncra/internal/server/database"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) listenWS() tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return wsErrorMsg{err: fmt.Errorf("no connection")}
		}
		for {
			_, data, err := m.conn.Conn.ReadMessage()
			if err != nil {
				return wsErrorMsg{err: err}
			}
			var p models.Packet
			if err := json.Unmarshal(data, &p); err != nil {
				continue
			}
			return wsMessage{packet: p}
		}
	}
}
func (m model) pollLocalChats() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return wsMessage{packet: models.Packet{Type: models.TypeSystem}} // A generic ping to trigger UI update
	})
}
func (m model) performSetup() tea.Cmd {
	return func() tea.Msg {
		var db *database.DB
		var err error
		if !m.isLocal {
			// 1. Connect to DB
			db, err = database.Connect()
			if err != nil {
				return setupResult{err: fmt.Errorf("failed to connect to server: %v", err)}
			}
			defer db.Close()

			ctx := context.Background()

			// 2. Validate Username uniqueness (again, just to be sure)
			taken, err := db.IsUsernameTaken(ctx, m.tempUsername)
			if err != nil {
				return setupResult{err: fmt.Errorf("failed to validate username: %v", err)}
			}
			if taken {
				return setupResult{err: fmt.Errorf("username already taken")}
			}
		}

		// 3. Generate Key Pair
		pub, priv, err := crypto.GenerateKeyPair()
		if err != nil {
			return setupResult{err: fmt.Errorf("failed to generate keys: %v", err)}
		}

		// 4. Initialize local structure
		if err := config.InitializeStructure(m.tempWorkspace); err != nil {
			return setupResult{err: fmt.Errorf("failed to create folders: %v", err)}
		}

		// 5. Save Private Key locally
		keyPath := filepath.Join(m.tempWorkspace, "syncra", "identities", "id_ed25519")
		if err := crypto.SavePrivateKey(keyPath, priv); err != nil {
			return setupResult{err: fmt.Errorf("failed to save private key: %v", err)}
		}

		// 6. Register on Server (If not local)
		pubHash := crypto.HashPublicKey(pub)
		pubHex := hex.EncodeToString(pub)
		if !m.isLocal {
			user := &models.User{
				Username:      m.tempUsername,
				FullName:      m.tempFullName,
				PublicKey:     pubHex,
				PublicKeyHash: pubHash,
			}
			if err := db.CreateUser(context.Background(), user); err != nil {
				return setupResult{err: fmt.Errorf("failed to register user on server: %v", err)}
			}
		}

		// 7. Save Config
		cfg := &config.Config{
			WorkspacePath: m.tempWorkspace,
			Username:      m.tempUsername,
			FullName:      m.tempFullName,
		}
		if err := config.SaveConfig(cfg); err != nil {
			return setupResult{err: fmt.Errorf("failed to save config: %v", err)}
		}

		return setupResult{cfg: cfg}
	}
}
func (m model) performSearch(query string) tea.Cmd {
	return func() tea.Msg {
		if m.isLocal {
			var users []*models.User
			if m.localNode != nil {
				for _, p := range m.localNode.GetPeers() {
					if query == "" || p.Username == query || p.FullName == query {
						users = append(users, &models.User{
							Username: p.Username,
							FullName: p.FullName,
						})
					}
				}
			}
			return searchResult{users: users}
		}

		db, err := database.Connect()
		if err != nil {
			return searchResult{err: fmt.Errorf("failed to connect to server: %v", err)}
		}
		defer db.Close()

		users, err := db.SearchUsers(context.Background(), query)
		if err != nil {
			return searchResult{err: fmt.Errorf("search failed: %v", err)}
		}

		return searchResult{users: users}
	}
}
func checkUsername(username string, isLocal bool) tea.Cmd {
	return func() tea.Msg {
		if isLocal {
			return usernameCheckResult{exists: false, err: nil}
		}

		db, err := database.Connect()
		if err != nil {
			return usernameCheckResult{err: err}
		}
		defer db.Close()

		taken, err := db.IsUsernameTaken(context.Background(), username)
		return usernameCheckResult{exists: taken, err: err}
	}
}
func (m model) performSelfDestruct() tea.Cmd {
	return func() tea.Msg {
		// 1. Delete from Server
		db, err := database.Connect()
		if err == nil {
			db.DeleteUser(context.Background(), m.cfg.Username)
			db.Close()
		}

		// 2. Delete local folder
		syncraPath := filepath.Join(m.cfg.WorkspacePath, "syncra")
		os.RemoveAll(syncraPath)

		// 3. Purge config
		config.PurgeConfig()

		return tea.Quit()
	}
}
