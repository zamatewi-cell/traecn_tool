package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// TraeAuth holds the auth info as stored (encrypted) in Trae CN's
// storage.json. It is a transport DTO; the pool works with TokenInfo.
type TraeAuth struct {
	Token            string    `json:"token"`
	RefreshToken     string    `json:"refreshToken"`
	ExpiredAt        time.Time `json:"expiredAt"`
	RefreshExpiredAt time.Time `json:"refreshExpiredAt"`
	UserID           string    `json:"userId"`
	Host             string    `json:"host"`
	UserRegion       struct {
		Region   string `json:"region"`
		AIRegion string `json:"_aiRegion"`
	} `json:"userRegion"`
	Account struct {
		Username   string `json:"username"`
		Scope      string `json:"scope"`
		LoginScope string `json:"loginScope"`
	} `json:"account"`
}

// IsExpired checks if token has expired
func (a *TraeAuth) IsExpired() bool {
	return time.Now().After(a.ExpiredAt)
}

// IsRefreshExpired checks if refresh token has expired
func (a *TraeAuth) IsRefreshExpired() bool {
	return time.Now().After(a.RefreshExpiredAt)
}

// ToTokenInfo converts the storage DTO into the pool's TokenInfo.
func (a *TraeAuth) ToTokenInfo() *TokenInfo {
	return &TokenInfo{
		AccessToken:      a.Token,
		RefreshToken:     a.RefreshToken,
		ExpiresAt:        a.ExpiredAt,
		RefreshExpiresAt: a.RefreshExpiredAt,
		UserID:           a.UserID,
	}
}

// LoadTokenFromStorage reads the auth info from Trae CN's storage.json,
// transparently decrypting the AES-128-CBC blob when necessary.
func LoadTokenFromStorage(storagePath string) (*TraeAuth, error) {
	data, err := os.ReadFile(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var storage map[string]json.RawMessage
	if err := json.Unmarshal(data, &storage); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	authKey := "iCubeAuthInfo://icube.cloudide"
	raw, ok := storage[authKey]
	if !ok {
		return nil, fmt.Errorf("auth key not found: %s", authKey)
	}

	// The value can be an encrypted base64 blob or a raw JSON string
	var authStr string
	if err := json.Unmarshal(raw, &authStr); err != nil {
		return nil, fmt.Errorf("failed to parse auth string: %w", err)
	}

	var authBytes []byte
	decrypted, err := DecryptTraeBlob(authStr)
	if err == nil {
		authBytes = decrypted
	} else {
		authBytes = []byte(authStr)
	}

	var auth TraeAuth
	if err := json.Unmarshal(authBytes, &auth); err != nil {
		return nil, fmt.Errorf("failed to parse auth object: %w", err)
	}

	return &auth, nil
}
