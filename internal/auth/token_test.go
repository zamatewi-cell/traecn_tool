package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTraeAuth_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiredAt time.Time
		want      bool
	}{
		{"expired token", time.Now().Add(-1 * time.Hour), true},
		{"valid token", time.Now().Add(1 * time.Hour), false},
		{"expires in 5 minutes", time.Now().Add(5 * time.Minute), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &TraeAuth{Token: "test_token", ExpiredAt: tt.expiredAt}
			if got := auth.IsExpired(); got != tt.want {
				t.Errorf("TraeAuth.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTraeAuth_IsRefreshExpired(t *testing.T) {
	auth := &TraeAuth{RefreshToken: "r", RefreshExpiredAt: time.Now().Add(-24 * time.Hour)}
	if !auth.IsRefreshExpired() {
		t.Error("expected refresh token to be expired")
	}
	auth.RefreshExpiredAt = time.Now().Add(24 * time.Hour)
	if auth.IsRefreshExpired() {
		t.Error("expected refresh token to be valid")
	}
}

func TestTraeAuth_ToTokenInfo(t *testing.T) {
	exp := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	rexp := time.Now().Add(72 * time.Hour).Truncate(time.Second)
	a := &TraeAuth{
		Token:            "tok",
		RefreshToken:     "rtok",
		ExpiredAt:        exp,
		RefreshExpiredAt: rexp,
		UserID:           "u-1",
	}
	ti := a.ToTokenInfo()
	if ti.AccessToken != "tok" || ti.RefreshToken != "rtok" || ti.UserID != "u-1" {
		t.Errorf("ToTokenInfo mapping wrong: %+v", ti)
	}
	if !ti.ExpiresAt.Equal(exp) || !ti.RefreshExpiresAt.Equal(rexp) {
		t.Errorf("ToTokenInfo time mapping wrong: %+v", ti)
	}
}

func TestTokenInfo_ExpiryHelpers(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name         string
		token        *TokenInfo
		wantExpired  bool
		wantNeeds5m  bool
		wantCanFresh bool
	}{
		{
			name:         "valid far-future",
			token:        &TokenInfo{AccessToken: "t", ExpiresAt: now.Add(time.Hour), RefreshToken: "r", RefreshExpiresAt: now.Add(24 * time.Hour)},
			wantExpired:  false,
			wantNeeds5m:  false,
			wantCanFresh: true,
		},
		{
			name:         "inside early window (4m)",
			token:        &TokenInfo{AccessToken: "t", ExpiresAt: now.Add(4 * time.Minute), RefreshToken: "r", RefreshExpiresAt: now.Add(24 * time.Hour)},
			wantExpired:  false,
			wantNeeds5m:  true,
			wantCanFresh: true,
		},
		{
			name:         "hard expired",
			token:        &TokenInfo{AccessToken: "t", ExpiresAt: now.Add(-time.Minute)},
			wantExpired:  true,
			wantNeeds5m:  true,
			wantCanFresh: false,
		},
		{
			name:         "no refresh token",
			token:        &TokenInfo{AccessToken: "t", ExpiresAt: now.Add(time.Hour)},
			wantExpired:  false,
			wantNeeds5m:  false,
			wantCanFresh: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsExpired(); got != tt.wantExpired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.wantExpired)
			}
			if got := tt.token.NeedsRefresh(5 * time.Minute); got != tt.wantNeeds5m {
				t.Errorf("NeedsRefresh(5m) = %v, want %v", got, tt.wantNeeds5m)
			}
			if got := tt.token.CanRefresh(); got != tt.wantCanFresh {
				t.Errorf("CanRefresh() = %v, want %v", got, tt.wantCanFresh)
			}
		})
	}
}

func TestLoadTokenFromStorage_FileNotFound(t *testing.T) {
	if _, err := LoadTokenFromStorage("/nonexistent/path/storage.json"); err == nil {
		t.Error("LoadTokenFromStorage() error = nil, want error for missing file")
	}
}

func TestLoadTokenFromStorage_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "storage.json")
	if err := os.WriteFile(storagePath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if _, err := LoadTokenFromStorage(storagePath); err == nil {
		t.Error("LoadTokenFromStorage() error = nil, want error for invalid JSON")
	}
}

func TestLoadTokenFromStorage_PlaintextAuth(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "storage.json")
	inner, _ := json.Marshal(map[string]string{
		"token":        "plain_tok",
		"refreshToken": "plain_rtok",
		"expiredAt":    time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
		"userId":       "u-9",
	})
	outer, _ := json.Marshal(map[string]string{
		"iCubeAuthInfo://icube.cloudide": string(inner),
	})
	if err := os.WriteFile(storagePath, outer, 0644); err != nil {
		t.Fatal(err)
	}
	a, err := LoadTokenFromStorage(storagePath)
	if err != nil {
		t.Fatalf("LoadTokenFromStorage() error = %v", err)
	}
	if a.Token != "plain_tok" || a.RefreshToken != "plain_rtok" || a.UserID != "u-9" {
		t.Errorf("unexpected auth: %+v", a)
	}
}
