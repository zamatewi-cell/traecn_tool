package proxy

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/models"
)

// TestAgentTask_ResolveConfig verifies that model names map to valid config and model identifiers.
func TestAgentTask_ResolveConfig(t *testing.T) {
	tests := []struct {
		input      string
		wantConfig string
		wantModel  string
	}{
		{"Doubao-Seed-Code", "Doubao-Seed-Code", "Doubao-Seed-Code__dev"},
		{"seed-code", "Doubao-Seed-Code", "Doubao-Seed-Code__dev"},
		{"deepseek-v4.1-flash", "DeepSeek-V4-Flash", "DeepSeek-V4-Flash__dev"},
		{"DeepSeek-V4.1-Flash", "DeepSeek-V4-Flash", "DeepSeek-V4-Flash__dev"},
		{"deepseek-v4-pro", "DeepSeek-V4-Pro", "DeepSeek-V4-Pro__dev"},
		{"glm-5.2", "glm-5.2", "glm-5.2__dev"},
		{"GLM-5.2", "glm-5.2", "glm-5.2__dev"},
		{"GLM-5.3", "glm-5.3", "glm-5.3__dev"},
		{"kimi-k3", "kimi-k2", "kimi-k2__dev"},
		{"minimax-m3", "minimax-m3", "minimax-m3__dev"},
		{"minimax-m2.5", "minimax-m2.5", "minimax-m2.5__dev"},
		{"qwen-3.7-plus", "qwen-3.7-plus", "qwen-3.7-plus__dev"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cfg, mdl := ResolveAgentTaskConfig(tt.input)
			if cfg != tt.wantConfig {
				t.Errorf("ResolveAgentTaskConfig(%q) cfg = %q, want %q", tt.input, cfg, tt.wantConfig)
			}
			if mdl != tt.wantModel {
				t.Errorf("ResolveAgentTaskConfig(%q) mdl = %q, want %q", tt.input, mdl, tt.wantModel)
			}
		})
	}
}

// TestAgentTask_BuildPayload verifies payload structure and 24-character hex IDs.
func TestAgentTask_BuildPayload(t *testing.T) {
	req := &ChatCompletionRequest{
		ModelName: "Doubao-Seed-Code",
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Write hello world in Python"},
		},
	}

	payload, err := buildAgentTaskPayload(req, "", "")
	if err != nil {
		t.Fatalf("buildAgentTaskPayload failed: %v", err)
	}

	if payload.AgentType != "builder_v3" {
		t.Errorf("invalid agent type: %s", payload.AgentType)
	}
	if payload.RenderContext.Variables == "" {
		t.Errorf("expected non-empty render_context variables")
	}
	if !strings.Contains(payload.RenderContext.Variables, "Write hello world in Python") {
		t.Errorf("expected render_context variables to contain user prompt, got: %s", payload.RenderContext.Variables)
	}
	if payload.ConfigName != "Doubao-Seed-Code" {
		t.Errorf("config_name = %q, want Doubao-Seed-Code", payload.ConfigName)
	}
	if payload.ModelName != "Doubao-Seed-Code__dev" {
		t.Errorf("model_name = %q, want Doubao-Seed-Code__dev", payload.ModelName)
	}
	if len(payload.ConversationID) != 24 {
		t.Errorf("conversation_id len = %d, want 24", len(payload.ConversationID))
	}
	if len(payload.SessionID) != 24 {
		t.Errorf("session_id len = %d, want 24", len(payload.SessionID))
	}
	if len(payload.UserInput.Messages) != 1 {
		t.Fatalf("user_input messages count = %d, want 1", len(payload.UserInput.Messages))
	}
	text := payload.UserInput.Messages[0].TextContent
	if !strings.Contains(text, "System: You are a helpful assistant.") || !strings.Contains(text, "User: Write hello world in Python") {
		t.Errorf("prompt formatting incomplete: %q", text)
	}
}

