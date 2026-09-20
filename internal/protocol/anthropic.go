package protocol

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/zamatewi-cell/traecn_tool/internal/models"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

// AnthropicHandler implements the Anthropic Messages API (POST /v1/messages)
// so Claude Code (via CC Switch) can connect directly.
type AnthropicHandler struct {
	proxy *proxy.TraeProxy
}

// NewAnthropicHandler creates the handler.
func NewAnthropicHandler(p *proxy.TraeProxy) *AnthropicHandler {
	return &AnthropicHandler{proxy: p}
}

// anthropicRequest is the inbound Messages API payload.
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      json.RawMessage    `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Stream      bool               `json:"stream,omitempty"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
	ToolChoice  json.RawMessage    `json:"tool_choice,omitempty"`
	Temperature *float64           `json:"temperature,omitempty"`
	TopP        *float64           `json:"top_p,omitempty"`
	StopSeqs    []string           `json:"stop_sequences,omitempty"`
	Metadata    json.RawMessage    `json:"metadata,omitempty"`
}

type anthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

// anthropicBlock is one content block inside a message.
type anthropicBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// parseContent accepts a string or an array of content blocks.
func (m *anthropicMessage) parseContent() ([]anthropicBlock, error) {
	if len(m.Content) == 0 || string(m.Content) == "null" {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(m.Content, &s); err == nil {
		return []anthropicBlock{{Type: "text", Text: s}}, nil
	}
	var blocks []anthropicBlock
	if err := json.Unmarshal(m.Content, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// systemText extracts the system prompt (string or block array form).
func systemText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []anthropicBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

// toUpstream converts the Anthropic request into the upstream payload.
func (r *anthropicRequest) toUpstream() (*proxy.ChatCompletionRequest, error) {
	out := &proxy.ChatCompletionRequest{
		Stream: r.Stream,
	}
	if m := models.Default().Resolve(r.Model); m != nil {
		out.ModelName = m.ModelID
	} else {
		out.ModelName = models.DefaultModel
	}

	if sys := systemText(r.System); sys != "" {
		out.Messages = append(out.Messages, proxy.Message{Role: "system", Content: sys})
	}

	for _, msg := range r.Messages {
		blocks, err := msg.parseContent()
		if err != nil {
			return nil, fmt.Errorf("invalid content in %s message: %w", msg.Role, err)
		}

		var texts []string
		var toolCalls []proxy.ToolCall
		var toolResults []proxy.Message

		for _, b := range blocks {
			switch b.Type {
			case "text":
				if b.Text != "" {
					texts = append(texts, b.Text)
				}
			case "image":
				texts = append(texts, "[image]")
			case "tool_use":
				args := "{}"
				if len(b.Input) > 0 {
					args = string(b.Input)
				}
				toolCalls = append(toolCalls, proxy.ToolCall{
					ID:   b.ID,
					Type: "function",
					Function: proxy.ToolFunction{
						Name:      b.Name,
						Arguments: args,
					},
				})
			case "tool_result":
				content := ""
				if len(b.Content) > 0 {
					var s string
					if err := json.Unmarshal(b.Content, &s); err == nil {
						content = s
					} else {
						var parts []anthropicBlock
						if err := json.Unmarshal(b.Content, &parts); err == nil {
							for _, p := range parts {
								if p.Type == "text" {
									content += p.Text
								}
							}
						}
					}
				}
				toolResults = append(toolResults, proxy.Message{
					Role:       "tool",
					Content:    content,
					ToolCallID: b.ToolUseID,
				})
			}
		}

		// tool_result blocks become standalone tool messages.
		out.Messages = append(out.Messages, toolResults...)

		if len(texts) > 0 || len(toolCalls) > 0 {
			out.Messages = append(out.Messages, proxy.Message{
				Role:      msg.Role,
				Content:   strings.Join(texts, "\n"),
				ToolCalls: toolCalls,
			})
		}
	}

	for _, t := range r.Tools {
		out.Tools = append(out.Tools, proxy.Tool{
			Type: "function",
			Function: proxy.ToolFunctionSpec{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	return out, nil
}

// mapStopReason converts OpenAI-style finish reasons to Anthropic ones.
func mapStopReason(finish string, hasToolCalls bool) string {
	if hasToolCalls || finish == "tool_calls" {
		return "tool_use"
	}
	switch finish {
	case "length":
		return "max_tokens"
	case "stop", "":
		return "end_turn"
	default:
		return finish
	}
}

// anthropicUsage is the usage block shape shared by response and stream.
type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func usageFromProxy(u *proxy.Usage) anthropicUsage {
	if u == nil {
		return anthropicUsage{}
	}
	return anthropicUsage{InputTokens: u.PromptTokens, OutputTokens: u.CompletionTokens}
}

// HandleMessages implements POST /v1/messages.
func (h *AnthropicHandler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeProtocolError(w, http.StatusMethodNotAllowed, "invalid_request_error", "Only POST method is allowed")
		return
	}

	var req anthropicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProtocolError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON: "+err.Error())
		return
	}
	if req.Model == "" {
		writeProtocolError(w, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	if len(req.Messages) == 0 {
		writeProtocolError(w, http.StatusBadRequest, "invalid_request_error", "messages is required")
		return
	}

	upstream, err := req.toUpstream()
	if err != nil {
		writeProtocolError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	upstream.Context = r.Context()

	if req.Stream {
		h.handleStreaming(w, upstream, req.Model)
	} else {
		h.handleNonStreaming(w, upstream, req.Model)
	}
}

func (h *AnthropicHandler) handleNonStreaming(w http.ResponseWriter, upstream *proxy.ChatCompletionRequest, model string) {
	c, err := collect(h.proxy, upstream)
	if err != nil {
		writeProtocolError(w, http.StatusBadGateway, "api_error", "Upstream error: "+err.Error())
		return
	}

	var blocks []anthropicBlock
	if c.reasoning != "" {
		blocks = append(blocks, anthropicBlock{Type: "thinking", Thinking: c.reasoning})
	}
	if c.content != "" {
		blocks = append(blocks, anthropicBlock{Type: "text", Text: c.content})
	}
	for _, idx := range c.toolOrder {
		tc := c.toolCalls[idx]
		input := json.RawMessage(tc.args)
		if len(input) == 0 || !json.Valid(input) {
			input = json.RawMessage("{}")
		}
		blocks = append(blocks, anthropicBlock{
			Type:  "tool_use",
			ID:    tc.id,
			Name:  tc.name,
			Input: input,
		})
	}

	resp := map[string]interface{}{
		"id":            "msg_" + uuid.New().String(),
		"type":          "message",
		"role":          "assistant",
		"model":         model,
		"content":       blocks,
		"stop_reason":   mapStopReason(c.finishReason, len(c.toolOrder) > 0),
		"stop_sequence": nil,
		"usage":         usageFromProxy(c.usage),
	}
	writeJSON(w, http.StatusOK, resp)
}

type streamToolCallAccumulator struct {
	id        string
	name      string
	arguments strings.Builder
}

func (h *AnthropicHandler) handleStreaming(w http.ResponseWriter, upstream *proxy.ChatCompletionRequest, model string) {
	em, ok := newSSEEmitter(w)
	if !ok {
		writeProtocolError(w, http.StatusInternalServerError, "api_error", "Streaming not supported")
		return
	}

	msgID := "msg_" + uuid.New().String()

	// message_start
	em.emit("message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            msgID,
			"type":          "message",
			"role":          "assistant",
			"model":         model,
			"content":       []interface{}{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage":         anthropicUsage{InputTokens: 0, OutputTokens: 0},
		},
	})

	// Content block state machine.
	blockOpen := false
	blockIndex := -1
	blockType := ""
	toolAccMap := make(map[int]*streamToolCallAccumulator)
	var toolOrder []int
	var hasEmittedToolCalls bool
	finishReason := "stop"
	var usage *proxy.Usage

	closeBlock := func() {
		if blockOpen {
			em.emit("content_block_stop", map[string]interface{}{
				"type":  "content_block_stop",
				"index": blockIndex,
			})
			blockOpen = false
		}
	}
	openBlock := func(payload map[string]interface{}) {
		blockIndex++
		em.emit("content_block_start", map[string]interface{}{
			"type":          "content_block_start",
			"index":         blockIndex,
			"content_block": payload,
		})
		blockOpen = true
	}

	flushToolCalls := func() {
		closeBlock()
		for _, idx := range toolOrder {
			acc := toolAccMap[idx]
			if acc == nil {
				continue
			}
			openBlock(map[string]interface{}{
				"type":  "tool_use",
				"id":    acc.id,
				"name":  acc.name,
				"input": map[string]interface{}{},
			})
			blockType = "tool_use"
			args := acc.arguments.String()
			if args != "" {
				em.emit("content_block_delta", map[string]interface{}{
					"type":  "content_block_delta",
					"index": blockIndex,
					"delta": map[string]string{"type": "input_json_delta", "partial_json": args},
				})
			}
			closeBlock()
		}
		if len(toolOrder) > 0 {
			hasEmittedToolCalls = true
		}
		toolOrder = nil
		toolAccMap = make(map[int]*streamToolCallAccumulator)
	}

	err := h.proxy.ChatCompletion(upstream, func(evt *proxy.StreamEvent) error {
		switch evt.Type {
		case proxy.EventReasoning:
			if !blockOpen || blockType != "thinking" {
				closeBlock()
				openBlock(map[string]interface{}{"type": "thinking", "thinking": ""})
				blockType = "thinking"
			}
			em.emit("content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": blockIndex,
				"delta": map[string]string{"type": "thinking_delta", "thinking": evt.Reasoning},
			})
		case proxy.EventText:
			if !blockOpen || blockType != "text" {
				closeBlock()
				openBlock(map[string]interface{}{"type": "text", "text": ""})
				blockType = "text"
			}
			em.emit("content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": blockIndex,
				"delta": map[string]string{"type": "text_delta", "text": evt.Text},
			})
		case proxy.EventToolCall:
			tc := evt.ToolCall
			acc, ok := toolAccMap[tc.Index]
			if !ok {
				id := tc.ID
				if id == "" {
					id = "toolu_" + uuid.New().String()
				}
				acc = &streamToolCallAccumulator{
					id:   id,
					name: tc.Name,
				}
				toolAccMap[tc.Index] = acc
				toolOrder = append(toolOrder, tc.Index)
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Name != "" {
				acc.name = tc.Name
			}
			if tc.Arguments != "" {
				acc.arguments.WriteString(tc.Arguments)
			}
		case proxy.EventFinish:
			finishReason = evt.FinishReason
		case proxy.EventUsage:
			usage = evt.Usage
		case proxy.EventError:
			return evt.Err
		}
		return nil
	})

	if err != nil {
		em.emit("error", map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "api_error",
				"message": "Stream error: " + err.Error(),
			},
		})
		return
	}

	flushToolCalls()
	closeBlock()

	u := usageFromProxy(usage)
	em.emit("message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   mapStopReason(finishReason, hasEmittedToolCalls || len(toolOrder) > 0),
			"stop_sequence": nil,
		},
		"usage": map[string]int{"output_tokens": u.OutputTokens},
	})
	em.emit("message_stop", map[string]interface{}{"type": "message_stop"})
}
