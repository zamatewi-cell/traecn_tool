package auth

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/queue"
)

// DefaultRefreshEarly is how long before expiry a token gets proactively
// refreshed: requests check the expiry clock before going out and trigger a
// refresh when the remaining lifetime drops below this window.
const DefaultRefreshEarly = 5 * time.Minute

// SourceType describes where an account credential comes from.
type SourceType string

const (
	SourceStorage SourceType = "storage" // IDE storage.json (sniffed or configured)
	SourceToken   SourceType = "token"   // manually provided token (config.json)
	SourceEnv     SourceType = "env"     // environment variable
)

// CredentialSource locates an account credential and knows how to reload it.
type CredentialSource struct {
	Type        SourceType
	StoragePath string // for SourceStorage
	EnvVar      string // for SourceEnv
}

// Load (re)loads a TokenInfo from the source. SourceToken is static and
// cannot be reloaded.
func (s CredentialSource) Load() (*TokenInfo, error) {
	switch s.Type {
	case SourceStorage:
		a, err := LoadTokenFromStorage(s.StoragePath)
		if err != nil {
			return nil, err
		}
		return a.ToTokenInfo(), nil
	case SourceEnv:
		v := strings.TrimSpace(os.Getenv(s.EnvVar))
		if v == "" {
			return nil, fmt.Errorf("env var %s is empty", s.EnvVar)
		}
		return DirectTokenInfo(v), nil
	default:
		return nil, fmt.Errorf("credential source type %q cannot be reloaded", s.Type)
	}
}

// DirectTokenInfo wraps a manually provided token with a far-future expiry
// (we cannot know its real lifetime; the upstream will 401 when it dies).
func DirectTokenInfo(token string) *TokenInfo {
	farFuture := time.Now().Add(10 * 365 * 24 * time.Hour)
	return &TokenInfo{
		AccessToken:      token,
		ExpiresAt:        farFuture,
		RefreshExpiresAt: farFuture,
	}
}

// IsExpired reports whether the access token has hard-expired.
func (t *TokenInfo) IsExpired() bool {
	return !time.Now().Before(t.ExpiresAt)
}

// NeedsRefresh reports whether the token expires within the early-refresh
// window (or already expired).
func (t *TokenInfo) NeedsRefresh(early time.Duration) bool {
	return time.Now().Add(early).After(t.ExpiresAt)
}

// CanRefresh reports whether an API refresh can be attempted.
func (t *TokenInfo) CanRefresh() bool {
	return t.RefreshToken != "" && time.Now().Before(t.RefreshExpiresAt)
}

// RefreshClient abstracts token refresh so it can be mocked in tests.
// *TokenRefresher implements it.
type RefreshClient interface {
	Refresh(ctx context.Context, refreshToken string) (*TokenInfo, error)
}

// Account is one credential in the pool with its own circuit breaker.
type Account struct {
	ID        string
	Name      string
	Source    CredentialSource
	IsCurrent bool

	mu      sync.Mutex
	token   *TokenInfo
	stale   bool // force reload on next use (e.g. after an upstream 401)
	breaker *queue.CircuitBreaker
	lastErr error
}

// PoolOptions configures a Pool.
type PoolOptions struct {
	// RefreshEarly is the proactive refresh window (default 5m).
	RefreshEarly time.Duration
	// Refresher performs API token refresh; nil disables it.
	Refresher RefreshClient
	// BreakerConfig tunes per-account circuit breakers.
	BreakerConfig queue.CircuitBreakerConfig
	// Logger for refresh/failure diagnostics; nil discards.
	Logger *slog.Logger
	// OnTokenRefreshed is triggered immediately after a successful API token refresh.
	OnTokenRefreshed func(accountName string, token *TokenInfo)
}

