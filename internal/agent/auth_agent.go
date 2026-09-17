package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/workflow"
)

// AuthAgent manages authentication and tokens
type AuthAgent struct {
	*BaseAgent
	TokenManager   *TokenManager
	AccountPool    *AccountPool
	CircuitBreaker *CircuitBreaker
}

// TokenManager manages Token lifecycle
type TokenManager struct {
	Tokens           map[string]*TokenInfo
	mu               interface{} // Should be sync.RWMutex
	RefreshThreshold int         // Seconds before expiry to refresh
}

// TokenInfo represents Token information
type TokenInfo struct {
	Token        string
	RefreshToken string
	ExpiresAt    time.Time
	AccountID    string
	Status       string // active, expired, revoked
}

// AccountPool manages account pool
type AccountPool struct {
	Accounts  []Account
	CurrentID string
	mu        interface{} // Should be sync.RWMutex
}

// Account represents an account
type Account struct {
	ID           string
	Username     string
	Password     string
	Token        string
	Status       string // active, limited, banned
	LastUsed     time.Time
	RequestCount int
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	State        string // closed, open, half-open
	FailureCount int
	Threshold    int
	ResetTimeout time.Duration
	LastFailure  time.Time
}

// NewAuthAgent creates an auth agent
func NewAuthAgent(ctxManager *workflow.ContextManager, taskManager *workflow.TaskManager, eventBus *workflow.EventBus) *AuthAgent {
	return &AuthAgent{
		BaseAgent: NewBaseAgent("auth-agent", "Auth Agent", ctxManager, taskManager, eventBus),
		TokenManager: &TokenManager{
			Tokens:           make(map[string]*TokenInfo),
			RefreshThreshold: 300, // 5 minutes
		},
		AccountPool: &AccountPool{
			Accounts: make([]Account, 0),
		},
		CircuitBreaker: &CircuitBreaker{
			State:        "closed",
			Threshold:    5,
			ResetTimeout: 60 * time.Second,
		},
	}
}

// Run starts auth management
func (a *AuthAgent) Run(ctx context.Context, task *workflow.Task) error {
	a.SetStatus(StateRunning)

	defer func() {
		if r := recover(); r != nil {
			a.SetStatus(StateFailed)
			a.PublishEvent(workflow.EventError, map[string]interface{}{
				"error": fmt.Sprintf("%v", r),
			})
		}
	}()

	// Execute task based on type
	switch task.Type {
	case "token_generate":
		return a.generateToken(ctx, task)
	case "token_refresh":
		return a.refreshToken(ctx, task)
	case "account_rotation":
		return a.rotateAccount(ctx, task)
	default:
		return a.generateToken(ctx, task)
	}
}

// generateToken generates a new Token
func (a *AuthAgent) generateToken(ctx context.Context, task *workflow.Task) error {
	a.SetProgress(30, "Generating Token", 1, 3)

	// TODO: Implement Token generation logic
	token := &TokenInfo{
		Token:     "generated_token_xxx",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Status:    "active",
	}

	a.TokenManager.Tokens["account1"] = token

	a.SetProgress(60, "Saving Token", 2, 3)
	a.SetProgress(100, "Token generated", 3, 3)

	a.SetStatus(StateCompleted)

	a.PublishEvent(workflow.EventTaskCompleted, map[string]interface{}{
		"task_id":    task.ID,
		"token_type": "access_token",
	})

	return nil
}

// refreshToken refreshes an expiring Token
func (a *AuthAgent) refreshToken(ctx context.Context, task *workflow.Task) error {
	a.SetProgress(30, "Checking Token expiry", 1, 3)

	// Check if needs refresh
	needsRefresh := false
	for _, token := range a.TokenManager.Tokens {
		if time.Until(token.ExpiresAt) < time.Duration(a.TokenManager.RefreshThreshold)*time.Second {
			needsRefresh = true
			break
		}
	}

	if !needsRefresh {
		a.SetProgress(100, "No refresh needed", 3, 3)
		a.SetStatus(StateCompleted)
		return nil
	}

	a.SetProgress(60, "Refreshing Token", 2, 3)

	// TODO: Implement Token refresh logic

	a.SetProgress(100, "Token refreshed", 3, 3)

	a.SetStatus(StateCompleted)

	a.PublishEvent(workflow.EventTaskCompleted, map[string]interface{}{
		"task_id": task.ID,
		"action":  "token_refreshed",
	})

	return nil
}

// rotateAccount rotates to next available account
func (a *AuthAgent) rotateAccount(ctx context.Context, task *workflow.Task) error {
	a.SetProgress(30, "Checking account status", 1, 3)

	// Find next available account
	var nextAccount *Account
	for i := range a.AccountPool.Accounts {
		acc := &a.AccountPool.Accounts[i]
		if acc.Status == "active" {
			nextAccount = acc
			break
		}
	}

	if nextAccount == nil {
		a.SetProgress(100, "No available accounts", 3, 3)
		a.SetStatus(StateCompleted)
		return nil
	}

	a.SetProgress(60, "Switching account", 2, 3)
	a.AccountPool.CurrentID = nextAccount.ID

	a.SetProgress(100, "Account rotated", 3, 3)

	a.SetStatus(StateCompleted)

	a.PublishEvent(workflow.EventTaskCompleted, map[string]interface{}{
		"task_id":    task.ID,
		"account_id": nextAccount.ID,
	})

	return nil
}

// AddAccount adds an account to the pool
func (a *AuthAgent) AddAccount(acc Account) {
	a.AccountPool.Accounts = append(a.AccountPool.Accounts, acc)
}

// GetActiveAccount gets the current active account
func (a *AuthAgent) GetActiveAccount() *Account {
	for i := range a.AccountPool.Accounts {
		if a.AccountPool.Accounts[i].ID == a.AccountPool.CurrentID {
			return &a.AccountPool.Accounts[i]
		}
	}
	return nil
}

// GetToken gets Token for an account
func (a *AuthAgent) GetToken(accountID string) (*TokenInfo, bool) {
	token, exists := a.TokenManager.Tokens[accountID]
	return token, exists
}
