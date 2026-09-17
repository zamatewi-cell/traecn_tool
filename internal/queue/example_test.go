package queue

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Example demonstrates queue manager usage
func Example() {
	// Create manager with default config
	config := DefaultManagerConfig()
	manager := NewManager(config)

	// Create request processor
	processor := &SimpleProcessor{
		ProcessFunc: func(req *QueuedRequest) {
			// Simulate processing
			time.Sleep(100 * time.Millisecond)

			// Send response
			req.Response <- Response{
				Data:       "Processed",
				StatusCode: 200,
			}
		},
	}

	// Start manager
	manager.Start(processor)
	defer manager.Stop()

	// Submit requests
	for i := 0; i < 10; i++ {
		respChan := make(chan Response, 1)
		req := &QueuedRequest{
			ID:        fmt.Sprintf("req-%d", i),
			Priority:  PriorityNormal,
			AccountID: "account-1",
			Model:     "claude-3-5-sonnet",
			Request:   nil,
			Response:  respChan,
		}

		if err := manager.SubmitRequest(req); err != nil {
			fmt.Printf("Failed to submit request: %v\n", err)
			continue
		}

		// Wait for response
		resp := <-respChan
		fmt.Printf("Request %s completed with status: %d\n", req.ID, resp.StatusCode)
	}

	// Get stats
	stats := manager.GetStatus()
	fmt.Printf("Queue length: %d\n", stats.QueueStats.TotalLength)
	fmt.Printf("Circuit breaker state: %s\n", stats.CircuitBreaker.State)
	fmt.Printf("SSE clients: %d\n", stats.SSEClientCount)
}

// SimpleProcessor is a simple request processor
type SimpleProcessor struct {
	ProcessFunc func(*QueuedRequest)
}

// Process processes a request
func (p *SimpleProcessor) Process(req *QueuedRequest) {
	if p.ProcessFunc != nil {
		p.ProcessFunc(req)
	}
}

// ExampleSSEBroker demonstrates SSE usage
func ExampleSSEBroker() {
	config := DefaultManagerConfig()
	manager := NewManager(config)
	manager.Start(&SimpleProcessor{})
	defer manager.Stop()

	// Create SSE handler
	http.HandleFunc("/queue/status", manager.GetSSEBroker().SSEHandler("claude-3-5-sonnet"))

	// Start server
	go http.ListenAndServe(":8080", nil)

	// Simulate queue status updates
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	position := 10
	for range ticker.C {
		position--
		estimatedWait := manager.GetSSEBroker().CalculateEstimatedWait(position)

		manager.GetSSEBroker().BroadcastQueueStatus(
			"claude-3-5-sonnet",
			position,
			estimatedWait,
		)

		if position <= 0 {
			break
		}
	}
}

// ExampleRateLimiter demonstrates rate limiter usage
func ExampleRateLimiter() {
	config := DefaultManagerConfig()
	config.RateLimiter.GlobalRPS = 10
	config.RateLimiter.UserRPM = 60
	config.RateLimiter.AccountRPM = 30

	manager := NewManager(config)
	manager.Start(&SimpleProcessor{})
	defer manager.Stop()

	// Submit requests with rate limiting
	for i := 0; i < 100; i++ {
		req := &QueuedRequest{
			ID:        fmt.Sprintf("req-%d", i),
			Priority:  PriorityNormal,
			AccountID: "account-1",
			Model:     "claude-3-5-sonnet",
			Response:  make(chan Response, 1),
		}

		if err := manager.SubmitRequest(req); err != nil {
			fmt.Printf("Request %d rejected: %v\n", i, err)
			continue
		}

		fmt.Printf("Request %d accepted\n", i)
	}
}

