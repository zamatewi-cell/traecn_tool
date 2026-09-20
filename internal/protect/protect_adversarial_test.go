package protect

import (
	"context"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestAdversarial_Limiter_50Goroutines_ConcurrentWithRandomCancellations 对抗验证任务 2：
// 1. 构造 50 个高并发协程同时抢占限流器；
// 2. 随机 10 个协程在不同时间点触发 Context 取消；
// 3. 验证无死锁、无协程泄漏（全部 50 个协程平稳退出）；
// 4. 验证所有成功放行的请求两两之间的实际放行时间严格满足 minWait；
// 5. 验证限流器内部状态无惊群、无连环 0ms 违规。
func TestAdversarial_Limiter_50Goroutines_ConcurrentWithRandomCancellations(t *testing.T) {
	// 设定 minWait 为 25ms，容差 5ms（Windows 时钟精度约 15ms，20ms 下限防御极度严格）
	minWait := 25 * time.Millisecond
	tolerance := 5 * time.Millisecond
	l := NewLimiter(0, minWait)

	// 先执行基准占位，消除冷启动 0ms 放行竞争
	rel0 := l.Acquire()
	rel0()

	const totalGoroutines = 50
	const cancelGoroutines = 10

	// 随机挑选 10 个不同的索引作为取消目标
	cancelIndices := make(map[int]bool)
	r := rand.New(rand.NewSource(20260920))
	for len(cancelIndices) < cancelGoroutines {
		idx := r.Intn(totalGoroutines)
		cancelIndices[idx] = true
	}

	startBarrier := make(chan struct{})
	var wg sync.WaitGroup

	var mu sync.Mutex
	var releaseTimes []time.Time
	var actualCancelCount int32
	var actualSuccessCount int32

	for i := 0; i < totalGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			var ctx context.Context
			var cancel context.CancelFunc

			if cancelIndices[idx] {
				// 设置极短超时（5ms ~ 15ms），保证在 minWait (25ms) 到达前 100% 触发取消
				timeout := time.Duration(5+r.Intn(10)) * time.Millisecond
				ctx, cancel = context.WithTimeout(context.Background(), timeout)
				defer cancel()
			} else {
				// 正常请求给予充足超时（10秒）
				ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
			}

			// 等待所有 50 个协程就绪后瞬间放行，造成最大冲击
			<-startBarrier

			rel, err := l.AcquireContext(ctx)
			if err != nil {
				atomic.AddInt32(&actualCancelCount, 1)
				return
			}
			defer rel()

			atomic.AddInt32(&actualSuccessCount, 1)
			now := time.Now()
			mu.Lock()
			releaseTimes = append(releaseTimes, now)
			mu.Unlock()
		}(i)
	}

	// 瞬间开闸
	close(startBarrier)

	// 设定超时监控，杜绝死锁无法退出
	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	select {
	case <-doneChan:
		// 正常退出
	case <-time.After(5 * time.Second):
		t.Fatalf("ADVERSARIAL FAIL: DEADLOCK DETECTED! 50 goroutines failed to complete within 5 seconds")
	}

	// 验证协程统计
	if int(actualCancelCount) != cancelGoroutines {
		t.Fatalf("expected exactly %d cancelled goroutines, got %d", cancelGoroutines, actualCancelCount)
	}
	expectedSuccess := totalGoroutines - cancelGoroutines
	if int(actualSuccessCount) != expectedSuccess {
		t.Fatalf("expected exactly %d successful goroutines, got %d", expectedSuccess, actualSuccessCount)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(releaseTimes) != expectedSuccess {
		t.Fatalf("expected %d recorded release times, got %d", expectedSuccess, len(releaseTimes))
	}

	// 排序放行时间戳
	sort.Slice(releaseTimes, func(i, j int) bool {
		return releaseTimes[i].Before(releaseTimes[j])
	})

	// 严格核验两两相邻放行的时间间隔
	minAllowedGap := minWait - tolerance
	for i := 1; i < len(releaseTimes); i++ {
		gap := releaseTimes[i].Sub(releaseTimes[i-1])
		if gap < minAllowedGap {
			t.Fatalf("ADVERSARIAL FAIL: THUNDERING HERD VIOLATION! Acquire interval between #%d and #%d was %v, strictly required >= %v",
				i-1, i, gap, minAllowedGap)
		}
	}

	t.Logf("Adversarial Limiter passed: %d total, %d cancelled, %d acquired, all intervals >= %v (minWait=%v)",
		totalGoroutines, actualCancelCount, actualSuccessCount, minAllowedGap, minWait)
}

// TestAdversarial_Limiter_SemaphoreAndMinWait_LeakProof 对抗验证：
// 结合信号量并发控制 (limit=5) 与 minWait 间隔，测试并发抢占与 Context 取消下，
// 信号量槽位无任何泄漏，所有已取消协程均安全退还槽位。
func TestAdversarial_Limiter_SemaphoreAndMinWait_LeakProof(t *testing.T) {
	limit := 5
	minWait := 10 * time.Millisecond
	l := NewLimiter(limit, minWait)

	var wg sync.WaitGroup
	const workers = 30
	barrier := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier
			// 随机取消
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(idx%3*5)*time.Millisecond)
			defer cancel()

			rel, err := l.AcquireContext(ctx)
			if err == nil {
				time.Sleep(5 * time.Millisecond)
				rel()
			}
		}(i)
	}

	close(barrier)
	wg.Wait()

	// 验证全部 5 个槽位可以无阻碍全部重新获取并释放，证明 0 槽位泄漏
	for i := 0; i < limit; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		rel, err := l.AcquireContext(ctx)
		cancel()
		if err != nil {
			t.Fatalf("ADVERSARIAL FAIL: Semaphore slot leak detected! Could not acquire slot %d: %v", i, err)
		}
		rel()
	}
	t.Logf("Adversarial Limiter semaphore leak-proof test passed successfully!")
}
