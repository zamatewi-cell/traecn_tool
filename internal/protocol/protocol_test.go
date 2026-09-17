package protocol

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// withUpstream runs f with a mock upstream; the captured request body is
// returned for assertions.
func withUpstream(t *testing.T, upstream http.HandlerFunc, f func(p *proxy.TraeProxy, gotBody *map[string]interface{})) {
	t.Helper()
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		decryptUpstreamBody(r, gotBody)
		upstream(w, r)
	}))
	defer srv.Close()

	old := config.AgentDomain
	config.AgentDomain = srv.URL
	defer func() { config.AgentDomain = old }()

	tokens := auth.NewTokenProvider()
	tokens.AddAccountWithToken("acc", "tok")
	f(proxy.NewTraeProxy(tokens, testLogger()), &gotBody)
}


// decryptUpstreamBody replaces the encrypted "message" field with its
// decrypted contents ("messages" and optional "tools") so test assertions
// can inspect what the proxy actually sent upstream.
func decryptUpstreamBody(r *http.Request, body map[string]interface{}) {
	msg, _ := body["message"].(string)
	if msg == "" {
		return
	}
	at, _ := strconv.ParseInt(r.Header.Get("x-requested-at"), 10, 64)
	plain, err := proxy.Demasticate(msg, r.Header.Get("x-request-pin"), at)
	if err != nil {
		return
	}
	var arr []interface{}
	if json.Unmarshal(plain, &arr) == nil {
		body["messages"] = arr
		return
	}
	var obj map[string]interface{}
	if json.Unmarshal(plain, &obj) == nil {
		for k, v := range obj {
			body[k] = v
		}
	}
}

func firstPartText(m map[string]interface{}) string {
	parts, ok := m["content"].([]interface{})
	if !ok || len(parts) == 0 {
		return ""
	}
	p, _ := parts[0].(map[string]interface{})
	s, _ := p["text"].(string)
	return s
}

func sseUpstream(events ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, e := range events {
			io.WriteString(w, "data: "+e+"\n\n")
		}
		io.WriteString(w, "data: [DONE]\n\n")
	}
}

// parseSSEEvents parses an SSE body into (event, data-json) pairs.
func parseSSEEvents(t *testing.T, body string) []map[string]interface{} {
	t.Helper()
	var out []map[string]interface{}
	var event string
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "event: ") {
			event = strings.TrimPrefix(line, "event: ")
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &m); err == nil {
				if event != "" {
					m["__event"] = event
				}
				out = append(out, m)
			}
			event = ""
		}
	}
	return out
}

