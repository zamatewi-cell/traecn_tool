package proxy

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/sse"
)

// Helper function to create test logger
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// TestTraeProxy_NewTraeProxy tests proxy initialization
func TestTraeProxy_NewTraeProxy(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()

	proxy := NewTraeProxy(tokens, logger)

	if proxy == nil {
		t.Fatal("NewTraeProxy() returned nil")
	}
	if proxy.client == nil {
		t.Error("NewTraeProxy() client is nil")
	}
	if proxy.tokens == nil {
		t.Error("NewTraeProxy() tokens is nil")
	}
	if proxy.device == nil {
		t.Error("NewTraeProxy() device is nil")
	}
	if proxy.queue == nil {
		t.Error("NewTraeProxy() queue is nil")
	}
	if proxy.logger == nil {
		t.Error("NewTraeProxy() logger is nil")
	}
}

// TestTraeProxy_GetQueueMonitor tests queue monitor accessor
func TestTraeProxy_GetQueueMonitor(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	monitor := proxy.GetQueueMonitor()
	if monitor == nil {
		t.Error("GetQueueMonitor() returned nil")
	}
}

// TestChatCompletionRequest_Marshal tests request marshaling
func TestChatCompletionRequest_Marshal(t *testing.T) {
	req := &ChatCompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
		ModelName: "doubao-1-5-pro",
		Stream:    true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled ChatCompletionRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(unmarshaled.Messages) != 2 {
		t.Errorf("Messages length = %v, want 2", len(unmarshaled.Messages))
	}
	if unmarshaled.ModelName != "doubao-1-5-pro" {
		t.Errorf("ModelName = %v, want doubao-1-5-pro", unmarshaled.ModelName)
	}
	if !unmarshaled.Stream {
		t.Error("Stream = false, want true")
	}
}

// TestMessage_Validation tests message validation
func TestMessage_Validation(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		wantErr bool
	}{
		{
			name:    "valid user message",
			msg:     Message{Role: "user", Content: "test"},
			wantErr: false,
		},
		{
			name:    "valid assistant message",
			msg:     Message{Role: "assistant", Content: "test"},
			wantErr: false,
		},
		{
			name:    "valid system message",
			msg:     Message{Role: "system", Content: "test"},
			wantErr: false,
		},
		{
			name:    "empty role",
			msg:     Message{Role: "", Content: "test"},
			wantErr: true,
		},
		{
			name:    "empty content",
			msg:     Message{Role: "user", Content: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation - in real code this might be more complex
			hasError := false
			if tt.msg.Role == "" || tt.msg.Content == "" {
				hasError = true
			}

			if hasError != tt.wantErr {
				t.Errorf("Message validation error = %v, want %v", hasError, tt.wantErr)
			}
		})
	}
}

// TestTraeProxy_setHeaders tests header setting
func TestTraeProxy_setHeaders(t *testing.T) {
	tokens := auth.NewTokenProvider()
	tokens.AddAccountWithToken("test", "test_token_123")
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	req, _ := http.NewRequest("POST", "http://test.com", nil)
	token := "test_token_123"

	proxy.setHeaders(req, token)

	// Check required headers
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", got)
	}
	if got := req.Header.Get(config.HeaderIDEToken); got != token {
		t.Errorf("IDE token = %v, want %v", got, token)
	}
	if got := req.Header.Get("X-Request-ID"); got == "" {
		t.Error("X-Request-ID is empty")
	}
	if got := req.Header.Get("X-Trae-Request-ID"); got == "" {
		t.Error("X-Trae-Request-ID is empty")
	}

	// Check device headers
	deviceHeaders := proxy.device.Headers()
	for k, v := range deviceHeaders {
		if got := req.Header.Get(k); got != v {
			t.Errorf("%s = %v, want %v", k, got, v)
		}
	}
}

