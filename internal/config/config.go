package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zamatewi-cell/traecn_tool/internal/protect"
)

// Config holds global proxy configuration
type Config struct {
	ListenAddr string          `json:"listen_addr"`
	Accounts   []AccountConfig `json:"accounts"`
	LogLevel   string          `json:"log_level"`
	Protect    protect.Config  `json:"protect"`
}

// AccountConfig holds a single Trae CN account config
type AccountConfig struct {
	Name        string `json:"name"`
	StoragePath string `json:"storage_path,omitempty"`
	Token       string `json:"token,omitempty"`
	EnvVar      string `json:"env_var,omitempty"`
	Weight      int    `json:"weight,omitempty"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		ListenAddr: ":9090",
		LogLevel:   "info",
		Protect:    protect.DefaultConfig(),
	}
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

// SaveConfig saves configuration to file
func SaveConfig(cfg *Config, filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}
	return os.WriteFile(filePath, data, 0o644)
}
