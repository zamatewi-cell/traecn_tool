package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/queue"
)

// mockRefresher implements RefreshClient for tests.
type mockRefresher struct {
	calls    atomic.Int32
	fail     bool
	newToken string
}

func (m *mockRefresher) Refresh(ctx context.Context, refreshToken string) (*TokenInfo, error) {
	m.calls.Add(1)
	if m.fail {
		return nil, fmt.Errorf("refresh boom")
	}
	tok := m.newToken
	if tok == "" {
		tok = "refreshed_token"
	}
	return &TokenInfo{
		AccessToken:      tok,
		RefreshToken:     "new_refresh",
		ExpiresAt:        time.Now().Add(2 * time.Hour),
		RefreshExpiresAt: time.Now().Add(72 * time.Hour),
	}, nil
}

func newTestPool(opts *PoolOptions) *Pool {
	return NewPool(opts)
}

// addRawAccount inserts an account with a prebuilt token (test helper).
func (p *Pool) addRawAccount(name string, src CredentialSource, tok *TokenInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.accounts = append(p.accounts, &Account{
		Name:    name,
		Source:  src,
		token:   tok,
		breaker: queue.NewCircuitBreaker(p.opts.BreakerConfig),
	})
}

func validToken(id string) *TokenInfo {
	return &TokenInfo{
		AccessToken:      "token_" + id,
		RefreshToken:     "refresh_" + id,
		ExpiresAt:        time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
	}
}

func TestPool_GetToken_RoundRobin(t *testing.T) {
	p := newTestPool(nil)
	for i := 1; i <= 3; i++ {
		id := string(rune('0' + i))
		p.addRawAccount("account-"+id, CredentialSource{Type: SourceToken}, validToken(id))
	}

	expected := []string{"token_1", "token_2", "token_3", "token_1", "token_2"}
	for i, want := range expected {
		got, _, err := p.GetToken()
		if err != nil {
			t.Fatalf("GetToken() call %d error = %v", i, err)
		}
		if got != want {
			t.Errorf("GetToken() call %d = %v, want %v", i, got, want)
		}
	}
}

func TestPool_GetToken_NoAccounts(t *testing.T) {
	p := newTestPool(nil)
	if _, _, err := p.GetToken(); err == nil {
		t.Error("GetToken() error = nil, want error for empty pool")
	}
}

func TestPool_GetToken_SkipExpired(t *testing.T) {
	p := newTestPool(nil)
	p.addRawAccount("expired", CredentialSource{Type: SourceToken}, &TokenInfo{
		AccessToken: "dead", ExpiresAt: time.Now().Add(-time.Hour),
	})
	p.addRawAccount("valid", CredentialSource{Type: SourceToken}, validToken("v"))

	tok, name, err := p.GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if tok != "token_v" || name != "valid" {
		t.Errorf("GetToken() = (%v,%v), want (token_v,valid)", tok, name)
	}
}

func TestPool_GetToken_AllExpired(t *testing.T) {
	p := newTestPool(nil)
	p.addRawAccount("e1", CredentialSource{Type: SourceToken}, &TokenInfo{
		AccessToken: "t1", ExpiresAt: time.Now().Add(-time.Hour),
	})
	p.addRawAccount("e2", CredentialSource{Type: SourceToken}, &TokenInfo{
		AccessToken: "t2", ExpiresAt: time.Now().Add(-2 * time.Hour),
	})
	if _, _, err := p.GetToken(); err == nil {
		t.Error("GetToken() error = nil, want error when all accounts expired")
	}
}

func TestPool_EarlyRefreshViaAPI(t *testing.T) {
	mr := &mockRefresher{newToken: "fresh_api_token"}
	p := newTestPool(&PoolOptions{Refresher: mr})

	// Token expires in 4 minutes — inside the 5m early window.
	p.addRawAccount("acc", CredentialSource{Type: SourceToken}, &TokenInfo{
		AccessToken:      "old_token",
		RefreshToken:     "rt",
		ExpiresAt:        time.Now().Add(4 * time.Minute),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
	})

	tok, _, err := p.GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if tok != "fresh_api_token" {
		t.Errorf("GetToken() = %v, want fresh_api_token (API refresh)", tok)
	}
	if mr.calls.Load() != 1 {
		t.Errorf("refresher calls = %d, want 1", mr.calls.Load())
	}
}

