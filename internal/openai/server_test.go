package openai

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/models"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
	"github.com/zamatewi-cell/traecn_tool/internal/version"
)

// Helper function to create test logger
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestServer_NewServer(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)

	// Test with nil config
	s1 := NewServer(p, logger, nil)
	if s1 == nil {
		t.Fatal("NewServer() with nil config returned nil")
	}

	// Test with empty config
	s2 := NewServer(p, logger, &ServerConfig{})
	if s2 == nil {
		t.Fatal("NewServer() with empty config returned nil")
	}

	// Test with API keys
	s3 := NewServer(p, logger, &ServerConfig{
		APIKeys: []string{"test-key-1", "test-key-2"},
	})
	if s3 == nil {
		t.Fatal("NewServer() with API keys returned nil")
	}
}

func TestServer_ServeHTTP(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)
	s := NewServer(p, logger, nil)

	// Test basic request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ServeHTTP() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("ServeHTTP() invalid JSON response: %v", err)
	}

	if response["name"] != "trae-proxy" {
		t.Errorf("ServeHTTP() name = %v, want 'trae-proxy'", response["name"])
	}
}

func TestServer_handleModels(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)
	s := NewServer(p, logger, nil)

	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleModels() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("handleModels() invalid JSON response: %v", err)
	}

	if response["object"] != "list" {
		t.Errorf("handleModels() object = %v, want 'list'", response["object"])
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatal("handleModels() data is not an array")
	}

	// Should have at least one model
	if len(data) == 0 {
		t.Error("handleModels() returned empty model list")
	}
}

func TestServer_handleHealth(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)
	s := NewServer(p, logger, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleHealth() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("handleHealth() invalid JSON response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("handleHealth() status = %v, want 'ok'", response["status"])
	}
	if response["version"] == "" {
		t.Error("handleHealth() version is empty")
	}
}

func TestServer_handleQueueStatus(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)
	s := NewServer(p, logger, nil)

	// Test without model parameter (get all status)
	req := httptest.NewRequest("GET", "/v1/queue/status", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleQueueStatus() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("handleQueueStatus() invalid JSON response: %v", err)
	}

	// Test with model parameter
	req = httptest.NewRequest("GET", "/v1/queue/status?model=test-model", nil)
	w = httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleQueueStatus(model) status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestServer_Routes(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)
	s := NewServer(p, logger, nil)

	tests := []struct {
		method       string
		path         string
		expectedCode int
	}{
		{"GET", "/", http.StatusOK},
		{"GET", "/health", http.StatusOK},
		{"GET", "/v1/models", http.StatusOK},
		{"GET", "/v1/queue/status", http.StatusOK},
		{"POST", "/v1/chat/completions", http.StatusBadRequest}, // Expected to fail without body
		{"POST", "/v1/completions", http.StatusBadRequest},      // Expected to fail without body
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, req)

		if w.Code != tt.expectedCode {
			t.Errorf("%s %s: status = %v, want %v", tt.method, tt.path, w.Code, tt.expectedCode)
		}
	}
}

func TestServer_APIKeyMiddleware(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)

	// Create server with API key
	s := NewServer(p, logger, &ServerConfig{
		APIKeys: []string{"valid-key"},
	})

	// Test without API key (should fail for protected routes)
	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	// Should be 401 or 403 without API key
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
		t.Errorf("handleModels() without API key: status = %v, want 401 or 403", w.Code)
	}

	// Test with valid API key
	req = httptest.NewRequest("GET", "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer valid-key")
	w = httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleModels() with valid API key: status = %v, want 200", w.Code)
	}

	// Test with invalid API key
	req = httptest.NewRequest("GET", "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer invalid-key")
	w = httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
		t.Errorf("handleModels() with invalid API key: status = %v, want 401 or 403", w.Code)
	}
}