// ExampleCircuitBreaker demonstrates circuit breaker usage
func ExampleCircuitBreaker() {
	config := DefaultManagerConfig()
	config.CircuitBreaker.FailureThreshold = 3
	config.CircuitBreaker.Timeout = 10 * time.Second

	manager := NewManager(config)
	manager.Start(&SimpleProcessor{})
	defer manager.Stop()

	// Simulate failures
	for i := 0; i < 10; i++ {
		req := &QueuedRequest{
			ID:        fmt.Sprintf("req-%d", i),
			Priority:  PriorityNormal,
			AccountID: "account-1",
			Model:     "claude-3-5-sonnet",
			Response:  make(chan Response, 1),
		}

		if err := manager.SubmitRequest(req); err != nil {
			fmt.Printf("Request %d rejected: %v\n", i, err)
			continue
		}

		// Simulate failure
		manager.RecordFailure()
		fmt.Printf("Request %d failed\n", i)
	}

	// Check circuit breaker state
	stats := manager.GetCircuitBreakerStats()
	fmt.Printf("Circuit breaker state: %s\n", stats.State)

	// Wait for timeout
	time.Sleep(11 * time.Second)

	// Try again
	req := &QueuedRequest{
		ID:        "req-retry",
		Priority:  PriorityNormal,
		AccountID: "account-1",
		Model:     "claude-3-5-sonnet",
		Response:  make(chan Response, 1),
	}

	if err := manager.SubmitRequest(req); err != nil {
		fmt.Printf("Retry request rejected: %v\n", err)
	} else {
		fmt.Printf("Retry request accepted (circuit breaker reset)\n")
	}
}

// ExamplePriority demonstrates priority queue usage
func ExamplePriority() {
	config := DefaultManagerConfig()
	manager := NewManager(config)
	manager.Start(&SimpleProcessor{})
	defer manager.Stop()

	// Submit requests with different priorities
	priorities := []Priority{
		PriorityNormal,
		PriorityVIP,
		PriorityLow,
		PriorityHigh,
		PriorityNormal,
	}

	for i, p := range priorities {
		req := &QueuedRequest{
			ID:        fmt.Sprintf("req-%d", i),
			Priority:  p,
			AccountID: "account-1",
			Model:     "claude-3-5-sonnet",
			Response:  make(chan Response, 1),
		}

		manager.SubmitRequest(req)
		fmt.Printf("Submitted request %s with priority %s\n", req.ID, p)
	}

	// Requests will be processed in priority order:
	// VIP -> High -> Normal -> Low
}

// ExampleDegradationPolicy demonstrates degradation policy usage
func ExampleDegradationPolicy() {
	config := DefaultManagerConfig()
	config.Degradation.QueueThreshold = 50
	config.Degradation.ErrorThreshold = 0.5
	config.Degradation.CooldownDuration = 30 * time.Second

	manager := NewManager(config)
	manager.Start(&SimpleProcessor{})
	defer manager.Stop()

	// Simulate high load
	for i := 0; i < 100; i++ {
		req := &QueuedRequest{
			ID:        fmt.Sprintf("req-%d", i),
			Priority:  PriorityNormal,
			AccountID: "account-1",
			Model:     "claude-3-5-sonnet",
			Response:  make(chan Response, 1),
		}

		if err := manager.SubmitRequest(req); err != nil {
			fmt.Printf("Request %d rejected: %v\n", i, err)

			// Check if degraded
			if manager.GetStatus().IsDegraded {
				fmt.Println("System is degraded, pausing submissions")
				break
			}
			continue
		}
	}

	// Check degradation level
	stats := manager.GetStatus()
	fmt.Printf("Is degraded: %v\n", stats.IsDegraded)
}

// ContextKey is a type for context keys
type ContextKey string

const (
	// QueueContextKey is the context key for queue manager
	QueueContextKey ContextKey = "queue_manager"
)

// WithQueueManager adds queue manager to context
func WithQueueManager(ctx context.Context, manager *Manager) context.Context {
	return context.WithValue(ctx, QueueContextKey, manager)
}

// GetQueueManagerFromContext gets queue manager from context
func GetQueueManagerFromContext(ctx context.Context) *Manager {
	if m, ok := ctx.Value(QueueContextKey).(*Manager); ok {
		return m
	}
	return nil
}
