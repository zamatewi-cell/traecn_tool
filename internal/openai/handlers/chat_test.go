package handlers

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

// withMockUpstream points the proxy at a local SSE server for the duration of f.
func withMockUpstream(t *testing.T, handler http.HandlerFunc, f func(*ChatHandler, *map[string]interface{})) {
	t.Helper()
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		decryptUpstreamBody(r, gotBody)
		handler(w, r)
	}))
	defer srv.Close()

	old := config.AgentDomain
	config.AgentDomain = srv.URL
	defer func() { config.AgentDomain = old }()

	tokens := auth.NewTokenProvider()
	tokens.AddAccountWithToken("acc", "tok")
	p := proxy.NewTraeProxy(tokens, testLogger())
	f(NewChatHandler(p), &gotBody)
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

func postChat(t *testing.T, h *ChatHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.HandleChatCompletions(rec, req)
	return rec
}

// collectDeltas parses an SSE body and returns all delta JSON objects.
func collectDeltas(t *testing.T, body string) []map[string]interface{} {
	t.Helper()
	var deltas []map[string]interface{}
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data: ") || line == "data: [DONE]" {
			continue
		}
		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk); err != nil {
			continue
		}
		choices, _ := chunk["choices"].([]interface{})
		for _, c := range choices {
			if m, ok := c.(map[string]interface{}); ok {
				if d, ok := m["delta"].(map[string]interface{}); ok {
					deltas = append(deltas, d)
				}
			}
		}
	}
	return deltas
}

func TestChatHandler_StreamingThinkSeparation(t *testing.T) {
	withMockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"<think>abc\"}}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"def</think>ans\"}}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"wer\"}}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}, func(h *ChatHandler, _ *map[string]interface{}) {
		rec := postChat(t, h, `{"model":"Seed-Code","messages":[{"role":"user","content":"hi"}],"stream":true}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		var reasoning, content string
		for _, d := range collectDeltas(t, rec.Body.String()) {
			if rc, ok := d["reasoning_content"].(string); ok {
				reasoning += rc
			}
			if c, ok := d["content"].(string); ok {
				content += c
			}
		}
		if reasoning != "abcdef" {
			t.Errorf("reasoning_content = %q, want abcdef", reasoning)
		}
		if content != "answer" {
			t.Errorf("content = %q, want answer", content)
		}
		if !strings.HasSuffix(rec.Body.String(), "data: [DONE]\n\n") {
			t.Error("missing [DONE] terminator")
		}
	})
}

func TestChatHandler_StreamingToolCalls(t *testing.T) {
	withMockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"\"}}]}}]}\n\n")
		io.WriteString(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path"}}]}}]}`+"\n\n")
		io.WriteString(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\":\"a.go\"}"}}]}}]}`+"\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"finish_reason\":\"tool_calls\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}, func(h *ChatHandler, gotBody *map[string]interface{}) {
		rec := postChat(t, h, `{
		  "model":"seed_m8",
		  "messages":[{"role":"user","content":"read a.go"}],
		  "stream":true,
		  "tools":[{"type":"function","function":{"name":"read_file","description":"read a file","parameters":{"type":"object","properties":{"path":{"type":"string"}}}}}],
		  "tool_choice":"auto"
		}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		// Client stream must carry incremental tool_calls deltas.
		var id, name, args string
		sawToolCalls := false
		for _, d := range collectDeltas(t, rec.Body.String()) {
			tcs, ok := d["tool_calls"].([]interface{})
			if !ok {
				continue
			}
			sawToolCalls = true
			for _, tci := range tcs {
				tc := tci.(map[string]interface{})
				if v, ok := tc["id"].(string); ok {
					id += v
				}
				if fn, ok := tc["function"].(map[string]interface{}); ok {
					if v, ok := fn["name"].(string); ok {
						name += v
					}
					if v, ok := fn["arguments"].(string); ok {
						args += v
					}
				}
			}
		}
		if !sawToolCalls {
			t.Fatal("no tool_calls deltas in client stream")
		}
		if id != "call_1" || name != "read_file" || args != `{"path":"a.go"}` {
			t.Errorf("reassembled tool call = (%q,%q,%q)", id, name, args)
		}

		// Upstream request must carry the tool declaration.
		tools, ok := (*gotBody)["tools"].([]interface{})
		if !ok || len(tools) != 1 {
			t.Fatalf("upstream tools = %v", (*gotBody)["tools"])
		}
		tool := tools[0].(map[string]interface{})
		fn := tool["function"].(map[string]interface{})
		if fn["name"] != "read_file" {
			t.Errorf("upstream tool name = %v", fn["name"])
		}
	})
}

func TestChatHandler_ToolResultMessagePassthrough(t *testing.T) {
	withMockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"done"},"finish_reason":"stop"}]}`)
	}, func(h *ChatHandler, gotBody *map[string]interface{}) {
		rec := postChat(t, h, `{
		  "model":"seed_m8",
		  "messages":[
		    {"role":"user","content":"read a.go"},
		    {"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.go\"}"}}]},
		    {"role":"tool","tool_call_id":"call_1","content":"package main"}
		  ],
		  "stream":false
		}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		msgs := (*gotBody)["messages"].([]interface{})
		if len(msgs) != 3 {
			t.Fatalf("upstream messages = %v", (*gotBody)["messages"])
		}
		assistant := msgs[1].(map[string]interface{})
		tcs := assistant["tool_calls"].([]interface{})
		if len(tcs) != 1 {
			t.Fatalf("assistant tool_calls = %v", assistant)
		}
		tc := tcs[0].(map[string]interface{})
		if tc["id"] != "call_1" {
			t.Errorf("assistant tool_call id = %v", tc["id"])
		}
		toolMsg := msgs[2].(map[string]interface{})
		if toolMsg["role"] != "tool" || toolMsg["tool_call_id"] != "call_1" || firstPartText(toolMsg) != "package main" {
			t.Errorf("tool result message = %v", toolMsg)
		}
	})
}

func TestChatHandler_NonStreamReasoningField(t *testing.T) {
	withMockUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"<think>hmm</think>42"},"finish_reason":"stop"}]}`)
	}, func(h *ChatHandler, _ *map[string]interface{}) {
		rec := postChat(t, h, `{"model":"seed_m8","messages":[{"role":"user","content":"?"}],"stream":false}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		msg := resp["choices"].([]interface{})[0].(map[string]interface{})["message"].(map[string]interface{})
		if msg["reasoning_content"] != "hmm" {
			t.Errorf("reasoning_content = %v", msg["reasoning_content"])
		}
		if msg["content"] != "42" {
			t.Errorf("content = %v", msg["content"])
		}
	})
}
