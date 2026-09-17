package e2e_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// OpenAI 流式 Chunk 结构体
type StreamChunkResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role             *string `json:"role,omitempty"`
			Content          *string `json:"content,omitempty"`
			ReasoningContent *string `json:"reasoning_content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// readSSEStream 读取整个 SSE 流，返回所有数据帧与是否正常以 [DONE] 结束
func readSSEStream(t *testing.T, resp *http.Response) (chunks []StreamChunkResponse, gotDone bool) {
	t.Helper()
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			dataContent := strings.TrimPrefix(line, "data: ")
			if dataContent == "[DONE]" {
				gotDone = true
				break
			}

			var chunk StreamChunkResponse
			if err := json.Unmarshal([]byte(dataContent), &chunk); err != nil {
				t.Logf("Warning: SSE chunk not unmarshaled into StreamChunkResponse: %v (raw: %s)", err, dataContent)
				continue
			}
			chunks = append(chunks, chunk)
		}
	}
	return chunks, gotDone
}

// TestTier4_StreamingBasic 测试 stream: true 模式下的基础 SSE 增量推送与协议头
func TestTier4_StreamingBasic(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	modelsToTest := []string{"Seed-Code", "seed_m8"}

	for _, model := range modelsToTest {
		t.Run("StreamBasic_"+model, func(t *testing.T) {
			reqPayload := map[string]interface{}{
				"model": model,
				"messages": []map[string]string{
					{"role": "user", "content": "Tell me a short greeting"},
				},
				"stream": true,
			}

			bs, _ := json.Marshal(reqPayload)
			httpReq, err := http.NewRequest("POST", tg.BaseURL+"/v1/chat/completions", bytes.NewReader(bs))
			if err != nil {
				t.Fatalf("http.NewRequest failed: %v", err)
			}
			httpReq.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(httpReq)
			if err != nil {
				t.Fatalf("client.Do failed: %v", err)
			}

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", resp.StatusCode)
			}

			// 验证 SSE 响应协议头
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "text/event-stream") {
				t.Errorf("expected Content-Type text/event-stream, got %q", contentType)
			}

			chunks, gotDone := readSSEStream(t, resp)
			if !gotDone {
				t.Errorf("Stream for model %s did not terminate with [DONE]", model)
			}
			if len(chunks) == 0 {
				t.Fatalf("No chunks received for model %s", model)
			}

			// 检查第一帧下发了 role: assistant
			hasRoleAssistant := false
			totalContent := ""
			for _, c := range chunks {
				if len(c.Choices) > 0 {
					if c.Choices[0].Delta.Role != nil && *c.Choices[0].Delta.Role == "assistant" {
						hasRoleAssistant = true
					}
					if c.Choices[0].Delta.Content != nil {
						totalContent += *c.Choices[0].Delta.Content
					}
				}
			}

			if !hasRoleAssistant {
				t.Errorf("Expected role 'assistant' in initial chunk deltas")
			}
			if totalContent == "" {
				t.Errorf("Expected non-empty aggregated content from stream")
			}

			t.Logf("Tier 4: StreamingBasic passed for %s, total chunks: %d, text: %q", model, len(chunks), totalContent)
		})
	}
}

// TestTier4_StreamingReasoningAndContent 验证流式模式下思考过程与正文内容的分离
func TestTier4_StreamingReasoningAndContent(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	// 使用支持思考过程的模型
	thinkingModel := "Doubao_1_5_thinking_pro"

	reqPayload := map[string]interface{}{
		"model": thinkingModel,
		"messages": []map[string]string{
			{"role": "user", "content": "Explain 1+1 with step by step reasoning"},
		},
		"stream": true,
	}

	bs, _ := json.Marshal(reqPayload)
	httpReq, err := http.NewRequest("POST", tg.BaseURL+"/v1/chat/completions", bytes.NewReader(bs))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("client.Do failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	chunks, gotDone := readSSEStream(t, resp)
	if !gotDone {
		t.Errorf("Stream did not terminate with [DONE]")
	}

	totalReasoning := ""
	totalContent := ""

	for _, c := range chunks {
		if len(c.Choices) > 0 {
			if c.Choices[0].Delta.ReasoningContent != nil {
				totalReasoning += *c.Choices[0].Delta.ReasoningContent
			}
			if c.Choices[0].Delta.Content != nil {
				totalContent += *c.Choices[0].Delta.Content
			}
		}
	}

	t.Logf("Reasoning output captured: %q", totalReasoning)
	t.Logf("Content output captured: %q", totalContent)

	if totalReasoning == "" {
		t.Errorf("Expected reasoning_content to be non-empty for thinking model %s", thinkingModel)
	}
	if totalContent == "" {
		t.Errorf("Expected content to be non-empty")
	}

	t.Logf("Tier 4: Streaming reasoning and content passed")
}

// TestTier4_StreamingDoneTermination 验证流式结束时携带 finish_reason 与 [DONE] 标记
func TestTier4_StreamingDoneTermination(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	reqPayload := map[string]interface{}{
		"model": "Seed-Code",
		"messages": []map[string]string{
			{"role": "user", "content": "Just say done"},
		},
		"stream": true,
	}

	bs, _ := json.Marshal(reqPayload)
	httpReq, err := http.NewRequest("POST", tg.BaseURL+"/v1/chat/completions", bytes.NewReader(bs))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("client.Do failed: %v", err)
	}
	defer resp.Body.Close()

	chunks, gotDone := readSSEStream(t, resp)
	if !gotDone {
		t.Fatalf("Stream did not emit [DONE] marker at the end")
	}

	hasFinishReason := false
	for _, c := range chunks {
		if len(c.Choices) > 0 && c.Choices[0].FinishReason != nil {
			if *c.Choices[0].FinishReason == "stop" {
				hasFinishReason = true
				break
			}
		}
	}

	if !hasFinishReason {
		t.Errorf("Expected at least one chunk to contain finish_reason='stop'")
	}

	t.Logf("Tier 4: Streaming [DONE] termination verified")
}
