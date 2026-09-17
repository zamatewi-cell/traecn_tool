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
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type OpenAIResponseNonStream struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func main() {
	tokenBytes, err := os.ReadFile("scripts/current_token.txt")
	if err != nil {
		fmt.Printf("Token read error: %v\n", err)
		os.Exit(1)
	}
	token := strings.TrimSpace(string(tokenBytes))

	testPort := "9096"
	baseURL := "http://127.0.0.1:" + testPort

	tempCfgPath := filepath.Join("scripts", "temp_cfg_9096.json")
	cfg := fmt.Sprintf(`{
  "listen_addr": ":%s",
  "log_level": "warn",
  "accounts": [{"name": "challenger", "token": "%s"}]
}`, testPort, token)

	_ = os.WriteFile(tempCfgPath, []byte(cfg), 0644)
	defer os.Remove(tempCfgPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "./trae-proxy.exe", "-config", tempCfgPath, "-listen", ":"+testPort)
	if err := cmd.Start(); err != nil {
		fmt.Printf("Start gateway failed: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		cancel()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	client := &http.Client{Timeout: 60 * time.Second}
	ready := false
	for i := 0; i < 30; i++ {
		time.Sleep(200 * time.Millisecond)
		resp, err := client.Get(baseURL + "/v1/models")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			ready = true
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	if !ready {
		fmt.Println("Gateway not ready")
		os.Exit(1)
	}
	fmt.Println("=== Gateway Started on " + baseURL + " ===")

	testModels := []string{
		"Doubao-Seed-Code",
		"glm-5.2",
		"DeepSeek-V4.1-Flash",
		"seed_m8",
	}

	fmt.Println("\n=======================================================")
	fmt.Println("PART 1: STREAMING (stream: true) LIVE TEST")
	fmt.Println("=======================================================")

	for _, model := range testModels {
		fmt.Printf("\n>>> Testing Stream for: %s <<<\n", model)
		reqBody, _ := json.Marshal(ChatRequest{
			Model:  model,
			Stream: true,
			Messages: []ChatMessage{
				{Role: "user", Content: "你好，请用一句话告诉我你的名字。"},
			},
		})

		start := time.Now()
		httpReq, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(httpReq)
		if err != nil {
			fmt.Printf("HTTP error: %v\n", err)
			continue
		}

		fmt.Printf("HTTP Status: %d %s\n", resp.StatusCode, resp.Status)
		fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))

		scanner := bufio.NewScanner(resp.Body)
		var sseLines []string
		var hasDone bool
		var reasoningStr strings.Builder
		var contentStr strings.Builder
		var chunkCount int
		var foundUsageInStream bool
		var promptTokens, completionTokens, totalTokens int

		for scanner.Scan() {
			line := scanner.Text()
			if line == "data: [DONE]" {
				hasDone = true
				continue
			}
			if strings.HasPrefix(line, "data: ") {
				chunkCount++
				payload := strings.TrimPrefix(line, "data: ")
				sseLines = append(sseLines, payload)

				var rawMap map[string]interface{}
				if err := json.Unmarshal([]byte(payload), &rawMap); err == nil {
					if u, ok := rawMap["usage"].(map[string]interface{}); ok {
						foundUsageInStream = true
						if pt, ok := u["prompt_tokens"].(float64); ok {
							promptTokens = int(pt)
						}
						if ct, ok := u["completion_tokens"].(float64); ok {
							completionTokens = int(ct)
						}
						if tt, ok := u["total_tokens"].(float64); ok {
							totalTokens = int(tt)
						}
					}
					if choices, ok := rawMap["choices"].([]interface{}); ok && len(choices) > 0 {
						if ch, ok := choices[0].(map[string]interface{}); ok {
							if delta, ok := ch["delta"].(map[string]interface{}); ok {
								if rc, ok := delta["reasoning_content"].(string); ok {
									reasoningStr.WriteString(rc)
								}
								if c, ok := delta["content"].(string); ok {
									contentStr.WriteString(c)
								}
							}
						}
					}
				}
			}
		}
		resp.Body.Close()
		elapsed := time.Since(start)

		fmt.Printf("Elapsed: %v\n", elapsed)
		fmt.Printf("Total SSE Chunks: %d\n", chunkCount)
		fmt.Printf("Received [DONE]: %v\n", hasDone)
		fmt.Printf("Reasoning Chars: %d\n", reasoningStr.Len())
		fmt.Printf("Content Chars: %d\n", contentStr.Len())
		fmt.Printf("Has Usage in Stream SSE: %v (prompt=%d, completion=%d, total=%d)\n", foundUsageInStream, promptTokens, completionTokens, totalTokens)
		if reasoningStr.Len() > 0 {
			rText := reasoningStr.String()
			if len(rText) > 80 {
				rText = rText[:80] + "..."
			}
			fmt.Printf("Reasoning Preview: %s\n", rText)
		}
		cText := contentStr.String()
		if len(cText) > 80 {
			cText = cText[:80] + "..."
		}
		fmt.Printf("Content Preview: %s\n", cText)
		if len(sseLines) > 0 {
			fmt.Printf("Sample First Chunk: %s\n", sseLines[0])
			fmt.Printf("Sample Last Chunk: %s\n", sseLines[len(sseLines)-1])
		}
	}

	fmt.Println("\n=======================================================")
	fmt.Println("PART 2: NON-STREAMING (stream: false) LIVE TEST")
	fmt.Println("=======================================================")

	for _, model := range testModels {
		fmt.Printf("\n>>> Testing Non-Stream for: %s <<<\n", model)
		reqBody, _ := json.Marshal(ChatRequest{
			Model:  model,
			Stream: false,
			Messages: []ChatMessage{
				{Role: "user", Content: "你好，请用一句话告诉我你的名字。"},
			},
		})

		start := time.Now()
		httpReq, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			fmt.Printf("HTTP error: %v\n", err)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		elapsed := time.Since(start)

		fmt.Printf("HTTP Status: %d %s\n", resp.StatusCode, resp.Status)
		fmt.Printf("Elapsed: %v\n", elapsed)

		var nonStreamResp OpenAIResponseNonStream
		if err := json.Unmarshal(bodyBytes, &nonStreamResp); err != nil {
			fmt.Printf("JSON unmarshal error: %v, body=%s\n", err, string(bodyBytes))
			continue
		}

		fmt.Printf("Response ID: %s, Model: %s\n", nonStreamResp.ID, nonStreamResp.Model)
		if len(nonStreamResp.Choices) > 0 {
			ch := nonStreamResp.Choices[0]
			fmt.Printf("Finish Reason: %s\n", ch.FinishReason)
			fmt.Printf("Reasoning Chars: %d\n", len(ch.Message.ReasoningContent))
			fmt.Printf("Content: %s\n", ch.Message.Content)
		}
		fmt.Printf("Usage: PromptTokens=%d, CompletionTokens=%d, TotalTokens=%d\n",
			nonStreamResp.Usage.PromptTokens, nonStreamResp.Usage.CompletionTokens, nonStreamResp.Usage.TotalTokens)
	}
}
