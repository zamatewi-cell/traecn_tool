package protect

import (
	"strings"
	"sync"
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
