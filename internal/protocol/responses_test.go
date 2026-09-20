package protocol

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

// TestResponses_InterleavedToolCalls_ArgumentAndIDIntegrity 验证反例 4：
// 当上游返回严格交错的工具流：`工具0(分片1) -> 工具1(分片1) -> 工具0(分片2) -> 工具1(分片2)` 时，
// 1. 工具 0 绝不在工具 1 首分片到达时过早关闭；
// 2. 两个工具调用的参数被精准隔离到各自独立容器，绝无参数交叉污染；
// 3. 每个 delta 事件严格引用属于自身工具的 item_id 与 output_index；
// 4. 流结束时统一发射各自的 output_item.done。
func TestResponses_InterleavedToolCalls_ArgumentAndIDIntegrity(t *testing.T) {
	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			t.Fatalf("expected http.Flusher")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl.Flush()

		// 交错分片下发：
		// Chunk 1: Tool 0 分片 1
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_zero\",\"function\":{\"name\":\"exec_cmd\",\"arguments\":\"{\\\"cmd\\\":\"}}]}}]}\n\n"))
		fl.Flush()

		// Chunk 2: Tool 1 分片 1 (此时工具 0 仍在传输中，绝不能被提前关闭)
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":1,\"id\":\"call_one\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\"}}]}}]}\n\n"))
		fl.Flush()

		// Chunk 3: Tool 0 分片 2 (续传参数，必须被追加到工具 0 自身缓冲区，item_id 必须是工具 0 的 ID)
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"ls -la\\\"}\"}}]}}]}\n\n"))
		fl.Flush()

		// Chunk 4: Tool 1 分片 2 (续传参数，追加到工具 1)
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":1,\"function\":{\"arguments\":\"\\\"/tmp/log\\\"}\"}}]}}]}\n\n"))
		fl.Flush()

		// Chunk 5: finish_reason: tool_calls
		w.Write([]byte("data: {\"choices\":[{\"finish_reason\":\"tool_calls\"}]}\n\n"))
		fl.Flush()
		w.Write([]byte("data: [DONE]\n\n"))
		fl.Flush()
	}, func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewResponsesHandler(p)
		req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{
			"model": "seed_m8",
			"input": "execute both tools concurrently",
			"stream": true
		}`))
		rec := httptest.NewRecorder()
		h.HandleResponses(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("HandleResponses returned code %d: %s", rec.Code, rec.Body.String())
		}

		events := parseSSEEvents(t, rec.Body.String())

		toolAddedIndex := make(map[int]string) // output_index -> fcID
		toolDoneArgs := make(map[int]string)   // output_index -> final arguments
		toolDeltas := make(map[int]string)     // output_index -> accumulated delta

		var prematureDoneFound bool

		for _, e := range events {
			typ, _ := e["type"].(string)
			switch typ {
			case "response.output_item.added":
				outIdx := int(e["output_index"].(float64))
				item, _ := e["item"].(map[string]interface{})
				fcID, _ := item["id"].(string)
				toolAddedIndex[outIdx] = fcID

			case "response.function_call_arguments.delta":
				outIdx := int(e["output_index"].(float64))
				itemID, _ := e["item_id"].(string)
				expectedID := toolAddedIndex[outIdx]
				if itemID != expectedID {
					t.Fatalf("DELTA PROTOCOL MISMATCH: output_index=%d delta got item_id=%s, expected %s",
						outIdx, itemID, expectedID)
				}
				delta, _ := e["delta"].(string)
				toolDeltas[outIdx] += delta

			case "response.output_item.done":
				outIdx := int(e["output_index"].(float64))
				item, _ := e["item"].(map[string]interface{})
				args, _ := item["arguments"].(string)
				toolDoneArgs[outIdx] = args

				// 如果工具 1 的 added 尚未到达就收到了 done，或者两个工具 done 在 delta 传输完之前发出，即为过早关闭
				if len(toolAddedIndex) < 2 {
					prematureDoneFound = true
				}
			}
		}

		if prematureDoneFound {
			t.Fatalf("PREMATURE CLOSE DETECTED: output_item.done was emitted before all parallel items completed")
		}

		// 验证工具 0 的参数完整性
		expectedTool0 := `{"cmd":"ls -la"}`
		if toolDeltas[0] != expectedTool0 {
			t.Fatalf("tool 0 delta arguments corrupted: got %q, want %q", toolDeltas[0], expectedTool0)
		}
		if toolDoneArgs[0] != expectedTool0 {
			t.Fatalf("tool 0 done arguments corrupted: got %q, want %q", toolDoneArgs[0], expectedTool0)
		}

		// 验证工具 1 的参数完整性
		expectedTool1 := `{"path":"/tmp/log"}`
		if toolDeltas[1] != expectedTool1 {
			t.Fatalf("tool 1 delta arguments corrupted: got %q, want %q", toolDeltas[1], expectedTool1)
		}
		if toolDoneArgs[1] != expectedTool1 {
			t.Fatalf("tool 1 done arguments corrupted: got %q, want %q", toolDoneArgs[1], expectedTool1)
		}
	})
}

// TestResponses_Incomplete_DoesNotEmitCompleted 验证任务 6 [P2-5]：
// 当上游以截断状态结束 (如 finish_reason: "length" 或 "incomplete") 时，
// Responses 流式事件流仅发射 response.done，严禁下发 response.completed 事件。
func TestResponses_Incomplete_DoesNotEmitCompleted(t *testing.T) {
	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fl := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl.Flush()

		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"truncated text...\"}}]}\n\n"))
		fl.Flush()
		w.Write([]byte("data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}]}\n\n"))
		fl.Flush()
		w.Write([]byte("data: [DONE]\n\n"))
		fl.Flush()
	}, func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewResponsesHandler(p)
		req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{
			"model": "seed_m8",
			"input": "generate a long essay",
			"stream": true
		}`))
		rec := httptest.NewRecorder()
		h.HandleResponses(rec, req)

		events := parseSSEEvents(t, rec.Body.String())
		var hasDone, hasCompleted bool
		for _, e := range events {
			typ, _ := e["type"].(string)
			if typ == "response.done" {
				hasDone = true
				respObj, _ := e["response"].(map[string]interface{})
				if respObj["status"] != "incomplete" {
					t.Fatalf("expected response.done status to be 'incomplete', got: %v", respObj["status"])
				}
			}
			if typ == "response.completed" {
				hasCompleted = true
			}
		}

		if !hasDone {
			t.Fatalf("expected response.done event to be emitted")
		}
		if hasCompleted {
			t.Fatalf("PROTOCOL VIOLATION: response.completed was emitted for incomplete response!")
		}
	})
}