func eventTypes(events []map[string]interface{}) []string {
	var out []string
	for _, e := range events {
		if t, ok := e["type"].(string); ok {
			out = append(out, t)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Anthropic Messages API
// ---------------------------------------------------------------------------

func TestAnthropic_NonStream(t *testing.T) {
	withUpstream(t, sseUpstream(
		`{"choices":[{"delta":{"content":"<think>hmm</think>The answer is 42."}}]}`,
		`{"choices":[{"finish_reason":"stop"}]}`,
		`{"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
	), func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewAnthropicHandler(p)
		req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{
		  "model": "claude-fake",
		  "max_tokens": 100,
		  "system": "You are helpful.",
		  "messages": [{"role":"user","content":"What is the answer?"}]
		}`))
		rec := httptest.NewRecorder()
		h.HandleMessages(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp["type"] != "message" || resp["role"] != "assistant" {
			t.Errorf("bad response envelope: %v", resp)
		}
		if !strings.HasPrefix(resp["id"].(string), "msg_") {
			t.Errorf("id = %v", resp["id"])
		}
		if resp["stop_reason"] != "end_turn" {
			t.Errorf("stop_reason = %v", resp["stop_reason"])
		}
		blocks := resp["content"].([]interface{})
		if len(blocks) != 2 {
			t.Fatalf("content blocks = %v", blocks)
		}
		b0 := blocks[0].(map[string]interface{})
		b1 := blocks[1].(map[string]interface{})
		if b0["type"] != "thinking" || b0["thinking"] != "hmm" {
			t.Errorf("thinking block = %v", b0)
		}
		if b1["type"] != "text" || b1["text"] != "The answer is 42." {
			t.Errorf("text block = %v", b1)
		}
		usage := resp["usage"].(map[string]interface{})
		if usage["input_tokens"].(float64) != 10 {
			t.Errorf("usage = %v", usage)
		}
	})
}

func TestAnthropic_RequestConversion(t *testing.T) {
	withUpstream(t, sseUpstream(`{"choices":[{"delta":{"content":"ok"}}]}`),
		func(p *proxy.TraeProxy, gotBody *map[string]interface{}) {
			h := NewAnthropicHandler(p)
			req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{
			  "model": "kimi",
			  "max_tokens": 100,
			  "system": [{"type":"text","text":"sys prompt"}],
			  "messages": [
			    {"role":"user","content":[{"type":"text","text":"read a.go"}]},
			    {"role":"assistant","content":[
			      {"type":"text","text":"Let me read it."},
			      {"type":"tool_use","id":"toolu_1","name":"read_file","input":{"path":"a.go"}}
			    ]},
			    {"role":"user","content":[
			      {"type":"tool_result","tool_use_id":"toolu_1","content":"package main"}
			    ]}
			  ],
			  "tools": [{"name":"read_file","description":"read","input_schema":{"type":"object","properties":{"path":{"type":"string"}}}}]
			}`))
			rec := httptest.NewRecorder()
			h.HandleMessages(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}

			// kimi -> Kimi-K3 (no verified UpstreamID) -> default preset seed_m8
			if (*gotBody)["model_name"] != "seed_m8" {
				t.Errorf("model_name = %v", (*gotBody)["model_name"])
			}
			msgs := (*gotBody)["messages"].([]interface{})
			// system + user + assistant(with tool_calls) + tool
			if len(msgs) != 4 {
				t.Fatalf("messages = %v", (*gotBody)["messages"])
			}
			if msgs[0].(map[string]interface{})["role"] != "system" {
				t.Errorf("first message = %v", msgs[0])
			}
			assistant := msgs[2].(map[string]interface{})
			tcs := assistant["tool_calls"].([]interface{})
			if len(tcs) != 1 {
				t.Fatalf("assistant tool_calls = %v", assistant)
			}
			tc := tcs[0].(map[string]interface{})
			fn := tc["function"].(map[string]interface{})
			if tc["id"] != "toolu_1" || fn["name"] != "read_file" {
				t.Errorf("tool_call = %v", tc)
			}
			if !strings.Contains(fn["arguments"].(string), "a.go") {
				t.Errorf("arguments = %v", fn["arguments"])
			}
			toolMsg := msgs[3].(map[string]interface{})
			if toolMsg["role"] != "tool" || toolMsg["tool_call_id"] != "toolu_1" || firstPartText(toolMsg) != "package main" {
				t.Errorf("tool result message = %v", toolMsg)
			}
			tools := (*gotBody)["tools"].([]interface{})
			if len(tools) != 1 {
				t.Fatalf("tools = %v", tools)
			}
		})
}

func TestAnthropic_Streaming(t *testing.T) {
	withUpstream(t, sseUpstream(
		`{"choices":[{"delta":{"reasoning_content":"thinking..."}}]}`,
		`{"choices":[{"delta":{"content":"Hello"}}]}`,
		`{"choices":[{"delta":{"content":" world"}}]}`,
		`{"choices":[{"finish_reason":"stop"}]}`,
	), func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewAnthropicHandler(p)
		req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{
		  "model": "kimi", "max_tokens": 100, "stream": true,
		  "messages": [{"role":"user","content":"hi"}]
		}`))
		rec := httptest.NewRecorder()
		h.HandleMessages(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
			t.Errorf("Content-Type = %v", ct)
		}

		events := parseSSEEvents(t, rec.Body.String())
		types := eventTypes(events)
		want := []string{
			"message_start",
			"content_block_start", "content_block_delta", "content_block_stop", // thinking
			"content_block_start", "content_block_delta", "content_block_delta", "content_block_stop", // text
			"message_delta", "message_stop",
		}
		if len(types) != len(want) {
			t.Fatalf("event sequence = %v", types)
		}
		for i := range want {
			if types[i] != want[i] {
				t.Fatalf("event %d = %v, want %v (full: %v)", i, types[i], want[i], types)
			}
		}

		// First delta must be a thinking delta.
		firstDelta := events[2]["delta"].(map[string]interface{})
		if firstDelta["type"] != "thinking_delta" || firstDelta["thinking"] != "thinking..." {
			t.Errorf("first delta = %v", firstDelta)
		}
		// Text deltas concatenate to the full reply.
		var text string
		for _, e := range events {
			if e["type"] == "content_block_delta" {
				d := e["delta"].(map[string]interface{})
				if d["type"] == "text_delta" {
					text += d["text"].(string)
				}
			}
		}
		if text != "Hello world" {
			t.Errorf("text = %q", text)
		}
		// message_delta carries stop_reason end_turn.
		md := events[len(events)-2]["delta"].(map[string]interface{})
		if md["stop_reason"] != "end_turn" {
			t.Errorf("message_delta = %v", md)
		}
	})
}

