package protocol

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zamatewi-cell/traecn_tool/internal/models"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

// ResponsesHandler implements the OpenAI Responses API (POST /v1/responses)
// so Codex CLI can connect directly.
type ResponsesHandler struct {
	proxy *proxy.TraeProxy
}

// NewResponsesHandler creates the handler.
func NewResponsesHandler(p *proxy.TraeProxy) *ResponsesHandler {
	return &ResponsesHandler{proxy: p}
}

// responsesRequest is the inbound Responses API payload.
type responsesRequest struct {
	Model           string          `json:"model"`
	Instructions    string          `json:"instructions,omitempty"`
	Input           json.RawMessage `json:"input"`
	Tools           []responsesTool `json:"tools,omitempty"`
	Stream          bool            `json:"stream,omitempty"`
	MaxOutputTokens int             `json:"max_output_tokens,omitempty"`
	Temperature     *float64        `json:"temperature,omitempty"`
}

type responsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Function    *struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Parameters  json.RawMessage `json:"parameters,omitempty"`
	} `json:"function,omitempty"`
}

// inputItem is one item of the input array.
type inputItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments string          `json:"arguments,omitempty"`
	Output    string          `json:"output,omitempty"`
}

// toUpstream converts the Responses API request into the upstream payload.
func (r *responsesRequest) toUpstream() (*proxy.ChatCompletionRequest, error) {
	out := &proxy.ChatCompletionRequest{Stream: r.Stream}
	if m := models.Default().Resolve(r.Model); m != nil {
		out.ModelName = m.ModelID
	} else {
		out.ModelName = models.DefaultModel
	}

	if r.Instructions != "" {
		out.Messages = append(out.Messages, proxy.Message{Role: "system", Content: r.Instructions})
	}

	// input may be a plain string or an array of items.
	if len(r.Input) > 0 {
		var s string
		if err := json.Unmarshal(r.Input, &s); err == nil {
			out.Messages = append(out.Messages, proxy.Message{Role: "user", Content: s})
		} else {
			var items []inputItem
			if err := json.Unmarshal(r.Input, &items); err != nil {
				return nil, fmt.Errorf("invalid input: %w", err)
			}
			for _, item := range items {
				switch item.Type {
				case "message", "":
					role := item.Role
					if role == "" {
						role = "user"
					}
					out.Messages = append(out.Messages, proxy.Message{
						Role:    role,
						Content: itemContentText(item.Content),
					})
				case "function_call":
					out.Messages = append(out.Messages, proxy.Message{
						Role: "assistant",
						ToolCalls: []proxy.ToolCall{{
							ID:   item.CallID,
							Type: "function",
							Function: proxy.ToolFunction{
								Name:      item.Name,
								Arguments: item.Arguments,
							},
						}},
					})
				case "function_call_output":
					out.Messages = append(out.Messages, proxy.Message{
						Role:       "tool",
						Content:    item.Output,
						ToolCallID: item.CallID,
					})
				}
			}
		}
	}

	for _, t := range r.Tools {
		spec := proxy.ToolFunctionSpec{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		}
		if t.Function != nil {
			spec.Name = t.Function.Name
			spec.Description = t.Function.Description
			spec.Parameters = t.Function.Parameters
		}
		typ := t.Type
		if typ == "" || typ == "function" {
			typ = "function"
		}
		out.Tools = append(out.Tools, proxy.Tool{Type: typ, Function: spec})
	}

	return out, nil
}

// itemContentText extracts text from message item content (string or parts).
func itemContentText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}
	var sb strings.Builder
	for i, p := range parts {
		if p.Type == "input_text" || p.Type == "output_text" || p.Type == "text" {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(p.Text)
		}
	}
	return sb.String()
}

