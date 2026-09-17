//go:build ignore

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature,omitempty"`
}

type Delta struct {
	Role             string `json:"role,omitempty"`
	Content          string `json:"content,omitempty"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

type Choice struct {
	Index        int    `json:"index"`
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type ModelVerificationResult struct {
	ModelName        string
	HTTPStatus       int
	ContentType      string
	TotalChunks      int
	ReasoningLength  int
	ContentLength    int
	ReasoningPreview string
	ContentPreview   string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	HasDone          bool
	Elapsed          time.Duration
	TTFT             time.Duration
	Error            string
	Passed           bool
}

func main() {
	tokenBytes, err := os.ReadFile("scripts/current_token.txt")
	if err != nil {
		fmt.Printf("FATAL: Failed to read token: %v\n", err)
		os.Exit(1)
	}
	token := strings.TrimSpace(string(tokenBytes))
	if len(token) == 0 {
		fmt.Println("FATAL: Token is empty")
		os.Exit(1)
	}

	testPort := "9095"
	baseURL := "http://127.0.0.1:" + testPort

	// 准备临时的测试配置文件
	tempCfgPath := filepath.Join("scripts", "temp_proxy_test_cfg.json")
	cfgContent := fmt.Sprintf(`{
  "listen_addr": ":%s",
  "log_level": "info",
  "accounts": [
    {
      "name": "challenger_live_token",
      "token": "%s"
    }
  ]
}`, testPort, token)

	if err := os.WriteFile(tempCfgPath, []byte(cfgContent), 0644); err != nil {
		fmt.Printf("FATAL: Failed to write temp config: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempCfgPath)

	// 启动 trae-proxy 进程
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "./trae-proxy.exe", "-config", tempCfgPath, "-listen", ":"+testPort)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("FATAL: Failed to start trae-proxy: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		cancel()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// 等待网关就绪
	client := &http.Client{Timeout: 60 * time.Second}
	ready := false
	for i := 0; i < 30; i++ {
		time.Sleep(200 * time.Millisecond)
		resp, err := client.Get(baseURL + "/v1/models")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			ready = true
			fmt.Printf(">>> trae-proxy is UP and HEALTHY at %s <<<\n\n", baseURL)
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	if !ready {
		fmt.Println("FATAL: trae-proxy failed to become healthy within 6s")
		os.Exit(1)
	}

	testModels := []string{
		"Doubao-Seed-Code",
		"glm-5.2",
		"DeepSeek-V4.1-Flash",
		"seed_m8",
	}

	results := make([]ModelVerificationResult, 0, len(testModels))

	for _, modelName := range testModels {
		fmt.Println("######################################################################")
		fmt.Printf("  CHALLENGE TEST: Model [%s] via HTTP Reverse Proxy\n", modelName)
		fmt.Println("######################################################################")

		res := verifyModelHTTP(client, baseURL, modelName)
		results = append(results, res)

		fmt.Printf("\n--- [RESULT: %s] ---\n", modelName)
		fmt.Printf("HTTP Status: %d\n", res.HTTPStatus)
		fmt.Printf("Content-Type: %s\n", res.ContentType)
		fmt.Printf("Time to First Token (TTFT): %v\n", res.TTFT)
		fmt.Printf("Total Elapsed: %v\n", res.Elapsed)
		fmt.Printf("Chunks Count: %d\n", res.TotalChunks)
		fmt.Printf("Received [DONE]: %v\n", res.HasDone)
		fmt.Printf("Reasoning Chars: %d\n", res.ReasoningLength)
		fmt.Printf("Content Chars: %d\n", res.ContentLength)
		fmt.Printf("Token Usage: prompt=%d, completion=%d, total=%d\n", res.PromptTokens, res.CompletionTokens, res.TotalTokens)
		if res.ReasoningPreview != "" {
			fmt.Printf("Reasoning Sample: %s\n", res.ReasoningPreview)
		}
		fmt.Printf("Content Sample: %s\n", res.ContentPreview)
		if res.Error != "" {
			fmt.Printf("Error: %s\n", res.Error)
		}
		if res.Passed {
			fmt.Printf(">>> VERDICT FOR %s: PASS <<<\n\n", modelName)
		} else {
			fmt.Printf(">>> VERDICT FOR %s: FAIL <<<\n\n", modelName)
		}
	}

	fmt.Println("======================================================================")
	fmt.Println("                      SUMMARY VERDICT MATRIX                          ")
	fmt.Println("======================================================================")
	allPassed := true
	for _, r := range results {
		statusStr := "PASS"
		if !r.Passed {
			statusStr = "FAIL"
			allPassed = false
		}
		fmt.Printf("[%-20s] Status:%d Chunks:%-3d Reasoning:%-4d Content:%-4d Tokens(P/C/T):%d/%d/%d [DONE]:%-5v -> %s\n",
			r.ModelName, r.HTTPStatus, r.TotalChunks, r.ReasoningLength, r.ContentLength,
			r.PromptTokens, r.CompletionTokens, r.TotalTokens, r.HasDone, statusStr)
	}
	fmt.Println("======================================================================")
	if allPassed {
		fmt.Println(">>> OVERALL STATUS: ALL CRITERIA MET (HTTP 200, SSE STREAMING, USAGE, [DONE]) <<<")
	} else {
		fmt.Println(">>> OVERALL STATUS: SOME CHECKS FAILED <<<")
		os.Exit(1)
	}
}

func verifyModelHTTP(client *http.Client, baseURL, modelName string) ModelVerificationResult {
	reqBody := ChatRequest{
		Model:  modelName,
		Stream: true,
		Messages: []ChatMessage{
			{Role: "user", Content: "你好，请用一句话告诉我你的名字是什么。"},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	startTime := time.Now()
	httpReq, err := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return ModelVerificationResult{ModelName: modelName, Error: fmt.Sprintf("failed to build request: %v", err)}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(httpReq)
	if err != nil {
		return ModelVerificationResult{ModelName: modelName, Error: fmt.Sprintf("http request failed: %v", err)}
	}
	defer resp.Body.Close()

	result := ModelVerificationResult{
		ModelName:   modelName,
		HTTPStatus:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Sprintf("HTTP %d error: %s", resp.StatusCode, string(body))
		return result
	}

	scanner := bufio.NewScanner(resp.Body)
	var reasoningBuilder strings.Builder
	var contentBuilder strings.Builder
	var firstTokenReceived bool

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			dataPayload := strings.TrimPrefix(line, "data: ")
			if dataPayload == "[DONE]" {
				result.HasDone = true
				continue
			}

			var chunk ChatChunk
			if err := json.Unmarshal([]byte(dataPayload), &chunk); err != nil {
				continue
			}

			result.TotalChunks++

			if !firstTokenReceived {
				result.TTFT = time.Since(startTime)
				firstTokenReceived = true
			}

			if chunk.Usage != nil {
				if chunk.Usage.PromptTokens > 0 {
					result.PromptTokens = chunk.Usage.PromptTokens
				}
				if chunk.Usage.CompletionTokens > 0 {
					result.CompletionTokens = chunk.Usage.CompletionTokens
				}
				if chunk.Usage.TotalTokens > 0 {
					result.TotalTokens = chunk.Usage.TotalTokens
				}
			}

			for _, ch := range chunk.Choices {
				if ch.Delta.ReasoningContent != "" {
					reasoningBuilder.WriteString(ch.Delta.ReasoningContent)
					fmt.Print(ch.Delta.ReasoningContent)
				}
				if ch.Delta.Content != "" {
					contentBuilder.WriteString(ch.Delta.Content)
					fmt.Print(ch.Delta.Content)
				}
			}
		}
	}

	result.Elapsed = time.Since(startTime)
	result.ReasoningLength = reasoningBuilder.Len()
	result.ContentLength = contentBuilder.Len()

	fullReasoning := reasoningBuilder.String()
	fullContent := contentBuilder.String()

	if len(fullReasoning) > 100 {
		result.ReasoningPreview = fullReasoning[:100] + "..."
	} else {
		result.ReasoningPreview = fullReasoning
	}

	if len(fullContent) > 120 {
		result.ContentPreview = fullContent[:120] + "..."
	} else {
		result.ContentPreview = fullContent
	}

	// 校验通过条件：
	// 1. HTTP 状态码必须是 200
	// 2. 必须包含 [DONE] 结束符
	// 3. 必须输出实质性内容（Content > 0 或 Reasoning > 0）
	// 4. Content-Type 必须包含 text/event-stream
	// 5. Token 必须有统计（PromptTokens > 0）
	if result.HTTPStatus == 200 &&
		result.HasDone &&
		(result.ContentLength > 0 || result.ReasoningLength > 0) &&
		strings.Contains(result.ContentType, "text/event-stream") &&
		result.PromptTokens > 0 {
		result.Passed = true
	} else {
		result.Passed = false
		if !result.HasDone {
			result.Error += " missing [DONE];"
		}
		if result.ContentLength == 0 && result.ReasoningLength == 0 {
			result.Error += " empty content and reasoning;"
		}
		if result.PromptTokens == 0 {
			result.Error += " prompt_tokens is 0;"
		}
	}

	return result
}
