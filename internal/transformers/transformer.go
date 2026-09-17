package transformers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/models"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

// OpenAIRequest represents an OpenAI-compatible chat completion request
type OpenAIRequest struct {
	Model            string          `json:"model"`
	Messages         []OpenAIMessage `json:"messages"`
	Stream           bool            `json:"stream"`
	Temperature      *float64        `json:"temperature,omitempty"`
	MaxTokens        *int            `json:"max_tokens,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	FrequencyPenalty *float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64        `json:"presence_penalty,omitempty"`
	Stop             []string        `json:"stop,omitempty"`
	User             string          `json:"user,omitempty"`
	Tools            []OpenAITool    `json:"tools,omitempty"`
	ToolChoice       json.RawMessage `json:"tool_choice,omitempty"`
}

// OpenAITool is an OpenAI function-calling tool declaration.
type OpenAITool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Parameters  json.RawMessage `json:"parameters,omitempty"`
	} `json:"function"`
}

// OpenAIToolCall is a tool call embedded in an assistant message.
type OpenAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// OpenAIMessage represents an OpenAI-format message. Content accepts both
// the plain string form and the multipart [{type:"text",text:...}] form.
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
}

// UnmarshalJSON accepts content as a string or as an array of content
// parts (text parts are concatenated; image parts degrade to a placeholder).
func (m *OpenAIMessage) UnmarshalJSON(data []byte) error {
	type rawPart struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	var raw struct {
		Role       string           `json:"role"`
		Content    json.RawMessage  `json:"content"`
		Name       string           `json:"name,omitempty"`
		ToolCallID string           `json:"tool_call_id,omitempty"`
		ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role, m.Name, m.ToolCallID, m.ToolCalls = raw.Role, raw.Name, raw.ToolCallID, raw.ToolCalls

	if len(raw.Content) == 0 || string(raw.Content) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw.Content, &s); err == nil {
		m.Content = s
		return nil
	}
	var parts []rawPart
	if err := json.Unmarshal(raw.Content, &parts); err == nil {
		var sb strings.Builder
		for i, p := range parts {
			switch p.Type {
			case "text":
				sb.WriteString(p.Text)
			case "image_url", "image":
				sb.WriteString("[image]")
			default:
				sb.WriteString(p.Text)
			}
			if i < len(parts)-1 && p.Type == "text" {
				sb.WriteString("\n")
			}
		}
		m.Content = sb.String()
		return nil
	}
	return fmt.Errorf("unsupported content shape")
}

// OpenAIResponse represents an OpenAI-compatible chat completion response
type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message,omitempty"`
	Delta        *Delta  `json:"delta,omitempty"`
	FinishReason *string `json:"finish_reason,omitempty"`
}