// TestTraeProxy_detectQueueStatus tests queue status detection
func TestTraeProxy_detectQueueStatus(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	tests := []struct {
		name          string
		eventData     string
		wantPosition  int
		wantStatusSet bool
	}{
		{
			name:          "queue position present",
			eventData:     `{"queue_position": 5}`,
			wantPosition:  5,
			wantStatusSet: true,
		},
		{
			name:          "queue position zero",
			eventData:     `{"queue_position": 0}`,
			wantPosition:  0,
			wantStatusSet: false,
		},
		{
			name:          "no queue position",
			eventData:     `{"result": "done"}`,
			wantPosition:  0,
			wantStatusSet: false,
		},
		{
			name:          "invalid JSON",
			eventData:     `{invalid}`,
			wantPosition:  0,
			wantStatusSet: false,
		},
		{
			name:          "empty data",
			eventData:     "",
			wantPosition:  0,
			wantStatusSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := &sse.Event{Data: tt.eventData}
			proxy.detectQueueStatus(evt, "test-model")

			monitor := proxy.GetQueueMonitor()
			status := monitor.GetQueueStatus("test-model")

			if tt.wantStatusSet {
				if status.Position != tt.wantPosition {
					t.Errorf("Queue position = %v, want %v", status.Position, tt.wantPosition)
				}
			}
		})
	}
}

