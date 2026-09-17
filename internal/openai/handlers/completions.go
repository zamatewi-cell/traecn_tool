package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/zamatewi-cell/traecn_tool/internal/transformers"
)

// CompletionsHandler handles legacy completions requests
type CompletionsHandler struct {
	// Similar to ChatHandler but for /v1/completions endpoint
	// This is a legacy endpoint, less commonly used
}

// NewCompletionsHandler creates a new completions handler
func NewCompletionsHandler() *CompletionsHandler {
	return &CompletionsHandler{}
}

// HandleCompletions handles POST /v1/completions
func (h *CompletionsHandler) HandleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	// Parse request
	var req struct {
		Model       string   `json:"model"`
		Prompt      string   `json:"prompt"`
		MaxTokens   int      `json:"max_tokens,omitempty"`
		Temperature float64  `json:"temperature,omitempty"`
		Stream      bool     `json:"stream,omitempty"`
		Stop        []string `json:"stop,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON: "+err.Error())
		return
	}

	if req.Model == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	// For now, return a not implemented error
	// This endpoint can be implemented later if needed
	h.writeError(w, http.StatusNotImplemented, "not_implemented", "/v1/completions endpoint is not implemented. Please use /v1/chat/completions")
}

func (h *CompletionsHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *CompletionsHandler) writeError(w http.ResponseWriter, status int, errType string, message string) {
	resp := transformers.ErrorResponse{
		Error: transformers.ErrorDetail{
			Message: message,
			Type:    transformers.ErrorCode(errType),
		},
	}
	h.writeJSON(w, status, resp)
}
