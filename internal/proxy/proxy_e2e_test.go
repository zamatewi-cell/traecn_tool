package proxy

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
)

func e2eLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// withUpstream points the proxy at a local test server for the duration of f.
func withUpstream(t *testing.T, handler http.HandlerFunc, f func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	old := config.AgentDomain
	config.AgentDomain = srv.URL
	defer func() { config.AgentDomain = old }()

	f()
}

func TestTraeProxy_E2E_HeadersAndBody(t *testing.T) {
	var gotAuth, gotUA, gotReqID, gotTraeReqID, gotDevice, gotMachine, gotIDEVersion string
	var gotGetSvc, gotPin, gotAt string
	var gotBody map[string]interface{}

	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get(config.HeaderIDEToken)
		gotUA = r.Header.Get("User-Agent")
		gotReqID = r.Header.Get("x-request-id")
		gotTraeReqID = r.Header.Get("x-trae-request-id")
		gotDevice = r.Header.Get("x-device-id")
		gotMachine = r.Header.Get("x-machine-id")
		gotIDEVersion = r.Header.Get("x-ide-version")
		gotGetSvc = r.Header.Get("get-svc")
		gotPin = r.Header.Get("x-request-pin")
		gotAt = r.Header.Get("x-requested-at")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)

		// The real llm_raw_chat SSE dialect.
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "event: metadata\ndata: {\"session_id\":\"s-1\"}\n\n")
		io.WriteString(w, "event: output\ndata: {\"response\":\"hi\",\"reasoning_content\":null,\"tool_calls\":null}\n\n")
		io.WriteString(w, "event: token_usage\ndata: {\"prompt_tokens\":5,\"completion_tokens\":1,\"total_tokens\":6}\n\n")
		io.WriteString(w, "event: done\ndata: {\"finish_reason\":\"stop\"}\n\n")
	}, func() {
		tokens := auth.NewTokenProvider()
		tokens.AddAccountWithToken("acc", "secret_token_xyz")
		p := NewTraeProxy(tokens, e2eLogger())

		var texts []string
		finish := ""
		var usage *Usage
		err := p.ChatCompletion(&ChatCompletionRequest{
			ModelName: "Seed-Code",
			Messages:  []Message{{Role: "user", Content: "hello"}},
			Stream:    true,
		}, func(evt *StreamEvent) error {
			switch evt.Type {
			case EventText:
				texts = append(texts, evt.Text)
			case EventFinish:
				finish = evt.FinishReason
			case EventUsage:
				usage = evt.Usage
			}
			return nil
		})
		if err != nil {
			t.Fatalf("ChatCompletion() error = %v", err)
		}
		if len(texts) != 1 || texts[0] != "hi" {
			t.Errorf("texts = %v", texts)
		}
		if finish != "stop" {
			t.Errorf("finish = %q", finish)
		}
		if usage == nil || usage.TotalTokens != 6 {
			t.Errorf("usage = %+v, want total 6", usage)
		}
	})

	if gotAuth != "secret_token_xyz" {
		t.Errorf("X-IDE-Token = %q", gotAuth)
	}
	if gotUA != "TraeClient/TTNet" {
		t.Errorf("User-Agent = %q", gotUA)
	}
	if !strings.HasPrefix(gotReqID, "req_") {
		t.Errorf("x-request-id = %q, want req_ prefix", gotReqID)
	}
	if gotTraeReqID == "" || gotDevice == "" || gotMachine == "" {
		t.Error("device fingerprint headers missing")
	}
	if gotIDEVersion != config.IDEVersion {
		t.Errorf("x-ide-version = %q, want %q", gotIDEVersion, config.IDEVersion)
	}
	if gotGetSvc != "1" {
		t.Errorf("get-svc = %q, want 1", gotGetSvc)
	}
	if gotPin == "" || gotAt == "" {
		t.Error("x-request-pin / x-requested-at missing")
	}

	// Body assertions: model_name + masticate-encrypted message, decrypted
	// with the pin/at pair echoed in the headers.
	if gotBody["model_name"] != "seed_m8" {
		t.Errorf("model_name = %v", gotBody["model_name"])
	}
	encMsg, _ := gotBody["message"].(string)
	if encMsg == "" {
		t.Fatal("encrypted message missing from body")
	}
	at, err := strconv.ParseInt(gotAt, 10, 64)
	if err != nil {
		t.Fatalf("x-requested-at not a unix timestamp: %q", gotAt)
	}
	plain, err := Demasticate(encMsg, gotPin, at)
	if err != nil {
		t.Fatalf("Demasticate() error = %v", err)
	}
	var msgs []map[string]interface{}
	if err := json.Unmarshal(plain, &msgs); err != nil {
		t.Fatalf("decrypted payload not a messages array: %v", err)
	}
	if len(msgs) != 1 || msgs[0]["role"] != "user" || firstPartText(msgs[0]) != "hello" {
		t.Errorf("decrypted messages wrong: %v", msgs)
	}
}

