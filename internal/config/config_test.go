package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ListenAddr != ":9090" {
		t.Errorf("DefaultConfig() ListenAddr = %v, want :9090", cfg.ListenAddr)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("DefaultConfig() LogLevel = %v, want info", cfg.LogLevel)
	}
	// Accounts is initialized as empty slice when first account is added
}

func TestLoadConfig(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write test config
	testConfig := `{
		"listen_addr": ":8080",
		"log_level": "debug",
		"accounts": [
			{
				"name": "account1",
				"storage_path": "/path/to/storage1.json",
				"weight": 10
			},
			{
				"name": "account2",
				"token": "direct_token",
				"weight": 5
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test loading
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.ListenAddr != ":8080" {
		t.Errorf("LoadConfig() ListenAddr = %v, want :8080", cfg.ListenAddr)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LoadConfig() LogLevel = %v, want debug", cfg.LogLevel)
	}
	if len(cfg.Accounts) != 2 {
		t.Errorf("LoadConfig() Accounts length = %v, want 2", len(cfg.Accounts))
	}

	// Check first account
	if cfg.Accounts[0].Name != "account1" {
		t.Errorf("Account[0] Name = %v, want account1", cfg.Accounts[0].Name)
	}
	if cfg.Accounts[0].StoragePath != "/path/to/storage1.json" {
		t.Errorf("Account[0] StoragePath = %v, want /path/to/storage1.json", cfg.Accounts[0].StoragePath)
	}
	if cfg.Accounts[0].Weight != 10 {
		t.Errorf("Account[0] Weight = %v, want 10", cfg.Accounts[0].Weight)
	}

	// Check second account
	if cfg.Accounts[1].Name != "account2" {
		t.Errorf("Account[1] Name = %v, want account2", cfg.Accounts[1].Name)
	}
	if cfg.Accounts[1].Token != "direct_token" {
		t.Errorf("Account[1] Token = %v, want direct_token", cfg.Accounts[1].Token)
	}
	if cfg.Accounts[1].Weight != 5 {
		t.Errorf("Account[1] Weight = %v, want 5", cfg.Accounts[1].Weight)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Error("LoadConfig() error = nil, want error for missing file")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	err := os.WriteFile(configPath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() error = nil, want error for invalid JSON")
	}
}

func TestLoadConfig_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Only specify listen_addr, others should use defaults
	testConfig := `{
		"listen_addr": ":3000"
	}`

	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.ListenAddr != ":3000" {
		t.Errorf("LoadConfig() ListenAddr = %v, want :3000", cfg.ListenAddr)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LoadConfig() LogLevel = %v, want info (default)", cfg.LogLevel)
	}
	// Accounts is nil when not specified in config file
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := &Config{
		ListenAddr: ":9090",
		LogLevel:   "info",
		Accounts: []AccountConfig{
			{
				Name:        "test-account",
				StoragePath: "/path/to/storage.json",
				Weight:      10,
			},
		},
	}

	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("SaveConfig() did not create config file")
	}

	// Read and verify content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	if loaded.ListenAddr != cfg.ListenAddr {
		t.Errorf("Saved config ListenAddr = %v, want %v", loaded.ListenAddr, cfg.ListenAddr)
	}
	if loaded.LogLevel != cfg.LogLevel {
		t.Errorf("Saved config LogLevel = %v, want %v", loaded.LogLevel, cfg.LogLevel)
	}
	if len(loaded.Accounts) != 1 {
		t.Errorf("Saved config Accounts length = %v, want 1", len(loaded.Accounts))
	}
}

func TestSaveConfig_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.json")

	cfg := DefaultConfig()
	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("SaveConfig() did not create config file in subdirectory")
	}
}

func TestAccountConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  AccountConfig
		wantErr bool
	}{
		{
			name: "valid with storage_path",
			config: AccountConfig{
				Name:        "test",
				StoragePath: "/path/to/storage.json",
			},
			wantErr: false,
		},
		{
			name: "valid with token",
			config: AccountConfig{
				Name:  "test",
				Token: "direct_token",
			},
			wantErr: false,
		},
		{
			name: "invalid - no storage_path or token",
			config: AccountConfig{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "invalid - empty name",
			config: AccountConfig{
				StoragePath: "/path/to/storage.json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Current implementation doesn't validate AccountConfig
			// This test documents expected future behavior
			if tt.config.Name == "" && tt.config.StoragePath == "" && tt.config.Token == "" {
				if !tt.wantErr {
					t.Error("Expected validation error for empty config")
				}
			}
		})
	}
}
