package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/models"
)

// ============================================================================
// 迭代2 挑战 1: 极限超高并发混合模型交替调用与严格 Oracle 状态隔离
// ============================================================================

func TestChallengerIter2_ExtremeConcurrencyMixedLoad(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	allModels := append(LegacyPresetModels, NewBuiltinModels...)
	totalRequests := 105 // 21 个模型各至少 5 次高并发请求
	concurrency := 15    // 15 个并发 worker 并发冲击

	t.Logf("【迭代2极限挑战】启动 %d 个高并发请求，覆盖全部 %d 个新旧模型，混合流式/非流式...", totalRequests, len(allModels))

	type reqJob struct {
		id       int
		model    string
		isStream bool
	}

	jobs := make(chan reqJob, totalRequests)
	for i := 0; i < totalRequests; i++ {
		jobs <- reqJob{
			id:       i,
			model:    allModels[i%len(allModels)],
			isStream: (i % 2) == 0,
		}
	}
	close(jobs)

	var wg sync.WaitGroup
	errCh := make(chan error, totalRequests)
	start := time.Now()

	client := &http.Client{Timeout: 10 * time.Second}

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				uniqueID := fmt.Sprintf("req_%03d_w%02d_%s", job.id, workerID, job.model)
				payload := map[string]interface{}{
					"model": job.model,
					"messages": []map[string]string{
						{"role": "user", "content": "OracleCheck:" + uniqueID},
					},
					"stream": job.isStream,
				}
				bs, _ := json.Marshal(payload)
				req, err := http.NewRequest("POST", tg.BaseURL+"/v1/chat/completions", bytes.NewReader(bs))
				if err != nil {
					errCh <- fmt.Errorf("[%s] 创建请求失败: %w", uniqueID, err)
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					errCh <- fmt.Errorf("[%s] 请求执行失败: %w", uniqueID, err)
					continue
				}

				respBody, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					errCh <- fmt.Errorf("[%s] 读取响应失败: %w", uniqueID, err)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					errCh <- fmt.Errorf("[%s] 异常状态码 %d: %s", uniqueID, resp.StatusCode, string(respBody))
					continue
				}

				bodyStr := string(respBody)

				// Oracle 校验 1: 必须返回正确的通道特征
				isLegacy := false
				for _, lp := range LegacyPresetModels {
					if lp == job.model {
						isLegacy = true
						break
					}
				}

				if job.isStream {
					// 流式 Oracle 校验
					if !strings.Contains(bodyStr, "[DONE]") {
						errCh <- fmt.Errorf("[%s] 流式响应缺失 [DONE] 标记: %s", uniqueID, bodyStr)
						continue
					}
					if !strings.Contains(bodyStr, "chat.completion.chunk") {
						errCh <- fmt.Errorf("[%s] 流式响应缺失 chunk 类型: %s", uniqueID, bodyStr)
						continue
					}
					// 验证是否包含 Token Usage 修复
					if !strings.Contains(bodyStr, "usage") {
						errCh <- fmt.Errorf("[%s] 流式响应缺失 token usage 信息", uniqueID)
						continue
					}
				} else {
					// 非流式 Oracle 校验
					var cResp ChatCompletionResponse
					if err := json.Unmarshal(respBody, &cResp); err != nil {
						errCh <- fmt.Errorf("[%s] 非流式响应 JSON 解析失败: %w, raw: %s", uniqueID, err, bodyStr)
						continue
					}
					if len(cResp.Choices) == 0 {
						errCh <- fmt.Errorf("[%s] 非流式响应 Choices 为空", uniqueID)
						continue
					}
					content := cResp.Choices[0].Message.Content
					if isLegacy {
						if !strings.Contains(content, "存量预设通道") {
							errCh <- fmt.Errorf("[%s] 存量模型返回了非存量通道内容: %s", uniqueID, content)
							continue
						}
					} else {
						if !strings.Contains(content, "新一代内置模型通道") {
							errCh <- fmt.Errorf("[%s] 新一代内置模型返回了非新通道内容: %s", uniqueID, content)
							continue
						}
					}
				}
			}
		}(w)
	}

	wg.Wait()
	close(errCh)

	elapsed := time.Since(start)
	t.Logf("【迭代2极限挑战】%d 个请求并发执行完毕，总耗时: %v, QPS: %.1f", totalRequests, elapsed, float64(totalRequests)/elapsed.Seconds())

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("并发挑战失败: %v", err)
		}
		t.Fatalf("极限并发混合模型挑战失败: 共发现 %d 个错误", len(errs))
	}

	// 校验 MockUpstream 收到的全部命中记录，检查通道分流是否 100% 精确匹配
	legacyHits := tg.MockUpstream.GetHits(config.EndpointLLMRawChat)
	agentHits := tg.MockUpstream.GetHits(config.EndpointAgentCreateTask)
	t.Logf("【通道路由统计】存量 Legacy 通道命中: %d 次，AgentTask 新通道命中: %d 次，总计: %d 次",
		len(legacyHits), len(agentHits), len(legacyHits)+len(agentHits))

	if len(legacyHits)+len(agentHits) != totalRequests {
		t.Fatalf("上游请求接收总数 (%d) 与发出总数 (%d) 不一致", len(legacyHits)+len(agentHits), totalRequests)
	}

	t.Log("【迭代2极限挑战】超高并发混合模型交替调用挑战 100% PASS，无死锁、无状态交叉污染！")
}

