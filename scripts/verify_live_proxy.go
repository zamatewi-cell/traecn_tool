//go:build ignore

package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

func main() {
	tokenBytes, err := os.ReadFile("scripts/current_token.txt")
	if err != nil {
		fmt.Printf("Failed to read token: %v\n", err)
		os.Exit(1)
	}
	token := strings.TrimSpace(string(tokenBytes))
	if len(token) == 0 {
		fmt.Println("Token is empty")
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	pool := auth.NewPool(&auth.PoolOptions{
		Logger: logger,
	})
	pool.AddAccountWithToken("real_token", token)

	p := proxy.NewTraeProxy(pool, logger)

	testModels := []string{
		"Doubao-Seed-Code",
		"glm-5.2",
		"DeepSeek-V4.1-Flash",
		"seed_m8",
	}

	for _, modelName := range testModels {
		fmt.Println("==================================================")
		fmt.Printf("Testing Model: %s\n", modelName)
		fmt.Println("==================================================")

		req := &proxy.ChatCompletionRequest{
			ModelName: modelName,
			Stream:    true,
			Messages: []proxy.Message{
				{Role: "user", Content: "Hello, reply in one short sentence: what model are you?"},
			},
		}

		var outputText strings.Builder
		var reasoningText strings.Builder
		var eventCount int
		var receivedFinish bool

		startTime := time.Now()
		err := p.ChatCompletion(req, func(evt *proxy.StreamEvent) error {
			eventCount++
			switch evt.Type {
			case proxy.EventReasoning:
				reasoningText.WriteString(evt.Reasoning)
				fmt.Printf("[Reasoning] %s", evt.Reasoning)
			case proxy.EventText:
				outputText.WriteString(evt.Text)
				fmt.Printf("[Text] %s", evt.Text)
			case proxy.EventUsage:
				if evt.Usage != nil {
					fmt.Printf("\n[Usage] prompt=%d completion=%d total=%d\n",
						evt.Usage.PromptTokens, evt.Usage.CompletionTokens, evt.Usage.TotalTokens)
				}
			case proxy.EventFinish:
				receivedFinish = true
				fmt.Printf("\n[Finish] reason=%s\n", evt.FinishReason)
			case proxy.EventError:
				fmt.Printf("\n[Error Event] %v\n", evt.Err)
			}
			return nil
		})
		elapsed := time.Since(startTime)

		fmt.Printf("\n--- Model %s Result ---\n", modelName)
		fmt.Printf("Elapsed: %v\n", elapsed)
		fmt.Printf("Events received: %d\n", eventCount)
		fmt.Printf("Finish received: %v\n", receivedFinish)
		if reasoningText.Len() > 0 {
			fmt.Printf("Total Reasoning: %s\n", reasoningText.String())
		}
		fmt.Printf("Total Output: %s\n", outputText.String())

		if err != nil {
			fmt.Printf("Execution Error: %v\n", err)
		} else if outputText.Len() > 0 || reasoningText.Len() > 0 {
			fmt.Printf(">>> SUCCESS for model %s <<<\n", modelName)
		} else {
			fmt.Printf(">>> FAILED for model %s (output_len=%d) <<<\n", modelName, outputText.Len())
		}
		fmt.Println()
	}
}