func TestPool_RefreshFailureFallsBackToValidOldToken(t *testing.T) {
	mr := &mockRefresher{fail: true}
	p := newTestPool(&PoolOptions{Refresher: mr})

	// Inside early window but NOT hard-expired: smooth fallback keeps it.
	p.addRawAccount("acc", CredentialSource{Type: SourceToken}, &TokenInfo{
		AccessToken:      "still_ok",
		RefreshToken:     "rt",
		ExpiresAt:        time.Now().Add(4 * time.Minute),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
	})

	tok, _, err := p.GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v, want smooth fallback to old token", err)
	}
	if tok != "still_ok" {
		t.Errorf("GetToken() = %v, want still_ok (fallback)", tok)
	}
}

func TestPool_RefreshFailureHardExpired(t *testing.T) {
	mr := &mockRefresher{fail: true}
	p := newTestPool(&PoolOptions{Refresher: mr})
	p.addRawAccount("acc", CredentialSource{Type: SourceToken}, &TokenInfo{
		AccessToken:      "dead",
		RefreshToken:     "rt",
		ExpiresAt:        time.Now().Add(-time.Minute),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
	})
	if _, _, err := p.GetToken(); err == nil {
		t.Error("GetToken() error = nil, want error for hard-expired token with failing refresh")
	}
}