func TestServer_CORSHeaders(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)
	s := NewServer(p, logger, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("ServeHTTP() missing Access-Control-Allow-Origin header")
	}
}

func TestServer_LoggerMiddleware(t *testing.T) {
	var buf bytes.Buffer
	tokens := auth.NewTokenProvider()
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	p := proxy.NewTraeProxy(tokens, logger)

	s := NewServer(p, logger, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	// Logger should have logged the request
	if buf.Len() == 0 {
		t.Error("ServeHTTP() logger middleware did not log request")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]string{"key": "value"}
	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("writeJSON() status = %v, want %v", w.Code, http.StatusOK)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("writeJSON() Content-Type = %v, want application/json", w.Header().Get("Content-Type"))
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("writeJSON() invalid JSON: %v", err)
	}

	if response["key"] != "value" {
		t.Errorf("writeJSON() data = %v, want value", response["key"])
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusBadRequest, "invalid_request", "test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("writeError() status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("writeError() invalid JSON: %v", err)
	}

	err, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("writeError() missing error object")
	}

	if err["type"] != "invalid_request" {
		t.Errorf("writeError() type = %v, want 'invalid_request'", err["type"])
	}
	if err["message"] != "test error message" {
		t.Errorf("writeError() message = %v, want 'test error message'", err["message"])
	}
}

func TestServer_IntegrationWithProxy(t *testing.T) {
	// Create proxy with queue
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)

	// Create server with proxy
	s := NewServer(p, logger, &ServerConfig{
		APIKeys: []string{"test-key"},
	})

	// Test queue status endpoint
	req := httptest.NewRequest("GET", "/v1/queue/status", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Integration queue status: status = %v, want 200", w.Code)
	}
}

func TestServer_KnownModels(t *testing.T) {
	// Verify that the model registry is populated
	list := models.Default().List()
	if len(list) == 0 {
		t.Error("model registry is empty")
	}

	// Check that every model has required fields
	for _, model := range list {
		if model.ModelID == "" {
			t.Error("Found model with empty ModelID")
		}
		if model.Provider == "" {
			t.Errorf("Model %s has empty Provider", model.ModelID)
		}
	}
}

func TestServer_CORS_UntrustedOriginBlockedWhenNoAuth(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)
	// Server running without API keys (open loopback mode)
	s := NewServer(p, logger, nil)

	// External malicious website trying to access local gateway
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "https://evil.attacker.com")
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("ServeHTTP() with external untrusted Origin = %d, want 403 Forbidden", w.Code)
	}

	// OPTIONS preflight from untrusted origin should also be blocked
	preflightReq := httptest.NewRequest("OPTIONS", "/v1/chat/completions", nil)
	preflightReq.Header.Set("Origin", "https://evil.attacker.com")
	preflightW := httptest.NewRecorder()

	s.ServeHTTP(preflightW, preflightReq)

	if preflightW.Code != http.StatusForbidden {
		t.Errorf("ServeHTTP() OPTIONS preflight with external untrusted Origin = %d, want 403 Forbidden", preflightW.Code)
	}
}

func TestServer_CORS_TrustedOriginsAllowedWhenNoAuth(t *testing.T) {
	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)
	// Server running without API keys
	s := NewServer(p, logger, nil)

	trustedOrigins := []string{
		"http://localhost:3000",
		"http://127.0.0.1:5173",
		"http://[::1]:8080",
		"vscode-webview://abc123xyz",
		"app://obsidian.md",
	}

	for _, origin := range trustedOrigins {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("ServeHTTP() with trusted Origin %s: status = %d, want 200", origin, w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("ServeHTTP() with trusted Origin %s: Access-Control-Allow-Origin = %q, want %q", origin, got, origin)
		}
	}
}

