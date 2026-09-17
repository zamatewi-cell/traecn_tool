package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryTokenStorage_Basic(t *testing.T) {
	storage := NewMemoryTokenStorage()

	now := time.Now()
	token := &TokenInfo{
		AccessToken:      "test_token",
		RefreshToken:     "test_refresh",
		ExpiresAt:        now.Add(24 * time.Hour),
		RefreshExpiresAt: now.Add(180 * 24 * time.Hour),
		LastRefreshedAt:  now,
		AccountID:        "test_account",
	}

	// Save token
	err := storage.Save("test_account", token)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	// Load token
	loaded, err := storage.Load("test_account")
	if err != nil {
		t.Fatalf("Failed to load token: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("Expected access token %s, got %s", token.AccessToken, loaded.AccessToken)
	}
}

func TestMemoryTokenStorage_Delete(t *testing.T) {
	storage := NewMemoryTokenStorage()

	token := &TokenInfo{
		AccessToken:      "to_delete",
		RefreshToken:     "to_delete",
		ExpiresAt:        time.Now().Add(1 * time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		LastRefreshedAt:  time.Now(),
	}

	// Save
	storage.Save("temp_account", token)

	// Verify exists
	if !storage.Exists("temp_account") {
		t.Error("Expected token to exist")
	}

	// Delete
	err := storage.Delete("temp_account")
	if err != nil {
		t.Fatalf("Failed to delete token: %v", err)
	}

	// Verify deleted
	if storage.Exists("temp_account") {
		t.Error("Expected token to be deleted")
	}

	// Verify load fails
	_, err = storage.Load("temp_account")
	if err == nil {
		t.Error("Expected error when loading deleted token")
	}
}

func TestMemoryTokenStorage_LoadAll(t *testing.T) {
	storage := NewMemoryTokenStorage()

	now := time.Now()
	// Add multiple tokens
	for i := 0; i < 3; i++ {
		token := &TokenInfo{
			AccessToken:      "token_" + string(rune(i)),
			RefreshToken:     "refresh_" + string(rune(i)),
			ExpiresAt:        now.Add(14 * 24 * time.Hour),
			RefreshExpiresAt: now.Add(180 * 24 * time.Hour),
			LastRefreshedAt:  now,
			AccountID:        "account_" + string(rune(i)),
		}
		storage.Save("account_"+string(rune(i)), token)
	}

	all, err := storage.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load all tokens: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("Expected 3 tokens, got %d", len(all))
	}
}

func TestFileTokenStorage_Basic(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tokens.json")

	storage, err := NewFileTokenStorage(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	now := time.Now()
	token := &TokenInfo{
		AccessToken:      "test_token",
		RefreshToken:     "test_refresh",
		ExpiresAt:        now.Add(24 * time.Hour),
		RefreshExpiresAt: now.Add(180 * 24 * time.Hour),
		LastRefreshedAt:  now,
		AccountID:        "test_account",
	}

	// Save token
	err = storage.Save("test_account", token)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Expected file to be created")
	}

	// Create new storage instance (simulate restart)
	storage2, err := NewFileTokenStorage(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create second storage instance: %v", err)
	}

	// Load token
	loaded, err := storage2.Load("test_account")
	if err != nil {
		t.Fatalf("Failed to load token: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("Expected access token %s, got %s", token.AccessToken, loaded.AccessToken)
	}
}

func TestFileTokenStorage_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tokens.json")

	storage, err := NewFileTokenStorage(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	now := time.Now()
	// Add multiple tokens
	for i := 0; i < 3; i++ {
		token := &TokenInfo{
			AccessToken:      "token_" + string(rune(i)),
			RefreshToken:     "refresh_" + string(rune(i)),
			ExpiresAt:        now.Add(14 * 24 * time.Hour),
			RefreshExpiresAt: now.Add(180 * 24 * time.Hour),
			LastRefreshedAt:  now,
			AccountID:        "account_" + string(rune(i)),
		}
		storage.Save("account_"+string(rune(i)), token)
	}

	// Create new instance to verify persistence
	storage2, err := NewFileTokenStorage(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create second storage instance: %v", err)
	}

	all, err := storage2.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load all tokens: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("Expected 3 tokens, got %d", len(all))
	}
}

func TestFileTokenStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tokens.json")

	storage, err := NewFileTokenStorage(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	token := &TokenInfo{
		AccessToken:      "to_delete",
		RefreshToken:     "to_delete",
		ExpiresAt:        time.Now().Add(1 * time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		LastRefreshedAt:  time.Now(),
	}

	// Save
	storage.Save("temp_account", token)

	// Verify exists
	if !storage.Exists("temp_account") {
		t.Error("Expected token to exist")
	}

	// Delete
	err = storage.Delete("temp_account")
	if err != nil {
		t.Fatalf("Failed to delete token: %v", err)
	}

	// Verify deleted
	if storage.Exists("temp_account") {
		t.Error("Expected token to be deleted")
	}
}

func TestFileTokenStorage_Cache(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tokens.json")

	storage, err := NewFileTokenStorage(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}

	token := &TokenInfo{
		AccessToken:      "cached_token",
		RefreshToken:     "cached_refresh",
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		RefreshExpiresAt: time.Now().Add(180 * 24 * time.Hour),
		LastRefreshedAt:  time.Now(),
	}

	// First save (writes to file)
	err = storage.Save("test_account", token)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	// Second load (should read from cache)
	loaded, err := storage.Load("test_account")
	if err != nil {
		t.Fatalf("Failed to load token: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("Expected access token %s, got %s", token.AccessToken, loaded.AccessToken)
	}
}
