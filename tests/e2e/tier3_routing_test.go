package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/zamatewi-cell/traecn_tool/internal/config"
)

// TestTier3_LegacyChannelRouting 验证存量预设模型严格派发至 /api/ide/v1/llm_raw_chat
func TestTier3_LegacyChannelRouting(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	if tg.MockUpstream == nil {
		t.Skip("Testing against external gateway, skip internal upstream hit assertion")
	}

	testModel := "seed_m8"
	reqPayload := map[string]interface{}{
		"model": testModel,
		"messages": []map[string]string{
			{"role": "user", "content": "Route verification for legacy model"},
		},
	}

	tg.MockUpstream.ClearHits()
	resp, body := sendRequest(t, "POST", tg.BaseURL+"/v1/chat/completions", reqPayload, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body: %s", resp.StatusCode, string(body))
	}

	legacyHits := tg.MockUpstream.GetHits(config.EndpointLLMRawChat)
	if len(legacyHits) == 0 {
		t.Fatalf("expected at least 1 hit on %s, got 0", config.EndpointLLMRawChat)
	}

	lastHit := legacyHits[len(legacyHits)-1]
	if lastHit.Endpoint != config.EndpointLLMRawChat {
		t.Errorf("expected endpoint %s, got %s", config.EndpointLLMRawChat, lastHit.Endpoint)
	}
	if agentHits := tg.MockUpstream.GetHits(config.EndpointAgentCreateTask); len(agentHits) > 0 {
		t.Errorf("legacy model %s must not hit AgentTask endpoint", testModel)
	}
	if lastHit.Headers.Get("get-svc") != "1" {
		t.Errorf("expected get-svc=1 in legacy upstream request header")
	}
	if lastHit.Headers.Get("x-request-pin") == "" || lastHit.Headers.Get("x-requested-at") == "" {
		t.Errorf("missing x-request-pin or x-requested-at in upstream request")
	}

	t.Logf("Tier 3: Legacy channel routing verified for %s (hit %s)", testModel, config.EndpointLLMRawChat)
}

// TestTier3_AgentTaskChannelRouting 验证新内置模型能够派发至相应通道并正常响应
func TestTier3_AgentTaskChannelRouting(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	testModels := []string{"DeepSeek-V4.1-Flash", "GLM-5.3", "Kimi-K3"}

	for _, model := range testModels {
		t.Run("Route_"+model, func(t *testing.T) {
			if tg.MockUpstream != nil {
				tg.MockUpstream.ClearHits()
			}

			reqPayload := map[string]interface{}{
				"model": model,
				"messages": []map[string]string{
					{"role": "user", "content": fmt.Sprintf("Route verification for %s", model)},
				},
			}

			resp, body := sendRequest(t, "POST", tg.BaseURL+"/v1/chat/completions", reqPayload, nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 OK for %s, got %d, body: %s", model, resp.StatusCode, string(body))
			}

			var chatResp ChatCompletionResponse
			if err := json.Unmarshal(body, &chatResp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
				t.Fatalf("expected valid choice content for %s", model)
			}

			if tg.MockUpstream != nil {
				allHits := tg.MockUpstream.AllHits()
				if len(allHits) == 0 {
					t.Fatalf("expected upstream to receive request for %s, got 0 hits", model)
				}
				lastHit := allHits[len(allHits)-1]
				if lastHit.Endpoint != config.EndpointAgentCreateTask {
					t.Fatalf("Model %s routed to %s, expected %s", model, lastHit.Endpoint, config.EndpointAgentCreateTask)
				}
				if legacyHits := tg.MockUpstream.GetHits(config.EndpointLLMRawChat); len(legacyHits) > 0 {
					t.Fatalf("Model %s hit legacy endpoint %s, expected AgentTask only", model, config.EndpointLLMRawChat)
				}
				t.Logf("Model %s routed upstream to: %s", model, lastHit.Endpoint)
			}
		})
	}
}

// TestTier3_AlternatingModelRouting 连续交替发起旧模型与新模型请求，验证路由隔离与无状态污染
func TestTier3_AlternatingModelRouting(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	sequence := []struct {
		model            string
		expectedEndpoint string
	}{
		{"seed_m8", config.EndpointLLMRawChat},
		{"DeepSeek-V4.1-Flash", config.EndpointAgentCreateTask},
		{"deepseek-R1", config.EndpointLLMRawChat},
		{"GLM-5.3", config.EndpointAgentCreateTask},
		{"Doubao_1_5_thinking_pro", config.EndpointLLMRawChat},
		{"Kimi-K3", config.EndpointAgentCreateTask},
	}

	for i, item := range sequence {
		t.Run(fmt.Sprintf("Step%d_%s", i+1, item.model), func(t *testing.T) {
			if tg.MockUpstream != nil {
				tg.MockUpstream.ClearHits()
			}

			reqPayload := map[string]interface{}{
				"model": item.model,
				"messages": []map[string]string{
					{"role": "user", "content": fmt.Sprintf("Alternating test step %d with %s", i+1, item.model)},
				},
			}

			resp, body := sendRequest(t, "POST", tg.BaseURL+"/v1/chat/completions", reqPayload, nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Step %d (%s) failed with status %d: %s", i+1, item.model, resp.StatusCode, string(body))
			}

			var chatResp ChatCompletionResponse
			if err := json.Unmarshal(body, &chatResp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if len(chatResp.Choices) == 0 {
				t.Fatalf("Step %d (%s) returned empty choices", i+1, item.model)
			}

			if tg.MockUpstream != nil {
				hits := tg.MockUpstream.AllHits()
				if len(hits) == 0 {
					t.Fatalf("expected upstream hits for %s", item.model)
				}
				lastHit := hits[len(hits)-1]
				if lastHit.Endpoint != item.expectedEndpoint {
					t.Errorf("model %s routed to %s, want %s", item.model, lastHit.Endpoint, item.expectedEndpoint)
				}
			}
		})
	}
}

// TestTier3_ConcurrentMixedRouting 高并发混合请求隔离性测试（并发发起新老模型请求）
func TestTier3_ConcurrentMixedRouting(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	modelsToTest := []string{
		"seed_m8",
		"DeepSeek-V4.1-Flash",
		"deepseek-R1",
		"GLM-5.3",
		"Seed-Code",
		"Kimi-K3",
		"MiniMax-M3",
		"deepseek-V3-0324",
	}

	concurrency := 16
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		model := modelsToTest[i%len(modelsToTest)]
		idx := i
		go func(m string, index int) {
			defer wg.Done()
			uniquePrompt := fmt.Sprintf("Concurrent request #%d for model %s", index, m)
			reqPayload := map[string]interface{}{
				"model": m,
				"messages": []map[string]string{
					{"role": "user", "content": uniquePrompt},
				},
			}

			resp, body := sendRequest(t, "POST", tg.BaseURL+"/v1/chat/completions", reqPayload, nil)
			if resp.StatusCode != http.StatusOK {
				errCh <- fmt.Errorf("concurrent request #%d (%s) returned %d: %s", index, m, resp.StatusCode, string(body))
				return
			}

			var chatResp ChatCompletionResponse
			if err := json.Unmarshal(body, &chatResp); err != nil {
				errCh <- fmt.Errorf("concurrent request #%d unmarshal error: %v", index, err)
				return
			}

			if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
				errCh <- fmt.Errorf("concurrent request #%d returned empty content", index)
				return
			}
		}(model, idx)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("Concurrency failure: %v", err)
	}

	t.Logf("Tier 3: Concurrent mixed routing passed with %d goroutines", concurrency)
}
