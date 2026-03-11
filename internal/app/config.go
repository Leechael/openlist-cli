package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type config struct {
	BaseURL string `json:"base_url,omitempty"`
	Token   string `json:"token,omitempty"`
}

func configPath() (string, error) {
	if v := strings.TrimSpace(os.Getenv("OPENLIST_CLI_CONFIG")); v != "" {
		return v, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "openlist-cli", "config.json"), nil
}

func loadConfig() (config, error) {
	path, err := configPath()
	if err != nil {
		return config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config{}, nil
		}
		return config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return config{}, nil
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

func saveConfig(cfg config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

func defaultBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("OPENLIST_BASE_URL")); v != "" {
		return v
	}
	cfg, err := loadConfig()
	if err == nil && strings.TrimSpace(cfg.BaseURL) != "" {
		return strings.TrimSpace(cfg.BaseURL)
	}
	return "http://localhost:5244"
}

func defaultToken() string {
	if v := strings.TrimSpace(os.Getenv("OPENLIST_TOKEN")); v != "" {
		return v
	}
	cfg, err := loadConfig()
	if err == nil {
		return strings.TrimSpace(cfg.Token)
	}
	return ""
}