func TestTraeProxy_E2E_RetryOn401(t *testing.T) {
	var calls atomic.Int32

	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, `{"error":"bad token"}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n")
	}, func() {
		tokens := auth.NewTokenProvider()
		tokens.AddAccountWithToken("bad", "bad_token")
		tokens.AddAccountWithToken("good", "good_token")
		p := NewTraeProxy(tokens, e2eLogger())

		var got string
		err := p.ChatCompletion(&ChatCompletionRequest{
			ModelName: "Seed-Code",
			Messages:  []Message{{Role: "user", Content: "hi"}},
			Stream:    true,
		}, func(evt *StreamEvent) error {
			if evt.Type == EventText {
				got += evt.Text
			}
			return nil
		})
		if err != nil {
			t.Fatalf("ChatCompletion() error = %v", err)
		}
		if got != "ok" {
			t.Errorf("got %q, want ok (retry with second account)", got)
		}
	})

	if calls.Load() != 2 {
		t.Errorf("upstream calls = %d, want 2 (401 then success)", calls.Load())
	}
}

func TestTraeProxy_E2E_NonStreamJSON(t *testing.T) {
	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"x","choices":[{"message":{"role":"assistant","content":"full answer"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)
	}, func() {
		tokens := auth.NewTokenProvider()
		tokens.AddAccountWithToken("acc", "tok")
		p := NewTraeProxy(tokens, e2eLogger())

		var content string
		var usage *Usage
		err := p.ChatCompletion(&ChatCompletionRequest{
			ModelName: "Seed-Code",
			Messages:  []Message{{Role: "user", Content: "hi"}},
			Stream:    false,
		}, func(evt *StreamEvent) error {
			switch evt.Type {
			case EventText:
				content += evt.Text
			case EventUsage:
				usage = evt.Usage
			}
			return nil
		})
		if err != nil {
			t.Fatalf("ChatCompletion() error = %v", err)
		}
		if content != "full answer" {
			t.Errorf("content = %q", content)
		}
		if usage == nil || usage.TotalTokens != 3 {
			t.Errorf("usage = %+v", usage)
		}
	})
}

func TestParseUpstreamData_Shapes(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []EventType
	}{
		{"openai delta", `{"choices":[{"delta":{"content":"a"}}]}`, []EventType{EventText}},
		{"reasoning delta", `{"choices":[{"delta":{"reasoning_content":"think"}}]}`, []EventType{EventReasoning}},
		{"tool call", `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"fn","arguments":"{"}}]}}]}`, []EventType{EventToolCall}},
		{"queue", `{"queue_position":4}`, []EventType{EventQueue}},
		{"task created", `{"event":"task_created","task_id":"t-1"}`, []EventType{EventTaskCreated}},
		{"message_delta", `{"message_delta":"hello"}`, []EventType{EventText}},
		{"finish", `{"choices":[{"finish_reason":"stop"}]}`, []EventType{EventFinish}},
		{"usage", `{"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`, []EventType{EventUsage}},
		{"done marker", `[DONE]`, []EventType{EventFinish}},
		{"error payload", `{"error":{"message":"boom","type":"rate_limit"}}`, []EventType{EventError}},
		{"raw text fallback", `not json at all`, []EventType{EventText}},
		{"empty", ``, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evts := parseUpstreamData(tt.data)
			if len(evts) != len(tt.want) {
				t.Fatalf("parseUpstreamData() emitted %d events, want %d: %+v", len(evts), len(tt.want), evts)
			}
			for i, want := range tt.want {
				if evts[i].Type != want {
					t.Errorf("event %d type = %v, want %v", i, evts[i].Type, want)
				}
			}
		})
	}
}

func TestParseUpstreamEvent_RealDialect(t *testing.T) {
	tests := []struct {
		name  string
		event string
		data  string
		want  []EventType
	}{
		{"metadata ignored", "metadata", `{"session_id":"s-1","ide_version":"3.3.37"}`, nil},
		{"output text", "output", `{"response":"Hi","reasoning_content":null,"tool_calls":null}`, []EventType{EventText}},
		{"output reasoning", "output", `{"response":"","reasoning_content":"thinking","tool_calls":null}`, []EventType{EventReasoning}},
		{"output tool call", "output", `{"response":"","tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"fn","arguments":"{}"}}]}`, []EventType{EventToolCall}},
		{"token usage", "token_usage", `{"prompt_tokens":50,"completion_tokens":1,"total_tokens":51}`, []EventType{EventUsage}},
		{"done", "done", `{"finish_reason":"stop"}`, []EventType{EventFinish}},
		{"error 4023", "error", `{"code":4023,"error":"","message":"the model is unknown","extra":null}`, []EventType{EventError}},
		{"unknown event falls back", "", `{"choices":[{"delta":{"content":"a"}}]}`, []EventType{EventText}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evts := parseUpstreamEvent(tt.event, tt.data)
			if len(evts) != len(tt.want) {
				t.Fatalf("parseUpstreamEvent() emitted %d events, want %d: %+v", len(evts), len(tt.want), evts)
			}
			for i, want := range tt.want {
				if evts[i].Type != want {
					t.Errorf("event %d type = %v, want %v", i, evts[i].Type, want)
				}
			}
		})
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

func TestParseUpstreamData_ToolCallFields(t *testing.T) {
	evts := parseUpstreamData(`{"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_9","type":"function","function":{"name":"read_file","arguments":"{\"path\":"}}]}}]}`)
	if len(evts) != 1 || evts[0].Type != EventToolCall {
		t.Fatalf("events = %+v", evts)
	}
	tc := evts[0].ToolCall
	if tc.Index != 1 || tc.ID != "call_9" || tc.Name != "read_file" || tc.Arguments != `{"path":` {
		t.Errorf("tool call delta = %+v", tc)
	}
}