// Message represents a completion message
type Message struct {
	Role             string           `json:"role"`
	Content          string           `json:"content"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	ToolCalls        []OpenAIToolCall `json:"tool_calls,omitempty"`
}

// DeltaToolCall is a streamed tool_call increment.
type DeltaToolCall struct {
	Index    int            `json:"index"`
	ID       string         `json:"id,omitempty"`
	Type     string         `json:"type,omitempty"`
	Function *DeltaFunction `json:"function,omitempty"`
}

// DeltaFunction carries streamed function name/argument increments.
type DeltaFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// Delta represents a streaming delta
type Delta struct {
	Role             *string         `json:"role,omitempty"`
	Content          *string         `json:"content,omitempty"`
	ReasoningContent *string         `json:"reasoning_content,omitempty"`
	ToolCalls        []DeltaToolCall `json:"tool_calls,omitempty"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// RequestTransformer converts OpenAI requests to the upstream payload.
type RequestTransformer struct{}

// NewRequestTransformer creates a new request transformer
func NewRequestTransformer() *RequestTransformer {
	return &RequestTransformer{}
}

// Transform converts an OpenAI request into the upstream chat payload:
// model alias resolution, role-preserving message mapping (system / user /
// assistant / tool), and tool declaration conversion.
func (rt *RequestTransformer) Transform(req *OpenAIRequest) (*proxy.ChatCompletionRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}

	out := &proxy.ChatCompletionRequest{
		ModelName: rt.mapModel(req.Model),
		Stream:    req.Stream,
	}

	for _, msg := range req.Messages {
		up := proxy.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
		for _, tc := range msg.ToolCalls {
			up.ToolCalls = append(up.ToolCalls, proxy.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: proxy.ToolFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		out.Messages = append(out.Messages, up)
	}

	for _, t := range req.Tools {
		out.Tools = append(out.Tools, proxy.Tool{
			Type: t.Type,
			Function: proxy.ToolFunctionSpec{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		})
	}

	return out, nil
}

// mapModel resolves a client-supplied model name to a canonical registry
// model ID via exact match / alias / fuzzy prefix; unknown names fall back
// to models.DefaultModel so clients sending foreign ids (e.g. "gpt-4o")
// still get served.
func (rt *RequestTransformer) mapModel(model string) string {
	if m := models.Default().Resolve(model); m != nil {
		return m.ModelID
	}
	return models.DefaultModel
}

// NewChunkID generates an OpenAI-style completion id.
func NewChunkID() string {
	return "chatcmpl-" + time.Now().Format("20060102150405")
}

// ---------------------------------------------------------------------------
// Legacy response-side types (retained for compatibility and tests)
// ---------------------------------------------------------------------------

// TraeRequest represents a Trae CN request (legacy shape)
type TraeRequest struct {
	ModelID        string                 `json:"model_id"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Messages       []proxy.Message        `json:"messages"`
	Stream         bool                   `json:"stream"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
}

// TraeResponse represents a Trae CN response
type TraeResponse struct {
	ID      string       `json:"id"`
	Choices []TraeChoice `json:"choices"`
	Usage   *TraeUsage   `json:"usage,omitempty"`
}

// TraeChoice represents a Trae CN choice
type TraeChoice struct {
	Delta        *TraeDelta   `json:"delta,omitempty"`
	Message      *TraeMessage `json:"message,omitempty"`
	FinishReason *string      `json:"finish_reason,omitempty"`
}

// TraeDelta represents a Trae CN streaming delta
type TraeDelta struct {
	Content string `json:"content"`
}

// TraeMessage represents a Trae CN message
type TraeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TraeUsage represents Trae CN token usage
type TraeUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ResponseTransformer converts Trae CN responses to OpenAI format
type ResponseTransformer struct{}

// NewResponseTransformer creates a new response transformer
func NewResponseTransformer() *ResponseTransformer {
	return &ResponseTransformer{}
}

// TransformChunk transforms a streaming Trae CN chunk to OpenAI format
func (rt *ResponseTransformer) TransformChunk(traeChunk *TraeResponse, model string) *OpenAIResponse {
	if len(traeChunk.Choices) == 0 {
		return nil
	}

	traeChoice := traeChunk.Choices[0]

	var delta *Delta
	if traeChoice.Delta != nil {
		content := traeChoice.Delta.Content
		delta = &Delta{
			Content: &content,
		}
	}

	return &OpenAIResponse{
		ID:      "chatcmpl-" + traeChunk.ID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []Choice{
			{
				Index: 0,
				Delta: delta,
			},
		},
	}
}

// TransformComplete transforms a complete Trae CN response to OpenAI format
func (rt *ResponseTransformer) TransformComplete(traeResp *TraeResponse, model string) *OpenAIResponse {
	if len(traeResp.Choices) == 0 {
		return nil
	}

	traeChoice := traeResp.Choices[0]

	var message Message
	if traeChoice.Message != nil {
		message = Message{
			Role:    traeChoice.Message.Role,
			Content: traeChoice.Message.Content,
		}
	}

	var usage *Usage
	if traeResp.Usage != nil {
		usage = &Usage{
			PromptTokens:     traeResp.Usage.PromptTokens,
			CompletionTokens: traeResp.Usage.CompletionTokens,
			TotalTokens:      traeResp.Usage.TotalTokens,
		}
	}

	return &OpenAIResponse{
		ID:      "chatcmpl-" + traeResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []Choice{
			{
				Index:        0,
				Message:      message,
				FinishReason: traeChoice.FinishReason,
			},
		},
		Usage: usage,
	}
}