// writePlainStorage writes a storage.json with a plaintext (unencrypted) auth value.
func writePlainStorage(t *testing.T, dir, token string, exp time.Time) string {
	t.Helper()
	path := filepath.Join(dir, "storage.json")
	inner, err := json.Marshal(map[string]string{
		"token":     token,
		"expiredAt": exp.UTC().Format(time.RFC3339Nano),
		"userId":    "u-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	outer, err := json.Marshal(map[string]string{
		"iCubeAuthInfo://icube.cloudide": string(inner),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, outer, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPool_StorageReloadOnExpiry(t *testing.T) {
	dir := t.TempDir()
	path := writePlainStorage(t, dir, "reloaded_token", time.Now().Add(time.Hour))

	p := newTestPool(nil) // no API refresher — must fall back to storage reload
	p.addRawAccount("acc", CredentialSource{Type: SourceStorage, StoragePath: path}, &TokenInfo{
		AccessToken: "stale",
		ExpiresAt:   time.Now().Add(time.Minute), // inside early window
	})

	tok, _, err := p.GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if tok != "reloaded_token" {
		t.Errorf("GetToken() = %v, want reloaded_token (storage reload)", tok)
	}
}

func TestPool_CircuitBreakerTripsAndSkips(t *testing.T) {
	cfg := queue.CircuitBreakerConfig{FailureThreshold: 2, Timeout: 50 * time.Millisecond, HalfOpenMaxCalls: 1}
	p := newTestPool(&PoolOptions{BreakerConfig: cfg})
	p.addRawAccount("bad", CredentialSource{Type: SourceToken}, &TokenInfo{
		AccessToken: "dead", ExpiresAt: time.Now().Add(-time.Hour),
	})
	p.addRawAccount("good", CredentialSource{Type: SourceToken}, validToken("g"))

	// Two failures should trip "bad" open.
	for i := 0; i < 2; i++ {
		if _, _, err := p.GetToken(); err != nil {
			t.Fatalf("GetToken() call %d error = %v", i, err)
		}
	}
	for _, info := range p.GetAccounts() {
		if info.Name == "bad" && info.CircuitState != string(queue.StatusOpen) {
			t.Errorf("bad account circuit = %v, want open", info.CircuitState)
		}
	}

	// Subsequent calls must go straight to "good" without touching "bad".
	tok, name, err := p.GetToken()
	if err != nil || name != "good" {
		t.Fatalf("GetToken() = (%v,%v,%v), want good account", tok, name, err)
	}

	// After the breaker timeout, half-open probe re-includes "bad" but it
	// still fails; pool keeps serving "good".
	time.Sleep(60 * time.Millisecond)
	if _, name, err := p.GetToken(); err != nil || name != "good" {
		t.Fatalf("GetToken() after cooldown = (%v,%v), want good", name, err)
	}
}

func TestPool_ReportFailureMarksStale(t *testing.T) {
	dir := t.TempDir()
	path := writePlainStorage(t, dir, "storage_fresh", time.Now().Add(time.Hour))

	p := newTestPool(nil)
	if err := p.AddAccount("acc", path); err != nil {
		t.Fatal(err)
	}

	if tok, _, _ := p.GetToken(); tok != "storage_fresh" {
		t.Fatalf("initial GetToken() = %v", tok)
	}

	// Simulate an upstream 401: account goes stale and gets reloaded.
	p.ReportFailure("acc")

	// Overwrite storage with a renewed token.
	writePlainStorage(t, dir, "storage_renewed", time.Now().Add(time.Hour))
	tok, _, err := p.GetToken()
	if err != nil {
		t.Fatalf("GetToken() after stale error = %v", err)
	}
	if tok != "storage_renewed" {
		t.Errorf("GetToken() = %v, want storage_renewed after stale reload", tok)
	}
}

func TestPool_AddAccountWithToken(t *testing.T) {
	p := newTestPool(nil)
	p.AddAccountWithToken("manual", "manual_tok")
	tok, name, err := p.GetToken()
	if err != nil || tok != "manual_tok" || name != "manual" {
		t.Errorf("GetToken() = (%v,%v,%v)", tok, name, err)
	}
	infos := p.GetAccounts()
	if len(infos) != 1 || infos[0].Source != string(SourceToken) {
		t.Errorf("GetAccounts() = %+v", infos)
	}
}

func TestPool_AddAccountWithCredentials(t *testing.T) {
	p := newTestPool(nil)
	p.AddAccountWithCredentials("acc_cred", "tok_val", "ref_val", "u_123", time.Time{}, time.Time{})

	tok, name, err := p.GetToken()
	if err != nil || tok != "tok_val" || name != "acc_cred" {
		t.Fatalf("GetToken() = (%v, %v, %v)", tok, name, err)
	}

	p.mu.RLock()
	acc := p.accounts[0]
	p.mu.RUnlock()

	acc.mu.Lock()
	defer acc.mu.Unlock()
	if acc.token.RefreshToken != "ref_val" {
		t.Errorf("RefreshToken = %v, want ref_val", acc.token.RefreshToken)
	}
	if acc.token.UserID != "u_123" {
		t.Errorf("UserID = %v, want u_123", acc.token.UserID)
	}
	// 验证时间非零且在当前时间之后（2小时短期兜底而非10年）
	if acc.token.ExpiresAt.IsZero() || !acc.token.ExpiresAt.After(time.Now()) {
		t.Errorf("ExpiresAt was not set properly: %v", acc.token.ExpiresAt)
	}
	if acc.token.ExpiresAt.After(time.Now().Add(3 * time.Hour)) {
		t.Errorf("ExpiresAt exceeds 2 hours fallback: %v", acc.token.ExpiresAt)
	}
}

func TestPool_OnTokenRefreshedCallback(t *testing.T) {
	var callbackCalled bool
	var callbackAcc string
	var callbackToken string

	mockRef := &mockRefresher{
		newToken: "renewed_token",
	}

	opts := &PoolOptions{
		Refresher: mockRef,
		OnTokenRefreshed: func(accName string, token *TokenInfo) {
			callbackCalled = true
			callbackAcc = accName
			callbackToken = token.AccessToken
		},
	}
	p := newTestPool(opts)
	// 添加一个即将过期的 token 触发刷新
	p.AddAccountWithCredentials("test_acc", "old_token", "old_refresh", "uid_1", time.Now().Add(-time.Minute), time.Now().Add(time.Hour))

	tok, name, err := p.GetToken()
	if err != nil {
		t.Fatalf("GetToken error: %v", err)
	}
	if tok != "renewed_token" || name != "test_acc" {
		t.Errorf("GetToken() = (%v, %v), want (renewed_token, test_acc)", tok, name)
	}
	if !callbackCalled {
		t.Error("expected OnTokenRefreshed callback to be invoked")
	}
	if callbackAcc != "test_acc" || callbackToken != "renewed_token" {
		t.Errorf("callback received (%v, %v)", callbackAcc, callbackToken)
	}
}

func TestPool_DuplicateNameIsolation_P2_6(t *testing.T) {
	cfg := queue.CircuitBreakerConfig{FailureThreshold: 2, Timeout: 50 * time.Millisecond}
	p := newTestPool(&PoolOptions{BreakerConfig: cfg})

	// 两个同名账号，不同 ID
	p.AddAccountWithCredentialsAndID("id-1", "default", "token-1", "", "", time.Time{}, time.Time{}, false)
	p.AddAccountWithCredentialsAndID("id-2", "default", "token-2", "", "", time.Time{}, time.Time{}, false)

	// 对 id-1 上报失败 2 次使其熔断
	p.ReportFailure("id-1")
	p.ReportFailure("id-1")

	accounts := p.GetAccounts()
	var acc1State, acc2State string
	for _, a := range accounts {
		if a.ID == "id-1" {
			acc1State = a.CircuitState
		} else if a.ID == "id-2" {
			acc2State = a.CircuitState
		}
	}

	if acc1State != string(queue.StatusOpen) {
		t.Errorf("id-1 circuit state = %v, want open", acc1State)
	}
	if acc2State != string(queue.StatusClosed) {
		t.Errorf("id-2 circuit state = %v, want closed (should not be affected by id-1)", acc2State)
	}

	// GetToken 应跳过 id-1，始终返回 id-2 的 token
	tok, id, err := p.GetToken()
	if err != nil {
		t.Fatalf("GetToken error: %v", err)
	}
	if tok != "token-2" || id != "id-2" {
		t.Errorf("GetToken() = (%v, %v), want (token-2, id-2)", tok, id)
	}
}

func TestPool_ConsecutiveAuthFailureCircuitBreaker_P2_7(t *testing.T) {
	cfg := queue.CircuitBreakerConfig{FailureThreshold: 2, Timeout: 50 * time.Millisecond}
	mr := &mockRefresher{newToken: "refreshed_tok"}
	p := newTestPool(&PoolOptions{BreakerConfig: cfg, Refresher: mr})

	p.AddAccountWithCredentials("acc", "tok", "refresh_tok", "u1", time.Now().Add(time.Hour), time.Now().Add(24*time.Hour))

	// 模拟真实网关场景：
	// 第 1 次请求：GetToken -> 上游 401 -> ReportFailure (failure=1)
	tok1, id1, err := p.GetToken()
	if err != nil || tok1 != "tok" {
		t.Fatalf("call 1 GetToken failed: %v", err)
	}
	p.ReportFailure(id1)

	// 第 2 次请求：GetToken 触发 refresh 拿到新 token，但不应清零失败计数！
	// 接着上游依然 401 -> ReportFailure (failure=2，达到阈值)
	tok2, id2, err := p.GetToken()
	if err != nil || tok2 != "refreshed_tok" {
		t.Fatalf("call 2 GetToken failed: %v", err)
	}
	p.ReportFailure(id2)

	// 此时失败达到 2 次，账号应当进入 Open 状态
	accs := p.GetAccounts()
	if len(accs) != 1 || accs[0].CircuitState != string(queue.StatusOpen) {
		t.Fatalf("expected circuit open after 2 failures, got: %+v", accs)
	}

	// 第 3 次请求：此时账号已熔断，GetToken 应当拒绝
	_, _, err = p.GetToken()
	if err == nil {
		t.Fatal("expected GetToken to fail when account is circuit-broken")
	}
}

func TestPool_ActiveAccountPreferredAndFallback_P2_10(t *testing.T) {
	cfg := queue.CircuitBreakerConfig{FailureThreshold: 2, Timeout: 50 * time.Millisecond}
	p := newTestPool(&PoolOptions{BreakerConfig: cfg})

	p.AddAccountWithCredentialsAndID("acc-a", "Account A", "token-a", "", "", time.Time{}, time.Time{}, false)
	p.AddAccountWithCredentialsAndID("acc-b", "Account B", "token-b", "", "", time.Time{}, time.Time{}, true) // Active

	// 正常情况下，无论调用多少次，都优先返回 active account (acc-b)
	for i := 0; i < 5; i++ {
		tok, id, err := p.GetToken()
		if err != nil {
			t.Fatalf("call %d GetToken failed: %v", i, err)
		}
		if tok != "token-b" || id != "acc-b" {
			t.Fatalf("call %d returned (%v, %v), want (token-b, acc-b)", i, tok, id)
		}
	}

	// 使 acc-b 熔断
	p.ReportFailure("acc-b")
	p.ReportFailure("acc-b")

	// acc-b 熔断后，应自动平滑降级返回 acc-a
	tok, id, err := p.GetToken()
	if err != nil {
		t.Fatalf("fallback GetToken failed: %v", err)
	}
	if tok != "token-a" || id != "acc-a" {
		t.Fatalf("fallback returned (%v, %v), want (token-a, acc-a)", tok, id)
	}
}
