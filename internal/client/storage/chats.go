package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syncra/internal/config"
	"syncra/internal/models"
)

var fileMutex sync.Mutex

// AppendMessage saves a chat message to the local storage
func AppendMessage(targetUsername string, msg models.LocalChatMessage) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	chatsDir := filepath.Join(cfg.WorkspacePath, "syncra", "chats")
	if err := os.MkdirAll(chatsDir, 0755); err != nil {
		return fmt.Errorf("failed to create chats directory: %v", err)
	}

	filePath := filepath.Join(chatsDir, targetUsername+".json")

	fileMutex.Lock()
	defer fileMutex.Unlock()

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open chat file: %v", err)
	}
	defer f.Close()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	return nil
}

// LoadMessages retrieves all messages for a specific conversation
func LoadMessages(targetUsername string) ([]models.LocalChatMessage, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(cfg.WorkspacePath, "syncra", "chats", targetUsername+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []models.LocalChatMessage{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parsing JSONL
	lines := strings.Split(string(data), "\n")
	var messages []models.LocalChatMessage
	for _, line := range lines {
		if line == "" {
			continue
		}
		var msg models.LocalChatMessage
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// ListChats returns a list of usernames that the user has a chat history with
func ListChats() ([]string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	chatsDir := filepath.Join(cfg.WorkspacePath, "syncra", "chats")
	files, err := os.ReadDir(chatsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var chats []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
			chats = append(chats, strings.TrimSuffix(f.Name(), ".json"))
		}
	}

	return chats, nil
}
