package e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// OpenAI 错误结构体
type OpenAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
}

// TestTier2_EmptyAndMalformedRequests 测试空请求与畸形请求体的拒绝与格式友好错误
func TestTier2_EmptyAndMalformedRequests(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	cases := []struct {
		name        string
		body        interface{}
		expectCode  int
		expectedErr string
	}{
		{
			name:        "EmptyRequestBody",
			body:        "",
			expectCode:  http.StatusBadRequest,
			expectedErr: "invalid_request_error",
		},
		{
			name:        "MalformedJSON",
			body:        `{"model": "Seed-Code", "messages": `,
			expectCode:  http.StatusBadRequest,
			expectedErr: "invalid_request_error",
		},
		{
			name: "MissingModelField",
			body: map[string]interface{}{
				"messages": []map[string]string{
					{"role": "user", "content": "hi"},
				},
			},
			expectCode:  http.StatusBadRequest,
			expectedErr: "invalid_request_error",
		},
		{
			name: "MissingMessagesField",
			body: map[string]interface{}{
				"model": "Seed-Code",
			},
			expectCode:  http.StatusBadRequest,
			expectedErr: "invalid_request_error",
		},
		{
			name: "EmptyMessagesArray",
			body: map[string]interface{}{
				"model":    "Seed-Code",
				"messages": []map[string]string{},
			},
			expectCode:  http.StatusBadRequest,
			expectedErr: "invalid_request_error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := sendRequest(t, "POST", tg.BaseURL+"/v1/chat/completions", tc.body, nil)
			if resp.StatusCode != tc.expectCode {
				t.Errorf("expected status code %d, got %d. Body: %s", tc.expectCode, resp.StatusCode, string(body))
			}

			var errResp OpenAIErrorResponse
			if err := json.Unmarshal(body, &errResp); err != nil {
				t.Fatalf("expected JSON error response, unmarshal failed: %v, raw: %s", err, string(body))
			}

			if errResp.Error.Type == "" && errResp.Error.Message == "" {
				t.Errorf("expected non-empty error.type or error.message, got %+v", errResp)
			}
		})
	}
}

// TestTier2_InvalidModelHandling 测试极端或未知模型名称，确保不崩溃并友好处理
func TestTier2_InvalidModelHandling(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	invalidModels := []string{
		"non-existent-model-xyz-999",
		"invalid/strange/model-name!@#",
		"__unknown_null__",
	}

	for _, badModel := range invalidModels {
		t.Run("Model_"+badModel, func(t *testing.T) {
			reqPayload := map[string]interface{}{
				"model": badModel,
				"messages": []map[string]string{
					{"role": "user", "content": "testing invalid model handling"},
				},
			}

			resp, body := sendRequest(t, "POST", tg.BaseURL+"/v1/chat/completions", reqPayload, nil)

			// 验证服务绝对没有 panic 挂死或返回非 JSON
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusBadGateway {
				t.Errorf("expected 200 (fallback) or 400/502 (error), got %d, body: %s", resp.StatusCode, string(body))
			}

			// 如果返回错误状态，必须是合法结构化 JSON
			if resp.StatusCode != http.StatusOK {
				var errResp OpenAIErrorResponse
				if err := json.Unmarshal(body, &errResp); err != nil {
					t.Fatalf("expected structured JSON error for invalid model %q, got: %s", badModel, string(body))
				}
				if errResp.Error.Message == "" {
					t.Errorf("expected descriptive error message for invalid model, got empty")
				}
			} else {
				// 如果按安全策略降级回退到默认模型，必须正常返回合法对话响应
				var chatResp ChatCompletionResponse
				if err := json.Unmarshal(body, &chatResp); err != nil {
					t.Fatalf("expected valid chat completion response on fallback, got: %s", string(body))
				}
				if len(chatResp.Choices) == 0 {
					t.Error("expected at least 1 choice on fallback completion")
				}
			}
		})
	}
}

// TestTier2_ExtremeParameters 测试极端参数（temperature, max_tokens）容错性
func TestTier2_ExtremeParameters(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	extremeCases := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name: "NegativeTemperature",
			params: map[string]interface{}{
				"temperature": -1.5,
			},
		},
		{
			name: "HighTemperature",
			params: map[string]interface{}{
				"temperature": 5.0,
			},
		},
		{
			name: "ZeroMaxTokens",
			params: map[string]interface{}{
				"max_tokens": 0,
			},
		},
		{
			name: "ExtremeMaxTokens",
			params: map[string]interface{}{
				"max_tokens": 99999999,
			},
		},
	}

	for _, ec := range extremeCases {
		t.Run(ec.name, func(t *testing.T) {
			reqPayload := map[string]interface{}{
				"model": "Seed-Code",
				"messages": []map[string]string{
					{"role": "user", "content": "Ping with extreme parameter"},
				},
			}
			for k, v := range ec.params {
				reqPayload[k] = v
			}

			resp, body := sendRequest(t, "POST", tg.BaseURL+"/v1/chat/completions", reqPayload, nil)

			// 验证网关保持健壮（200 成功或 400 提示参数越界，严禁 500 崩溃）
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("extreme parameter %s caused unexpected status %d, body: %s", ec.name, resp.StatusCode, string(body))
			}
		})
	}
}
