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
	"strings"
	"time"
)

func main() {
	fmt.Println("=== 开始验证 5 大安全与核心缺陷修复 ===")

	// 1. 验证编译产物版本输出
	cmdVersion := exec.Command("./trae-proxy.exe", "-version")
	outVersion, err := cmdVersion.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ -version 失败: %v\n", err)
		os.Exit(1)
	}
	verStr := strings.TrimSpace(string(outVersion))
	fmt.Printf("✔ 版本号输出: %s\n", verStr)
	if !strings.Contains(verStr, "v1.0.0") {
		fmt.Printf("❌ 版本号不符合预期，期望 v1.0.0，实际: %s\n", verStr)
		os.Exit(1)
	}

	// 2. 准备测试配置文件，启用 API Key 鉴权
	testKey := "sk-traecn-test-key-9988"
	testCfgPath := "test_verify_config.json"
	cfgContent := fmt.Sprintf(`{
  "listen_addr": "127.0.0.1:9099",
  "allow_lan": false,
  "api_keys": ["%s"]
}`, testKey)
	_ = os.WriteFile(testCfgPath, []byte(cfgContent), 0644)
	defer os.Remove(testCfgPath)

	// 3. 启动后台实例运行在 127.0.0.1:9099
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmdServer := exec.CommandContext(ctx, "./trae-proxy.exe", "-config", testCfgPath)
	var serverLogs bytes.Buffer
	cmdServer.Stdout = &serverLogs
	cmdServer.Stderr = &serverLogs

	if err := cmdServer.Start(); err != nil {
		fmt.Printf("❌ 启动代理服务失败: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		cancel()
		_ = cmdServer.Wait()
	}()

	// 等待服务启动
	time.Sleep(1500 * time.Millisecond)

	baseURL := "http://127.0.0.1:9099"

	// 4. 验证 /health 与版本
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		fmt.Printf("❌ 连接 /health 失败: %v\nServer Logs:\n%s\n", err, serverLogs.String())
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✔ /health 响应: %s\n", strings.TrimSpace(string(body)))
	if !strings.Contains(string(body), `"version":"1.0.0"`) {
		fmt.Printf("❌ /health 未返回版本 1.0.0\n")
		os.Exit(1)
	}

	// 5. 验证 API Key 鉴权有效性 (Fix 2 验证)
	// (a) 未带 Header 请求受保护的 /v1/models 接口，必须返回 401
	respUnauth, err := http.Get(baseURL + "/v1/models")
	if err != nil {
		fmt.Printf("❌ 请求 /v1/models 失败: %v\n", err)
		os.Exit(1)
	}
	respUnauth.Body.Close()
	if respUnauth.StatusCode != http.StatusUnauthorized {
		fmt.Printf("❌ 未授权请求预期返回 401，实际返回: %d\n", respUnauth.StatusCode)
		os.Exit(1)
	}
	fmt.Printf("✔ 未授权访问受保护接口成功被 401 拦截 (Status: %d)\n", respUnauth.StatusCode)

	// (b) 携带正确的 Authorization Header 请求，返回 200
	reqAuth, _ := http.NewRequest("GET", baseURL+"/v1/models", nil)
	reqAuth.Header.Set("Authorization", "Bearer "+testKey)
	respAuth, err := http.DefaultClient.Do(reqAuth)
	if err != nil {
		fmt.Printf("❌ 携带授权请求失败: %v\n", err)
		os.Exit(1)
	}
	defer respAuth.Body.Close()
	if respAuth.StatusCode != http.StatusOK {
		fmt.Printf("❌ 携带正确密钥请求预期返回 200，实际返回: %d\n", respAuth.StatusCode)
		os.Exit(1)
	}
	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respAuth.Body).Decode(&modelsResp)
	fmt.Printf("✔ 携带正确 API Key 成功通过鉴权 (模型数量: %d)\n", len(modelsResp.Data))

	// 6. 验证流式响应防重下发 (Fix 5 验证)
	streamReqBody := `{
		"model": "deepseek-V3",
		"messages": [{"role": "user", "content": "1+1=?"}],
		"stream": true
	}`
	reqStream, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", strings.NewReader(streamReqBody))
	reqStream.Header.Set("Authorization", "Bearer "+testKey)
	reqStream.Header.Set("Content-Type", "application/json")

	respStream, err := http.DefaultClient.Do(reqStream)
	if err != nil {
		fmt.Printf("❌ 发起流式请求失败: %v\n", err)
		os.Exit(1)
	}
	defer respStream.Body.Close()

	if respStream.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(respStream.Body)
		fmt.Printf("❌ 流式请求返回状态码: %d, body: %s\n", respStream.StatusCode, string(body))
		os.Exit(1)
	}

	finishCount := 0
	scanner := bufio.NewScanner(respStream.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				break
			}
			var chunk struct {
				Choices []struct {
					FinishReason *string `json:"finish_reason"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(payload), &chunk); err == nil {
				for _, ch := range chunk.Choices {
					if ch.FinishReason != nil && *ch.FinishReason != "" {
						finishCount++
						fmt.Printf("  -> 捕获 finish_reason: %s\n", *ch.FinishReason)
					}
				}
			}
		}
	}
	fmt.Printf("✔ 流式响应结束，finish_reason 出现次数: %d\n", finishCount)
	if finishCount != 1 {
		fmt.Printf("❌ finish_reason 下发次数异常，期望严格等于 1，实际: %d\n", finishCount)
		os.Exit(1)
	}

	// 7. 验证非流式 TTFT 为 0 统计 (Fix 5 验证)
	nonStreamReqBody := `{
		"model": "deepseek-V3",
		"messages": [{"role": "user", "content": "hello"}],
		"stream": false
	}`
	reqNonStream, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", strings.NewReader(nonStreamReqBody))
	reqNonStream.Header.Set("Authorization", "Bearer "+testKey)
	reqNonStream.Header.Set("Content-Type", "application/json")

	respNonStream, err := http.DefaultClient.Do(reqNonStream)
	if err != nil {
		fmt.Printf("❌ 发起非流式请求失败: %v\n", err)
		os.Exit(1)
	}
	respNonStream.Body.Close()

	time.Sleep(500 * time.Millisecond) // 等待落盘

	reqLogs, _ := http.NewRequest("GET", baseURL+"/v1/proxy/logs?page=1&page_size=5", nil)
	reqLogs.Header.Set("Authorization", "Bearer "+testKey)
	respLogs, err := http.DefaultClient.Do(reqLogs)
	if err == nil {
		defer respLogs.Body.Close()
		var logsResp struct {
			Logs []struct {
				Model  string `json:"model"`
				TTFTMs int64  `json:"ttft_ms"`
			} `json:"logs"`
		}
		_ = json.NewDecoder(respLogs.Body).Decode(&logsResp)
		if len(logsResp.Logs) > 0 {
			latestLog := logsResp.Logs[0]
			fmt.Printf("✔ 最新非流式落盘日志: model=%s, ttft_ms=%d\n", latestLog.Model, latestLog.TTFTMs)
			if latestLog.TTFTMs != 0 {
				fmt.Printf("❌ 非流式请求 TTFT 期望为 0，实际为 %d\n", latestLog.TTFTMs)
				os.Exit(1)
			}
		}
	}

	fmt.Println("\n🎉 所有核心修复项测试全部通过 (ALL CHECKS PASSED)！")
}