func (o *PoolOptions) withDefaults() PoolOptions {
	out := PoolOptions{
		RefreshEarly:  DefaultRefreshEarly,
		BreakerConfig: queue.DefaultCircuitBreakerConfig(),
		Logger:        slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}
	if o != nil {
		if o.RefreshEarly > 0 {
			out.RefreshEarly = o.RefreshEarly
		}
		if o.Refresher != nil {
			out.Refresher = o.Refresher
		}
		if o.BreakerConfig.FailureThreshold > 0 {
			out.BreakerConfig = o.BreakerConfig
		}
		if o.Logger != nil {
			out.Logger = o.Logger
		}
		if o.OnTokenRefreshed != nil {
			out.OnTokenRefreshed = o.OnTokenRefreshed
		}
	}
	return out
}

// Pool manages multiple accounts with round-robin selection, proactive
// token refresh and per-account circuit breaking / degradation.
type Pool struct {
	mu              sync.RWMutex
	accounts        []*Account
	index           int
	activeAccountID string
	opts            PoolOptions
}

// NewPool creates an empty account pool.
func NewPool(opts *PoolOptions) *Pool {
	return &Pool{opts: opts.withDefaults()}
}

// SetActiveAccount sets the currently active account ID for preferred dispatch.
func (p *Pool) SetActiveAccount(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeAccountID = id
}

// ActiveAccount returns the current active account ID.
func (p *Pool) ActiveAccount() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeAccountID
}

// AddSource registers an account and eagerly loads its credential.
func (p *Pool) AddSource(name string, src CredentialSource) error {
	return p.AddSourceWithID(name, name, src, false)
}