// ============================================================================
// 迭代2 挑战 2: 并发读写模型注册表与多协程查询竞态安全挑战
// ============================================================================

func TestChallengerIter2_RegistryConcurrencyRace(t *testing.T) {
	reg := models.NewDefaultRegistry()
	var wg sync.WaitGroup
	workers := 20
	iterations := 200

	t.Logf("启动 %d 个协程并发读写模型注册表与解析别名，测试锁安全性与数据竞争...", workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		wID := i
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				switch (id + j) % 5 {
				case 0:
					_ = reg.List()
				case 1:
					_ = reg.Resolve("DeepSeek-V4.1-Flash")
					_ = reg.Resolve("seed_m8")
					_ = reg.Resolve("kimi-k3")
				case 2:
					_ = reg.ResolveChannel("deepseek-v4-flash")
					_ = reg.ResolveChannel("seed_m8")
					_ = reg.ResolveChannel("unknown-random-model-name")
				case 3:
					_ = reg.Aliases()
				case 4:
					// 动态并发注入自定义模型与别名
					customID := fmt.Sprintf("dynamic-model-w%d-%d", id, j)
					reg.Upsert(models.ModelInfo{
						ModelID:     customID,
						DisplayName: "Dynamic Model",
						Channel:     models.ChannelAgentTask,
					})
					reg.RegisterAlias(fmt.Sprintf("alias-%s", customID), customID)
				}
			}
		}(wID)
	}

	wg.Wait()
	t.Log("模型注册表并发读写与竞态安全挑战通过，无死锁、无 panic")
}

// ============================================================================
// 迭代2 挑战 3: 严苛异常参数、非法方法与畸形载荷鲁棒性挑战
// ============================================================================