func TestAnthropic_StreamingToolUse(t *testing.T) {
	withUpstream(t, sseUpstream(
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":""}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\""}}]}}]}`,
		`{"choices":[{"finish_reason":"tool_calls"}]}`,
	), func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewAnthropicHandler(p)
		req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{
		  "model": "kimi", "max_tokens": 100, "stream": true,
		  "messages": [{"role":"user","content":"read a.go"}],
		  "tools": [{"name":"read_file","input_schema":{"type":"object"}}]
		}`))
		rec := httptest.NewRecorder()
		h.HandleMessages(rec, req)

		events := parseSSEEvents(t, rec.Body.String())
		var sawToolUseStart, sawJSONDelta bool
		var stopReason string
		for _, e := range events {
			switch e["type"] {
			case "content_block_start":
				cb := e["content_block"].(map[string]interface{})
				if cb["type"] == "tool_use" {
					sawToolUseStart = true
					if cb["name"] != "read_file" {
						t.Errorf("tool_use block = %v", cb)
					}
				}
			case "content_block_delta":
				d := e["delta"].(map[string]interface{})
				if d["type"] == "input_json_delta" {
					sawJSONDelta = true
				}
			case "message_delta":
				stopReason = e["delta"].(map[string]interface{})["stop_reason"].(string)
			}
		}
		if !sawToolUseStart || !sawJSONDelta {
			t.Errorf("tool_use stream blocks missing (start=%v delta=%v): %v", sawToolUseStart, sawJSONDelta, eventTypes(events))
		}
		if stopReason != "tool_use" {
			t.Errorf("stop_reason = %q, want tool_use", stopReason)
		}
	})
}

// ---------------------------------------------------------------------------
// OpenAI Responses API
// ---------------------------------------------------------------------------

func TestResponses_NonStream(t *testing.T) {
	withUpstream(t, sseUpstream(
		`{"choices":[{"delta":{"content":"pong"}}]}`,
		`{"choices":[{"finish_reason":"stop"}]}`,
	), func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewResponsesHandler(p)
		req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{
		  "model": "deepseek-v4", "input": "ping"
		}`))
		rec := httptest.NewRecorder()
		h.HandleResponses(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["object"] != "response" || resp["status"] != "completed" {
			t.Errorf("envelope = %v", resp)
		}
		output := resp["output"].([]interface{})
		msg := output[0].(map[string]interface{})
		content := msg["content"].([]interface{})[0].(map[string]interface{})
		if content["type"] != "output_text" || content["text"] != "pong" {
			t.Errorf("output = %v", output)
		}
	})
}

func TestResponses_InputItemsConversion(t *testing.T) {
	withUpstream(t, sseUpstream(`{"choices":[{"delta":{"content":"ok"}}]}`),
		func(p *proxy.TraeProxy, gotBody *map[string]interface{}) {
			h := NewResponsesHandler(p)
			req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{
			  "model": "qwen",
			  "instructions": "be terse",
			  "input": [
			    {"type":"message","role":"user","content":[{"type":"input_text","text":"run ls"}]},
			    {"type":"function_call","call_id":"call_1","name":"shell","arguments":"{\"cmd\":\"ls\"}"},
			    {"type":"function_call_output","call_id":"call_1","output":"a.go b.go"}
			  ]
			}`))
			rec := httptest.NewRecorder()
			h.HandleResponses(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}

			// qwen has no preset upstream model -> default fallback seed_m8
			if (*gotBody)["model_name"] != "seed_m8" {
				t.Errorf("model_name = %v", (*gotBody)["model_name"])
			}
			msgs := (*gotBody)["messages"].([]interface{})
			if len(msgs) != 4 {
				t.Fatalf("messages = %v", (*gotBody)["messages"])
			}
			if msgs[0].(map[string]interface{})["role"] != "system" {
				t.Errorf("system message missing: %v", msgs[0])
			}
			assistant := msgs[2].(map[string]interface{})
			if assistant["role"] != "assistant" {
				t.Errorf("function_call item -> %v", assistant)
			}
			toolMsg := msgs[3].(map[string]interface{})
			if toolMsg["role"] != "tool" || toolMsg["tool_call_id"] != "call_1" || firstPartText(toolMsg) != "a.go b.go" {
				t.Errorf("function_call_output item -> %v", toolMsg)
			}
		})
}

func TestResponses_Streaming(t *testing.T) {
	withUpstream(t, sseUpstream(
		`{"choices":[{"delta":{"content":"Hel"}}]}`,
		`{"choices":[{"delta":{"content":"lo"}}]}`,
		`{"choices":[{"finish_reason":"stop"}]}`,
	), func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewResponsesHandler(p)
		req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{
		  "model": "deepseek-v4", "input": "hi", "stream": true
		}`))
		rec := httptest.NewRecorder()
		h.HandleResponses(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		events := parseSSEEvents(t, rec.Body.String())
		types := eventTypes(events)

		if types[0] != "response.created" {
			t.Fatalf("first event = %v", types)
		}
		if types[len(types)-1] != "response.completed" {
			t.Fatalf("last event = %v", types)
		}

		var text string
		for _, e := range events {
			if e["type"] == "response.output_text.delta" {
				text += e["delta"].(string)
			}
		}
		if text != "Hello" {
			t.Errorf("streamed text = %q", text)
		}

		completed := events[len(events)-1]["response"].(map[string]interface{})
		if completed["status"] != "completed" {
			t.Errorf("completed response = %v", completed)
		}
	})
}