// AddSourceWithID registers an account with an explicit ID and isCurrent flag.
func (p *Pool) AddSourceWithID(id, name string, src CredentialSource, isCurrent bool) error {
	tok, err := src.Load()
	if err != nil {
		return fmt.Errorf("failed to load credential for %s: %w", name, err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.accounts = append(p.accounts, &Account{
		ID:        id,
		Name:      name,
		Source:    src,
		IsCurrent: isCurrent,
		token:     tok,
		breaker:   queue.NewCircuitBreaker(p.opts.BreakerConfig),
	})
	if isCurrent && p.activeAccountID == "" {
		p.activeAccountID = id
	}
	return nil
}

// AddAccount registers an account backed by a storage.json file.
func (p *Pool) AddAccount(name, storagePath string) error {
	return p.AddSource(name, CredentialSource{Type: SourceStorage, StoragePath: storagePath})
}

// AddAccountFromEnv registers an account backed by an environment variable.
func (p *Pool) AddAccountFromEnv(name, envVar string) error {
	return p.AddSource(name, CredentialSource{Type: SourceEnv, EnvVar: envVar})
}

// AddAccountWithToken registers an account with a static token.
func (p *Pool) AddAccountWithToken(name, token string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.accounts = append(p.accounts, &Account{
		ID:      name,
		Name:    name,
		Source:  CredentialSource{Type: SourceToken},
		token:   DirectTokenInfo(token),
		breaker: queue.NewCircuitBreaker(p.opts.BreakerConfig),
	})
}

// AddAccountWithCredentials registers an account with token, refresh credentials, and expiration metadata.
func (p *Pool) AddAccountWithCredentials(name, token, refreshToken, userID string, expiresAt, refreshExpiresAt time.Time) {
	p.AddAccountWithCredentialsAndID(name, name, token, refreshToken, userID, expiresAt, refreshExpiresAt, false)
}

// AddAccountWithCredentialsAndID registers an account with explicit ID, credentials, and isCurrent flag.
func (p *Pool) AddAccountWithCredentialsAndID(id, name, token, refreshToken, userID string, expiresAt, refreshExpiresAt time.Time, isCurrent bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 真实保底逻辑：若无明确过期时间，默认赋予 2 小时短期兜底，而非 10 年伪造
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(2 * time.Hour)
	}
	if refreshExpiresAt.IsZero() && refreshToken != "" {
		refreshExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	}

	tok := &TokenInfo{
		AccessToken:      token,
		RefreshToken:     refreshToken,
		ExpiresAt:        expiresAt,
		RefreshExpiresAt: refreshExpiresAt,
		UserID:           userID,
	}

	p.accounts = append(p.accounts, &Account{
		ID:        id,
		Name:      name,
		Source:    CredentialSource{Type: SourceToken},
		IsCurrent: isCurrent,
		token:     tok,
		breaker:   queue.NewCircuitBreaker(p.opts.BreakerConfig),
	})
	if isCurrent && p.activeAccountID == "" {
		p.activeAccountID = id
	}
}

// ensureFresh makes sure the account holds a usable token, refreshing
// proactively inside the early-refresh window. Refresh chain: API refresh →
// reload from origin source (the IDE itself may have renewed storage.json) →
// smooth fallback to the still-valid old token.
//
// Must be called with acc.mu held.
func (p *Pool) ensureFresh(acc *Account) error {
	tok := acc.token
	if tok == nil {
		nt, err := acc.Source.Load()
		if err != nil {
			return err
		}
		acc.token, tok = nt, nt
	}

	if !acc.stale && tok.AccessToken != "" && !tok.NeedsRefresh(p.opts.RefreshEarly) {
		return nil
	}

	// (a) API refresh via refresh token.
	if p.opts.Refresher != nil && tok.CanRefresh() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		nt, err := p.opts.Refresher.Refresh(ctx, tok.RefreshToken)
		cancel()
		if err == nil && nt != nil && nt.AccessToken != "" {
			nt.AccountID, nt.TenantID, nt.UserID = tok.AccountID, tok.TenantID, tok.UserID
			acc.token, acc.stale = nt, false
			if p.opts.OnTokenRefreshed != nil {
				p.opts.OnTokenRefreshed(acc.Name, nt)
			}
			return nil
		}
		if err != nil {
			p.opts.Logger.Warn("api token refresh failed, trying source reload", "account", acc.Name, "error", err)
		}
	}

	// (b) Reload from the origin source.
	if nt, err := acc.Source.Load(); err == nil && nt.AccessToken != "" && !nt.IsExpired() {
		acc.token, acc.stale = nt, false
		return nil
	}

	// (c) Smooth fallback: keep using the old token while it is not
	// hard-expired, even if it lives inside the early-refresh window.
	if !acc.stale && !tok.IsExpired() && tok.AccessToken != "" {
		return nil
	}

	return fmt.Errorf("credential expired and all refresh strategies failed")
}

// GetToken returns a usable token via round-robin, refreshing proactively
// and skipping circuit-broken / failing accounts. If an active account is configured,
// it is preferred as long as it is healthy and available.
func (p *Pool) GetToken() (string, string, error) {
	p.mu.RLock()
	n := len(p.accounts)
	activeID := p.activeAccountID
	p.mu.RUnlock()
	if n == 0 {
		return "", "", fmt.Errorf("no accounts available")
	}

	// 优先调度活动账号 (Active Account Preferred)
	if activeID != "" {
		p.mu.RLock()
		var activeAcc *Account
		for _, acc := range p.accounts {
			if acc.ID == activeID || (acc.ID == "" && acc.Name == activeID) {
				activeAcc = acc
				break
			}
		}
		p.mu.RUnlock()

		if activeAcc != nil && activeAcc.breaker.Allow() {
			activeAcc.mu.Lock()
			err := p.ensureFresh(activeAcc)
			if err == nil {
				token := activeAcc.token.AccessToken
				activeAcc.stale = false
				activeAcc.mu.Unlock()
				key := activeAcc.ID
				if key == "" {
					key = activeAcc.Name
				}
				return token, key, nil
			}
			activeAcc.lastErr = err
			activeAcc.mu.Unlock()

			activeAcc.breaker.RecordFailure()
			p.opts.Logger.Warn("active account unavailable, falling back to pool", "account", activeAcc.Name, "error", err)
		}
	}

	var lastErr error
	for i := 0; i < n; i++ {
		p.mu.Lock()
		idx := (p.index + i) % n
		acc := p.accounts[idx]
		p.mu.Unlock()

		if !acc.breaker.Allow() {
			continue
		}

		acc.mu.Lock()
		err := p.ensureFresh(acc)
		if err == nil {
			token := acc.token.AccessToken
			acc.stale = false
			acc.mu.Unlock()

			p.mu.Lock()
			p.index = (idx + 1) % n
			p.mu.Unlock()

			key := acc.ID
			if key == "" {
				key = acc.Name
			}
			return token, key, nil
		}
		acc.lastErr = err
		acc.mu.Unlock()

		acc.breaker.RecordFailure()
		lastErr = err
		p.opts.Logger.Warn("account unavailable", "account", acc.Name, "error", err)
	}

	if lastErr != nil {
		return "", "", fmt.Errorf("all accounts unavailable: %w", lastErr)
	}
	return "", "", fmt.Errorf("all accounts are circuit-broken")
}

