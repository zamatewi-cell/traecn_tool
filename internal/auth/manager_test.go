package auth

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestTokenManager_Basic(t *testing.T) {
	// Create token manager
	config := DefaultTokenConfig()
	tm := NewTokenManager(config)

	// Create test token
	now := time.Now()
	token := &TokenInfo{
		AccessToken:      "test_access_token",
		RefreshToken:     "test_refresh_token",
		ExpiresAt:        now.Add(14 * 24 * time.Hour),
		RefreshExpiresAt: now.Add(180 * 24 * time.Hour),
		LastRefreshedAt:  now,
		AccountID:        "test_account",
		TenantID:         "test_tenant",
		UserID:           "test_user",
	}

	// Add token
	tm.AddToken("test_account", token)

	// Get token
	retrieved, err := tm.GetToken(context.Background(), "test_account")
	if err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}

	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("Expected access token %s, got %s", token.AccessToken, retrieved.AccessToken)
	}
}

func TestTokenManager_CheckStatus_Valid(t *testing.T) {
	config := DefaultTokenConfig()
	tm := NewTokenManager(config)

	now := time.Now()
	token := &TokenInfo{
		AccessToken:      "valid_token",
		RefreshToken:     "valid_refresh",
		ExpiresAt:        now.Add(10 * 24 * time.Hour),
		RefreshExpiresAt: now.Add(180 * 24 * time.Hour),
		LastRefreshedAt:  now,
	}

	status := tm.CheckTokenStatus(token)

	if !status.IsValid {
		t.Error("Expected token to be valid")
	}

	if status.IsExpired {
		t.Error("Expected token to not be expired")
	}

	if status.NeedsRefresh {
		t.Error("Expected token to not need refresh yet")
	}

	if !status.CanRefresh {
		t.Error("Expected token to be refreshable")
	}
}

func TestTokenManager_CheckStatus_Expired(t *testing.T) {
	config := DefaultTokenConfig()
	tm := NewTokenManager(config)

	now := time.Now()
	token := &TokenInfo{
		AccessToken:      "expired_token",
		RefreshToken:     "valid_refresh",
		ExpiresAt:        now.Add(-1 * time.Hour),
		RefreshExpiresAt: now.Add(180 * 24 * time.Hour),
		LastRefreshedAt:  now,
	}

	status := tm.CheckTokenStatus(token)

	if status.IsValid {
		t.Error("Expected token to be invalid")
	}

	if !status.IsExpired {
		t.Error("Expected token to be expired")
	}

	if !status.NeedsRefresh {
		t.Error("Expected expired token to need refresh")
	}
}

func TestTokenManager_CheckStatus_NeedsRefresh(t *testing.T) {
	config := DefaultTokenConfig()
	tm := NewTokenManager(config)

	now := time.Now()
	token := &TokenInfo{
		AccessToken:      "soon_expired_token",
		RefreshToken:     "valid_refresh",
		ExpiresAt:        now.Add(2 * 24 * time.Hour),
		RefreshExpiresAt: now.Add(180 * 24 * time.Hour),
		LastRefreshedAt:  now,
	}

	status := tm.CheckTokenStatus(token)

	if !status.IsValid {
		t.Error("Expected token to still be valid")
	}

	if status.IsExpired {
		t.Error("Expected token to not be expired yet")
	}

	if !status.NeedsRefresh {
		t.Error("Expected token to need refresh (within 3 day threshold)")
	}
}

func TestTokenManager_CheckStatus_RefreshExpired(t *testing.T) {
	config := DefaultTokenConfig()
	tm := NewTokenManager(config)

	now := time.Now()
	token := &TokenInfo{
		AccessToken:      "token",
		RefreshToken:     "expired_refresh",
		ExpiresAt:        now.Add(10 * 24 * time.Hour),
		RefreshExpiresAt: now.Add(-1 * time.Hour),
		LastRefreshedAt:  now,
	}

	status := tm.CheckTokenStatus(token)

	if status.IsValid {
		t.Error("Expected token to be invalid when refresh token expired")
	}

	if !status.RefreshExpired {
		t.Error("Expected refresh token to be expired")
	}

	if status.CanRefresh {
		t.Error("Expected token to not be refreshable")
	}
}

func TestTokenManager_RemoveToken(t *testing.T) {
	config := DefaultTokenConfig()
	tm := NewTokenManager(config)

	token := &TokenInfo{
		AccessToken:      "to_remove",
		RefreshToken:     "to_remove",
		ExpiresAt:        time.Now().Add(1 * time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		LastRefreshedAt:  time.Now(),
	}

	tm.AddToken("temp_account", token)

	// Verify added
	_, err := tm.GetToken(context.Background(), "temp_account")
	if err != nil {
		t.Fatalf("Failed to get token before removal: %v", err)
	}

	// Delete token
	tm.RemoveToken("temp_account")

	// Verify deleted
	_, err = tm.GetToken(context.Background(), "temp_account")
	if err == nil {
		t.Error("Expected error when getting removed token")
	}
}

func TestTokenManager_GetAllTokens(t *testing.T) {
	config := DefaultTokenConfig()
	tm := NewTokenManager(config)

	now := time.Now()
	// Add multiple tokens
	for i := 0; i < 3; i++ {
		token := &TokenInfo{
			AccessToken:      fmt.Sprintf("token_%d", i),
			RefreshToken:     fmt.Sprintf("refresh_%d", i),
			ExpiresAt:        now.Add(14 * 24 * time.Hour),
			RefreshExpiresAt: now.Add(180 * 24 * time.Hour),
			LastRefreshedAt:  now,
			AccountID:        fmt.Sprintf("account_%d", i),
		}
		tm.AddToken(fmt.Sprintf("account_%d", i), token)
	}

	allTokens := tm.GetAllTokens()

	if len(allTokens) != 3 {
		t.Errorf("Expected 3 tokens, got %d", len(allTokens))
	}
}

func TestDefaultTokenConfig(t *testing.T) {
	config := DefaultTokenConfig()

	if config.RefreshThreshold != 72*time.Hour {
		t.Errorf("Expected refresh threshold 72h, got %v", config.RefreshThreshold)
	}

	if config.AutoRefreshInterval != 1*time.Hour {
		t.Errorf("Expected auto refresh interval 1h, got %v", config.AutoRefreshInterval)
	}

	if !config.EnableAutoRefresh {
		t.Error("Expected auto refresh to be enabled by default")
	}
}