func TestDashboard_HTML_StaticIntegrityAndXSSProtection(t *testing.T) {
	// 1. 验证 dashboardHTML 变量正确嵌入
	if len(dashboardHTML) == 0 {
		t.Fatal("dashboardHTML is empty or not embedded")
	}

	content := string(dashboardHTML)

	// 2. 验证文件的第一个非 BOM/空白内容必须严格是 <!DOCTYPE html>
	trimmed := strings.TrimLeft(content, "\xef\xbb\xbf \t\r\n")
	if !strings.HasPrefix(strings.ToLower(trimmed), "<!doctype html>") {
		firstLine := trimmed
		if idx := strings.IndexAny(firstLine, "\r\n"); idx != -1 {
			firstLine = firstLine[:idx]
		}
		t.Fatalf("dashboard.html must strictly start with <!DOCTYPE html>, but got: %q", firstLine)
	}

	// 3. 验证无外部 CDN 运行时脚本依赖
	if strings.Contains(content, "cdn.tailwindcss.com") {
		t.Error("dashboard.html must not import external cdn.tailwindcss.com runtime script")
	}

	// 4. 验证关键函数定义的唯一性，杜绝重复定义或插入文件头
	checkSingleDefinition := func(funcPattern string, expectedCount int) {
		count := strings.Count(content, funcPattern)
		if count != expectedCount {
			t.Errorf("expected function %q to be defined %d times, but found %d times", funcPattern, expectedCount, count)
		}
	}

	checkSingleDefinition("async function loadLiveLogsFromBackend", 1)
	checkSingleDefinition("async function loadCloudBilling", 1)
	checkSingleDefinition("function renderModels()", 1)
	checkSingleDefinition("function populateSelect()", 1)
	checkSingleDefinition("function escapeHtml(", 1)

	// 5. 验证不再有动态拼接的 onclick 注入点
	if strings.Contains(content, "onclick=\"selectForTest(") {
		t.Error("dashboard.html still contains inline dynamic onclick='selectForTest(...)', should use data-model-id and event delegation")
	}

	// 6. 验证日志表格和账单表格中的关键动态字段均被 escapeHtml 保护
	if !strings.Contains(content, "escapeHtml(r.model") {
		t.Error("loadLiveLogsFromBackend missing escapeHtml for r.model")
	}
	if !strings.Contains(content, "escapeHtml(r.error_msg") {
		t.Error("loadLiveLogsFromBackend missing escapeHtml for r.error_msg")
	}
	if !strings.Contains(content, "escapeHtml(r.client_ip") {
		t.Error("loadLiveLogsFromBackend missing escapeHtml for r.client_ip")
	}
	if !strings.Contains(content, "escapeHtml(e.message)") {
		t.Error("loadLiveLogsFromBackend missing escapeHtml for e.message")
	}

	// 7. 验证转义规则对于典型 XSS payload 的过滤能力
	maliciousPayload := `<img src=x onerror=alert(1)>`
	escapeTest := func(s string) string {
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = strings.ReplaceAll(s, "\"", "&quot;")
		s = strings.ReplaceAll(s, "'", "&#39;")
		return s
	}
	escaped := escapeTest(maliciousPayload)
	if strings.Contains(escaped, "<") || strings.Contains(escaped, ">") {
		t.Errorf("escapeHtml should eliminate < and >, got: %s", escaped)
	}
	if escaped != "&lt;img src=x onerror=alert(1)&gt;" {
		t.Errorf("unexpected escaped payload result: %s", escaped)
	}
}

func TestServer_VersionNormalization(t *testing.T) {
	origVer := version.Version
	defer func() { version.Version = origVer }()

	// 模拟外部 ldflags 注入了带 "v" 前缀的版本号
	version.Version = "v1.0.0"

	tokens := auth.NewTokenProvider()
	logger := newTestLogger()
	p := proxy.NewTraeProxy(tokens, logger)
	s := NewServer(p, logger, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse /health json: %v", err)
	}

	if resp["version"] != "1.0.0" {
		t.Errorf("expected normalized version '1.0.0', got '%s'", resp["version"])
	}
}



