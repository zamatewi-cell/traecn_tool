package proxy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Message is a chat message in the upstream payload. Roles follow the
// OpenAI convention (system / user / assistant / tool).
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall is a native function call issued by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction carries the function name and JSON arguments.
type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolSpec declares an available tool to the model.
type ToolSpec struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunctionSpec is the declaration side of a tool (schema).
type ToolFunctionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// Tool is the upstream tool declaration wrapper.
type Tool struct {
	Type     string           `json:"type"`
	Function ToolFunctionSpec `json:"function"`
}

// ChatCompletionRequest is the upstream Trae chat request payload.
type ChatCompletionRequest struct {
	ModelName      string    `json:"model_name"`
	Messages       []Message `json:"messages"`
	Stream         bool      `json:"stream"`
	SessionID      string    `json:"session_id,omitempty"`
	ConversationID string    `json:"conversation_id,omitempty"`
	TaskID         string    `json:"task_id,omitempty"`
	Tools          []Tool    `json:"tools,omitempty"`
}

// RequestIDs bundles the per-request UUIDs injected into headers/body.
type RequestIDs struct {
	RequestID      string // req_<uuid> header id
	TraeRequestID  string // <uuid> trace id
	SessionID      string
	ConversationID string
	TaskID         string
}

// NewRequestIDs generates a fresh spec-compliant id set.
func NewRequestIDs() RequestIDs {
	return RequestIDs{
		RequestID:      "req_" + uuid.New().String(),
		TraeRequestID:  uuid.New().String(),
		SessionID:      uuid.New().String(),
		ConversationID: uuid.New().String(),
		TaskID:         uuid.New().String(),
	}
}

// upstreamChunk tolerantly models the upstream SSE data payloads. The real
// backend speaks an OpenAI-like dialect with extra task/queue fields.
type upstreamChunk struct {
	ID           string `json:"id"`
	Event        string `json:"event"`
	TaskID       string `json:"task_id"`
	MessageDelta string `json:"message_delta"`
	QueuePos     *int   `json:"queue_position"`
	Code         int    `json:"code"`
	Message      string `json:"message"`
	Finish       string `json:"finish_reason"`
	Error        *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
	Choices []struct {
		Index int `json:"index"`
		Delta *struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		Message *struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// parseUpstreamEvent converts one upstream SSE event into normalized events.
// The live llm_raw_chat backend speaks a named-event dialect:
//
//	event: metadata     {"session_id":...}                       -> ignored
//	event: output       {"response","reasoning_content",...}     -> text / reasoning / tool calls
//	event: token_usage  {"prompt_tokens","completion_tokens",...}-> usage
//	event: done         {"finish_reason":"stop"}                 -> finish
//	event: error        {"code":4023,"message":"..."}            -> error
//
// Unknown / empty event names fall back to shape-based legacy parsing.
func parseUpstreamEvent(eventName, data string) []*StreamEvent {
	switch eventName {
	case "metadata":
		return nil
	case "output":
		return parseOutputEvent(data)
	case "token_usage":
		return parseTokenUsageEvent(data)
	case "error":
		return parseErrorEvent(data)
	default:
		return parseUpstreamData(data)
	}
}

// parseErrorEvent handles the real dialect's "error" event, whose payload
// carries "error" as a plain string (""), which the shape-based legacy
// parser cannot unmarshal into its error struct.
func parseErrorEvent(data string) []*StreamEvent {
	var e struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(data)), &e); err != nil {
		return []*StreamEvent{{Type: EventError, Err: &UpstreamError{Message: data}}}
	}
	if e.Message == "" && e.Code == 0 {
		return nil
	}
	typ := "upstream_error"
	if e.Code != 0 {
		typ = fmt.Sprintf("upstream_code_%d", e.Code)
	}
	return []*StreamEvent{{
		Type: EventError,
		Err:  &UpstreamError{Message: e.Message, Type: typ},
	}}
}

