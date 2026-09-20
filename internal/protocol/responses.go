package protocol

import (
	"encoding/json"
	"errors"
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
	ToolChoice      json.RawMessage `json:"tool_choice,omitempty"`
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
	out := &proxy.ChatCompletionRequest{
		Stream:     r.Stream,
		ToolChoice: r.ToolChoice,
	}
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

	out.ToolChoice = r.ToolChoice

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

// responseObject builds the final Responses API object using stable item IDs.
func responseObject(id, model string, c *collected, stableIDs map[string]string) map[string]interface{} {
	var output []interface{}

	status := "completed"
	var incompleteDetails map[string]interface{}
	if c.finishReason == "length" || c.finishReason == "max_tokens" {
		status = "incomplete"
		incompleteDetails = map[string]interface{}{
			"reason": "max_output_tokens",
		}
	} else if c.finishReason == "content_filter" {
		status = "incomplete"
		incompleteDetails = map[string]interface{}{
			"reason": "content_filter",
		}
	} else if c.finishReason == "incomplete" {
		status = "incomplete"
		incompleteDetails = map[string]interface{}{
			"reason": "unknown",
		}
	}

	itemStatus := status

	if c.reasoning != "" {
		rsID := ""
		if stableIDs != nil {
			rsID = stableIDs["reasoning"]
		}
		if rsID == "" {
			rsID = "rs_" + uuid.New().String()
		}
		output = append(output, map[string]interface{}{
			"id":     rsID,
			"type":   "reasoning",
			"status": itemStatus,
			"summary": []interface{}{
				map[string]interface{}{"type": "summary_text", "text": c.reasoning},
			},
		})
	}

	if c.content != "" {
		msgID := ""
		if stableIDs != nil {
			msgID = stableIDs["message"]
		}
		if msgID == "" {
			msgID = "msg_" + uuid.New().String()
		}
		output = append(output, map[string]interface{}{
			"id":      msgID,
			"type":    "message",
			"status":  itemStatus,
			"role":    "assistant",
			"content": []interface{}{
				map[string]interface{}{"type": "output_text", "text": c.content, "annotations": []interface{}{}},
			},
		})
	}

	for i, idx := range c.toolOrder {
		tc := c.toolCalls[idx]
		fcKey := fmt.Sprintf("function_call:%d", i)
		fcID := ""
		if stableIDs != nil {
			fcID = stableIDs[fcKey]
		}
		if fcID == "" {
			fcID = "fc_" + uuid.New().String()
		}
		output = append(output, map[string]interface{}{
			"id":        fcID,
			"type":      "function_call",
			"status":    itemStatus,
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

	res := map[string]interface{}{
		"id":         id,
		"object":     "response",
		"created_at": time.Now().Unix(),
		"status":     status,
		"model":      model,
		"output":     output,
		"usage":      usage,
	}
	if incompleteDetails != nil {
		res["incomplete_details"] = incompleteDetails
	}
	return res
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
	upstream.Context = r.Context()

	// 统一非 Chat 接口工具前置守卫：针对不支持工具的渠道直接返回 400，严禁 502
	if models.ResolveChannel(upstream.ModelName) == models.ChannelAgentTask {
		if err := proxy.ValidateAgentTaskRequest(upstream); err != nil {
			if errors.Is(err, proxy.ErrAgentTaskToolsUnsupported) || strings.Contains(err.Error(), "unsupported_channel_feature") {
				writeProtocolError(w, http.StatusBadRequest, "unsupported_channel_feature", err.Error())
				return
			}
		}
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
		if errors.Is(err, proxy.ErrAgentTaskToolsUnsupported) || strings.Contains(err.Error(), "unsupported_channel_feature") {
			writeProtocolError(w, http.StatusBadRequest, "unsupported_channel_feature", err.Error())
			return
		}
		writeProtocolError(w, http.StatusBadGateway, "api_error", "Upstream error: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, responseObject("resp_"+uuid.New().String(), req.Model, c, nil))
}

type streamItemMeta struct {
	id          string
	typ         string // "reasoning", "message"
	role        string
	callID      string
	name        string
	outputIndex int
	textBuffer  strings.Builder
}

type toolItemStreamMeta struct {
	fcID         string
	callID       string
	name         string
	outputIndex  int
	textBuffer   strings.Builder
	addedEmitted bool
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
	stableIDs := make(map[string]string)

	itemOpen := false
	itemIndex := -1
	partOpen := false
	var currentItem *streamItemMeta
	toolStreamMap := make(map[int]*toolItemStreamMeta)
	var toolStreamOrder []int

	closePart := func() {
		if partOpen && currentItem != nil && currentItem.typ == "message" {
			partObj := map[string]interface{}{
				"type":        "output_text",
				"text":        currentItem.textBuffer.String(),
				"annotations": []interface{}{},
			}
			em.emit("response.content_part.done", map[string]interface{}{
				"type":            "response.content_part.done",
				"sequence_number": nextSeq(),
				"response_id":     respID,
				"item_id":         currentItem.id,
				"output_index":    currentItem.outputIndex,
				"content_index":   0,
				"part":            partObj,
			})
			partOpen = false
		}
	}

	closeItem := func() {
		closePart()
		if itemOpen && currentItem != nil {
			itemStatus := "completed"
			var incompleteDetails map[string]interface{}
			if c.finishReason == "length" || c.finishReason == "max_tokens" {
				itemStatus = "incomplete"
				incompleteDetails = map[string]interface{}{
					"reason": "max_output_tokens",
				}
			} else if c.finishReason == "content_filter" {
				itemStatus = "incomplete"
				incompleteDetails = map[string]interface{}{
					"reason": "content_filter",
				}
			} else if c.finishReason == "incomplete" {
				itemStatus = "incomplete"
				incompleteDetails = map[string]interface{}{
					"reason": "unknown",
				}
			}

			itemObj := map[string]interface{}{
				"id":     currentItem.id,
				"type":   currentItem.typ,
				"status": itemStatus,
			}
			if incompleteDetails != nil {
				itemObj["incomplete_details"] = incompleteDetails
			}
			switch currentItem.typ {
			case "reasoning":
				itemObj["summary"] = []interface{}{
					map[string]interface{}{
						"type": "summary_text",
						"text": currentItem.textBuffer.String(),
					},
				}
			case "message":
				itemObj["role"] = currentItem.role
				itemObj["content"] = []interface{}{
					map[string]interface{}{
						"type":        "output_text",
						"text":        currentItem.textBuffer.String(),
						"annotations": []interface{}{},
					},
				}
			}

			em.emit("response.output_item.done", map[string]interface{}{
				"type":            "response.output_item.done",
				"sequence_number": nextSeq(),
				"response_id":     respID,
				"output_index":    currentItem.outputIndex,
				"item":            itemObj,
			})
			itemOpen = false
			currentItem = nil
		}
	}

	openTextItem := func() {
		itemIndex++
		msgID := "msg_" + uuid.New().String()
		stableIDs["message"] = msgID
		currentItem = &streamItemMeta{
			id:          msgID,
			typ:         "message",
			role:        "assistant",
			outputIndex: itemIndex,
		}
		em.emit("response.output_item.added", map[string]interface{}{
			"type":            "response.output_item.added",
			"sequence_number": nextSeq(),
			"response_id":     respID,
			"output_index":    itemIndex,
			"item": map[string]interface{}{
				"id":      msgID,
				"type":    "message",
				"status":  "in_progress",
				"role":    "assistant",
				"content": []interface{}{},
			},
		})
		em.emit("response.content_part.added", map[string]interface{}{
			"type":            "response.content_part.added",
			"sequence_number": nextSeq(),
			"response_id":     respID,
			"item_id":         msgID,
			"output_index":    itemIndex,
			"content_index":   0,
			"part":            map[string]interface{}{"type": "output_text", "text": "", "annotations": []interface{}{}},
		})
		itemOpen, partOpen = true, true
	}

	openReasoningItem := func() {
		itemIndex++
		rsID := "rs_" + uuid.New().String()
		stableIDs["reasoning"] = rsID
		currentItem = &streamItemMeta{
			id:          rsID,
			typ:         "reasoning",
			outputIndex: itemIndex,
		}
		em.emit("response.output_item.added", map[string]interface{}{
			"type":            "response.output_item.added",
			"sequence_number": nextSeq(),
			"response_id":     respID,
			"output_index":    itemIndex,
			"item": map[string]interface{}{
				"id":      rsID,
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
			if !itemOpen || currentItem == nil || currentItem.typ != "reasoning" {
				closeItem()
				openReasoningItem()
			}
			if currentItem != nil {
				currentItem.textBuffer.WriteString(evt.Reasoning)
			}
			em.emit("response.reasoning_summary_text.delta", map[string]interface{}{
				"type":            "response.reasoning_summary_text.delta",
				"sequence_number": nextSeq(),
				"response_id":     respID,
				"item_id":         currentItem.id,
				"output_index":    currentItem.outputIndex,
				"summary_index":   0,
				"delta":           evt.Reasoning,
			})
		case proxy.EventText:
			c.content += evt.Text
			if !itemOpen || currentItem == nil || currentItem.typ != "message" || !partOpen {
				closeItem()
				openTextItem()
			}
			if currentItem != nil {
				currentItem.textBuffer.WriteString(evt.Text)
			}
			em.emit("response.output_text.delta", map[string]interface{}{
				"type":            "response.output_text.delta",
				"sequence_number": nextSeq(),
				"response_id":     respID,
				"item_id":         currentItem.id,
				"output_index":    currentItem.outputIndex,
				"content_index":   0,
				"delta":           evt.Text,
			})
		case proxy.EventToolCall:
			tc := evt.ToolCall

			// 若此前正在输出文本或思考项，将其安全闭合；绝不关闭其他正在并行的工具！
			if itemOpen && currentItem != nil {
				closeItem()
			}

			toolMeta, exists := toolStreamMap[tc.Index]
			if !exists {
				itemIndex++
				callID := tc.ID
				if callID == "" {
					callID = "call_" + uuid.New().String()
				}
				fcID := "fc_" + uuid.New().String()
				toolMeta = &toolItemStreamMeta{
					fcID:        fcID,
					callID:      callID,
					name:        tc.Name,
					outputIndex: itemIndex,
				}
				toolStreamMap[tc.Index] = toolMeta
				toolStreamOrder = append(toolStreamOrder, tc.Index)

				fcKey := fmt.Sprintf("function_call:%d", len(toolStreamOrder)-1)
				stableIDs[fcKey] = fcID

				em.emit("response.output_item.added", map[string]interface{}{
					"type":            "response.output_item.added",
					"sequence_number": nextSeq(),
					"response_id":     respID,
					"output_index":    toolMeta.outputIndex,
					"item": map[string]interface{}{
						"id":        fcID,
						"type":      "function_call",
						"status":    "in_progress",
						"call_id":   callID,
						"name":      tc.Name,
						"arguments": "",
					},
				})
				toolMeta.addedEmitted = true
			}

			acc, existsAcc := c.toolCalls[tc.Index]
			if !existsAcc {
				acc = &toolCallAcc{typ: "function", id: toolMeta.callID, name: tc.Name}
				c.toolCalls[tc.Index] = acc
				c.toolOrder = append(c.toolOrder, tc.Index)
			}
			if tc.ID != "" {
				acc.id = tc.ID
				toolMeta.callID = tc.ID
			}
			if tc.Name != "" {
				acc.name = tc.Name
				toolMeta.name = tc.Name
			}
			acc.args += tc.Arguments
			if tc.Arguments != "" {
				toolMeta.textBuffer.WriteString(tc.Arguments)
				em.emit("response.function_call_arguments.delta", map[string]interface{}{
					"type":            "response.function_call_arguments.delta",
					"sequence_number": nextSeq(),
					"response_id":     respID,
					"item_id":         toolMeta.fcID,
					"output_index":    toolMeta.outputIndex,
					"delta":           tc.Arguments,
				})
			}
		case proxy.EventFinish:
			if c.finishReason == "" || (evt.FinishReason != "stop" && evt.FinishReason != "") {
				c.finishReason = evt.FinishReason
			}
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

	// 按顺序为每一个并行的工具调用发射 output_item.done
	for _, idx := range toolStreamOrder {
		meta := toolStreamMap[idx]
		itemStatus := "completed"
		if c.finishReason == "length" || c.finishReason == "max_tokens" || c.finishReason == "incomplete" {
			itemStatus = "incomplete"
		}
		em.emit("response.output_item.done", map[string]interface{}{
			"type":            "response.output_item.done",
			"sequence_number": nextSeq(),
			"response_id":     respID,
			"output_index":    meta.outputIndex,
			"item": map[string]interface{}{
				"id":        meta.fcID,
				"type":      "function_call",
				"status":    itemStatus,
				"call_id":   meta.callID,
				"name":      meta.name,
				"arguments": meta.textBuffer.String(),
			},
		})
	}

	completed := responseObject(respID, req.Model, c, stableIDs)
	status, _ := completed["status"].(string)
	em.emit("response.done", map[string]interface{}{
		"type":            "response.done",
		"sequence_number": nextSeq(),
		"response":        completed,
	})
	if status == "completed" {
		em.emit("response.completed", map[string]interface{}{
			"type":            "response.completed",
			"sequence_number": nextSeq(),
			"response":        completed,
		})
	} else if status == "incomplete" {
		em.emit("response.incomplete", map[string]interface{}{
			"type":            "response.incomplete",
			"sequence_number": nextSeq(),
			"response":        completed,
		})
	}
}