// TestAgentTask_StreamParser verifies that agent_task SSE events are translated to StreamEvents.
func TestAgentTask_StreamParser(t *testing.T) {
	sseData := "event: task_created\n" +
		"data: {\"task_id\":\"task-123\",\"agent_run_id\":\"run-456\"}\n\n" +
		"event: thought\n" +
		"data: {\"first_data\":true,\"reasoning_content\":\"thinking about it...\",\"thought\":\"\"}\n\n" +
		"event: thought\n" +
		"data: {\"first_data\":false,\"reasoning_content\":\"\",\"thought\":\"Here is your code:\"}\n\n" +
		"event: token_usage\n" +
		"data: {\"prompt_tokens\":10,\"completion_tokens\":20,\"total_tokens\":30}\n\n" +
		"event: turn_completion\n" +
		"data: {\"task_completion\":true}\n\n"

	tokens := auth.NewTokenProvider()
	tokens.AddAccountWithToken("acc", "tok")
	p := NewTraeProxy(tokens, newTestLogger())

	var events []*StreamEvent
	err := p.streamAgentTaskEvents(strings.NewReader(sseData), "Doubao-Seed-Code", func(evt *StreamEvent) error {
		events = append(events, evt)
		return nil
	})

	if err != nil {
		t.Fatalf("streamAgentTaskEvents failed: %v", err)
	}

	var sawTaskCreated, sawReasoning, sawText, sawUsage, sawFinish bool
	var reasoningText, outputText string

	for _, e := range events {
		switch e.Type {
		case EventTaskCreated:
			sawTaskCreated = true
			if e.TaskID != "task-123" {
				t.Errorf("task_id = %q, want task-123", e.TaskID)
			}
		case EventReasoning:
			sawReasoning = true
			reasoningText += e.Reasoning
		case EventText:
			sawText = true
			outputText += e.Text
		case EventUsage:
			sawUsage = true
			if e.Usage.TotalTokens != 30 {
				t.Errorf("total_tokens = %d, want 30", e.Usage.TotalTokens)
			}
		case EventFinish:
			sawFinish = true
		}
	}

	if !sawTaskCreated {
		t.Error("task_created event missing")
	}
	if !sawReasoning || reasoningText != "thinking about it..." {
		t.Errorf("reasoning missing or mismatch: %q", reasoningText)
	}
	if !sawText || outputText != "Here is your code:" {
		t.Errorf("text output missing or mismatch: %q", outputText)
	}
	if !sawUsage {
		t.Error("usage event missing")
	}
	if !sawFinish {
		t.Error("finish event missing")
	}
}