// responseObject builds the final Responses API object.
func responseObject(id, model string, c *collected) map[string]interface{} {
	var output []interface{}

	if c.reasoning != "" {
		output = append(output, map[string]interface{}{
			"id":     "rs_" + uuid.New().String(),
			"type":   "reasoning",
			"status": "completed",
			"summary": []interface{}{
				map[string]interface{}{"type": "summary_text", "text": c.reasoning},
			},
		})
	}

	if c.content != "" {
		output = append(output, map[string]interface{}{
			"id":     "msg_" + uuid.New().String(),
			"type":   "message",
			"status": "completed",
			"role":   "assistant",
			"content": []interface{}{
				map[string]interface{}{"type": "output_text", "text": c.content, "annotations": []interface{}{}},
			},
		})
	}

	for _, idx := range c.toolOrder {
		tc := c.toolCalls[idx]
		output = append(output, map[string]interface{}{
			"id":        "fc_" + uuid.New().String(),
			"type":      "function_call",
			"status":    "completed",
			"call_id":   tc.id,
			"name":      tc.name,
			"arguments": tc.args,
		})
	}

	usage := map[string]int{"input_tokens": 0, "output_tokens": 0, "total_tokens": 0}
	if c.usage != nil {
		usage["input_tokens"] = c.usage.PromptTokens
		usage["output_tokens"] = c.usage.CompletionTokens
		usage["total_tokens"] = c.usage.TotalTokens
	}

	return map[string]interface{}{
		"id":         id,
		"object":     "response",
		"created_at": time.Now().Unix(),
		"status":     "completed",
		"model":      model,
		"output":     output,
		"usage":      usage,
	}
}

