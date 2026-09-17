package e2e_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/openai"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

// 5 个存量预设模型清单（严格受保护模型）
var LegacyPresetModels = []string{
	"seed_m8",
	"Doubao_1_5_thinking_pro",
	"deepseek-R1",
	"deepseek-V3",
	"deepseek-V3-0324",
}

// 16 个新一代内置模型清单（涵盖全生态主力模型与别名）
var NewBuiltinModels = []string{
	"Seed-Code",
	"Seed-Evolving",
	"Seed-2.1-Pro-0915",
	"Seed-2.1-Turbo",
	"GLM-5.3-Flash",
	"GLM-5.3",
	"GLM-5.2",
	"DeepSeek-V4.1-Flash",
	"DeepSeek-V4-Flash",
	"DeepSeek-V4-Pro",
	"Kimi-K3",
	"Kimi-K2.8-Preview",
	"MiniMax-M3",
	"Qwen3.8-Flash",
	"Qwen3.8-Max",
	"Qwen3.7-Plus",
}

// RecordedRequest 记录 Mock 上游捕获到的真实请求
type RecordedRequest struct {
	Endpoint         string
	Method           string
	Headers          http.Header
	RawBody          []byte
	DecryptedContent string
	ModelName        string
	Timestamp        time.Time
}

// MockUpstreamServer 负责模拟 Trae CN 双通道上游（HTTPS 存量通道与 AgentTask 通道）
type MockUpstreamServer struct {
	Server *httptest.Server
	URL    string

	mu   sync.Mutex
	hits []RecordedRequest
}

// NewMockUpstreamServer 构建高保真双通道 Mock 上游服务器
func NewMockUpstreamServer() *MockUpstreamServer {
	m := &MockUpstreamServer{}
	mux := http.NewServeMux()

	// 存量通道：/api/ide/v1/llm_raw_chat
	mux.HandleFunc(config.EndpointLLMRawChat, m.handleLegacyLLMRawChat)

	// AgentTask 新通道：/api/agent/v3/create_agent_task
	mux.HandleFunc(config.EndpointAgentCreateTask, m.handleAgentCreateTask)

	// 模型列表刷新端点：/api/ide/v1/model_list
	mux.HandleFunc(config.EndpointModelList, m.handleModelList)

	m.Server = httptest.NewServer(mux)
	m.URL = m.Server.URL
	return m
}

// Close 关闭 Mock 上游
func (m *MockUpstreamServer) Close() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// ClearHits 清空记录
func (m *MockUpstreamServer) ClearHits() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits = nil
}

// GetHits 返回符合特定路径前缀的捕获记录
func (m *MockUpstreamServer) GetHits(endpointPrefix string) []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []RecordedRequest
	for _, h := range m.hits {
		if strings.HasPrefix(h.Endpoint, endpointPrefix) {
			res = append(res, h)
		}
	}
	return res
}

// AllHits 获取所有捕获记录
func (m *MockUpstreamServer) AllHits() []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]RecordedRequest, len(m.hits))
	copy(cp, m.hits)
	return cp
}

func (m *MockUpstreamServer) record(endpoint, method string, headers http.Header, body []byte, decrypted string, model string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits = append(m.hits, RecordedRequest{
		Endpoint:         endpoint,
		Method:           method,
		Headers:          headers.Clone(),
		RawBody:          body,
		DecryptedContent: decrypted,
		ModelName:        model,
		Timestamp:        time.Now(),
	})
}

