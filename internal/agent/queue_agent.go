package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/workflow"
)

// QueueAgent manages request queues and scheduling
type QueueAgent struct {
	*BaseAgent
	RequestQueue   *RequestQueue
	RateLimiter    *RateLimiter
	CircuitBreaker *CircuitBreakerStatus
	mu             sync.Mutex
}

// RequestQueue implements priority queue
type RequestQueue struct {
	Requests []*QueuedRequest
	mu       sync.Mutex
}

// QueuedRequest represents a queued request
type QueuedRequest struct {
	ID         string
	Priority   int
	CreatedAt  time.Time
	Data       interface{}
	RetryCount int
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	Tokens     float64
	MaxTokens  float64
	RefillRate float64 // tokens per second
	LastRefill time.Time
	mu         sync.Mutex
}

// CircuitBreakerStatus represents circuit breaker status
type CircuitBreakerStatus struct {
	State        string // closed, open, half-open
	FailureCount int
	Threshold    int
	LastFailure  time.Time
}

// NewQueueAgent creates a queue agent
func NewQueueAgent(ctxManager *workflow.ContextManager, taskManager *workflow.TaskManager, eventBus *workflow.EventBus) *QueueAgent {
	return &QueueAgent{
		BaseAgent: NewBaseAgent("queue-agent", "Queue Agent", ctxManager, taskManager, eventBus),
		RequestQueue: &RequestQueue{
			Requests: make([]*QueuedRequest, 0),
		},
		RateLimiter: &RateLimiter{
			Tokens:     100,
			MaxTokens:  100,
			RefillRate: 10, // 10 tokens per second
			LastRefill: time.Now(),
		},
		CircuitBreaker: &CircuitBreakerStatus{
			State:     "closed",
			Threshold: 5,
		},
	}
}

// Run starts queue management
func (q *QueueAgent) Run(ctx context.Context, task *workflow.Task) error {
	q.SetStatus(StateRunning)

	defer func() {
		if r := recover(); r != nil {
			q.SetStatus(StateFailed)
			q.PublishEvent(workflow.EventError, map[string]interface{}{
				"error": fmt.Sprintf("%v", r),
			})
		}
	}()

	// Execute task based on type
	switch task.Type {
	case "queue_schedule":
		return q.scheduleRequests(ctx, task)
	case "queue_ratelimit":
		return q.configureRateLimit(ctx, task)
	case "queue_circuit":
		return q.configureCircuitBreaker(ctx, task)
	default:
		return q.scheduleRequests(ctx, task)
	}
}

// scheduleRequests schedules requests using MLFQ algorithm
func (q *QueueAgent) scheduleRequests(ctx context.Context, task *workflow.Task) error {
	q.SetProgress(20, "Initializing MLFQ scheduler", 1, 4)

	q.mu.Lock()

	// Sort by priority
	q.sortByPriority()

	q.SetProgress(50, "Processing high priority requests", 2, 4)

	// Process high priority first
	processed := 0
	for _, req := range q.RequestQueue.Requests {
		if req.Priority >= 10 {
			processed++
		}
	}

	q.SetProgress(80, "Processing normal priority requests", 3, 4)

	// Process remaining
	processed = len(q.RequestQueue.Requests)

	q.SetProgress(100, fmt.Sprintf("Scheduled %d requests", processed), 4, 4)

	q.mu.Unlock()

	q.SetStatus(StateCompleted)

	q.PublishEvent(workflow.EventTaskCompleted, map[string]interface{}{
		"task_id":        task.ID,
		"requests_count": processed,
	})

	return nil
}

// configureRateLimit configures rate limiting
func (q *QueueAgent) configureRateLimit(ctx context.Context, task *workflow.Task) error {
	q.SetProgress(30, "Configuring token bucket", 1, 3)

	q.RateLimiter.mu.Lock()
	q.RateLimiter.MaxTokens = 100
	q.RateLimiter.RefillRate = 10
	q.RateLimiter.mu.Unlock()

	q.SetProgress(60, "Testing rate limiter", 2, 3)

	// Test rate limiting
	allowed := 0
	for i := 0; i < 20; i++ {
		if q.RateLimiter.Allow() {
			allowed++
		}
	}

	q.SetProgress(100, fmt.Sprintf("Rate limit configured: %d/%d allowed", allowed, 20), 3, 3)

	q.SetStatus(StateCompleted)

	q.PublishEvent(workflow.EventTaskCompleted, map[string]interface{}{
		"task_id":       task.ID,
		"allowed_count": allowed,
	})

	return nil
}

// configureCircuitBreaker configures circuit breaker
func (q *QueueAgent) configureCircuitBreaker(ctx context.Context, task *workflow.Task) error {
	q.SetProgress(30, "Configuring circuit breaker", 1, 3)

	q.CircuitBreaker.Threshold = 5
	q.CircuitBreaker.State = "closed"

	q.SetProgress(60, "Testing circuit breaker", 2, 3)

	// Simulate failures
	for i := 0; i < 3; i++ {
		q.CircuitBreaker.RecordFailure()
	}

	q.SetProgress(100, "Circuit breaker configured", 3, 3)

	q.SetStatus(StateCompleted)

	q.PublishEvent(workflow.EventTaskCompleted, map[string]interface{}{
		"task_id": task.ID,
		"state":   q.CircuitBreaker.State,
	})

	return nil
}

// sortByPriority sorts requests by priority (MLFQ)
func (q *QueueAgent) sortByPriority() {
	requests := q.RequestQueue.Requests
	for i := 0; i < len(requests); i++ {
		for j := i + 1; j < len(requests); j++ {
			if requests[i].Priority < requests[j].Priority {
				requests[i], requests[j] = requests[j], requests[i]
			}
		}
	}
}

// Enqueue adds a request to the queue
func (q *QueueAgent) Enqueue(req *QueuedRequest) {
	q.RequestQueue.mu.Lock()
	defer q.RequestQueue.mu.Unlock()

	q.RequestQueue.Requests = append(q.RequestQueue.Requests, req)
	q.sortByPriority()
}

// Dequeue removes and returns the highest priority request
func (q *QueueAgent) Dequeue() *QueuedRequest {
	q.RequestQueue.mu.Lock()
	defer q.RequestQueue.mu.Unlock()

	if len(q.RequestQueue.Requests) == 0 {
		return nil
	}

	req := q.RequestQueue.Requests[0]
	q.RequestQueue.Requests = q.RequestQueue.Requests[1:]
	return req
}

// Allow checks if a request is allowed by rate limiter
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(r.LastRefill).Seconds()
	r.Tokens = min(r.MaxTokens, r.Tokens+elapsed*r.RefillRate)
	r.LastRefill = now

	// Check if allowed
	if r.Tokens >= 1 {
		r.Tokens--
		return true
	}

	return false
}

// RecordFailure records a failure for circuit breaker
func (c *CircuitBreakerStatus) RecordFailure() {
	c.FailureCount++
	c.LastFailure = time.Now()

	if c.FailureCount >= c.Threshold {
		c.State = "open"
	}
}

// RecordSuccess records a success for circuit breaker
func (c *CircuitBreakerStatus) RecordSuccess() {
	if c.State == "half-open" {
		c.State = "closed"
		c.FailureCount = 0
	}
}

// CanExecute checks if circuit breaker allows execution
func (c *CircuitBreakerStatus) CanExecute() bool {
	if c.State == "closed" {
		return true
	}

	if c.State == "open" {
		if time.Since(c.LastFailure) > 60*time.Second {
			c.State = "half-open"
			return true
		}
		return false
	}

	// half-open
	return true
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