// TestAgentTask_E2E_Mock verifies the full flow through ChatCompletion with Masticate decrypt verification.
func TestAgentTask_E2E_Mock(t *testing.T) {
	var capturedPin, capturedAt string
	var capturedBody string
	var capturedAuth, capturedBridge string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPin = r.Header.Get("X-Request-Pin")
		capturedAt = r.Header.Get("X-Requested-At")
		capturedAuth = r.Header.Get(config.HeaderIDEToken)
		capturedBridge = r.Header.Get("x-bridge-transport")

		b, _ := io.ReadAll(r.Body)
		capturedBody = string(b)

		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "event: task_created\ndata: {\"task_id\":\"t-100\"}\n\n")
		io.WriteString(w, "event: thought\ndata: {\"reasoning_content\":\"deep thought\",\"thought\":\"hello world\"}\n\n")
		io.WriteString(w, "event: turn_completion\ndata: {\"task_completion\":true}\n\n")
	}))
	defer srv.Close()

	old := config.AgentDomain
	config.AgentDomain = srv.URL
	defer func() { config.AgentDomain = old }()

	tokens := auth.NewTokenProvider()
	tokens.AddAccountWithToken("test-acc", "test-token-val")
	p := NewTraeProxy(tokens, newTestLogger())

	var collectedReasoning, collectedContent string
	var finished bool

	req := &ChatCompletionRequest{
		ModelName: "Doubao-Seed-Code",
		Messages: []Message{
			{Role: "user", Content: "Hello from test"},
		},
		Stream: true,
	}

	err := p.ChatCompletion(req, func(evt *StreamEvent) error {
		switch evt.Type {
		case EventReasoning:
			collectedReasoning += evt.Reasoning
		case EventText:
			collectedContent += evt.Text
		case EventFinish:
			finished = true
		}
		return nil
	})

	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}

	if capturedAuth != "test-token-val" {
		t.Errorf("Header Token = %q, want test-token-val", capturedAuth)
	}
	if capturedBridge != "aha" {
		t.Errorf("x-bridge-transport = %q, want aha", capturedBridge)
	}
	if capturedPin == "" || capturedAt == "" {
		t.Errorf("pin/at missing: pin=%q at=%q", capturedPin, capturedAt)
	}

	// Verify Masticate decryption of the wire message
	at, _ := strconv.ParseInt(capturedAt, 10, 64)
	plain, err := Demasticate(capturedBody, capturedPin, at)
	if err != nil {
		t.Fatalf("Demasticate failed on agent_task wire body: %v", err)
	}

	var payload AgentTaskPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		t.Fatalf("unmarshal decrypted payload failed: %v", err)
	}

	if payload.ConfigName != "Doubao-Seed-Code" {
		t.Errorf("config_name in decrypted payload = %q", payload.ConfigName)
	}
	if len(payload.UserInput.Messages) != 1 || payload.UserInput.Messages[0].TextContent != "Hello from test" {
		t.Errorf("user_input prompt mismatch: %+v", payload.UserInput)
	}

	if collectedReasoning != "deep thought" {
		t.Errorf("collectedReasoning = %q, want 'deep thought'", collectedReasoning)
	}
	if collectedContent != "hello world" {
		t.Errorf("collectedContent = %q, want 'hello world'", collectedContent)
	}
	if !finished {
		t.Error("stream did not finish")
	}
}

// TestModels_ResolveChannel verifies channel assignment logic.
func TestModels_ResolveChannel(t *testing.T) {
	// Preset models -> ChannelLegacyHTTPS
	if ch := models.ResolveChannel("seed_m8"); ch != models.ChannelLegacyHTTPS {
		t.Errorf("seed_m8 channel = %v, want ChannelLegacyHTTPS", ch)
	}
	if ch := models.ResolveChannel("Doubao_1_5_thinking_pro"); ch != models.ChannelLegacyHTTPS {
		t.Errorf("Doubao_1_5_thinking_pro channel = %v, want ChannelLegacyHTTPS", ch)
	}
	if ch := models.ResolveChannel("deepseek-R1"); ch != models.ChannelLegacyHTTPS {
		t.Errorf("deepseek-R1 channel = %v, want ChannelLegacyHTTPS", ch)
	}
	if ch := models.ResolveChannel("deepseek-V3"); ch != models.ChannelLegacyHTTPS {
		t.Errorf("deepseek-V3 channel = %v, want ChannelLegacyHTTPS", ch)
	}
	if ch := models.ResolveChannel("deepseek-V3-0324"); ch != models.ChannelLegacyHTTPS {
		t.Errorf("deepseek-V3-0324 channel = %v, want ChannelLegacyHTTPS", ch)
	}

	// 16 New builtin models -> ChannelAgentTask
	newModels := []string{
		"Doubao-Seed-Code",
		"DeepSeek-V4.1-Flash",
		"glm-5.2",
		"GLM-5.3",
		"GLM-5.1",
		"GLM-4.7",
		"kimi-k2",
		"minimax-m3",
		"minimax-m2.5",
		"qwen-3.7-plus",
		"qwen-3.5",
		"qwen3-coder",
		"gemini-3.1-pro",
		"gpt-5-mini",
		"Seed-Evolving",
		"Seed-2.1-Pro-0915",
	}

	for _, m := range newModels {
		if ch := models.ResolveChannel(m); ch != models.ChannelAgentTask {
			t.Errorf("model %q channel = %v, want ChannelAgentTask", m, ch)
		}
	}
}

