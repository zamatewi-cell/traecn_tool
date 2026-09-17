package workflow

import (
	"fmt"
	"sync"
	"time"
)

// EventType represents event type
type EventType string

const (
	// Task events
	EventTaskCreated   EventType = "task.created"
	EventTaskStarted   EventType = "task.started"
	EventTaskProgress  EventType = "task.progress"
	EventTaskCompleted EventType = "task.completed"
	EventTaskFailed    EventType = "task.failed"
	EventTaskCancelled EventType = "task.cancelled"

	// Agent events
	EventAgentStarted   EventType = "agent.started"
	EventAgentStopped   EventType = "agent.stopped"
	EventAgentPaused    EventType = "agent.paused"
	EventAgentResumed   EventType = "agent.resumed"
	EventAgentError     EventType = "agent.error"
	EventProgressUpdate EventType = "progress.update"

	// System events
	EventSystemShutdown EventType = "system.shutdown"
	EventSystemConfig   EventType = "system.config"
	EventSystemError    EventType = "system.error"
	EventError          EventType = "error"
)

// Event represents an event
type Event struct {
	ID        string
	Type      EventType
	Timestamp time.Time
	Source    string // Event source (Agent ID or system component)
	Payload   interface{}
}

// EventHandler is event handler function
type EventHandler func(event *Event)

// EventBus is event bus for inter-agent communication
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]EventHandler
	history     []*Event
	maxHistory  int
}

// NewEventBus creates new event bus
func NewEventBus(maxHistory int) *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]EventHandler),
		history:     make([]*Event, 0),
		maxHistory:  maxHistory,
	}
}

// Subscribe subscribes to specific event type
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
	fmt.Printf("[EventBus] Subscribed to event: %s\n", eventType)
}

// SubscribeMultiple subscribes to multiple event types
func (eb *EventBus) SubscribeMultiple(eventTypes []EventType, handler EventHandler) {
	for _, eventType := range eventTypes {
		eb.Subscribe(eventType, handler)
	}
}

// Unsubscribe unsubscribes from event type
func (eb *EventBus) Unsubscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers := eb.subscribers[eventType]
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			eb.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Publish publishes event
func (eb *EventBus) Publish(event *Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Set timestamp
	event.Timestamp = time.Now()

	// Add to history
	eb.history = append(eb.history, event)
	if len(eb.history) > eb.maxHistory {
		eb.history = eb.history[1:]
	}

	// Notify subscribers
	handlers := eb.subscribers[event.Type]
	for _, handler := range handlers {
		go handler(event)
	}

	fmt.Printf("[EventBus] Published event: %s from %s\n", event.Type, event.Source)
}

// GetHistory gets event history
func (eb *EventBus) GetHistory() []*Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	result := make([]*Event, len(eb.history))
	copy(result, eb.history)
	return result
}

// GetHistoryByType gets event history filtered by type
func (eb *EventBus) GetHistoryByType(eventType EventType) []*Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	result := make([]*Event, 0)
	for _, event := range eb.history {
		if event.Type == eventType {
			result = append(result, event)
		}
	}
	return result
}

// GetHistoryBySource gets event history filtered by source
func (eb *EventBus) GetHistoryBySource(source string) []*Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	result := make([]*Event, 0)
	for _, event := range eb.history {
		if event.Source == source {
			result = append(result, event)
		}
	}
	return result
}

// ClearHistory clears event history
func (eb *EventBus) ClearHistory() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.history = make([]*Event, 0)
	fmt.Println("[EventBus] Event history cleared")
}

// GetStats gets event bus statistics
func (eb *EventBus) GetStats() EventBusStats {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subscriberCount := 0
	for _, handlers := range eb.subscribers {
		subscriberCount += len(handlers)
	}

	return EventBusStats{
		TotalEvents:     len(eb.history),
		SubscriberCount: subscriberCount,
		EventTypes:      len(eb.subscribers),
	}
}

// EventBusStats represents event bus statistics
type EventBusStats struct {
	TotalEvents     int
	SubscriberCount int
	EventTypes      int
}
