package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RefreshTokenRequest Token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse Token refresh response
type RefreshTokenResponse struct {
	Data struct {
		Token            string `json:"token"`
		RefreshToken     string `json:"refresh_token"`
		ExpiredAt        int64  `json:"expired_at"`
		RefreshExpiredAt int64  `json:"refresh_expired_at"`
	} `json:"data"`
}

// TokenRefresher Token refresher
type TokenRefresher struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
}

// NewTokenRefresher creates new token refresher
func NewTokenRefresher(baseURL string, headers map[string]string) *TokenRefresher {
	// Set default timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &TokenRefresher{
		httpClient: httpClient,
		baseURL:    baseURL,
		headers:    headers,
	}
}

// Refresh executes token refresh
func (tr *TokenRefresher) Refresh(ctx context.Context, refreshToken string) (*TokenInfo, error) {
	// Build request
	reqBody := RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", tr.baseURL+"/api/auth/refresh_token", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range tr.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := tr.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var respData RefreshTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Build TokenInfo
	now := time.Now()
	tokenInfo := &TokenInfo{
		AccessToken:      respData.Data.Token,
		RefreshToken:     respData.Data.RefreshToken,
		ExpiresAt:        time.Unix(respData.Data.ExpiredAt, 0),
		RefreshExpiresAt: time.Unix(respData.Data.RefreshExpiredAt, 0),
		LastRefreshedAt:  now,
	}

	return tokenInfo, nil
}

// SetHeader sets request header
func (tr *TokenRefresher) SetHeader(key, value string) {
	if tr.headers == nil {
		tr.headers = make(map[string]string)
	}
	tr.headers[key] = value
}

// SetTimeout sets HTTP timeout
func (tr *TokenRefresher) SetTimeout(timeout time.Duration) {
	tr.httpClient.Timeout = timeout
}
