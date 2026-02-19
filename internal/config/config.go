package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	WorkspacePath string `json:"workspace_path"`
}

func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	syncraDir := filepath.Join(configDir, "syncra")
	if err := os.MkdirAll(syncraDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(syncraDir, "config.json"), nil
}

func LoadConfig() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Not initialized
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func InitializeStructure(basePath string) error {
	// Root syncra folder
	syncraPath := filepath.Join(basePath, "syncra")
	folders := []string{
		syncraPath,
		filepath.Join(syncraPath, "logs"),
		filepath.Join(syncraPath, "data"),
		filepath.Join(syncraPath, "config"),
		filepath.Join(syncraPath, "identities"),
	}

	for _, folder := range folders {
		if err := os.MkdirAll(folder, 0755); err != nil {
			return fmt.Errorf("failed to create folder %s: %v", folder, err)
		}
	}

	return nil
}
