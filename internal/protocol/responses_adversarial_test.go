package protocol

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

// TestAdversarial_Responses_ChaoticInterleavedTools 对抗验证任务 1：
// 构造 4 个工具调用在流式 SSE 中深度乱序交错并混合 reasoning 的极端场景：
// 工具 0 -> 工具 1 -> reasoning -> 工具 2 -> 工具 0 -> 工具 3 -> 工具 1 -> 工具 2 -> 工具 0 -> 工具 3 -> 工具 1 -> 工具 2 -> 工具 3
// 验证：
// 1. 各个工具参数绝对隔离，零串扰，精准匹配目标 JSON；
// 2. 每个 delta 事件的 item_id 绝对精准引用各自工具 output_item.added 的 fcID；
// 3. 流结束时各自发射合法的 output_item.done，且无过早关闭。
func TestAdversarial_Responses_ChaoticInterleavedTools(t *testing.T) {
	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			t.Fatalf("expected http.Flusher")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl.Flush()

		// 14 个深度交错混沌分片下发：
		chunks := []string{
			// Tool 0 chunk 1
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_zero","function":{"name":"run_query","arguments":"{\"query\":"}}]}}]}`,
			// Tool 1 chunk 1
			`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_one","function":{"name":"fetch_metrics","arguments":"{\"metric\":"}}]}}]}`,
			// 混入 reasoning_content
			`data: {"choices":[{"delta":{"reasoning_content":"Analyzing system metrics and queries..."}}]}`,
			// Tool 2 chunk 1
			`data: {"choices":[{"delta":{"tool_calls":[{"index":2,"id":"call_two","function":{"name":"send_alert","arguments":"{\"level\":"}}]}}]}`,
			// Tool 0 chunk 2 (续传)
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"select * from "}}]}}]}`,
			// Tool 3 chunk 1 (新增第 4 个工具)
			`data: {"choices":[{"delta":{"tool_calls":[{"index":3,"id":"call_three","function":{"name":"update_config","arguments":"{\"dry_run\":"}}]}}]}`,
			// Tool 1 chunk 2 (续传)
			`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"\"cpu_utilization\","}}]}}]}`,
			// Tool 2 chunk 2 (续传)
			`data: {"choices":[{"delta":{"tool_calls":[{"index":2,"function":{"arguments":"\"CRITICAL\","}}]}}]}`,
			// Tool 0 chunk 3 (完结 Tool 0)
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"users where id > 100\"}"}}]}}]}`,
			// Tool 3 chunk 2 (续传)
			`data: {"choices":[{"delta":{"tool_calls":[{"index":3,"function":{"arguments":"true,"}}]}}]}`,
			// Tool 1 chunk 3 (完结 Tool 1)
			`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"\"interval\":\"1m\"}"}}]}}]}`,
			// Tool 2 chunk 3 (完结 Tool 2)
			`data: {"choices":[{"delta":{"tool_calls":[{"index":2,"function":{"arguments":"\"channel\":\"slack\"}"}}]}}]}`,
			// Tool 3 chunk 3 (完结 Tool 3)
			`data: {"choices":[{"delta":{"tool_calls":[{"index":3,"function":{"arguments":"\"workers\":16}"}}]}}]}`,
			// Finish
			`data: {"choices":[{"finish_reason":"tool_calls"}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
			fl.Flush()
		}
	}, func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewResponsesHandler(p)
		req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{
			"model": "seed_m8",
			"input": "execute 4 chaotic tools with reasoning",
			"stream": true
		}`))
		rec := httptest.NewRecorder()
		h.HandleResponses(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("HandleResponses returned status %d: %s", rec.Code, rec.Body.String())
		}

		events := parseSSEEvents(t, rec.Body.String())

		toolAddedIndex := make(map[int]string) // output_index -> fcID
		toolDoneArgs := make(map[int]string)   // output_index -> arguments in output_item.done
		toolDeltas := make(map[int]string)     // output_index -> concatenated arguments delta
		var prematureDoneFound bool
		var reasoningObserved bool

		for _, e := range events {
			typ, _ := e["type"].(string)
			switch typ {
			case "response.output_item.added":
				outIdx := int(e["output_index"].(float64))
				item, _ := e["item"].(map[string]interface{})
				itemType, _ := item["type"].(string)
				if itemType == "function_call" {
					fcID, _ := item["id"].(string)
					if fcID == "" {
						t.Fatalf("output_item.added function_call missing id: %v", item)
					}
					toolAddedIndex[outIdx] = fcID
				} else if itemType == "reasoning" {
					reasoningObserved = true
				}

			case "response.function_call_arguments.delta":
				outIdx := int(e["output_index"].(float64))
				itemID, _ := e["item_id"].(string)
				expectedID, exists := toolAddedIndex[outIdx]
				if !exists {
					t.Fatalf("delta received for unopened output_index=%d", outIdx)
				}
				if itemID != expectedID {
					t.Fatalf("ADVERSARIAL FAIL: delta item_id drift! output_index=%d got %s, expected %s",
						outIdx, itemID, expectedID)
				}
				delta, _ := e["delta"].(string)
				toolDeltas[outIdx] += delta

			case "response.output_item.done":
				outIdx := int(e["output_index"].(float64))
				item, _ := e["item"].(map[string]interface{})
				itemType, _ := item["type"].(string)
				if itemType == "function_call" {
					args, _ := item["arguments"].(string)
					toolDoneArgs[outIdx] = args
					// 如果在 4 个工具全部 added 之前就收到了任何工具的 done，即为过早关闭
					if len(toolAddedIndex) < 4 {
						prematureDoneFound = true
					}
				}
			}
		}

		if prematureDoneFound {
			t.Fatalf("ADVERSARIAL FAIL: premature output_item.done emitted before all parallel items were fully received")
		}

		if len(toolAddedIndex) != 4 {
			t.Fatalf("expected exactly 4 function_call items added, got %d", len(toolAddedIndex))
		}

		// 验证 4 个工具的期望参数
		expectedArgs := map[int]string{
			0: `{"query":"select * from users where id > 100"}`,
			1: `{"metric":"cpu_utilization","interval":"1m"}`,
			2: `{"level":"CRITICAL","channel":"slack"}`,
			3: `{"dry_run":true,"workers":16}`,
		}

		var matchedTools int
		for _, expectedJSON := range expectedArgs {
			for actualOutIdx, doneArg := range toolDoneArgs {
				if doneArg == expectedJSON {
					matchedTools++
					actualDelta := toolDeltas[actualOutIdx]
					if actualDelta != expectedJSON {
						t.Fatalf("tool arguments mismatch between delta (%q) and done (%q)", actualDelta, doneArg)
					}
				}
			}
		}

		if matchedTools != 4 {
			t.Fatalf("expected 4 matched tool arguments, got %d. Got doneArgs: %v, expected: %v",
				matchedTools, toolDoneArgs, expectedArgs)
		}

		t.Logf("Adversarial chaotic 4-tool interleaved stream passed! (reasoning mixed: %v)", reasoningObserved)
	})
}

// TestAdversarial_Responses_ChaoticInterleavedTools_LengthTruncated 对抗验证：
// 3 个并行工具乱序传输途中遭遇 finish_reason: "length" 截断，
// 验证：
// 1. 各工具发射的 output_item.done 的 status 严格为 "incomplete"；
// 2. 整个流终态严禁下发 response.completed，必须下发 status="incomplete" 的 response.done。
func TestAdversarial_Responses_ChaoticInterleavedTools_LengthTruncated(t *testing.T) {
	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fl := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl.Flush()

		// 3 个工具途中截断
		chunks := []string{
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_0","function":{"name":"t0","arguments":"{\"a\":1,"}}]}}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_1","function":{"name":"t1","arguments":"{\"b\":2,"}}]}}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":2,"id":"call_2","function":{"name":"t2","arguments":"{\"c\":3,"}}]}}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"trunc\"}}]}}]}`,
			`data: {"choices":[{"finish_reason":"length"}]}`,
			`data: [DONE]`,
		}

		for _, c := range chunks {
			w.Write([]byte(c + "\n\n"))
			fl.Flush()
		}
	}, func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewResponsesHandler(p)
		req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{
			"model": "seed_m8",
			"input": "test truncation during parallel tools",
			"stream": true
		}`))
		rec := httptest.NewRecorder()
		h.HandleResponses(rec, req)

		events := parseSSEEvents(t, rec.Body.String())
		var sawCompleted bool
		var sawDone bool
		var incompleteItemsCount int

		for _, e := range events {
			typ, _ := e["type"].(string)
			switch typ {
			case "response.completed":
				sawCompleted = true
			case "response.done":
				sawDone = true
				respObj, _ := e["response"].(map[string]interface{})
				if respObj["status"] != "incomplete" {
					t.Fatalf("expected response.done status 'incomplete', got: %v", respObj["status"])
				}
			case "response.output_item.done":
				item, _ := e["item"].(map[string]interface{})
				t.Logf("output_item.done: type=%v status=%v id=%v", item["type"], item["status"], item["id"])
				if item["type"] == "function_call" && item["status"] == "incomplete" {
					incompleteItemsCount++
				}
			}
		}

		if sawCompleted {
			t.Fatalf("ADVERSARIAL FAIL: response.completed was emitted during length truncation!")
		}
		if !sawDone {
			t.Fatalf("ADVERSARIAL FAIL: response.done was NOT emitted!")
		}
		if incompleteItemsCount != 3 {
			t.Fatalf("expected all 3 parallel tools to have status 'incomplete', got %d", incompleteItemsCount)
		}
		t.Logf("Adversarial parallel tools truncation test passed! (incomplete items: %d)", incompleteItemsCount)
	})
}

// TestAdversarial_NonChat_UnsupportedToolCombinations 对抗验证任务 4：
// 构造各种仅有 tool_choice、畸形参数组合或不合法工具声明，请求非 Chat (AgentTask) 模型：
// 验证统一返回本地 HTTP 400 (unsupported_channel_feature)，严禁穿透到上游，严禁返回 502。
func TestAdversarial_NonChat_UnsupportedToolCombinations(t *testing.T) {
	testCases := []struct {
		name     string
		endpoint string // "responses" or "anthropic"
		payload  string
	}{
		// Responses 端点对抗测试用例
		{
			name:     "Responses_OnlyToolChoiceAuto",
			endpoint: "responses",
			payload: `{
				"model": "Seed-Code",
				"input": "hello",
				"tool_choice": "auto"
			}`,
		},
		{
			name:     "Responses_OnlyToolChoiceRequired",
			endpoint: "responses",
			payload: `{
				"model": "Seed-Code",
				"input": "hello",
				"tool_choice": "required"
			}`,
		},
		{
			name:     "Responses_ToolChoiceNamedFunction",
			endpoint: "responses",
			payload: `{
				"model": "Seed-Code",
				"input": "hello",
				"tool_choice": {"type": "function", "function": {"name": "test_fn"}}
			}`,
		},
		{
			name:     "Responses_EmptyToolsArrayWithToolChoice",
			endpoint: "responses",
			payload: `{
				"model": "Seed-Code",
				"input": "hello",
				"tools": [],
				"tool_choice": "auto"
			}`,
		},
		{
			name:     "Responses_ToolsDeclared",
			endpoint: "responses",
			payload: `{
				"model": "GLM-5.3",
				"input": "hello",
				"tools": [{"type": "function", "name": "query_db", "description": "query sql"}]
			}`,
		},
		{
			name:     "Responses_HistoricalFunctionCallInInput",
			endpoint: "responses",
			payload: `{
				"model": "Seed-Code",
				"input": [
					{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "run test"}]},
					{"type": "function_call", "call_id": "call_123", "name": "test", "arguments": "{}"}
				]
			}`,
		},

		// Anthropic 端点对抗测试用例
		{
			name:     "Anthropic_ToolsDeclared",
			endpoint: "anthropic",
			payload: `{
				"model": "Seed-Code",
				"messages": [{"role": "user", "content": "hi"}],
				"tools": [{"name": "exec_cmd", "description": "bash", "input_schema": {}}]
			}`,
		},
		{
			name:     "Anthropic_ToolChoiceNamed",
			endpoint: "anthropic",
			payload: `{
				"model": "Seed-Code",
				"messages": [{"role": "user", "content": "hi"}],
				"tool_choice": {"type": "tool", "name": "exec_cmd"}
			}`,
		},
		{
			name:     "Anthropic_ToolChoiceAuto",
			endpoint: "anthropic",
			payload: `{
				"model": "Seed-Code",
				"messages": [{"role": "user", "content": "hi"}],
				"tool_choice": {"type": "auto"}
			}`,
		},
		{
			name:     "Anthropic_HistoricalToolUse",
			endpoint: "anthropic",
			payload: `{
				"model": "Seed-Code",
				"messages": [
					{"role": "user", "content": "hello"},
					{"role": "assistant", "content": [{"type": "tool_use", "id": "t1", "name": "f", "input": {}}]}
				]
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("CRITICAL BREACH: request unexpectedly reached upstream server! Must be blocked locally.")
			}, func(p *proxy.TraeProxy, _ *map[string]interface{}) {
				var rec *httptest.ResponseRecorder
				if tc.endpoint == "responses" {
					h := NewResponsesHandler(p)
					req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(tc.payload))
					rec = httptest.NewRecorder()
					h.HandleResponses(rec, req)
				} else {
					h := NewAnthropicHandler(p)
					req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(tc.payload))
					rec = httptest.NewRecorder()
					h.HandleMessages(rec, req)
				}

				if rec.Code != http.StatusBadRequest {
					t.Fatalf("[%s] expected HTTP 400 Bad Request, got %d (body: %s)",
						tc.name, rec.Code, rec.Body.String())
				}

				var body map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("[%s] response is not valid JSON: %v", tc.name, err)
				}

				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatalf("[%s] response missing 'error' object: %v", tc.name, body)
				}

				code, _ := errObj["code"].(string)
				errType, _ := errObj["type"].(string)

				if code != "unsupported_channel_feature" && errType != "unsupported_channel_feature" {
					t.Fatalf("[%s] expected error code/type 'unsupported_channel_feature', got code=%q type=%q",
						tc.name, code, errType)
				}
			})
		})
	}
}