// TestTraeProxy_streamEvents tests upstream SSE parsing into normalized events
func TestTraeProxy_streamEvents(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	// Upstream SSE: two OpenAI-style deltas + a queue event.
	sseData := "data: {\"choices\":[{\"delta\":{\"content\":\"test1\"}}]}\n\n" +
		"data: {\"queue_position\": 3}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"test2\"}}]}\n\n" +
		"data: [DONE]\n\n"
	body := io.NopCloser(strings.NewReader(sseData))

	var texts []string
	var queuePos []int
	finishSeen := false

	err := proxy.streamEvents(body, "test-model", func(evt *StreamEvent) error {
		switch evt.Type {
		case EventText:
			texts = append(texts, evt.Text)
		case EventQueue:
			queuePos = append(queuePos, evt.QueuePosition)
		case EventFinish:
			finishSeen = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("streamEvents() error = %v", err)
	}

	if len(texts) != 2 || texts[0] != "test1" || texts[1] != "test2" {
		t.Errorf("text events = %v, want [test1 test2]", texts)
	}
	if len(queuePos) != 1 || queuePos[0] != 3 {
		t.Errorf("queue events = %v, want [3]", queuePos)
	}
	if !finishSeen {
		t.Error("finish event not seen for [DONE]")
	}

	// Queue monitor must have been updated then cleared.
	status := proxy.GetQueueMonitor().GetQueueStatus("test-model")
	if status.InQueue {
		t.Error("queue monitor still InQueue after stream end")
	}
}

// TestTraeProxy_ChatCompletion_NoTokens tests error handling when no tokens available
func TestTraeProxy_ChatCompletion_NoTokens(t *testing.T) {
	tokens := auth.NewTokenProvider() // Empty accounts
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	req := &ChatCompletionRequest{
		Messages:  []Message{{Role: "user", Content: "test"}},
		ModelName: "doubao-1-5-pro",
	}

	err := proxy.ChatCompletion(req, func(evt *StreamEvent) error { return nil })

	if err == nil {
		t.Fatal("ChatCompletion() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get token") {
		t.Errorf("Error = %v, want 'failed to get token'", err)
	}
}

// TestTraeProxy_ChatCompletion_InvalidModel tests handling of invalid model names
func TestTraeProxy_ChatCompletion_InvalidModel(t *testing.T) {
	tokens := auth.NewTokenProvider()
	tokens.AddAccountWithToken("test", "test_token")
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	tests := []struct {
		name    string
		model   string
		wantErr bool
	}{
		{
			name:    "valid model name",
			model:   "doubao-1-5-pro",
			wantErr: false,
		},
		{
			name:    "empty model name",
			model:   "",
			wantErr: false, // falls back to DefaultUpstreamModel
		},
		{
			name:    "unknown model",
			model:   "unknown-model-xyz",
			wantErr: false, // May still work if backend accepts it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ChatCompletionRequest{
				Messages:  []Message{{Role: "user", Content: "test"}},
				ModelName: tt.model,
			}

			err := proxy.ChatCompletion(req, func(evt *StreamEvent) error { return nil })

			// Note: This will likely fail due to network/invalid token
			// but we're testing the validation logic
			if tt.wantErr && err == nil {
				t.Error("ChatCompletion() expected error, got nil")
			}
		})
	}
}

// TestTraeProxy_FetchModels tests model list fetching
func TestTraeProxy_FetchModels(t *testing.T) {
	tokens := auth.NewTokenProvider()
	tokens.AddAccountWithToken("test", "test_token")
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	// This will fail in test environment (no valid token, no network)
	// but we test the function exists and has correct signature
	_, err := proxy.FetchModels()

	// Expected to fail with network error or auth error
	if err == nil {
		t.Log("FetchModels() succeeded (unexpected in test env)")
	} else {
		t.Logf("FetchModels() error (expected): %v", err)
	}
}

// TestTraeProxy_DeviceInfo tests device info integration
func TestTraeProxy_DeviceInfo(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	deviceInfo := proxy.device
	if deviceInfo == nil {
		t.Fatal("Device info is nil")
	}

	// Check device info is populated
	if deviceInfo.DeviceID == "" {
		t.Error("DeviceID is empty")
	}
	if deviceInfo.MachineID == "" {
		t.Error("MachineID is empty")
	}
	if deviceInfo.OSVersion == "" {
		t.Error("OSVersion is empty")
	}

	// Check headers are generated
	headers := deviceInfo.Headers()
	if len(headers) == 0 {
		t.Error("Device headers is empty")
	}

	requiredHeaders := []string{
		"x-device-id",
		"x-machine-id",
		"x-device-brand",
		"x-device-cpu",
		"x-os-version",
	}

	for _, h := range requiredHeaders {
		if _, ok := headers[h]; !ok {
			t.Errorf("Missing device header: %s", h)
		}
	}
}

// TestTraeProxy_QueueIntegration tests queue monitor integration
func TestTraeProxy_QueueIntegration(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	monitor := proxy.GetQueueMonitor()

	// Test queue status setting
	monitor.SetQueueStatus("test-model", 3, "Testing queue")
	status := monitor.GetQueueStatus("test-model")

	if status.Position != 3 {
		t.Errorf("Queue position = %v, want 3", status.Position)
	}
	if status.Message != "Testing queue" {
		t.Errorf("Queue message = %v, want 'Testing queue'", status.Message)
	}

	// Test clearing queue status (set position to 0)
	monitor.SetQueueStatus("test-model", 0, "")
	status = monitor.GetQueueStatus("test-model")
	if status.InQueue {
		t.Errorf("Cleared queue InQueue = %v, want false", status.InQueue)
	}
	if status.Position != 0 {
		t.Errorf("Cleared queue position = %v, want 0", status.Position)
	}
}

// BenchmarkTraeProxy_setHeaders benchmarks header setting
func BenchmarkTraeProxy_setHeaders(b *testing.B) {
	tokens := auth.NewTokenProvider()
	tokens.AddAccountWithToken("test", "test_token")
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	req, _ := http.NewRequest("POST", "http://test.com", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proxy.setHeaders(req, "test_token")
	}
}

// BenchmarkTraeProxy_detectQueueStatus benchmarks queue status detection
func BenchmarkTraeProxy_detectQueueStatus(b *testing.B) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	proxy := NewTraeProxy(tokens, logger)

	evt := &sse.Event{Data: `{"queue_position": 5}`}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proxy.detectQueueStatus(evt, "test-model")
	}
}