func TestAgentTask_ValidateAgentTaskRequest(t *testing.T) {
	// 1. 无工具、无 ToolChoice、普通消息 -> 允许通过
	reqOk := &ChatCompletionRequest{
		ModelName: "Seed-Code",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
		},
	}
	if err := ValidateAgentTaskRequest(reqOk); err != nil {
		t.Errorf("expected nil error for valid request, got: %v", err)
	}

	// 2. tool_choice: "none" -> 允许通过
	reqNone := &ChatCompletionRequest{
		ModelName:  "Seed-Code",
		ToolChoice: json.RawMessage(`"none"`),
	}
	if err := ValidateAgentTaskRequest(reqNone); err != nil {
		t.Errorf("expected nil error for tool_choice none, got: %v", err)
	}

	// 3. 自定义 tools -> 必须拦截
	reqTools := &ChatCompletionRequest{
		ModelName: "Seed-Code",
		Tools: []Tool{
			{Type: "function", Function: ToolFunctionSpec{Name: "get_weather"}},
		},
	}
	if err := ValidateAgentTaskRequest(reqTools); err == nil || !errors.Is(err, ErrAgentTaskToolsUnsupported) {
		t.Errorf("expected ErrAgentTaskToolsUnsupported for req with tools, got: %v", err)
	}

	// 4. tool_choice: "auto" -> 必须拦截
	reqAuto := &ChatCompletionRequest{
		ModelName:  "Seed-Code",
		ToolChoice: json.RawMessage(`"auto"`),
	}
	if err := ValidateAgentTaskRequest(reqAuto); err == nil || !errors.Is(err, ErrAgentTaskToolsUnsupported) {
		t.Errorf("expected ErrAgentTaskToolsUnsupported for tool_choice auto, got: %v", err)
	}

	// 5. 历史消息中包含 role: tool -> 必须拦截
	reqToolMsg := &ChatCompletionRequest{
		ModelName: "Seed-Code",
		Messages: []Message{
			{Role: "tool", Content: "tool result", ToolCallID: "call_123"},
		},
	}
	if err := ValidateAgentTaskRequest(reqToolMsg); err == nil || !errors.Is(err, ErrAgentTaskToolsUnsupported) {
		t.Errorf("expected ErrAgentTaskToolsUnsupported for message with role tool, got: %v", err)
	}

	// 6. 历史消息中 assistant 包含 tool_calls -> 必须拦截
	reqToolCalls := &ChatCompletionRequest{
		ModelName: "Seed-Code",
		Messages: []Message{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{ID: "call_123", Type: "function", Function: ToolFunction{Name: "calc"}},
				},
			},
		},
	}
	if err := ValidateAgentTaskRequest(reqToolCalls); err == nil || !errors.Is(err, ErrAgentTaskToolsUnsupported) {
		t.Errorf("expected ErrAgentTaskToolsUnsupported for message with tool_calls, got: %v", err)
	}
}

func TestAgentTask_PrematureEOF_And_TurnCompletion(t *testing.T) {
	p := &TraeProxy{}

	// 1. 模拟提前 EOF（只有 thought，没有 turn_completion）
	incompleteSSE := "event: thought\ndata: {\"thought\":\"thinking...\"}\n\n"
	err := p.streamAgentTaskEvents(strings.NewReader(incompleteSSE), "Seed-Code", func(evt *StreamEvent) error {
		return nil
	})
	if err == nil || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected io.ErrUnexpectedEOF on premature stream close, got: %v", err)
	}

	// 2. 模拟正常完成且携带 length 截断
	completedSSE := "event: thought\ndata: {\"thought\":\"done\"}\n\nevent: turn_completion\ndata: {\"task_completion\":false,\"finish_reason\":\"length\"}\n\n"
	var lastFinishReason string
	err = p.streamAgentTaskEvents(strings.NewReader(completedSSE), "Seed-Code", func(evt *StreamEvent) error {
		if evt.Type == EventFinish {
			lastFinishReason = evt.FinishReason
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error for completed SSE: %v", err)
	}
	if lastFinishReason != "length" {
		t.Fatalf("expected finish_reason 'length', got: %q", lastFinishReason)
	}
}