// handleLegacyLLMRawChat 处理存量预设模型通道
func (m *MockUpstreamServer) handleLegacyLLMRawChat(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	pin := r.Header.Get("x-request-pin")
	atStr := r.Header.Get("x-requested-at")
	at, _ := strconv.ParseInt(atStr, 10, 64)

	var payload struct {
		ModelName string `json:"model_name"`
		Message   string `json:"message"`
	}
	_ = json.Unmarshal(body, &payload)

	decryptedStr := ""
	if payload.Message != "" && pin != "" && at > 0 {
		if plain, err := proxy.Demasticate(payload.Message, pin, at); err == nil {
			decryptedStr = string(plain)
		}
	}

	m.record(r.URL.Path, r.Method, r.Header, body, decryptedStr, payload.ModelName)

	// 输出符合 llm_raw_chat 规范的标准 SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)
	sessionID := "s_" + randomHex(8)

	// 1. metadata
	fmt.Fprintf(w, "event: metadata\ndata: {\"session_id\":%q}\n\n", sessionID)
	if flusher != nil {
		flusher.Flush()
	}

	// 2. 如果包含思考推理模型
	if strings.Contains(strings.ToLower(payload.ModelName), "thinking") || strings.Contains(strings.ToLower(payload.ModelName), "r1") {
		fmt.Fprintf(w, "event: output\ndata: {\"response\":\"\",\"reasoning_content\":\"正在推理思考存量预设模型...\",\"tool_calls\":null}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}

	// 3. 正常文本增量
	fmt.Fprintf(w, "event: output\ndata: {\"response\":\"你好！我是来自存量预设通道的回答。模型：\"}\n\n")
	if flusher != nil {
		flusher.Flush()
	}
	fmt.Fprintf(w, "event: output\ndata: {\"response\":%q,\"reasoning_content\":null,\"tool_calls\":null}\n\n", payload.ModelName)
	if flusher != nil {
		flusher.Flush()
	}

	// 4. token_usage
	fmt.Fprintf(w, "event: token_usage\ndata: {\"prompt_tokens\":12,\"completion_tokens\":8,\"total_tokens\":20}\n\n")
	if flusher != nil {
		flusher.Flush()
	}

	// 5. done
	fmt.Fprintf(w, "event: done\ndata: {\"finish_reason\":\"stop\"}\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

// handleAgentCreateTask 处理 AgentTask 新通道
func (m *MockUpstreamServer) handleAgentCreateTask(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	pin := r.Header.Get("x-request-pin")
	atStr := r.Header.Get("x-requested-at")
	at, _ := strconv.ParseInt(atStr, 10, 64)

	rawBodyStr := strings.TrimSpace(string(body))
	decryptedStr := ""
	modelName := ""

	// 尝试作为 Base64 密文直接解密
	if pin != "" && at > 0 {
		if plain, err := proxy.Demasticate(rawBodyStr, pin, at); err == nil {
			decryptedStr = string(plain)
			var parsed map[string]interface{}
			if err := json.Unmarshal(plain, &parsed); err == nil {
				if m, ok := parsed["model_name"].(string); ok {
					modelName = m
				} else if c, ok := parsed["config_name"].(string); ok {
					modelName = c
				}
			}
		}
	}

	m.record(r.URL.Path, r.Method, r.Header, body, decryptedStr, modelName)

	// 输出符合 create_agent_task 规范的标准 SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)

	// 1. task_created
	taskID := "task_" + randomHex(12)
	runID := "run_" + randomHex(12)
	fmt.Fprintf(w, "event: task_created\ndata: {\"task_id\":%q,\"agent_run_id\":%q}\n\n", taskID, runID)
	if flusher != nil {
		flusher.Flush()
	}

	// 2. metadata
	fmt.Fprintf(w, "event: metadata\ndata: {\"model\":%q,\"session_id\":%q}\n\n", modelName, "sess_"+randomHex(8))
	if flusher != nil {
		flusher.Flush()
	}

	// 3. thought (思考过程)
	fmt.Fprintf(w, "event: thought\ndata: {\"reasoning_content\":\"正在深度推理并解析用户需求...\",\"thought\":\"\"}\n\n")
	if flusher != nil {
		flusher.Flush()
	}

	// 4. thought (生成正文)
	fmt.Fprintf(w, "event: thought\ndata: {\"reasoning_content\":\"\",\"thought\":\"你好！我是来自新一代内置模型通道的回答。\"}\n\n")
	if flusher != nil {
		flusher.Flush()
	}

	// 5. token_usage
	fmt.Fprintf(w, "event: token_usage\ndata: {\"prompt_tokens\":25,\"completion_tokens\":15,\"total_tokens\":40}\n\n")
	if flusher != nil {
		flusher.Flush()
	}

	// 6. turn_completion 正常结束
	fmt.Fprintf(w, "event: turn_completion\ndata: {\"task_completion\":true,\"finish_reason\":\"stop\"}\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

// handleModelList 提供模型元数据刷新
func (m *MockUpstreamServer) handleModelList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"model_info_list": []map[string]interface{}{
				{"model_name": "seed_m8", "display_name": "Doubao-1.5-pro", "is_preset": true, "status": true},
				{"model_name": "Doubao_1_5_thinking_pro", "display_name": "Doubao-1.5-thinking-pro", "is_preset": true, "status": true},
				{"model_name": "deepseek-R1", "display_name": "DeepSeek-Reasoner（R1）", "is_preset": true, "status": true},
				{"model_name": "deepseek-V3", "display_name": "DeepSeek-V3", "is_preset": true, "status": true},
				{"model_name": "deepseek-V3-0324", "display_name": "DeepSeek-V3-0324", "is_preset": true, "status": true},
			},
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// TestGateway 提供可自启或连接外部的统一网关测试脚手架
type TestGateway struct {
	BaseURL      string
	MockUpstream *MockUpstreamServer
	server       *httptest.Server
	origDomain   string
}

// NewTestGateway 自动构造并启动网关测试环境
func NewTestGateway(t *testing.T) *TestGateway {
	t.Helper()

	// 检查是否有外部指定网关 URL
	if extURL := os.Getenv("TEST_GATEWAY_URL"); extURL != "" {
		return &TestGateway{
			BaseURL: strings.TrimRight(extURL, "/"),
		}
	}

	mockUpstream := NewMockUpstreamServer()
	origDomain := config.AgentDomain
	config.AgentDomain = mockUpstream.URL

	// 准备带有测试 Token 的池
	tokens := auth.NewTokenProvider()
	tokens.AddAccountWithToken("test-suite-account", "mock_e2e_jwt_token_for_testing")

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	p := proxy.NewTraeProxy(tokens, logger)
	openaiSrv := openai.NewServer(p, logger, nil)

	testHTTPSrv := httptest.NewServer(openaiSrv)

	return &TestGateway{
		BaseURL:      testHTTPSrv.URL,
		MockUpstream: mockUpstream,
		server:       testHTTPSrv,
		origDomain:   origDomain,
	}
}

// Close 清理测试网关及 Mock 资源
func (tg *TestGateway) Close() {
	if tg.server != nil {
		tg.server.Close()
	}
	if tg.MockUpstream != nil {
		tg.MockUpstream.Close()
	}
	if tg.origDomain != "" {
		config.AgentDomain = tg.origDomain
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// sendRequest 统一向测试网关发起 HTTP 请求辅助函数
func sendRequest(t *testing.T, method, url string, body interface{}, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			bs, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			bodyReader = bytes.NewReader(bs)
		}
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do failed: %v", err)
	}

	respBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, respBytes
}
