package queue

import (
	"sync"
	"time"
)

// TokenBucket implements token bucket rate limiter
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	refillRate float64 // tokens/second
	lastRefill time.Time
}

// TokenBucketConfig holds token bucket configuration
type TokenBucketConfig struct {
	Capacity   float64 // Maximum tokens
	RefillRate float64 // Tokens refilled per second
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(config TokenBucketConfig) *TokenBucket {
	return &TokenBucket{
		tokens:     config.Capacity,
		capacity:   config.Capacity,
		refillRate: config.RefillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = min(tb.capacity, tb.tokens+elapsed*tb.refillRate)
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// Wait waits until a token is available
func (tb *TokenBucket) Wait() {
	for !tb.Allow() {
		time.Sleep(10 * time.Millisecond)
	}
}

// GetTokens returns current token count
func (tb *TokenBucket) GetTokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens
}

// SlidingWindowCounter implements sliding window rate limiter
type SlidingWindowCounter struct {
	mu        sync.Mutex
	window    time.Duration
	maxCount  int
	timestamp []time.Time
}

// SlidingWindowConfig holds sliding window configuration
type SlidingWindowConfig struct {
	WindowSize time.Duration // Window size (e.g., 1 minute)
	MaxCount   int           // Max requests per window
}

// NewSlidingWindowCounter creates a new sliding window rate limiter
func NewSlidingWindowCounter(config SlidingWindowConfig) *SlidingWindowCounter {
	return &SlidingWindowCounter{
		window:    config.WindowSize,
		maxCount:  config.MaxCount,
		timestamp: make([]time.Time, 0),
	}
}

// Allow checks if a request is allowed
func (sw *SlidingWindowCounter) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	// Remove old timestamps
	valid := make([]time.Time, 0, len(sw.timestamp))
	for _, ts := range sw.timestamp {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	sw.timestamp = valid

	// Check if under limit
	if len(sw.timestamp) < sw.maxCount {
		sw.timestamp = append(sw.timestamp, now)
		return true
	}

	return false
}

// GetCount returns current request count in window
func (sw *SlidingWindowCounter) GetCount() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	count := 0
	for _, ts := range sw.timestamp {
		if ts.After(cutoff) {
			count++
		}
	}
	return count
}

// RateLimiter combines multiple rate limiters
type RateLimiter struct {
	global     *TokenBucket
	perUser    map[string]*SlidingWindowCounter
	perAcct    map[string]*SlidingWindowCounter
	UserRPM    int
	AccountRPM int
	mu         sync.RWMutex
}

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	GlobalRPS         float64 // Global requests per second
	UserRPM           int     // Per-user requests per minute
	AccountRPM        int     // Per-account requests per minute
	AccountConcurrent int     // Per-account concurrent requests
}

// DefaultRateLimiterConfig returns default rate limiter config
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		GlobalRPS:         100,
		UserRPM:           60,
		AccountRPM:        30,
		AccountConcurrent: 3,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		global:     NewTokenBucket(TokenBucketConfig{Capacity: config.GlobalRPS * 1.5, RefillRate: config.GlobalRPS}),
		perUser:    make(map[string]*SlidingWindowCounter),
		perAcct:    make(map[string]*SlidingWindowCounter),
		UserRPM:    config.UserRPM,
		AccountRPM: config.AccountRPM,
	}
}

// Allow checks if request is allowed for all rate limiters
func (rl *RateLimiter) Allow(userID, accountID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check global limit
	if !rl.global.Allow() {
		return false
	}

	// Check per-user limit
	if _, exists := rl.perUser[userID]; !exists {
		rl.perUser[userID] = NewSlidingWindowCounter(SlidingWindowConfig{
			WindowSize: time.Minute,
			MaxCount:   rl.UserRPM,
		})
	}
	if !rl.perUser[userID].Allow() {
		return false
	}

	// Check per-account limit
	if _, exists := rl.perAcct[accountID]; !exists {
		rl.perAcct[accountID] = NewSlidingWindowCounter(SlidingWindowConfig{
			WindowSize: time.Minute,
			MaxCount:   rl.AccountRPM,
		})
	}
	if !rl.perAcct[accountID].Allow() {
		return false
	}

	return true
}

// Wait waits until request is allowed
func (rl *RateLimiter) Wait(userID, accountID string) {
	for !rl.Allow(userID, accountID) {
		time.Sleep(10 * time.Millisecond)
	}
}

// Status represents queue status for a model
type Status struct {
	InQueue   bool      `json:"in_queue"`
	Position  int       `json:"position"`
	Message   string    `json:"message,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
}

// Monitor tracks queue status for models
type Monitor struct {
	mu     sync.RWMutex
	queues map[string]*Status
}

// NewMonitor creates a queue monitor
func NewMonitor() *Monitor {
	return &Monitor{
		queues: make(map[string]*Status),
	}
}

// SetQueueStatus sets queue status for a model
func (m *Monitor) SetQueueStatus(model string, position int, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if position > 0 {
		m.queues[model] = &Status{
			InQueue:   true,
			Position:  position,
			Message:   message,
			StartTime: time.Now(),
		}
	} else {
		delete(m.queues, model)
	}
}

// GetQueueStatus gets queue status for a model
func (m *Monitor) GetQueueStatus(model string) *Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if s, ok := m.queues[model]; ok {
		return s
	}
	return &Status{InQueue: false}
}

// GetAllStatus gets all queue statuses
func (m *Monitor) GetAllStatus() map[string]*Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*Status, len(m.queues))
	for k, v := range m.queues {
		result[k] = v
	}
	return result
}