// HandleResponses implements POST /v1/responses.
func (h *ResponsesHandler) HandleResponses(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeProtocolError(w, http.StatusMethodNotAllowed, "invalid_request_error", "Only POST method is allowed")
		return
	}

	var req responsesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProtocolError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON: "+err.Error())
		return
	}
	if req.Model == "" {
		writeProtocolError(w, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	if len(req.Input) == 0 {
		writeProtocolError(w, http.StatusBadRequest, "invalid_request_error", "input is required")
		return
	}

	upstream, err := req.toUpstream()
	if err != nil {
		writeProtocolError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	if req.Stream {
		h.handleStreaming(w, upstream, &req)
	} else {
		h.handleNonStreaming(w, upstream, &req)
	}
}

func (h *ResponsesHandler) handleNonStreaming(w http.ResponseWriter, upstream *proxy.ChatCompletionRequest, req *responsesRequest) {
	c, err := collect(h.proxy, upstream)
	if err != nil {
		writeProtocolError(w, http.StatusBadGateway, "api_error", "Upstream error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, responseObject("resp_"+uuid.New().String(), req.Model, c))
}

func (h *ResponsesHandler) handleStreaming(w http.ResponseWriter, upstream *proxy.ChatCompletionRequest, req *responsesRequest) {
	em, ok := newSSEEmitter(w)
	if !ok {
		writeProtocolError(w, http.StatusInternalServerError, "api_error", "Streaming not supported")
		return
	}

	respID := "resp_" + uuid.New().String()
	seq := 0
	nextSeq := func() int { seq++; return seq }

	em.emit("response.created", map[string]interface{}{
		"type":            "response.created",
		"sequence_number": nextSeq(),
		"response": map[string]interface{}{
			"id":         respID,
			"object":     "response",
			"created_at": time.Now().Unix(),
			"status":     "in_progress",
			"model":      req.Model,
			"output":     []interface{}{},
		},
	})

	// Accumulate while streaming for the final response.completed object.
	c := newCollected()

	// Output item / content part state (single message item for text, one
	// item per tool call, optional leading reasoning item).
	itemOpen := false
	itemIndex := -1
	partOpen := false
	currentItemID := ""
	toolItemMap := map[int]int{}

	closePart := func() {
		if partOpen {
			em.emit("response.content_part.done", map[string]interface{}{
				"type":            "response.content_part.done",
				"sequence_number": nextSeq(),
				"item_id":         currentItemID,
				"output_index":    itemIndex,
				"content_index":   0,
			})
			partOpen = false
		}
	}
	closeItem := func() {
		closePart()
		if itemOpen {
			em.emit("response.output_item.done", map[string]interface{}{
				"type":            "response.output_item.done",
				"sequence_number": nextSeq(),
				"output_index":    itemIndex,
			})
			itemOpen = false
		}
	}
	openTextItem := func() {
		itemIndex++
		currentItemID = "msg_" + uuid.New().String()
		em.emit("response.output_item.added", map[string]interface{}{
			"type":            "response.output_item.added",
			"sequence_number": nextSeq(),
			"output_index":    itemIndex,
			"item": map[string]interface{}{
				"id":      currentItemID,
				"type":    "message",
				"status":  "in_progress",
				"role":    "assistant",
				"content": []interface{}{},
			},
		})
		em.emit("response.content_part.added", map[string]interface{}{
			"type":            "response.content_part.added",
			"sequence_number": nextSeq(),
			"item_id":         currentItemID,
			"output_index":    itemIndex,
			"content_index":   0,
			"part":            map[string]interface{}{"type": "output_text", "text": "", "annotations": []interface{}{}},
		})
		itemOpen, partOpen = true, true
	}
	openReasoningItem := func() {
		itemIndex++
		currentItemID = "rs_" + uuid.New().String()
		em.emit("response.output_item.added", map[string]interface{}{
			"type":            "response.output_item.added",
			"sequence_number": nextSeq(),
			"output_index":    itemIndex,
			"item": map[string]interface{}{
				"id":      currentItemID,
				"type":    "reasoning",
				"status":  "in_progress",
				"summary": []interface{}{},
			},
		})
		itemOpen = true
	}

	err := h.proxy.ChatCompletion(upstream, func(evt *proxy.StreamEvent) error {
		switch evt.Type {
		case proxy.EventReasoning:
			c.reasoning += evt.Reasoning
			if !itemOpen {
				openReasoningItem()
			}
			em.emit("response.reasoning_summary_text.delta", map[string]interface{}{
				"type":            "response.reasoning_summary_text.delta",
				"sequence_number": nextSeq(),
				"item_id":         currentItemID,
				"output_index":    itemIndex,
				"summary_index":   0,
				"delta":           evt.Reasoning,
			})
		case proxy.EventText:
			c.content += evt.Text
			if !itemOpen || currentItemID == "" || !partOpen {
				closeItem()
				openTextItem()
			}
			em.emit("response.output_text.delta", map[string]interface{}{
				"type":            "response.output_text.delta",
				"sequence_number": nextSeq(),
				"item_id":         currentItemID,
				"output_index":    itemIndex,
				"content_index":   0,
				"delta":           evt.Text,
			})
		case proxy.EventToolCall:
			tc := evt.ToolCall
			acc, exists := c.toolCalls[tc.Index]
			if !exists {
				acc = &toolCallAcc{typ: "function", id: tc.ID}
				c.toolCalls[tc.Index] = acc
				c.toolOrder = append(c.toolOrder, tc.Index)
				closeItem()
				itemIndex++
				toolItemMap[tc.Index] = itemIndex
				callID := tc.ID
				if callID == "" {
					callID = "call_" + uuid.New().String()
				}
				currentItemID = "fc_" + uuid.New().String()
				em.emit("response.output_item.added", map[string]interface{}{
					"type":            "response.output_item.added",
					"sequence_number": nextSeq(),
					"output_index":    itemIndex,
					"item": map[string]interface{}{
						"id":        currentItemID,
						"type":      "function_call",
						"status":    "in_progress",
						"call_id":   callID,
						"name":      tc.Name,
						"arguments": "",
					},
				})
				itemOpen = true
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Name != "" {
				acc.name = tc.Name
			}
			acc.args += tc.Arguments
			if tc.Arguments != "" {
				em.emit("response.function_call_arguments.delta", map[string]interface{}{
					"type":            "response.function_call_arguments.delta",
					"sequence_number": nextSeq(),
					"item_id":         currentItemID,
					"output_index":    toolItemMap[tc.Index],
					"delta":           tc.Arguments,
				})
			}
		case proxy.EventFinish:
			c.finishReason = evt.FinishReason
		case proxy.EventUsage:
			c.usage = evt.Usage
		case proxy.EventError:
			return evt.Err
		}
		return nil
	})

	if err != nil {
		em.emit("response.failed", map[string]interface{}{
			"type":            "response.failed",
			"sequence_number": nextSeq(),
			"response": map[string]interface{}{
				"id":     respID,
				"object": "response",
				"status": "failed",
				"error":  map[string]string{"code": "upstream_error", "message": err.Error()},
			},
		})
		return
	}

	closeItem()

	completed := responseObject(respID, req.Model, c)
	em.emit("response.completed", map[string]interface{}{
		"type":            "response.completed",
		"sequence_number": nextSeq(),
		"response":        completed,
	})
}
