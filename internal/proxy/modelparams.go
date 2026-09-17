package proxy

import (
	"encoding/json"
	"strings"
)

// partsMessage converts a plain-text message into the upstream parts-array
// content shape (verified via live probing: string content is rejected,
// [{"type":"text","text":...}] is accepted).
func partsMessage(m Message) map[string]interface{} {
	out := map[string]interface{}{
		"role": m.Role,
		"content": []map[string]string{
			{"type": "text", "text": m.Content},
		},
	}
	if m.ToolCallID != "" {
		out["tool_call_id"] = m.ToolCallID
	}
	if len(m.ToolCalls) > 0 {
		out["tool_calls"] = m.ToolCalls
	}
	return out
}

// buildChatBody assembles the llm_raw_chat request payload in the shape the
// upstream actually accepts: {"model_name": ..., "message": <encrypted
// messages>}. It returns the pin/timestamp pair that must be echoed back in
// the X-Request-Pin / X-Requested-At headers — the timestamp is the GCM AAD,
// so header and ciphertext must carry the exact same value.
func buildChatBody(req *ChatCompletionRequest, modelName string) (body []byte, pin string, requestAt int64, err error) {
	messages := make([]map[string]interface{}, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, partsMessage(m))
	}

	// The encrypted payload is the bare messages array (verified against the
	// live backend). When tools are present they travel inside the same
	// encrypted blob as a {"messages","tools"} object.
	var plaintext []byte
	if len(req.Tools) > 0 {
		plaintext, err = json.Marshal(map[string]interface{}{
			"messages": messages,
			"tools":    req.Tools,
		})
	} else {
		plaintext, err = json.Marshal(messages)
	}
	if err != nil {
		return nil, "", 0, err
	}

	msg, pin, at, err := masticate(plaintext)
	if err != nil {
		return nil, "", 0, err
	}

	body, err = json.Marshal(map[string]interface{}{
		"model_name": modelName,
		"message":    msg,
	})
	if err != nil {
		return nil, "", 0, err
	}
	return body, pin, at, nil
}

// isSSEResponse reports whether the payload looks like an SSE stream
// (the upstream answers 200 + SSE error events even for bad requests).
func isSSEResponse(contentType, bodyPrefix string) bool {
	if strings.Contains(contentType, "text/event-stream") {
		return true
	}
	return strings.HasPrefix(bodyPrefix, "event:") || strings.HasPrefix(bodyPrefix, "data:")
}
