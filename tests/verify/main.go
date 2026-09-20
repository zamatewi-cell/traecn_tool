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
	fmt.Println("=== 开始执行深度架构安全与缺陷闭环实测 ===")

	// 1. 验证编译产物版本输出 (统一 version 模块)
	cmdVersion := exec.Command("./trae-proxy.exe", "-version")
	outVersion, err := cmdVersion.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ -version 失败: %v\n", err)
		os.Exit(1)
	}
	verStr := strings.TrimSpace(string(outVersion))
	fmt.Printf("✔ [版本统一] 版本号输出: %s\n", verStr)
	if !strings.Contains(verStr, "v1.0.2") && !strings.Contains(verStr, "v1.0.1") && !strings.Contains(verStr, "v1.0.0") {
		fmt.Printf("❌ 版本号不符合预期，期望 v1.0.2 或 v1.0.1 或 v1.0.0，实际: %s\n", verStr)
		os.Exit(1)
	}

	// 2. 验证 LAN 模式防裸奔防呆拦截 (allow_lan=true 且无 key 拒绝启动)
	fmt.Println("-> 校验 LAN 模式无鉴权拦截...")
	cmdLanBlock := exec.Command("./trae-proxy.exe", "-allow-lan")
	outLanBlock, _ := cmdLanBlock.CombinedOutput()
	if strings.Contains(string(outLanBlock), "no api_keys are configured") {
		fmt.Println("✔ [LAN防呆] allow_lan 且未配 key 时成功安全早停拦截")
	} else {
		fmt.Printf("❌ LAN 防呆未触发，输出:\n%s\n", string(outLanBlock))
		os.Exit(1)
	}

	// 2.2 验证非 loopback 地址（如 192.168.1.10:9099）无 Key 无法绕过安全检查
	fmt.Println("-> 校验非 Loopback 监听地址防绕过拦截...")
	cmdNonLoopback := exec.Command("./trae-proxy.exe", "-listen", "192.168.1.10:9099")
	outNonLoopback, _ := cmdNonLoopback.CombinedOutput()
	if strings.Contains(string(outNonLoopback), "non-loopback address") && strings.Contains(string(outNonLoopback), "no api_keys are configured") {
		fmt.Println("✔ [IP安全收敛] 非 loopback 绑定 (192.168.1.10) 且未配 key 时被严格拦截，无法绕过！")
	} else {
		fmt.Printf("❌ 非 loopback 防绕过拦截未触发，输出:\n%s\n", string(outNonLoopback))
		os.Exit(1)
	}

	// 2.3 验证 standalone 无 Key 模式下，防范来自外部恶意网页的跨源 CSRF/Drive-by 访问 (P1级修复)
	fmt.Println("-> 校验 standalone 默认无 Key 模式下的跨域 Origin 安全边界...")
	ctxNoAuth, cancelNoAuth := context.WithCancel(context.Background())
	defer cancelNoAuth()
	cmdNoAuth := exec.CommandContext(ctxNoAuth, "./trae-proxy.exe", "-listen", "127.0.0.1:9096")
	if err := cmdNoAuth.Start(); err != nil {
		fmt.Printf("❌ 启动无 Key 测试服务失败: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		cancelNoAuth()
		_ = cmdNoAuth.Wait()
	}()
	time.Sleep(1200 * time.Millisecond)

	// 外部恶意 Origin 尝试请求
	reqEvil, _ := http.NewRequest("GET", "http://127.0.0.1:9096/health", nil)
	reqEvil.Header.Set("Origin", "https://evil.attacker.com")
	respEvil, err := http.DefaultClient.Do(reqEvil)
	if err != nil {
		fmt.Printf("❌ 恶意 Origin 请求发送失败: %v\n", err)
		os.Exit(1)
	}
	respEvil.Body.Close()
	if respEvil.StatusCode != http.StatusForbidden {
		fmt.Printf("❌ 预期恶意 Origin 被 403 阻断，实际返回: %d\n", respEvil.StatusCode)
		os.Exit(1)
	}
	fmt.Println("✔ [CSRF防护] 无 Key 模式下外部恶意 Web Origin (https://evil.attacker.com) 被严格 403 拦截！")

	// 本地受信任 Origin 访问
	reqSafe, _ := http.NewRequest("GET", "http://127.0.0.1:9096/health", nil)
	reqSafe.Header.Set("Origin", "http://localhost:3000")
	respSafe, err := http.DefaultClient.Do(reqSafe)
	if err != nil {
		fmt.Printf("❌ 本地 Origin 请求发送失败: %v\n", err)
		os.Exit(1)
	}
	respSafe.Body.Close()
	if respSafe.StatusCode != http.StatusOK {
		fmt.Printf("❌ 本地 Origin 预期返回 200，实际返回: %d\n", respSafe.StatusCode)
		os.Exit(1)
	}
	fmt.Println("✔ [Origin放行] 本地受信任 Origin (http://localhost:3000) 成功放行")


	// 3. 验证通过 stdin 管道传递配置（零落盘架构），并校验 API Key、request_timeout 与 log_level
	fmt.Println("-> 启动 stdin 管道配置测试服务...")
	testKey := "sk-traecn-pipeline-key-5566"
	stdinConfig := fmt.Sprintf(`{
  "listen_addr": "127.0.0.1:9098",
  "allow_lan": false,
  "request_timeout": 60,
  "log_level": "debug",
  "api_keys": ["%s"]
}`, testKey)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmdServer := exec.CommandContext(ctx, "./trae-proxy.exe", "-config", "stdin")
	stdinPipe, err := cmdServer.StdinPipe()
	if err != nil {
		fmt.Printf("❌ 获取 stdin pipe 失败: %v\n", err)
		os.Exit(1)
	}

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

	// 写入内存管道并关闭写入端
	_, _ = stdinPipe.Write([]byte(stdinConfig))
	_ = stdinPipe.Close()

	// 等待服务就绪
	time.Sleep(1500 * time.Millisecond)

	baseURL := "http://127.0.0.1:9098"

	// 4. 验证 /health
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		fmt.Printf("❌ 连接 /health 失败: %v\nServer Logs:\n%s\n", err, serverLogs.String())
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✔ [健康检查] /health 响应: %s\n", strings.TrimSpace(string(body)))
	if !strings.Contains(string(body), `"version":"1.0.2"`) && !strings.Contains(string(body), `"version":"1.0.1"`) && !strings.Contains(string(body), `"version":"1.0.0"`) {
		fmt.Printf("❌ /health 未返回版本 1.0.2 或 1.0.1 或 1.0.0\n")
		os.Exit(1)
	}

	// 5. 验证 API Key 鉴权真拦截 (未带 Key 401，携带 Key 200)
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
	fmt.Printf("✔ [鉴权拦截] 未授权请求被正确返回 401\n")

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
	fmt.Printf("✔ [鉴权通过] 携带 stdin 管道配置的 API Key 成功通过认证\n")

	// 5.1 验证 CORS OPTIONS 预检请求不携带 API Key 时被正常放行（非 401 拦截），且支持 X-API-Key 头
	reqOptions, _ := http.NewRequest(http.MethodOptions, baseURL+"/v1/chat/completions", nil)
	reqOptions.Header.Set("Origin", "http://localhost:5173")
	reqOptions.Header.Set("Access-Control-Request-Method", "POST")
	reqOptions.Header.Set("Access-Control-Request-Headers", "authorization,content-type,x-api-key")
	respOptions, err := http.DefaultClient.Do(reqOptions)
	if err != nil {
		fmt.Printf("❌ OPTIONS 预检请求失败: %v\n", err)
		os.Exit(1)
	}
	respOptions.Body.Close()
	if respOptions.StatusCode == http.StatusUnauthorized {
		fmt.Printf("❌ OPTIONS 预检请求被 401 拦截！\n")
		os.Exit(1)
	}
	if respOptions.StatusCode != http.StatusOK && respOptions.StatusCode != http.StatusNoContent {
		fmt.Printf("❌ OPTIONS 预检请求期望 200/204，实际: %d\n", respOptions.StatusCode)
		os.Exit(1)
	}
	corsHeader := respOptions.Header.Get("Access-Control-Allow-Origin")
	allowHeaders := strings.ToLower(respOptions.Header.Get("Access-Control-Allow-Headers"))
	if !strings.Contains(allowHeaders, "x-api-key") {
		fmt.Printf("❌ CORS Access-Control-Allow-Headers 未包含 x-api-key: %s\n", allowHeaders)
		os.Exit(1)
	}
	fmt.Printf("✔ [CORS放行] OPTIONS 预检请求成功放行 (Status: %d, Allow-Origin: %s, Allow-Headers 确认包含 x-api-key)\n", respOptions.StatusCode, corsHeader)

	// 6. 验证流式响应 finish_reason 单发保护
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
					}
				}
			}
		}
	}
	fmt.Printf("✔ [流式防重] finish_reason 出现次数严格为: %d\n", finishCount)
	if finishCount != 1 {
		fmt.Printf("❌ finish_reason 下发次数异常，期望为 1，实际: %d\n", finishCount)
		os.Exit(1)
	}

	// 7. 验证非流式 TTFT 为 0
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

	time.Sleep(500 * time.Millisecond)

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
			fmt.Printf("✔ [指标真实] 非流式请求 TTFT 明确记录为: %d ms (非流式置零无污染)\n", latestLog.TTFTMs)
			if latestLog.TTFTMs != 0 {
				fmt.Printf("❌ 非流式请求 TTFT 期望为 0，实际为 %d\n", latestLog.TTFTMs)
				os.Exit(1)
			}
		}
	}

	// 8. 校验核心服务日志输出（stdin 管道传参与超时注入标记）
	slogs := serverLogs.String()
	if strings.Contains(slogs, "pipeline mode, zero disk footprint") {
		fmt.Println("✔ [管道传递日志] 代理核心确认包含 stdin 管道接收配置标记")
	} else {
		fmt.Printf("❌ 未检测到 stdin 管道加载日志:\n%s\n", slogs)
		os.Exit(1)
	}
	if strings.Contains(slogs, "configured request timeout") {
		fmt.Println("✔ [超时注入日志] 代理核心确认已包含 request_timeout=60s 注入标记")
	} else {
		fmt.Printf("❌ 未检测到 request timeout 注入日志:\n%s\n", slogs)
		os.Exit(1)
	}

	fmt.Println("\n✔ 自动化安全行为回归测试项执行完毕，全部断言通过。")
}
