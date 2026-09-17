package queue

import (
	"sync"
	"time"
)

// CircuitBreakerStatus represents circuit breaker state
type CircuitBreakerStatus string

const (
	StatusClosed   CircuitBreakerStatus = "closed"    // Normal state
	StatusOpen     CircuitBreakerStatus = "open"      // Circuit open state
	StatusHalfOpen CircuitBreakerStatus = "half-open" // Half-open state
)

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	mu sync.Mutex

	state            CircuitBreakerStatus
	failureCount     int
	successCount     int
	threshold        int
	halfOpenMaxCalls int
	timeout          time.Duration
	lastFailureTime  time.Time
	lastStateChange  time.Time
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           // Failure count threshold
	Timeout          time.Duration // Circuit timeout
	HalfOpenMaxCalls int           // Max calls in half-open state
}

// DefaultCircuitBreakerConfig returns default circuit breaker config
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
		HalfOpenMaxCalls: 3,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StatusClosed,
		threshold:        config.FailureThreshold,
		timeout:          config.Timeout,
		halfOpenMaxCalls: config.HalfOpenMaxCalls,
		lastStateChange:  time.Now(),
	}
}

// Allow checks if request is allowed
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StatusClosed:
		return true

	case StatusOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = StatusHalfOpen
			cb.successCount = 0
			cb.lastStateChange = time.Now()
			return true
		}
		return false

	case StatusHalfOpen:
		if cb.successCount < cb.halfOpenMaxCalls {
			return true
		}
		return false

	default:
		return false
	}
}

// RecordSuccess records a successful call
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StatusHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.halfOpenMaxCalls {
			cb.state = StatusClosed
			cb.failureCount = 0
			cb.lastStateChange = time.Now()
		}
	} else if cb.state == StatusClosed {
		cb.failureCount = 0
	}
}

// RecordFailure records a failed call
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StatusHalfOpen {
		cb.state = StatusOpen
		cb.lastStateChange = time.Now()
	} else if cb.state == StatusClosed && cb.failureCount >= cb.threshold {
		cb.state = StatusOpen
		cb.lastStateChange = time.Now()
	}
}

// GetState returns current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerStatus {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() *CircuitBreakerStats {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return &CircuitBreakerStats{
		State:        cb.state,
		FailureCount: cb.failureCount,
		SuccessCount: cb.successCount,
		Threshold:    cb.threshold,
		LastFailure:  cb.lastFailureTime,
		StateChanged: cb.lastStateChange,
	}
}

// CircuitBreakerStats holds circuit breaker statistics
type CircuitBreakerStats struct {
	State        CircuitBreakerStatus
	FailureCount int
	SuccessCount int
	Threshold    int
	LastFailure  time.Time
	StateChanged time.Time
}

// DegradationPolicy implements automatic degradation
type DegradationPolicy struct {
	mu sync.Mutex

	queueThreshold   int
	errorThreshold   float64
	cooldownDuration time.Duration
	degradedUntil    time.Time
	isDegraded       bool
}

// DegradationConfig holds degradation configuration
type DegradationConfig struct {
	QueueThreshold   int           // Queue length threshold
	ErrorThreshold   float64       // Error rate threshold (0.0-1.0)
	CooldownDuration time.Duration // Degradation cooldown
}

// DefaultDegradationConfig returns default degradation config
func DefaultDegradationConfig() DegradationConfig {
	return DegradationConfig{
		QueueThreshold:   100,
		ErrorThreshold:   0.5,
		CooldownDuration: 60 * time.Second,
	}
}

// NewDegradationPolicy creates a new degradation policy
func NewDegradationPolicy(config DegradationConfig) *DegradationPolicy {
	return &DegradationPolicy{
		queueThreshold:   config.QueueThreshold,
		errorThreshold:   config.ErrorThreshold,
		cooldownDuration: config.CooldownDuration,
	}
}

// ShouldDegrade checks if system should degrade
func (dp *DegradationPolicy) ShouldDegrade(stats *QueueStats) bool {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	// Check if in cooldown
	if dp.isDegraded && time.Now().Before(dp.degradedUntil) {
		return true
	}

	// Reset if cooldown expired
	if dp.isDegraded && time.Now().After(dp.degradedUntil) {
		dp.isDegraded = false
	}

	// Check queue length
	if stats.TotalLength > dp.queueThreshold {
		dp.isDegraded = true
		dp.degradedUntil = time.Now().Add(dp.cooldownDuration)
		return true
	}

	return false
}

// IsDegraded returns current degradation state
func (dp *DegradationPolicy) IsDegraded() bool {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	return dp.isDegraded && time.Now().Before(dp.degradedUntil)
}

// GetDegradationLevel returns degradation level (0-3)
func (dp *DegradationPolicy) GetDegradationLevel(stats *QueueStats) int {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if !dp.isDegraded {
		return 0
	}

	// Level 1: Pause low priority
	if stats.TotalLength > dp.queueThreshold {
		return 1
	}
	// Level 2: Pause normal priority
	if stats.TotalLength > dp.queueThreshold*2 {
		return 2
	}
	// Level 3: Pause all except VIP
	if stats.TotalLength > dp.queueThreshold*3 {
		return 3
	}

	return 1
}