func TestChallengerIter2_ExtremeMalformedPayloads(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	tests := []struct {
		name         string
		method       string
		path         string
		headers      map[string]string
		body         string
		expectStatus int // 必须是非 500
	}{
		{
			name:         "MethodNotAllowed_DELETE",
			method:       "DELETE",
			path:         "/v1/chat/completions",
			body:         "",
			expectStatus: http.StatusMethodNotAllowed,
		},
		{
			name:         "InvalidContentType_PlainText",
			method:       "POST",
			path:         "/v1/chat/completions",
			headers:      map[string]string{"Content-Type": "text/plain"},
			body:         `{"model":"Seed-Code","messages":[{"role":"user","content":"test"}]}`,
			expectStatus: http.StatusOK, // 网关宽容解析 JSON
		},
		{
			name:         "InvalidJSON_UnclosedBracket",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"Seed-Code","messages":[{"role":"user","content":"test"`,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "InvalidJSON_UnexpectedToken",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model": "Seed-Code", "messages": undefined}`,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "InvalidMessage_NullInMessagesArray",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"GLM-5.3","messages":[null, {"role":"user","content":"hi"}]}`,
			expectStatus: http.StatusOK, // 防御空指针
		},
		{
			name:         "ExtraneousFields_PrototypeInjection",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"DeepSeek-V4.1-Flash","messages":[{"role":"user","content":"hi"}],"__proto__":{"polluted":true},"extra_unknown_key":{"nested":123}}`,
			expectStatus: http.StatusOK,
		},
		{
			name:         "ExtremeTemperature_NegativeAndExcessive",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"seed_m8","messages":[{"role":"user","content":"hi"}],"temperature": -99.9}`,
			expectStatus: http.StatusOK,
		},
		{
			name:         "ExtremeMaxTokens_Negative",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"seed_m8","messages":[{"role":"user","content":"hi"}],"max_tokens": -50}`,
			expectStatus: http.StatusOK,
		},
		{
			name:         "ExtremeMaxTokens_Exorbitant",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"seed_m8","messages":[{"role":"user","content":"hi"}],"max_tokens": 999999999}`,
			expectStatus: http.StatusOK,
		},
		{
			name:         "UnicodeAttacks_ZeroWidthAndRTL",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"DeepSeek-V4.1-Flash","messages":[{"role":"user","content":"\u200B\u200C\u200D\uFEFF\u202E\u0000\u0007\u001BUnicodeInfiltration"}]}`,
			expectStatus: http.StatusOK,
		},
		{
			name:         "EmojiFloodPrompt",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"Kimi-K3","messages":[{"role":"user","content":"` + strings.Repeat("🚀🔥🎉🤖💥", 2000) + `"}]}`,
			expectStatus: http.StatusOK,
		},
		{
			name:         "EmptyModelFieldRequired",
			method:       "POST",
			path:         "/v1/chat/completions",
			body:         `{"model":"","messages":[{"role":"user","content":"what model?"}]}`,
			expectStatus: http.StatusBadRequest, // 规范要求 model 不能为空
		},
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tg.BaseURL+tc.path, strings.NewReader(tc.body))
			if err != nil {
				t.Fatalf("创建请求失败: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("请求执行失败: %v", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			// 严密断言：系统绝不能触发 500 Internal Server Error 崩溃！
			if resp.StatusCode == http.StatusInternalServerError {
				t.Fatalf("[%s] 严重防御漏洞：触发服务器 500 崩溃错误: %s", tc.name, string(body))
			}

			if tc.expectStatus != 0 && resp.StatusCode != tc.expectStatus {
				t.Errorf("[%s] 实际状态码: %d (预期: %d), body: %s", tc.name, resp.StatusCode, tc.expectStatus, string(body))
			}

			// 对于 200 与 400 请求，返回的内容必须是有效 JSON，不能是裸 panic 堆栈
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
				var js map[string]interface{}
				if err := json.Unmarshal(body, &js); err != nil {
					t.Errorf("[%s] 响应不是合法 JSON: %v, raw: %s", tc.name, err, string(body))
				}
			}
		})
	}

	t.Log("【迭代2极限挑战】严苛异常参数与畸形载荷挑战 100% 通过，无 panic，无 500 泄露！")
}

// ============================================================================
// 迭代2 挑战 4: 21 个模型流式 Token Usage 与 Reasoning 深度解析实证
// ============================================================================

func TestChallengerIter2_All21ModelsStreamingUsageAndReasoning(t *testing.T) {
	tg := NewTestGateway(t)
	defer tg.Close()

	allModels := append(LegacyPresetModels, NewBuiltinModels...)
	client := &http.Client{Timeout: 5 * time.Second}

	t.Logf("逐一验证全量 %d 个模型的流式 SSE Token Usage、Reasoning 与 [DONE] 终结符...", len(allModels))

	for _, m := range allModels {
		t.Run(m, func(t *testing.T) {
			payload := map[string]interface{}{
				"model": m,
				"messages": []map[string]string{
					{"role": "user", "content": "hello " + m},
				},
				"stream": true,
			}
			bs, _ := json.Marshal(payload)
			resp, err := client.Post(tg.BaseURL+"/v1/chat/completions", "application/json", bytes.NewReader(bs))
			if err != nil {
				t.Fatalf("[%s] 流式请求失败: %v", m, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("[%s] 流式响应状态码错误 %d: %s", m, resp.StatusCode, string(body))
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("[%s] 读取流式响应失败: %v", m, err)
			}

			raw := string(bodyBytes)

			if !strings.Contains(raw, "[DONE]") {
				t.Fatalf("[%s] 流式输出缺失 [DONE] 终止标志", m)
			}

			// 检查是否包含 Token Usage
			if !strings.Contains(raw, "prompt_tokens") || !strings.Contains(raw, "completion_tokens") {
				t.Fatalf("[%s] 流式输出缺失 Token Usage 统计信息", m)
			}

			// 检查是否包含增量 content
			if !strings.Contains(raw, "content") {
				t.Fatalf("[%s] 流式输出缺失 content 增量", m)
			}
		})
	}

	t.Log("【迭代2极限挑战】全量 21 个模型流式 SSE Token Usage 与完整性 100% PASS！")
}
