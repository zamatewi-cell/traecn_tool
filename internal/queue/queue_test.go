package queue

import (
	"testing"
	"time"
)

// Test TokenBucket
func TestTokenBucket_NewTokenBucket(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:   10,
		RefillRate: 2,
	}
	tb := NewTokenBucket(config)
	if tb == nil {
		t.Fatal("NewTokenBucket() returned nil")
	}
	if tb.tokens != 10 {
		t.Errorf("NewTokenBucket() tokens = %v, want 10", tb.tokens)
	}
	if tb.capacity != 10 {
		t.Errorf("NewTokenBucket() capacity = %v, want 10", tb.capacity)
	}
	if tb.refillRate != 2 {
		t.Errorf("NewTokenBucket() refillRate = %v, want 2", tb.refillRate)
	}
}

func TestTokenBucket_Allow(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:   5,
		RefillRate: 1,
	}
	tb := NewTokenBucket(config)

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Errorf("Allow() request %d = false, want true", i+1)
		}
	}

	// 6th request should be denied (no tokens left)
	if tb.Allow() {
		t.Error("Allow() 6th request = true, want false (no tokens)")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:   2,
		RefillRate: 10, // 10 tokens/second for fast test
	}
	tb := NewTokenBucket(config)

	// Consume all tokens
	tb.Allow()
	tb.Allow()

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should have tokens again
	if !tb.Allow() {
		t.Error("Allow() after refill = false, want true")
	}
}

func TestTokenBucket_GetTokens(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:   10,
		RefillRate: 1,
	}
	tb := NewTokenBucket(config)

	tokens := tb.GetTokens()
	if tokens != 10 {
		t.Errorf("GetTokens() = %v, want 10", tokens)
	}

	// Consume one
	tb.Allow()
	tokens = tb.GetTokens()
	if tokens != 9 {
		t.Errorf("GetTokens() after Allow = %v, want 9", tokens)
	}
}

// Test SlidingWindowCounter
func TestSlidingWindowCounter_NewSlidingWindowCounter(t *testing.T) {
	config := SlidingWindowConfig{
		WindowSize: time.Minute,
		MaxCount:   10,
	}
	sw := NewSlidingWindowCounter(config)
	if sw == nil {
		t.Fatal("NewSlidingWindowCounter() returned nil")
	}
	if sw.window != time.Minute {
		t.Errorf("NewSlidingWindowCounter() window = %v, want 1m", sw.window)
	}
	if sw.maxCount != 10 {
		t.Errorf("NewSlidingWindowCounter() maxCount = %v, want 10", sw.maxCount)
	}
}

func TestSlidingWindowCounter_Allow(t *testing.T) {
	config := SlidingWindowConfig{
		WindowSize: time.Second,
		MaxCount:   3,
	}
	sw := NewSlidingWindowCounter(config)

	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		if !sw.Allow() {
			t.Errorf("Allow() request %d = false, want true", i+1)
		}
	}

	// 4th request should be denied
	if sw.Allow() {
		t.Error("Allow() 4th request = true, want false (over limit)")
	}
}

func TestSlidingWindowCounter_WindowExpiry(t *testing.T) {
	config := SlidingWindowConfig{
		WindowSize: 100 * time.Millisecond,
		MaxCount:   2,
	}
	sw := NewSlidingWindowCounter(config)

	// Use up allowance
	sw.Allow()
	sw.Allow()

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !sw.Allow() {
		t.Error("Allow() after window expiry = false, want true")
	}
}

func TestSlidingWindowCounter_GetCount(t *testing.T) {
	config := SlidingWindowConfig{
		WindowSize: time.Second,
		MaxCount:   5,
	}
	sw := NewSlidingWindowCounter(config)

	// Make 3 requests
	sw.Allow()
	sw.Allow()
	sw.Allow()

	count := sw.GetCount()
	if count != 3 {
		t.Errorf("GetCount() = %v, want 3", count)
	}
}

// Test RateLimiter
func TestRateLimiter_DefaultConfig(t *testing.T) {
	config := DefaultRateLimiterConfig()

	if config.GlobalRPS != 100 {
		t.Errorf("DefaultRateLimiterConfig() GlobalRPS = %v, want 100", config.GlobalRPS)
	}
	if config.UserRPM != 60 {
		t.Errorf("DefaultRateLimiterConfig() UserRPM = %v, want 60", config.UserRPM)
	}
	if config.AccountRPM != 30 {
		t.Errorf("DefaultRateLimiterConfig() AccountRPM = %v, want 30", config.AccountRPM)
	}
	if config.AccountConcurrent != 3 {
		t.Errorf("DefaultRateLimiterConfig() AccountConcurrent = %v, want 3", config.AccountConcurrent)
	}
}

