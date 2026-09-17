package queue

import (
	"context"
	"sync"
	"time"
)

// Manager coordinates queue, scheduler, rate limiter, and SSE
type Manager struct {
	mu             sync.RWMutex
	scheduler      *Scheduler
	rateLimiter    *RateLimiter
	circuitBreaker *CircuitBreaker
	degradation    *DegradationPolicy
	sseBroker      *SSEBroker
	config         ManagerConfig
	ctx            context.Context
	cancel         context.CancelFunc
}

// ManagerConfig holds manager configuration
type ManagerConfig struct {
	Scheduler      SchedulerConfig
	RateLimiter    RateLimiterConfig
	CircuitBreaker CircuitBreakerConfig
	Degradation    DegradationConfig
	SSETimeout     time.Duration
}

// DefaultManagerConfig returns default manager config
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		Scheduler:      DefaultSchedulerConfig(),
		RateLimiter:    DefaultRateLimiterConfig(),
		CircuitBreaker: DefaultCircuitBreakerConfig(),
		Degradation:    DefaultDegradationConfig(),
		SSETimeout:     5 * time.Minute,
	}
}

// NewManager creates a new queue manager
func NewManager(config ManagerConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		scheduler:      NewScheduler(config.Scheduler),
		rateLimiter:    NewRateLimiter(config.RateLimiter),
		circuitBreaker: NewCircuitBreaker(config.CircuitBreaker),
		degradation:    NewDegradationPolicy(config.Degradation),
		sseBroker:      NewSSEBroker(nil), // Will be set below
		config:         config,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Link scheduler to SSE broker
	m.sseBroker.queue = m.scheduler

	return m
}

// Start starts the queue manager
func (m *Manager) Start(processor RequestProcessor) {
	m.scheduler.StartWorkers(processor)
	m.sseBroker.StartHealthChecker(m.config.SSETimeout)
}

// Stop stops the queue manager
func (m *Manager) Stop() {
	m.cancel()
	m.scheduler.Stop()
}

// SubmitRequest submits a request to the queue
func (m *Manager) SubmitRequest(req *QueuedRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check circuit breaker
	if !m.circuitBreaker.Allow() {
		return ErrCircuitOpen
	}

	// Check rate limiter
	if !m.rateLimiter.Allow(req.AccountID, req.AccountID) {
		return ErrRateLimited
	}

	// Check degradation
	stats := m.scheduler.GetQueueStats()
	if m.degradation.ShouldDegrade(stats) {
		level := m.degradation.GetDegradationLevel(stats)
		if level >= 2 && req.Priority < PriorityHigh {
			return ErrDegraded
		}
	}

	// Enqueue request
	return m.scheduler.Enqueue(req)
}

// GetQueueStats returns queue statistics
func (m *Manager) GetQueueStats() *QueueStats {
	return m.scheduler.GetQueueStats()
}

// GetCircuitBreakerStats returns circuit breaker stats
func (m *Manager) GetCircuitBreakerStats() *CircuitBreakerStats {
	return m.circuitBreaker.GetStats()
}

// RecordSuccess records successful request
func (m *Manager) RecordSuccess() {
	m.circuitBreaker.RecordSuccess()
}

// RecordFailure records failed request
func (m *Manager) RecordFailure() {
	m.circuitBreaker.RecordFailure()
}

// GetSSEBroker returns SSE broker
func (m *Manager) GetSSEBroker() *SSEBroker {
	return m.sseBroker
}

// GetScheduler returns scheduler
func (m *Manager) GetScheduler() *Scheduler {
	return m.scheduler
}

// GetRateLimiter returns rate limiter
func (m *Manager) GetRateLimiter() *RateLimiter {
	return m.rateLimiter
}

// IsHealthy checks if manager is healthy
func (m *Manager) IsHealthy() bool {
	cbState := m.circuitBreaker.GetState()
	return cbState != StatusOpen
}

// GetStatus returns comprehensive status
func (m *Manager) GetStatus() *ManagerStatus {
	return &ManagerStatus{
		QueueStats:     m.scheduler.GetQueueStats(),
		CircuitBreaker: m.circuitBreaker.GetStats(),
		SSEClientCount: m.sseBroker.GetClientCount(),
		IsHealthy:      m.IsHealthy(),
		IsDegraded:     m.degradation.IsDegraded(),
	}
}

// ManagerStatus holds comprehensive manager status
type ManagerStatus struct {
	QueueStats     *QueueStats
	CircuitBreaker *CircuitBreakerStats
	SSEClientCount int
	IsHealthy      bool
	IsDegraded     bool
}

// Error definitions
var (
	ErrCircuitOpen = &ManagerError{message: "circuit breaker is open"}
	ErrRateLimited = &ManagerError{message: "rate limit exceeded"}
	ErrDegraded    = &ManagerError{message: "system is degraded"}
	ErrQueueFull   = &ManagerError{message: "queue is full"}
)

// ManagerError represents manager error
type ManagerError struct {
	message string
}

func (e *ManagerError) Error() string {
	return e.message
}
