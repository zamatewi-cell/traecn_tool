package auth

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type concurrentMockRefresher struct {
	mu       sync.Mutex
	callLogs []string
}

func (m *concurrentMockRefresher) Refresh(ctx context.Context, refreshToken string) (*TokenInfo, error) {
	m.mu.Lock()
	m.callLogs = append(m.callLogs, refreshToken)
	m.mu.Unlock()

	// 模拟账号 2 和 4 上游 401 彻底失效（包含刷新令牌被撤销）
	if refreshToken == "ref_2" || refreshToken == "ref_4" {
		return nil, fmt.Errorf("upstream token revoked: HTTP 401 Unauthorized")
	}

	return &TokenInfo{
		AccessToken:  "refreshed_tok_" + refreshToken,
		RefreshToken: "refreshed_ref_" + refreshToken,
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}, nil
}

// TestAdversarial_Pool_5SameNameAccounts_ConcurrentRefreshAndFailureIsolation 对抗验证任务 3：
// 1. 构造 5 个名称完全相同（全部叫 "default"）但拥有不同稳定唯一 ID 的账号；
// 2. 并发模拟其中某些账号刷新、某些账号报告 401 失败以及并发模糊歧义 ReportFailure；
// 3. 验证仅有真正失败的 ID 被标记失效，其余同名健康账号不受任何干扰；
// 4. 验证刷新回调精确归属目标 ID，未刷新的同名账号凭据绝无被篡改或覆盖；
// 5. 并发压力下验证 GetToken() 100% 自动降级至同名健康账号，绝不向调用方泄露已失效账号。
func TestAdversarial_Pool_5SameNameAccounts_ConcurrentRefreshAndFailureIsolation(t *testing.T) {
	mockRef := &concurrentMockRefresher{}

	var refreshEvents []string
	var eventMu sync.Mutex

	p := newTestPool(&PoolOptions{
		Refresher: mockRef,
		OnTokenRefreshed: func(id, name string, tok *TokenInfo) {
			eventMu.Lock()
			refreshEvents = append(refreshEvents, fmt.Sprintf("%s:%s:%s", id, name, tok.AccessToken))
			eventMu.Unlock()
		},
	})

	const numAccounts = 5
	for i := 1; i <= numAccounts; i++ {
		id := fmt.Sprintf("acc-id-%d", i)
		name := "default"
		initToken := fmt.Sprintf("token_%d_init", i)
		refToken := fmt.Sprintf("ref_%d", i)

		// 账号 1 和 账号 3 的 Token 设置为过期，以便触发刷新
		var exp time.Time
		if i == 1 || i == 3 {
			exp = time.Now().Add(-time.Minute)
		} else {
			exp = time.Now().Add(time.Hour)
		}

		p.AddAccountWithCredentialsAndID(id, name, initToken, refToken, fmt.Sprintf("user-%d", i), exp, time.Now().Add(24*time.Hour), false)
	}

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	// 协程组 1: 并发高频刷新 acc-id-1 与 acc-id-3
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(gIdx int) {
			defer wg.Done()
			<-startBarrier
			targetID := "acc-id-1"
			if gIdx%2 == 1 {
				targetID = "acc-id-3"
			}
			acc, ok := p.FindAccountByID(targetID)
			if ok {
				acc.mu.Lock()
				_ = p.ensureFresh(acc)
				acc.mu.Unlock()
			}
		}(i)
	}

	// 协程组 2: 并发向 acc-id-2 与 acc-id-4 上报 401 失败
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(gIdx int) {
			defer wg.Done()
			<-startBarrier
			targetID := "acc-id-2"
			if gIdx%2 == 1 {
				targetID = "acc-id-4"
			}
			p.ReportFailure(targetID)
		}(i)
	}

	// 协程组 3: 并发高频尝试使用重名 "default" 模糊上报失败（验证同名歧义拦截保护）
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startBarrier
			p.ReportFailure("default")
		}()
	}

	close(startBarrier)
	wg.Wait()

	// 协程组 4: 此时并发刷新与失败上报已完成，启动多协程并发高频抢占调用 GetToken
	// 验证池运转安全无死锁，且 100% 平滑降级至健康账号
	var getWg sync.WaitGroup
	for i := 0; i < 8; i++ {
		getWg.Add(1)
		go func() {
			defer getWg.Done()
			for j := 0; j < 20; j++ {
				tok, id, err := p.GetToken()
				if err != nil {
					t.Errorf("GetToken failed: %v", err)
					return
				}
				// 必须绝不返回已被标记失效且刷新失败的 acc-id-2 或 acc-id-4
				if id == "acc-id-2" || id == "acc-id-4" {
					t.Errorf("ADVERSARIAL FAIL: GetToken returned stale account %s with token %s", id, tok)
				}
			}
		}()
	}
	getWg.Wait()

	// 验证最终状态隔离性
	accMap := make(map[string]*Account)
	for i := 1; i <= numAccounts; i++ {
		id := fmt.Sprintf("acc-id-%d", i)
		acc, ok := p.FindAccountByID(id)
		if !ok {
			t.Fatalf("account %s missing from pool", id)
		}
		accMap[id] = acc
	}

	// 1. 验证被指定上报失败的账号精确被标记失效
	if !accMap["acc-id-2"].stale {
		t.Fatalf("ADVERSARIAL FAIL: acc-id-2 was NOT marked stale after ReportFailure('acc-id-2')")
	}
	if !accMap["acc-id-4"].stale {
		t.Fatalf("ADVERSARIAL FAIL: acc-id-4 was NOT marked stale after ReportFailure('acc-id-4')")
	}

	// 2. 验证其余同名健康账号 100% 未被误杀
	if accMap["acc-id-1"].stale {
		t.Fatalf("ADVERSARIAL FAIL: acc-id-1 was wrongly marked stale due to name collision or ambiguous failure!")
	}
	if accMap["acc-id-3"].stale {
		t.Fatalf("ADVERSARIAL FAIL: acc-id-3 was wrongly marked stale due to name collision or ambiguous failure!")
	}
	if accMap["acc-id-5"].stale {
		t.Fatalf("ADVERSARIAL FAIL: acc-id-5 was wrongly marked stale due to name collision or ambiguous failure!")
	}

	// 3. 验证未参与刷新的同名账号 5 凭据绝无被同名覆盖
	acc5 := accMap["acc-id-5"]
	acc5.mu.Lock()
	if acc5.token.AccessToken != "token_5_init" {
		t.Fatalf("ADVERSARIAL FAIL: acc-id-5 token overwritten! got %s, want token_5_init", acc5.token.AccessToken)
	}
	acc5.mu.Unlock()

	// 4. 验证刷新成功的账号凭据
	acc1 := accMap["acc-id-1"]
	acc1.mu.Lock()
	if acc1.token.AccessToken != "refreshed_tok_ref_1" {
		t.Fatalf("acc-id-1 token mismatch: got %s, want refreshed_tok_ref_1", acc1.token.AccessToken)
	}
	acc1.mu.Unlock()

	acc3 := accMap["acc-id-3"]
	acc3.mu.Lock()
	if acc3.token.AccessToken != "refreshed_tok_ref_3" {
		t.Fatalf("acc-id-3 token mismatch: got %s, want refreshed_tok_ref_3", acc3.token.AccessToken)
	}
	acc3.mu.Unlock()

	// 5. 验证回调事件中 ID 绝对精确
	eventMu.Lock()
	defer eventMu.Unlock()
	for _, ev := range refreshEvents {
		if ev[:9] != "acc-id-1:" && ev[:9] != "acc-id-3:" {
			t.Fatalf("ADVERSARIAL FAIL: unexpected ID in refresh event: %s", ev)
		}
	}

	t.Logf("Adversarial pool isolation passed! 5 same-name accounts, strictly isolated by unique ID.")
}