// TestResponses_NonChat_UnsupportedToolFeature_Returns400 验证任务 5 [P2-4]：
// 向不支持工具的 AgentTask 通道发送工具请求时，统一返回本地 HTTP 400 (unsupported_channel_feature)，严禁 502。
func TestResponses_NonChat_UnsupportedToolFeature_Returns400(t *testing.T) {
	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("upstream should not be reached when request is blocked by tool guard")
	}, func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewResponsesHandler(p)

		// 携带 tools
		reqBody := `{
			"model": "Seed-Code",
			"input": "test tool guard",
			"tools": [{"type": "function", "name": "dummy_tool", "description": "test"}]
		}`
		req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBody))
		rec := httptest.NewRecorder()
		h.HandleResponses(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for unsupported tool feature, got %d: %s", rec.Code, rec.Body.String())
		}
		var errResp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		errObj, _ := errResp["error"].(map[string]interface{})
		code, _ := errObj["code"].(string)
		errType, _ := errObj["type"].(string)
		if code != "unsupported_channel_feature" && errType != "unsupported_channel_feature" {
			t.Errorf("expected error code/type 'unsupported_channel_feature', got code=%v type=%v", code, errType)
		}

		// 携带 tool_choice
		reqBodyChoice := `{
			"model": "Seed-Code",
			"input": "test tool choice",
			"tool_choice": "auto"
		}`
		reqChoice := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(reqBodyChoice))
		recChoice := httptest.NewRecorder()
		h.HandleResponses(recChoice, reqChoice)

		if recChoice.Code != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for tool_choice, got %d: %s", recChoice.Code, recChoice.Body.String())
		}
	})
}

// TestAnthropic_NonChat_UnsupportedToolFeature_Returns400 验证任务 5 [P2-4]：
// 向 /v1/messages 发送不支持工具的 AgentTask 模型请求时，统一返回本地 HTTP 400 (unsupported_channel_feature)。
func TestAnthropic_NonChat_UnsupportedToolFeature_Returns400(t *testing.T) {
	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("upstream should not be reached when request is blocked by tool guard")
	}, func(p *proxy.TraeProxy, _ *map[string]interface{}) {
		h := NewAnthropicHandler(p)

		reqBody := `{
			"model": "Seed-Code",
			"messages": [{"role": "user", "content": "hello"}],
			"tools": [{"name": "some_tool", "description": "test", "input_schema": {}}]
		}`
		req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(reqBody))
		rec := httptest.NewRecorder()
		h.HandleMessages(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 from Anthropic endpoint, got %d: %s", rec.Code, rec.Body.String())
		}
		var errResp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		errObj, _ := errResp["error"].(map[string]interface{})
		errType, _ := errObj["type"].(string)
		if errType != "unsupported_channel_feature" {
			t.Errorf("expected error type 'unsupported_channel_feature', got: %v", errType)
		}
	})
}
