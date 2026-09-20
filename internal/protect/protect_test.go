package protect

import (
	"context"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTruncateHeadTail(t *testing.T) {
	text := strings.Repeat("a", 100) + strings.Repeat("b", 100)
	got := TruncateHeadTail(text, 100)
	if len(got) > 120 { // budget + marker tolerance
		t.Errorf("truncated length = %d, want around 100", len(got))
	}
	if !strings.Contains(got, "[middle truncated]") {
		t.Error("missing truncation marker")
	}
	if !strings.HasPrefix(got, "aaa") || !strings.HasSuffix(got, "bbb") {
		t.Errorf("head/tail not preserved: %q...%q", got[:10], got[len(got)-10:])
	}

	// Under budget: unchanged.
	if got := TruncateHeadTail("short", 100); got != "short" {
		t.Errorf("short text changed: %q", got)
	}
	// Zero budget: unchanged.
	if got := TruncateHeadTail("anything", 0); got != "anything" {
		t.Errorf("zero budget changed text: %q", got)
	}
}

func TestTruncateHeadTail_UTF8Safe(t *testing.T) {
	text := strings.Repeat("中", 60) // 3 bytes each, 180 bytes total
	got := TruncateHeadTail(text, 50)
	if !strings.Contains(got, "[middle truncated]") {
		t.Fatalf("missing marker: %q", got)
	}
	// Must be valid UTF-8 — no replacement chars from rune splits.
	if strings.ContainsRune(got, '�') {
		t.Errorf("invalid UTF-8 boundary: %q", got)
	}
}

func TestStripStackTraces(t *testing.T) {
	input := "Error: build failed\nmain.go:42:13: undefined: foo\n    at compile (main.go:42)\ngoroutine 1 [running]:\nsome context line"
	got := StripStackTraces(input)
	if !strings.Contains(got, "Error: build failed") {
		t.Error("error headline removed")
	}
	if strings.Contains(got, "at compile") {
		t.Error("stack line not removed")
	}
	if !strings.Contains(got, "stack trace lines removed") {
		t.Error("missing removal note")
	}
	if !strings.Contains(got, "some context line") {
		t.Error("context line wrongly removed")
	}
}

func TestProjector_TotalBudget(t *testing.T) {
	p := NewProjector(6000, 0)
	big := strings.Repeat("x", 5000)
	msgs := []MessageView{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: big},
		{Role: "assistant", Content: big},
		{Role: "user", Content: "recent question"},
	}
	out := p.Project(msgs)

	// System and last message stay intact.
	if out[0].Content != "sys" || out[3].Content != "recent question" {
		t.Error("protected messages were truncated")
	}
	// Older messages got shrunk: total must fit roughly within budget.
	total := 0
	for _, m := range out {
		total += len(m.Content) + 32
	}
	if total > 6000 {
		t.Errorf("total = %d, want <= 6000", total)
	}
}

func TestFilter(t *testing.T) {
	f := NewFilter(true, map[string]string{"secret-token": "[redacted]"})
	if got := f.Apply("my secret-token here"); got != "my [redacted] here" {
		t.Errorf("Apply = %q", got)
	}

	disabled := NewFilter(false, map[string]string{"a": "b"})
	if got := disabled.Apply("a"); got != "a" {
		t.Error("disabled filter changed text")
	}
}

func TestLimiter_ConcurrencyCap(t *testing.T) {
	l := NewLimiter(1, 0)
	release1 := l.Acquire()

	acquired := make(chan struct{})
	go func() {
		r2 := l.Acquire()
		close(acquired)
		r2()
	}()

	select {
	case <-acquired:
		t.Fatal("second acquire succeeded while first held (cap=1)")
	case <-time.After(50 * time.Millisecond):
	}

	release1()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second acquire did not proceed after release")
	}
}

func TestLimiter_MinInterval(t *testing.T) {
	l := NewLimiter(0, 80*time.Millisecond)
	start := time.Now()
	r1 := l.Acquire()
	r1()
	r2 := l.Acquire()
	r2()
	if elapsed := time.Since(start); elapsed < 70*time.Millisecond {
		t.Errorf("min interval not enforced: %v", elapsed)
	}
}

func TestLimiter_NilSafe(t *testing.T) {
	var l *Limiter
	release := l.Acquire()
	release() // must not panic
}

func TestLimiter_ConcurrentStress(t *testing.T) {
	l := NewLimiter(3, 0)
	var mu sync.Mutex
	maxSeen, current := 0, 0
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := l.Acquire()
			mu.Lock()
			current++
			if current > maxSeen {
				maxSeen = current
			}
			mu.Unlock()
			time.Sleep(5 * time.Millisecond)
			mu.Lock()
			current--
			mu.Unlock()
			r()
		}()
	}
	wg.Wait()
	if maxSeen > 3 {
		t.Errorf("concurrency exceeded cap: %d", maxSeen)
	}
}

func TestLimiter_AcquireContext_Cancellation(t *testing.T) {
	l := NewLimiter(1, 0)

	rel1, err := l.AcquireContext(context.Background())
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err = l.AcquireContext(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}

	rel1()
	rel2, err := l.AcquireContext(context.Background())
	if err != nil {
		t.Fatalf("acquire after release failed: %v", err)
	}
	rel2()
}

// TestLimiter_MinInterval_ConcurrentThunderingHerdAndCancellation 验证反例 7：
// 在 10 个高并发请求同时涌入且包含 Context 取消的极端场景下，
// 互斥锁保护下的 Re-check Loop 确保任意相邻两个成功放行的请求间隔严格 >= 235ms，
// 彻底消除由于等待前解锁导致的惊群连环 0ms 放行回归。
func TestLimiter_MinInterval_ConcurrentThunderingHerdAndCancellation(t *testing.T) {
	minWait := 250 * time.Millisecond
	l := NewLimiter(0, minWait)

	// 先执行一次基准请求占位，使 lastAt 确立为当前时间，确保后续并发涌入的请求必须排队等待 minWait
	rel0 := l.Acquire()
	rel0()

	concurrency := 10
	startBarrier := make(chan struct{})
	var wg sync.WaitGroup

	var mu sync.Mutex
	var releaseTimes []time.Time
	var cancelCount int32

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var ctx context.Context
			var cancel context.CancelFunc

			if idx >= 8 {
				// 后 2 个 goroutine 设置 50ms 极短超时模拟取消
				ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()
			} else {
				ctx = context.Background()
			}

			<-startBarrier // 栅栏同时放行，制造瞬时高并发

			rel, err := l.AcquireContext(ctx)
			if err != nil {
				atomic.AddInt32(&cancelCount, 1)
				return
			}
			now := time.Now()
			mu.Lock()
			releaseTimes = append(releaseTimes, now)
			mu.Unlock()
			rel()
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	if cancelCount != 2 {
		t.Fatalf("expected exactly 2 cancelled requests, got %d", cancelCount)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(releaseTimes) != 8 {
		t.Fatalf("expected 8 successful acquires, got %d", len(releaseTimes))
	}

	sort.Slice(releaseTimes, func(i, j int) bool {
		return releaseTimes[i].Before(releaseTimes[j])
	})

	// 验证相邻请求之间的放行间隔严格 >= 235ms (250ms - 15ms 调度容差)
	for i := 1; i < len(releaseTimes); i++ {
		gap := releaseTimes[i].Sub(releaseTimes[i-1])
		if gap < 235*time.Millisecond {
			t.Fatalf("THUNDERING HERD VIOLATION: gap between acquire %d and %d was %v, strictly expected >= 235ms",
				i-1, i, gap)
		}
	}
}

