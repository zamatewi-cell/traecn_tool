// Package protocol implements the client-facing protocol adapters
// (Anthropic Messages API, OpenAI Responses API) on top of the normalized
// upstream event stream.
package protocol

import (
	"encoding/json"
	"net/http"

	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

// collected accumulates a full response from the normalized event stream
// (shared by the non-streaming paths of all protocol adapters).
type collected struct {
	content      string
	reasoning    string
	toolCalls    map[int]*toolCallAcc
	toolOrder    []int
	finishReason string
	usage        *proxy.Usage
}

type toolCallAcc struct {
	id   string
	typ  string
	name string
	args string
}

func newCollected() *collected {
	return &collected{toolCalls: make(map[int]*toolCallAcc), finishReason: ""}
}

// collect drains a chat completion into a collected response.
func collect(p *proxy.TraeProxy, req *proxy.ChatCompletionRequest) (*collected, error) {
	c := newCollected()
	err := p.ChatCompletion(req, func(evt *proxy.StreamEvent) error {
		switch evt.Type {
		case proxy.EventText:
			c.content += evt.Text
		case proxy.EventReasoning:
			c.reasoning += evt.Reasoning
		case proxy.EventToolCall:
			tc := evt.ToolCall
			acc, ok := c.toolCalls[tc.Index]
			if !ok {
				acc = &toolCallAcc{typ: "function"}
				c.toolCalls[tc.Index] = acc
				c.toolOrder = append(c.toolOrder, tc.Index)
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Type != "" {
				acc.typ = tc.Type
			}
			if tc.Name != "" {
				acc.name = tc.Name
			}
			acc.args += tc.Arguments
		case proxy.EventFinish:
			c.finishReason = evt.FinishReason
		case proxy.EventUsage:
			c.usage = evt.Usage
		case proxy.EventError:
			return evt.Err
		}
		return nil
	})
	return c, err
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeProtocolError(w http.ResponseWriter, status int, errType, message string) {
	writeJSON(w, status, map[string]interface{}{
		"type": "error",
		"error": map[string]string{
			"type":    errType,
			"message": message,
		},
	})
}

// sseEmitter writes named SSE events with flushing.
type sseEmitter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newSSEEmitter(w http.ResponseWriter) (*sseEmitter, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	return &sseEmitter{w: w, flusher: flusher}, true
}

// emit sends one named event with a JSON payload.
func (e *sseEmitter) emit(event string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	e.w.Write([]byte("event: " + event + "\ndata: " + string(data) + "\n\n"))
	e.flusher.Flush()
}