// outputEvent is the llm_raw_chat "output" SSE payload.
type outputEvent struct {
	Response         string `json:"response"`
	ReasoningContent string `json:"reasoning_content"`
	ToolCalls        []struct {
		Index    int    `json:"index"`
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls"`
}

func parseOutputEvent(data string) []*StreamEvent {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return nil
	}
	var out outputEvent
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return []*StreamEvent{{Type: EventText, Text: data}}
	}
	var evts []*StreamEvent
	if out.ReasoningContent != "" {
		evts = append(evts, &StreamEvent{Type: EventReasoning, Reasoning: out.ReasoningContent})
	}
	if out.Response != "" {
		evts = append(evts, &StreamEvent{Type: EventText, Text: out.Response})
	}
	for _, tc := range out.ToolCalls {
		evts = append(evts, &StreamEvent{
			Type: EventToolCall,
			ToolCall: &ToolCallDelta{
				Index:     tc.Index,
				ID:        tc.ID,
				Type:      tc.Type,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return evts
}

func parseTokenUsageEvent(data string) []*StreamEvent {
	var u struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(data)), &u); err != nil {
		return nil
	}
	return []*StreamEvent{{
		Type: EventUsage,
		Usage: &Usage{
			PromptTokens:     u.PromptTokens,
			CompletionTokens: u.CompletionTokens,
			TotalTokens:      u.TotalTokens,
		},
	}}
}

// parseUpstreamData converts one upstream SSE data payload into normalized
// events. Non-JSON payloads degrade to raw text events (legacy behavior).
func parseUpstreamData(data string) []*StreamEvent {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return nil
	}
	if trimmed == "[DONE]" {
		return []*StreamEvent{{Type: EventFinish, FinishReason: "stop"}}
	}

	var chunk upstreamChunk
	if err := json.Unmarshal([]byte(trimmed), &chunk); err != nil {
		// Legacy / plain-text upstream payload.
		return []*StreamEvent{{Type: EventText, Text: data}}
	}

	var out []*StreamEvent

	if chunk.Error != nil && chunk.Error.Message != "" {
		out = append(out, &StreamEvent{
			Type: EventError,
			Err:  &UpstreamError{Message: chunk.Error.Message, Type: chunk.Error.Type},
		})
	}
	// Upstream SSE "event: error" envelope: {"code":4001,"message":"..."}
	if chunk.Code != 0 && chunk.Message != "" {
		out = append(out, &StreamEvent{
			Type: EventError,
			Err:  &UpstreamError{Message: chunk.Message, Type: fmt.Sprintf("upstream_code_%d", chunk.Code)},
		})
	}
	// Bare {"finish_reason":...} payload (upstream "event: done") with no choices.
	if len(chunk.Choices) == 0 && strings.Contains(trimmed, `"finish_reason"`) {
		reason := chunk.Finish
		if reason == "" {
			reason = "stop"
		}
		out = append(out, &StreamEvent{Type: EventFinish, FinishReason: reason})
		return out
	}
	if chunk.QueuePos != nil && *chunk.QueuePos > 0 {
		out = append(out, &StreamEvent{
			Type:          EventQueue,
			QueuePosition: *chunk.QueuePos,
		})
	}
	if chunk.Event == "task_created" || chunk.TaskID != "" {
		out = append(out, &StreamEvent{Type: EventTaskCreated, TaskID: chunk.TaskID})
	}
	if chunk.MessageDelta != "" {
		out = append(out, &StreamEvent{Type: EventText, Text: chunk.MessageDelta})
	}

	for _, choice := range chunk.Choices {
		if d := choice.Delta; d != nil {
			if d.ReasoningContent != "" {
				out = append(out, &StreamEvent{Type: EventReasoning, Reasoning: d.ReasoningContent})
			}
			if d.Content != "" {
				out = append(out, &StreamEvent{Type: EventText, Text: d.Content})
			}
			for _, tc := range d.ToolCalls {
				out = append(out, &StreamEvent{
					Type: EventToolCall,
					ToolCall: &ToolCallDelta{
						Index:     tc.Index,
						ID:        tc.ID,
						Type:      tc.Type,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
		if m := choice.Message; m != nil && m.Content != "" {
			out = append(out, &StreamEvent{Type: EventText, Text: m.Content})
		}
		if choice.FinishReason != nil && *choice.FinishReason != "" {
			out = append(out, &StreamEvent{Type: EventFinish, FinishReason: *choice.FinishReason})
		}
	}

	if chunk.Usage != nil {
		out = append(out, &StreamEvent{
			Type: EventUsage,
			Usage: &Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			},
		})
	}

	return out
}

// UpstreamError represents an error returned by the upstream API.
type UpstreamError struct {
	Message string
	Type    string
}

func (e *UpstreamError) Error() string {
	if e.Type != "" {
		return e.Type + ": " + e.Message
	}
	return e.Message
}