func TestRateLimiter_NewRateLimiter(t *testing.T) {
	config := RateLimiterConfig{
		GlobalRPS:         50,
		UserRPM:           30,
		AccountRPM:        20,
		AccountConcurrent: 2,
	}
	rl := NewRateLimiter(config)
	if rl == nil {
		t.Fatal("NewRateLimiter() returned nil")
	}
	if rl.global == nil {
		t.Error("NewRateLimiter() global limiter is nil")
	}
	if rl.perUser == nil {
		t.Error("NewRateLimiter() perUser map is nil")
	}
	if rl.perAcct == nil {
		t.Error("NewRateLimiter() perAcct map is nil")
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	config := RateLimiterConfig{
		GlobalRPS:         100,
		UserRPM:           60,
		AccountRPM:        30,
		AccountConcurrent: 3,
	}
	rl := NewRateLimiter(config)

	// First request should be allowed (global has capacity * 1.5 tokens)
	if !rl.Allow("user1", "account1") {
		t.Error("Allow() first request = false, want true")
	}
}

func TestRateLimiter_PerUserLimit(t *testing.T) {
	config := RateLimiterConfig{
		GlobalRPS:         1000, // High global limit
		UserRPM:           3,    // Low per-user limit for testing
		AccountRPM:        100,
		AccountConcurrent: 3,
	}
	rl := NewRateLimiter(config)

	// Make 3 requests (should all pass)
	for i := 0; i < 3; i++ {
		if !rl.Allow("user-test", "account-test") {
			t.Errorf("Allow() request %d = false, want true", i+1)
		}
	}

	// 4th request should fail (user limit reached)
	if rl.Allow("user-test", "account-test") {
		t.Error("Allow() 4th request = true, want false (user limit)")
	}

	// Different user should be allowed
	if !rl.Allow("user-test-2", "account-test") {
		t.Error("Allow() different user = false, want true")
	}
}

func TestRateLimiter_PerAccountLimit(t *testing.T) {
	config := RateLimiterConfig{
		GlobalRPS:         1000, // High global limit
		UserRPM:           100,  // High per-user limit
		AccountRPM:        3,    // Low per-account limit for testing
		AccountConcurrent: 3,
	}
	rl := NewRateLimiter(config)

	// Make 3 requests (should all pass)
	for i := 0; i < 3; i++ {
		if !rl.Allow("user-test", "account-limit-test") {
			t.Errorf("Allow() request %d = false, want true", i+1)
		}
	}

	// 4th request should fail (account limit reached)
	if rl.Allow("user-test", "account-limit-test") {
		t.Error("Allow() 4th request = true, want false (account limit)")
	}

	// Different account should be allowed
	if !rl.Allow("user-test", "account-limit-test-2") {
		t.Error("Allow() different account = false, want true")
	}
}

// Test Monitor
func TestMonitor_NewMonitor(t *testing.T) {
	m := NewMonitor()
	if m == nil {
		t.Fatal("NewMonitor() returned nil")
	}
	if m.queues == nil {
		t.Error("NewMonitor() queues map is nil")
	}
}

func TestMonitor_SetQueueStatus_InQueue(t *testing.T) {
	m := NewMonitor()
	m.SetQueueStatus("deepseek-v3", 5, "Waiting in queue")

	status := m.GetQueueStatus("deepseek-v3")
	if status == nil {
		t.Fatal("GetQueueStatus() returned nil")
	}
	if !status.InQueue {
		t.Error("SetQueueStatus() InQueue = false, want true")
	}
	if status.Position != 5 {
		t.Errorf("SetQueueStatus() Position = %v, want 5", status.Position)
	}
	if status.Message != "Waiting in queue" {
		t.Errorf("SetQueueStatus() Message = %v, want 'Waiting in queue'", status.Message)
	}
	if status.StartTime.IsZero() {
		t.Error("SetQueueStatus() StartTime is zero")
	}
}

func TestMonitor_SetQueueStatus_Remove(t *testing.T) {
	m := NewMonitor()

	// Add to queue
	m.SetQueueStatus("deepseek-v3", 5, "Waiting")

	// Remove from queue (position = 0)
	m.SetQueueStatus("deepseek-v3", 0, "")

	status := m.GetQueueStatus("deepseek-v3")
	if status.InQueue {
		t.Error("SetQueueStatus(0) InQueue = true, want false")
	}
}

func TestMonitor_GetQueueStatus_NotFound(t *testing.T) {
	m := NewMonitor()

	status := m.GetQueueStatus("nonexistent-model")
	if status == nil {
		t.Fatal("GetQueueStatus() returned nil")
	}
	if status.InQueue {
		t.Error("GetQueueStatus() InQueue = true, want false for nonexistent model")
	}
}

func TestMonitor_GetAllStatus(t *testing.T) {
	m := NewMonitor()

	// Add multiple queues
	m.SetQueueStatus("model1", 3, "Waiting")
	m.SetQueueStatus("model2", 7, "Waiting")
	m.SetQueueStatus("model3", 1, "Waiting")

	allStatus := m.GetAllStatus()
	if len(allStatus) != 3 {
		t.Errorf("GetAllStatus() length = %v, want 3", len(allStatus))
	}

	if _, exists := allStatus["model1"]; !exists {
		t.Error("GetAllStatus() missing model1")
	}
	if _, exists := allStatus["model2"]; !exists {
		t.Error("GetAllStatus() missing model2")
	}
	if _, exists := allStatus["model3"]; !exists {
		t.Error("GetAllStatus() missing model3")
	}
}

func TestMonitor_GetAllStatus_Empty(t *testing.T) {
	m := NewMonitor()

	allStatus := m.GetAllStatus()
	if len(allStatus) != 0 {
		t.Errorf("GetAllStatus() length = %v, want 0", len(allStatus))
	}
}

func TestStatus_Struct(t *testing.T) {
	status := Status{
		InQueue:   true,
		Position:  10,
		Message:   "Your turn is coming",
		StartTime: time.Now(),
	}

	if !status.InQueue {
		t.Error("Status InQueue = false, want true")
	}
	if status.Position != 10 {
		t.Errorf("Status Position = %v, want 10", status.Position)
	}
	if status.Message != "Your turn is coming" {
		t.Errorf("Status Message = %v, want 'Your turn is coming'", status.Message)
	}
}
