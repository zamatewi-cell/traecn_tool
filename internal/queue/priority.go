package queue

import (
	"time"
)

// Priority represents request priority level
type Priority int

const (
	PriorityLow    Priority = 1   // Low priority - processed when system idle
	PriorityNormal Priority = 5   // Normal priority - processed in FIFO order
	PriorityHigh   Priority = 10  // High priority - processed when VIP idle
	PriorityVIP    Priority = 100 // VIP priority - processed immediately (preemptive)
)

// String returns priority string representation
func (p Priority) String() string {
	switch p {
	case PriorityVIP:
		return "VIP"
	case PriorityHigh:
		return "HIGH"
	case PriorityNormal:
		return "NORMAL"
	case PriorityLow:
		return "LOW"
	default:
		return "UNKNOWN"
	}
}

// QueuedRequest represents a request in the queue
type QueuedRequest struct {
	ID         string
	Priority   Priority
	AccountID  string
	Model      string
	Request    interface{} // *http.Request
	Response   chan<- Response
	AddedAt    time.Time
	UpdatedAt  time.Time
	RetryCount int
}

// Response represents queue response
type Response struct {
	Data       interface{}
	Error      error
	StatusCode int
	Headers    map[string]string
}
