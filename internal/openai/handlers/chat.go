package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/db"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
	"github.com/zamatewi-cell/traecn_tool/internal/transformers"
)

// ChatHandler handles chat completion requests
type ChatHandler struct {
	proxy               *proxy.TraeProxy
	requestTransformer  *transformers.RequestTransformer
	responseTransformer *transformers.ResponseTransformer
	errorMapper         *transformers.ErrorMapper
}

// NewChatHandler creates a new chat handler
func NewChatHandler(p *proxy.TraeProxy) *ChatHandler {
	return &ChatHandler{
		proxy:               p,
		requestTransformer:  transformers.NewRequestTransformer(),
		responseTransformer: transformers.NewResponseTransformer(),
		errorMapper:         transformers.NewErrorMapper(),
	}
}

// HandleChatCompletions handles POST /v1/chat/completions
func (h *ChatHandler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	var openaiReq transformers.OpenAIRequest
	if err := json.NewDecoder(r.Body).Decode(&openaiReq); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON: "+err.Error())
		return
	}

	if openaiReq.Model == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	if len(openaiReq.Messages) == 0 {
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", "messages is required")
		return
	}

	upstreamReq, err := h.requestTransformer.Transform(&openaiReq)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", "Failed to transform request: "+err.Error())
		return
	}

	if openaiReq.Stream {
		h.handleStreaming(w, r, upstreamReq, openaiReq.Model)
	} else {
		h.handleNonStreaming(w, r, upstreamReq, openaiReq.Model)
	}
}

// streamState accumulates normalized events for rendering.
type streamState struct {
	chunkID   string
	created   int64
	finished  bool
	finishWhy string
	usage     *proxy.Usage
}

func newStreamState() *streamState {
	return &streamState{
		chunkID:   transformers.NewChunkID(),
		created:   time.Now().Unix(),
		finishWhy: "stop",
	}
}

