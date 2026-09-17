// Throwaway upstream endpoint probe (not part of the build).
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
)

func main() {
	a, err := auth.LoadTokenFromStorage(auth.DefaultStoragePath())
	if err != nil {
		fmt.Println("load token failed:", err)
		os.Exit(1)
	}
	fmt.Println("token loaded, user:", a.UserID, "expires:", a.ExpiredAt)

	chatBody := `{"model_name":"DeepSeek-V4-Flash","messages":[{"role":"user","content":"hi"}],"stream":true}`
	probes := []struct {
		method, path, body string
	}{
		{"POST", config.EndpointModelList, "{}"},
		{"POST", config.EndpointGetDetailParam, "{}"},
		{"POST", config.EndpointChatCompletion, chatBody},
		{"POST", config.EndpointLLMRawChat, chatBody},
		{"POST", config.EndpointFeatures, "{}"},
		{"GET", "/api/ide/v1/chat_completion", ""},
	}

	client := &http.Client{Timeout: 20 * time.Second}
	for _, p := range probes {
		req, _ := http.NewRequest(p.method, config.AgentDomain+p.path, bytes.NewReader([]byte(p.body)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-IDE-Token", a.Token)
		req.Header.Set("User-Agent", "TraeClient/TTNet")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("%-6s %-40s ERR %v\n", p.method, p.path, err)
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		resp.Body.Close()
		fmt.Printf("%-6s %-40s -> %d  %s\n", p.method, p.path, resp.StatusCode, string(body))
	}
}
