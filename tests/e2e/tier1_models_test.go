package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// OpenAI 模型列表响应结构体
type ModelListResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID            string `json:"id"`
		Object        string `json:"object"`
		Created       int64  `json:"created"`
		OwnedBy       string `json:"owned_by"`
		DisplayName   string `json:"display_name,omitempty"`
		MaxTokens     int    `json:"max_tokens,omitempty"`
		ContextWindow int    `json:"context_window,omitempty"`
	} `json:"data"`
}

// OpenAI 聊天响应结构体
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int `json:"index"`
		Message      struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
		} `json:"message"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// TestTier1_ModelsIntegrity 测试 /v1/models 接口的完整性与模型清单
func TestTier1_ModelsIntegrity(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	resp, body := sendRequest(t, "GET", tg.BaseURL+"/v1/models", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/models returned status %d, body: %s", resp.StatusCode, string(body))
	}

	var modelList ModelListResponse
	if err := json.Unmarshal(body, &modelList); err != nil {
		t.Fatalf("Failed to parse /v1/models JSON response: %v", err)
	}

	if modelList.Object != "list" {
		t.Errorf("expected object 'list', got %q", modelList.Object)
	}
	if len(modelList.Data) == 0 {
		t.Fatal("expected non-empty models list, got 0 models")
	}

	idMap := make(map[string]bool)
	for _, m := range modelList.Data {
		idMap[m.ID] = true
		if m.Object != "model" {
			t.Errorf("model %s object is %q, want 'model'", m.ID, m.Object)
		}
		if m.OwnedBy == "" {
			t.Errorf("model %s owned_by is empty", m.ID)
		}
	}

	// 1. 验证 16 个新一代内置核心模型存在于列表
	for _, expectedModel := range NewBuiltinModels {
		if !idMap[expectedModel] {
			t.Errorf("Builtin model %q not found in /v1/models list", expectedModel)
		}
	}

	// 2. 验证 5 个存量预设模型存在于列表
	for _, expectedModel := range LegacyPresetModels {
		if !idMap[expectedModel] {
			t.Errorf("Legacy model %q not found in /v1/models list", expectedModel)
		}
	}

	if len(modelList.Data) != 21 {
		t.Errorf("expected exactly 21 models in /v1/models list, got %d", len(modelList.Data))
	}

	t.Logf("Tier 1: /v1/models verification passed, total models: %d", len(modelList.Data))
}

// TestTier1_LegacyPresetModelsCompletion 覆盖 5 个存量预设模型的合法非流式请求响应
func TestTier1_LegacyPresetModelsCompletion(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	for _, model := range LegacyPresetModels {
		t.Run("LegacyModel_"+model, func(t *testing.T) {
			reqPayload := map[string]interface{}{
				"model": model,
				"messages": []map[string]string{
					{"role": "user", "content": fmt.Sprintf("Hello from legacy test for %s", model)},
				},
				"stream": false,
			}

			resp, body := sendRequest(t, "POST", tg.BaseURL+"/v1/chat/completions", reqPayload, nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("POST /v1/chat/completions for model %s returned status %d, body: %s", model, resp.StatusCode, string(body))
			}

			var chatResp ChatCompletionResponse
			if err := json.Unmarshal(body, &chatResp); err != nil {
				t.Fatalf("Failed to parse chat completion response: %v", err)
			}

			if chatResp.Object != "chat.completion" {
				t.Errorf("expected object 'chat.completion', got %q", chatResp.Object)
			}
			if len(chatResp.Choices) == 0 {
				t.Fatal("expected at least 1 choice in response")
			}
			choice := chatResp.Choices[0]
			if choice.Message.Role != "assistant" {
				t.Errorf("expected role 'assistant', got %q", choice.Message.Role)
			}
			if choice.Message.Content == "" {
				t.Errorf("expected non-empty message content for model %s", model)
			}
		})
	}
}

// TestTier1_NewBuiltinModelsCompletion 覆盖 16 个新内置模型的合法非流式请求响应
func TestTier1_NewBuiltinModelsCompletion(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	for _, model := range NewBuiltinModels {
		t.Run("NewModel_"+model, func(t *testing.T) {
			reqPayload := map[string]interface{}{
				"model": model,
				"messages": []map[string]string{
					{"role": "user", "content": fmt.Sprintf("Hello from builtin test for %s", model)},
				},
				"stream": false,
			}

			resp, body := sendRequest(t, "POST", tg.BaseURL+"/v1/chat/completions", reqPayload, nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("POST /v1/chat/completions for new model %s returned status %d, body: %s", model, resp.StatusCode, string(body))
			}

			var chatResp ChatCompletionResponse
			if err := json.Unmarshal(body, &chatResp); err != nil {
				t.Fatalf("Failed to parse chat completion response: %v", err)
			}

			if chatResp.Object != "chat.completion" {
				t.Errorf("expected object 'chat.completion', got %q", chatResp.Object)
			}
			if len(chatResp.Choices) == 0 {
				t.Fatal("expected at least 1 choice in response")
			}
			choice := chatResp.Choices[0]
			if choice.Message.Role != "assistant" {
				t.Errorf("expected role 'assistant', got %q", choice.Message.Role)
			}
			if choice.Message.Content == "" {
				t.Errorf("expected non-empty message content for new model %s", model)
			}
		})
	}
}
