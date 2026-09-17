package queue

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEEvent represents SSE event
type SSEEvent struct {
	Object        string      `json:"object"`
	Position      int         `json:"position,omitempty"`
	EstimatedWait string      `json:"estimated_wait,omitempty"`
	Message       string      `json:"message,omitempty"`
	Data          interface{} `json:"data,omitempty"`
	ID            string      `json:"id,omitempty"`
}

// SSEClient represents an SSE client connection
type SSEClient struct {
	ID           string
	Model        string
	Writer       http.ResponseWriter
	Flusher      http.Flusher
	Done         chan struct{}
	LastActivity time.Time
}

// SSEBroker manages SSE client connections
type SSEBroker struct {
	mu      sync.RWMutex
	clients map[string]*SSEClient
	queue   *Scheduler
}

// NewSSEBroker creates a new SSE broker
func NewSSEBroker(queue *Scheduler) *SSEBroker {
	return &SSEBroker{
		clients: make(map[string]*SSEClient),
		queue:   queue,
	}
}

// AddClient adds an SSE client
func (b *SSEBroker) AddClient(clientID string, client *SSEClient) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[clientID] = client
}

// RemoveClient removes an SSE client
func (b *SSEBroker) RemoveClient(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, clientID)
}

// BroadcastQueueStatus broadcasts queue status to all clients
func (b *SSEBroker) BroadcastQueueStatus(model string, position int, estimatedWait time.Duration) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	event := SSEEvent{
		Object:        "queue.status",
		Position:      position,
		EstimatedWait: estimatedWait.String(),
	}

	data, _ := json.Marshal(event)

	for _, client := range b.clients {
		if client.Model == model {
			select {
			case <-client.Done:
				continue
			default:
				fmt.Fprintf(client.Writer, "data: %s\n\n", data)
				client.Flusher.Flush()
				client.LastActivity = time.Now()
			}
		}
	}
}

// SendToClient sends event to specific client
func (b *SSEBroker) SendToClient(clientID string, event SSEEvent) error {
	b.mu.RLock()
	client, exists := b.clients[clientID]
	b.mu.RUnlock()

	if !exists {
		return ErrClientNotFound
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	select {
	case <-client.Done:
		return ErrClientDisconnected
	default:
		fmt.Fprintf(client.Writer, "data: %s\n\n", data)
		client.Flusher.Flush()
		client.LastActivity = time.Now()
		return nil
	}
}

// BroadcastToModel broadcasts to all clients subscribed to a model
func (b *SSEBroker) BroadcastToModel(model string, event SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	data, _ := json.Marshal(event)

	for _, client := range b.clients {
		if client.Model == model {
			select {
			case <-client.Done:
				continue
			default:
				fmt.Fprintf(client.Writer, "data: %s\n\n", data)
				client.Flusher.Flush()
			}
		}
	}
}

// StartHealthChecker starts health checking for inactive clients
func (b *SSEBroker) StartHealthChecker(timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			b.mu.Lock()
			now := time.Now()
			for id, client := range b.clients {
				if now.Sub(client.LastActivity) > timeout {
					close(client.Done)
					delete(b.clients, id)
				}
			}
			b.mu.Unlock()
		}
	}()
}

// GetClientCount returns number of connected clients
func (b *SSEBroker) GetClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// GetClientsByModel returns clients subscribed to a model
func (b *SSEBroker) GetClientsByModel(model string) []*SSEClient {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var clients []*SSEClient
	for _, client := range b.clients {
		if client.Model == model {
			clients = append(clients, client)
		}
	}
	return clients
}

// CalculateEstimatedWait calculates estimated wait time based on queue position
func (b *SSEBroker) CalculateEstimatedWait(position int) time.Duration {
	if position <= 0 {
		return 0
	}

	// Assume average 5 seconds per request
	avgProcessingTime := 5 * time.Second
	return time.Duration(position) * avgProcessingTime
}

// SSEHandler creates HTTP handler for SSE endpoint
func (b *SSEBroker) SSEHandler(model string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		clientID := r.RemoteAddr
		client := &SSEClient{
			ID:           clientID,
			Model:        model,
			Writer:       w,
			Flusher:      flusher,
			Done:         make(chan struct{}),
			LastActivity: time.Now(),
		}

		b.AddClient(clientID, client)
		defer b.RemoveClient(clientID)

		// Send initial connection event
		b.SendToClient(clientID, SSEEvent{
			Object:  "connection.opened",
			Message: "Connected to queue status stream",
		})

		// Keep connection alive
		<-r.Context().Done()
	}
}

// QueueStatusEvent creates queue status event
func QueueStatusEvent(position int, estimatedWait time.Duration) SSEEvent {
	return SSEEvent{
		Object:        "queue.status",
		Position:      position,
		EstimatedWait: estimatedWait.String(),
	}
}

// ChatCompletionEvent creates chat completion event
func ChatCompletionEvent(id string, content string) SSEEvent {
	return SSEEvent{
		ID:     id,
		Object: "chat.completion.chunk",
		Data: map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"delta": map[string]string{
						"content": content,
					},
				},
			},
		},
	}
}

// Error events
var (
	ErrClientNotFound     = &ClientError{message: "client not found"}
	ErrClientDisconnected = &ClientError{message: "client disconnected"}
)

// ClientError represents client error
type ClientError struct {
	message string
}

func (e *ClientError) Error() string {
	return e.message
}
