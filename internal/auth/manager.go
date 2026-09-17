package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenInfo represents complete token information
type TokenInfo struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	LastRefreshedAt  time.Time `json:"last_refreshed_at"`
	AccountID        string    `json:"account_id"`
	TenantID         string    `json:"tenant_id"`
	UserID           string    `json:"user_id"`
	Source           string    `json:"source"`
	SourceID         string    `json:"source_id"`
}

// TokenStatus represents current token status
type TokenStatus struct {
	IsValid        bool          `json:"is_valid"`
	IsExpired      bool          `json:"is_expired"`
	NeedsRefresh   bool          `json:"needs_refresh"`
	TimeToExpiry   time.Duration `json:"time_to_expiry"`
	CanRefresh     bool          `json:"can_refresh"`
	RefreshExpired bool          `json:"refresh_expired"`
}

// TokenConfig Token manager configuration
type TokenConfig struct {
	// Refresh threshold (default 3 days)
	RefreshThreshold time.Duration
	// Auto refresh interval (default 1 hour)
	AutoRefreshInterval time.Duration
	// Enable auto refresh
	EnableAutoRefresh bool
	// Storage path
	StoragePath string
}

// DefaultTokenConfig returns default configuration
func DefaultTokenConfig() *TokenConfig {
	return &TokenConfig{
		RefreshThreshold:    72 * time.Hour, // 3 days
		AutoRefreshInterval: 1 * time.Hour,
		EnableAutoRefresh:   true,
		StoragePath:         "", // Use default path
	}
}

// TokenManager Thread-safe token manager
type TokenManager struct {
	mu        sync.RWMutex
	tokens    map[string]*TokenInfo // accountID -> TokenInfo
	config    *TokenConfig
	pubKey    *rsa.PublicKey // For JWT verification
	refresher *TokenRefresher
	storage   TokenStorage
	stopChan  chan struct{}
	isRunning bool
}

// NewTokenManager creates new token manager
func NewTokenManager(config *TokenConfig) *TokenManager {
	if config == nil {
		config = DefaultTokenConfig()
	}

	return &TokenManager{
		tokens:    make(map[string]*TokenInfo),
		config:    config,
		stopChan:  make(chan struct{}),
		isRunning: false,
	}
}

// SetRefresher sets token refresher
func (tm *TokenManager) SetRefresher(refresher *TokenRefresher) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.refresher = refresher
}

// SetPublicKey sets RSA public key for JWT verification
func (tm *TokenManager) SetPublicKey(pubKey *rsa.PublicKey) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.pubKey = pubKey
}

// SetStorage sets token storage
func (tm *TokenManager) SetStorage(storage TokenStorage) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.storage = storage
}

// LoadFromStorage loads tokens from storage
func (tm *TokenManager) LoadFromStorage() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.storage == nil {
		return fmt.Errorf("storage not configured")
	}

	tokens, err := tm.storage.LoadAll()
	if err != nil {
		return fmt.Errorf("failed to load from storage: %w", err)
	}

	tm.tokens = tokens
	return nil
}

// SaveToStorage saves tokens to storage
func (tm *TokenManager) SaveToStorage() error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.storage == nil {
		return fmt.Errorf("storage not configured")
	}

	// Save all tokens
	for accountID, token := range tm.tokens {
		if err := tm.storage.Save(accountID, token); err != nil {
			return fmt.Errorf("failed to save token for account %s: %w", accountID, err)
		}
	}

	return nil
}

// Start starts token manager (begins auto-refresh task)
func (tm *TokenManager) Start(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.isRunning {
		return fmt.Errorf("token manager already running")
	}

	tm.isRunning = true
	go tm.autoRefreshLoop(ctx)

	return nil
}

// Stop stops token manager
func (tm *TokenManager) Stop() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.isRunning {
		return nil
	}

	close(tm.stopChan)
	tm.isRunning = false

	return nil
}