// ReportFailure lets the forwarding layer report an upstream auth failure
// (e.g. HTTP 401): the account is marked stale so its credential is forcibly
// refreshed on next use, and the failure feeds the circuit breaker.
// It matches by account ID first, and falls back to account Name.
func (p *Pool) ReportFailure(accountKey string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, acc := range p.accounts {
		if acc.ID != "" && acc.ID == accountKey {
			acc.mu.Lock()
			acc.stale = true
			acc.mu.Unlock()
			acc.breaker.RecordFailure()
			return
		}
	}
	for _, acc := range p.accounts {
		if acc.Name == accountKey {
			acc.mu.Lock()
			acc.stale = true
			acc.mu.Unlock()
			acc.breaker.RecordFailure()
			return
		}
	}
}

// ReportSuccess records a successful upstream call for the account.
// It matches by account ID first, and falls back to account Name.
func (p *Pool) ReportSuccess(accountKey string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, acc := range p.accounts {
		if acc.ID != "" && acc.ID == accountKey {
			acc.breaker.RecordSuccess()
			return
		}
	}
	for _, acc := range p.accounts {
		if acc.Name == accountKey {
			acc.breaker.RecordSuccess()
			return
		}
	}
}

// AccountInfo holds account summary info.
type AccountInfo struct {
	ID           string    `json:"id,omitempty"`
	Name         string    `json:"name"`
	UserID       string    `json:"user_id"`
	Expired      bool      `json:"expired"`
	ExpiresAt    time.Time `json:"expires_at"`
	Source       string    `json:"source"`
	CircuitState string    `json:"circuit_state"`
	IsCurrent    bool      `json:"is_current,omitempty"`
}

// GetAccounts returns summary info for all accounts.
func (p *Pool) GetAccounts() []AccountInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	infos := make([]AccountInfo, 0, len(p.accounts))
	for _, acc := range p.accounts {
		acc.mu.Lock()
		isCurrent := acc.IsCurrent || (p.activeAccountID != "" && (acc.ID == p.activeAccountID || acc.Name == p.activeAccountID))
		info := AccountInfo{
			ID:           acc.ID,
			Name:         acc.Name,
			Source:       string(acc.Source.Type),
			CircuitState: string(acc.breaker.GetState()),
			IsCurrent:    isCurrent,
		}
		if acc.token != nil {
			info.UserID = acc.token.UserID
			info.Expired = acc.token.IsExpired()
			info.ExpiresAt = acc.token.ExpiresAt
		}
		acc.mu.Unlock()
		infos = append(infos, info)
	}
	return infos
}

// TokenProvider is kept as a backwards-compatible alias of Pool.
type TokenProvider = Pool

// NewTokenProvider creates a Pool with default options.
// Deprecated: use NewPool.
func NewTokenProvider() *Pool {
	return NewPool(nil)
}
