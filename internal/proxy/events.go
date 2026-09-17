package proxy

// EventType classifies normalized upstream stream events.
type EventType int

const (
	// EventText carries an increment of the assistant reply.
	EventText EventType = iota
	// EventReasoning carries an increment of the model's thinking chain.
	EventReasoning
	// EventToolCall carries an increment of a native tool call.
	EventToolCall
	// EventQueue reports upstream queueing (queue_position).
	EventQueue
	// EventTaskCreated reports the upstream task id (task_created).
	EventTaskCreated
	// EventFinish signals completion with a finish reason.
	EventFinish
	// EventUsage carries token usage statistics.
	EventUsage
	// EventError carries an upstream or transport error.
	EventError
)

// ToolCallDelta is one increment of a streamed tool call.
type ToolCallDelta struct {
	Index     int    `json:"index"`
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// Usage mirrors token usage statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CachedTokens     int `json:"cached_tokens,omitempty"`
}

// StreamEvent is the protocol-neutral event emitted by the forwarding
// layer; each client-facing protocol (OpenAI / Anthropic / Responses)
// renders it into its own SSE dialect.
type StreamEvent struct {
	Type          EventType
	Text          string         // EventText payload
	Reasoning     string         // EventReasoning payload
	ToolCall      *ToolCallDelta // EventToolCall payload
	QueuePosition int            // EventQueue payload
	QueueMessage  string         // EventQueue human message
	TaskID        string         // EventTaskCreated payload
	FinishReason  string         // EventFinish payload
	Usage         *Usage         // EventUsage payload
	Err           error          // EventError payload
}

// StreamHandler consumes normalized stream events; returning an error
// aborts the stream.
type StreamHandler func(evt *StreamEvent) error