// GetToken gets token for specified account (auto-refresh if expired)
func (tm *TokenManager) GetToken(ctx context.Context, accountID string) (*TokenInfo, error) {
	tm.mu.RLock()
	token, exists := tm.tokens[accountID]
	tm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("token not found for account: %s", accountID)
	}

	// Check token status
	status := tm.CheckTokenStatus(token)

	if !status.IsValid {
		// Token invalid, need refresh
		err := tm.RefreshToken(ctx, accountID)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		// Get refreshed token
		tm.mu.RLock()
		token = tm.tokens[accountID]
		tm.mu.RUnlock()
	}

	return token, nil
}

// RefreshToken manually refreshes token for specified account
func (tm *TokenManager) RefreshToken(ctx context.Context, accountID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	token, exists := tm.tokens[accountID]
	if !exists {
		return fmt.Errorf("token not found for account: %s", accountID)
	}

	// Check if refresh token expired
	if time.Now().After(token.RefreshExpiresAt) {
		return fmt.Errorf("refresh token expired")
	}

	// Check if refresher is configured
	if tm.refresher == nil {
		return fmt.Errorf("token refresher not configured")
	}

	// Call refresh API to get new token
	newToken, err := tm.refresher.Refresh(ctx, token.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update token
	newToken.AccountID = accountID
	newToken.TenantID = token.TenantID
	newToken.UserID = token.UserID
	tm.tokens[accountID] = newToken

	return nil
}

// CheckTokenStatus checks token status
func (tm *TokenManager) CheckTokenStatus(token *TokenInfo) *TokenStatus {
	now := time.Now()
	timeToExpiry := token.ExpiresAt.Sub(now)
	refreshExpired := now.After(token.RefreshExpiresAt)

	return &TokenStatus{
		IsValid:        !refreshExpired && timeToExpiry > 0,
		IsExpired:      timeToExpiry <= 0,
		NeedsRefresh:   timeToExpiry < tm.config.RefreshThreshold && !refreshExpired,
		TimeToExpiry:   timeToExpiry,
		CanRefresh:     !refreshExpired,
		RefreshExpired: refreshExpired,
	}
}

// AddToken adds token to manager
func (tm *TokenManager) AddToken(accountID string, token *TokenInfo) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.tokens[accountID] = token

	// If storage configured, save asynchronously
	if tm.storage != nil {
		go func() {
			if err := tm.storage.Save(accountID, token); err != nil {
				fmt.Printf("Failed to save token to storage: %v\n", err)
			}
		}()
	}
}

// RemoveToken removes token from manager
func (tm *TokenManager) RemoveToken(accountID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.tokens, accountID)
}

// GetAllTokens gets all tokens (read-only copy)
func (tm *TokenManager) GetAllTokens() map[string]*TokenInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Return copy to prevent external modification
	result := make(map[string]*TokenInfo, len(tm.tokens))
	for k, v := range tm.tokens {
		result[k] = v
	}
	return result
}

// autoRefreshLoop auto-refresh background task
func (tm *TokenManager) autoRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(tm.config.AutoRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tm.stopChan:
			return
		case <-ticker.C:
			tm.checkAndRefreshAll()
		}
	}
}

// checkAndRefreshAll checks and refreshes all tokens that need refresh
func (tm *TokenManager) checkAndRefreshAll() {
	tm.mu.RLock()
	accountsToRefresh := make([]string, 0)

	for accountID, token := range tm.tokens {
		status := tm.CheckTokenStatus(token)
		if status.NeedsRefresh && status.CanRefresh {
			accountsToRefresh = append(accountsToRefresh, accountID)
		}
	}
	tm.mu.RUnlock()

	// Refresh tokens that need it
	for _, accountID := range accountsToRefresh {
		// Use independent context to avoid cancellation
		ctx := context.Background()
		err := tm.RefreshToken(ctx, accountID)
		if err != nil {
			// TODO: Log error, send alert
			fmt.Printf("Failed to refresh token for account %s: %v\n", accountID, err)
		}
	}
}

// ParseJWT parses JWT token
func ParseJWT(tokenString string, pubKey *rsa.PublicKey) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// ValidateTokenExpiry validates token expiration
func ValidateTokenExpiry(claims jwt.MapClaims) bool {
	exp, ok := claims["exp"].(float64)
	if !ok {
		return false
	}

	expiry := time.Unix(int64(exp), 0)
	return time.Now().Before(expiry)
}