func (h *ChatHandler) handleNonStreaming(w http.ResponseWriter, r *http.Request, upstreamReq *proxy.ChatCompletionRequest, model string) {
	startTime := time.Now()
	state := newStreamState()
	var content, reasoning string
	toolCalls := map[int]*transformers.OpenAIToolCall{}
	toolOrder := []int{}

	err := h.proxy.ChatCompletion(upstreamReq, func(evt *proxy.StreamEvent) error {
		switch evt.Type {
		case proxy.EventText:
			content += evt.Text
		case proxy.EventReasoning:
			reasoning += evt.Reasoning
		case proxy.EventToolCall:
			tc := evt.ToolCall
			acc, ok := toolCalls[tc.Index]
			if !ok {
				acc = &transformers.OpenAIToolCall{Type: "function"}
				toolCalls[tc.Index] = acc
				toolOrder = append(toolOrder, tc.Index)
			}
			if tc.ID != "" {
				acc.ID = tc.ID
			}
			if tc.Type != "" {
				acc.Type = tc.Type
			}
			if tc.Name != "" {
				acc.Function.Name = tc.Name
			}
			acc.Function.Arguments += tc.Arguments
		case proxy.EventFinish:
			state.finished = true
			state.finishWhy = evt.FinishReason
		case proxy.EventUsage:
			state.usage = evt.Usage
		case proxy.EventError:
			return evt.Err
		}
		return nil
	})

	elapsedSec := time.Since(startTime).Seconds()
	var tps float64
	var promptTokens, compTokens, totalTokens, cachedTokens int64
	if state.usage != nil {
		promptTokens = int64(state.usage.PromptTokens)
		compTokens = int64(state.usage.CompletionTokens)
		totalTokens = int64(state.usage.TotalTokens)
		cachedTokens = int64(state.usage.CachedTokens)
		if elapsedSec > 0 && compTokens > 0 {
			tps = float64(compTokens) / elapsedSec
		}
	}

	statusCode := http.StatusOK
	var errMsg string
	if err != nil {
		if errors.Is(err, proxy.ErrAgentTaskToolsUnsupported) {
			statusCode = http.StatusBadRequest
		} else {
			statusCode = http.StatusBadGateway
		}
		errMsg = err.Error()
	}

	db.GetGlobalStore().RecordLog(&db.LogRecord{
		TraceID:          state.chunkID,
		Model:            model,
		ClientIP:         r.RemoteAddr,
		PromptTokens:     promptTokens,
		CompletionTokens: compTokens,
		TotalTokens:      totalTokens,
		TTFTMs:           0, // 非流式请求不伪造 TTFT，置 0 避免污染性能大盘
		TokensPerSecond:  tps,
		CachedTokens:     cachedTokens,
		StatusCode:       statusCode,
		ErrorMsg:         errMsg,
		CreatedAt:        time.Now().Unix(),
	})

	if err != nil {
		if errors.Is(err, proxy.ErrAgentTaskToolsUnsupported) {
			h.writeError(w, http.StatusBadRequest, "unsupported_channel_feature", err.Error())
			return
		}
		h.writeError(w, http.StatusBadGateway, "upstream_error", "Proxy error: "+err.Error())
		return
	}

	msg := transformers.Message{
		Role:             "assistant",
		Content:          content,
		ReasoningContent: reasoning,
	}
	for _, idx := range toolOrder {
		msg.ToolCalls = append(msg.ToolCalls, *toolCalls[idx])
	}
	finishReason := state.finishWhy
	if len(msg.ToolCalls) > 0 && finishReason == "stop" {
		finishReason = "tool_calls"
	}

	resp := transformers.OpenAIResponse{
		ID:      state.chunkID,
		Object:  "chat.completion",
		Created: state.created,
		Model:   model,
		Choices: []transformers.Choice{
			{Index: 0, Message: msg, FinishReason: &finishReason},
		},
	}
	if state.usage != nil {
		resp.Usage = &transformers.Usage{
			PromptTokens:     state.usage.PromptTokens,
			CompletionTokens: state.usage.CompletionTokens,
			TotalTokens:      state.usage.TotalTokens,
		}
		if state.usage.CachedTokens > 0 {
			resp.Usage.PromptTokensDetails = &transformers.PromptTokensDetails{
				CachedTokens: state.usage.CachedTokens,
			}
		}
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *ChatHandler) handleStreaming(w http.ResponseWriter, r *http.Request, upstreamReq *proxy.ChatCompletionRequest, model string) {
	startTime := time.Now()
	var ttftMs int64

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Streaming not supported")
		return
	}

	state := newStreamState()
	var finishSent bool

	recordCompletion := func(err error) {
		elapsedSec := time.Since(startTime).Seconds()
		var tps float64
		var promptTokens, compTokens, totalTokens, cachedTokens int64
		if state.usage != nil {
			promptTokens = int64(state.usage.PromptTokens)
			compTokens = int64(state.usage.CompletionTokens)
			totalTokens = int64(state.usage.TotalTokens)
			cachedTokens = int64(state.usage.CachedTokens)
			if elapsedSec > 0 && compTokens > 0 {
				tps = float64(compTokens) / elapsedSec
			}
		}

		statusCode := http.StatusOK
		var errMsg string
		if err != nil {
			if errors.Is(err, proxy.ErrAgentTaskToolsUnsupported) {
				statusCode = http.StatusBadRequest
			} else {
				statusCode = http.StatusBadGateway
			}
			errMsg = err.Error()
		}

		db.GetGlobalStore().RecordLog(&db.LogRecord{
			TraceID:          state.chunkID,
			Model:            model,
			ClientIP:         r.RemoteAddr,
			PromptTokens:     promptTokens,
			CompletionTokens: compTokens,
			TotalTokens:      totalTokens,
			TTFTMs:           ttftMs,
			TokensPerSecond:  tps,
			CachedTokens:     cachedTokens,
			StatusCode:       statusCode,
			ErrorMsg:         errMsg,
			CreatedAt:        time.Now().Unix(),
		})
	}

	sendChunk := func(delta *transformers.Delta, finishReason *string) {
		if finishReason != nil {
			finishSent = true
		}
		resp := transformers.OpenAIResponse{
			ID:      state.chunkID,
			Object:  "chat.completion.chunk",
			Created: state.created,
			Model:   model,
			Choices: []transformers.Choice{{Index: 0, Delta: delta, FinishReason: finishReason}},
		}
		if state.usage != nil {
			resp.Usage = &transformers.Usage{
				PromptTokens:     state.usage.PromptTokens,
				CompletionTokens: state.usage.CompletionTokens,
				TotalTokens:      state.usage.TotalTokens,
			}
			if state.usage.CachedTokens > 0 {
				resp.Usage.PromptTokensDetails = &transformers.PromptTokensDetails{
					CachedTokens: state.usage.CachedTokens,
				}
			}
		}
		h.sendSSE(w, flusher, resp)
	}

	// Initial role chunk.
	role := "assistant"
	sendChunk(&transformers.Delta{Role: &role}, nil)

	err := h.proxy.ChatCompletion(upstreamReq, func(evt *proxy.StreamEvent) error {
		switch evt.Type {
		case proxy.EventText:
			if ttftMs == 0 {
				ttftMs = time.Since(startTime).Milliseconds()
			}
			text := evt.Text
			sendChunk(&transformers.Delta{Content: &text}, nil)
		case proxy.EventReasoning:
			if ttftMs == 0 {
				ttftMs = time.Since(startTime).Milliseconds()
			}
			reasoning := evt.Reasoning
			sendChunk(&transformers.Delta{ReasoningContent: &reasoning}, nil)
		case proxy.EventToolCall:
			if ttftMs == 0 {
				ttftMs = time.Since(startTime).Milliseconds()
			}
			tc := evt.ToolCall
			dtc := transformers.DeltaToolCall{Index: tc.Index}
			if tc.ID != "" {
				dtc.ID = tc.ID
			}
			if tc.Type != "" {
				dtc.Type = tc.Type
			}
			if tc.Name != "" || tc.Arguments != "" {
				dtc.Function = &transformers.DeltaFunction{Name: tc.Name, Arguments: tc.Arguments}
			}
			sendChunk(&transformers.Delta{ToolCalls: []transformers.DeltaToolCall{dtc}}, nil)
		case proxy.EventQueue, proxy.EventTaskCreated:
			// Internal signals: surfaced via /v1/queue/status, not to clients.
		case proxy.EventFinish:
			state.finished = true
			state.finishWhy = evt.FinishReason
			if !finishSent {
				sendChunk(&transformers.Delta{}, &state.finishWhy)
			}
		case proxy.EventUsage:
			state.usage = evt.Usage
			if state.finished && !finishSent {
				sendChunk(&transformers.Delta{}, &state.finishWhy)
			} else if finishSent {
				// 已发送过 finishReason，追加推送带有 Usage 信息的 chunk（choices 不再重复 finishReason）
				sendChunk(&transformers.Delta{}, nil)
			}
		case proxy.EventError:
			return evt.Err
		}
		return nil
	})

	recordCompletion(err)

	if err != nil {
		errType := "upstream_error"
		if errors.Is(err, proxy.ErrAgentTaskToolsUnsupported) {
			errType = "unsupported_channel_feature"
		}
		errorResp := transformers.ErrorResponse{
			Error: transformers.ErrorDetail{
				Message: "Stream error: " + err.Error(),
				Type:    transformers.ErrorCode(errType),
			},
		}
		h.sendSSE(w, flusher, errorResp)
		h.sendDone(w, flusher)
		return
	}

	if !finishSent {
		sendChunk(&transformers.Delta{}, &state.finishWhy)
	}
	h.sendDone(w, flusher)
}

func (h *ChatHandler) sendSSE(w http.ResponseWriter, flusher http.Flusher, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	_, err = w.Write([]byte("data: " + string(jsonData) + "\n\n"))
	if err != nil {
		return
	}
	flusher.Flush()
}

func (h *ChatHandler) sendDone(w http.ResponseWriter, flusher http.Flusher) {
	_, err := w.Write([]byte("data: [DONE]\n\n"))
	if err != nil {
		return
	}
	flusher.Flush()
}

func (h *ChatHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *ChatHandler) writeError(w http.ResponseWriter, status int, errType string, message string) {
	resp := transformers.ErrorResponse{
		Error: transformers.ErrorDetail{
			Message: message,
			Type:    transformers.ErrorCode(errType),
		},
	}
	h.writeJSON(w, status, resp)
}
