package proxy

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/models"
	"github.com/zamatewi-cell/traecn_tool/internal/sse"
)

// ErrAgentTaskToolsUnsupported is returned when tools, tool_choice or tool history are passed to AgentTask channel.
var ErrAgentTaskToolsUnsupported = errors.New("unsupported_channel_feature: AgentTask channel does not support tools or tool history")

// isToolChoiceNone 判断客户端传入的 tool_choice 是否显式为 "none"
func isToolChoiceNone(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed == `"none"` || trimmed == `none`
}

// ValidateAgentTaskRequest enforces Fail-Closed policy for models using ChannelAgentTask.
// It inspects tool declarations, tool_choice and conversation history for any tool-related artifacts.
func ValidateAgentTaskRequest(req *ChatCompletionRequest) error {
	if req == nil {
		return nil
	}

	// 1. 检查是否显式声明了自定义工具列表
	if len(req.Tools) > 0 {
		return fmt.Errorf("%w: requested model %q does not support function/tool declarations via AgentTask channel; please use preset models (e.g., deepseek-V3, seed_m8) for tool calling",
			ErrAgentTaskToolsUnsupported, req.ModelName)
	}

	// 2. 检查是否显式指定了非 "none" 的 ToolChoice (AI-3 边界加固)
	if len(req.ToolChoice) > 0 && !isToolChoiceNone(req.ToolChoice) {
		return fmt.Errorf("%w: requested model %q does not support tool_choice declarations via AgentTask channel; please use preset models for tool calling",
			ErrAgentTaskToolsUnsupported, req.ModelName)
	}

	// 3. 检查会话历史中是否包含工具交互残留 (tool_calls, role: tool 或 role: function)
	for i, msg := range req.Messages {
		if strings.EqualFold(msg.Role, "tool") || strings.EqualFold(msg.Role, "function") {
			return fmt.Errorf("%w: message at index %d contains role %q which is unsupported in AgentTask channel",
				ErrAgentTaskToolsUnsupported, i, msg.Role)
		}
		if len(msg.ToolCalls) > 0 {
			return fmt.Errorf("%w: assistant message at index %d contains tool_calls history which is unsupported in AgentTask channel",
				ErrAgentTaskToolsUnsupported, i)
		}
	}

	return nil
}

// AgentRenderContext wraps the environment variables for upstream prompt template rendering.
type AgentRenderContext struct {
	Variables  string                 `json:"variables"`
	References map[string]interface{} `json:"references"`
}

// AgentTaskPayload is the JSON payload sent to /api/agent/v3/create_agent_task.
type AgentTaskPayload struct {
	AgentID           *string                `json:"agent_id"`
	AgentType         string                 `json:"agent_type"`
	ConfigName        string                 `json:"config_name"`
	ModelName         string                 `json:"model_name"`
	ConfigSource      int                    `json:"config_source"`
	IDEVersion        string                 `json:"ide_version"`
	UserID            string                 `json:"user_id"`
	DeviceID          string                 `json:"device_id"`
	ConversationID    string                 `json:"conversation_id"`
	SessionID         string                 `json:"session_id"`
	UserInput         AgentUserInput         `json:"user_input"`
	HistoryIDList     []string               `json:"history_id_list"`
	AvailableToolList []interface{}          `json:"available_tool_list"`
	RenderContext     AgentRenderContext     `json:"render_context"`
	ExtraConfig       map[string]interface{} `json:"extra_config"`
	AgentVersion      string                 `json:"agent_version,omitempty"`
	ModeType          int                    `json:"mode_type"`
	RequestSeq        int                    `json:"request_seq"`
}

// AgentUserInput wraps the user's prompt messages in an agent_task.
type AgentUserInput struct {
	ID       string                 `json:"id"`
	Messages []AgentUserMessageItem `json:"messages"`
}

// AgentUserMessageItem is one message inside user_input.
type AgentUserMessageItem struct {
	Type        string `json:"type"`
	TextContent string `json:"text_content"`
}

// AgentThoughtEvent matches the payload of an event: thought SSE chunk.
type AgentThoughtEvent struct {
	FirstData        bool   `json:"first_data"`
	ReasoningContent string `json:"reasoning_content"`
	Thought          string `json:"thought"`
}

// AgentTokenUsageEvent matches the payload of an event: token_usage SSE chunk.
type AgentTokenUsageEvent struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	ReasoningTokens  int `json:"reasoning_tokens"`
}

// AgentTaskCreatedEvent matches event: task_created.
type AgentTaskCreatedEvent struct {
	TaskID     string `json:"task_id"`
	AgentRunID string `json:"agent_run_id"`
}

// AgentTurnCompletionEvent matches event: turn_completion.
type AgentTurnCompletionEvent struct {
	TaskCompletion bool `json:"task_completion"`
}

// randomHex generates an n-byte cryptographically secure random string as hex.
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		u := strings.ReplaceAll(uuid.New().String(), "-", "")
		if len(u) >= n*2 {
			return u[:n*2]
		}
		return u
	}
	return hex.EncodeToString(b)
}

// ResolveAgentTaskConfig resolves the client-supplied model name into the
// internal config_name and upstream model_name accepted by /api/agent/v3/create_agent_task.
func ResolveAgentTaskConfig(reqModel string) (configName, modelName string) {
	norm := models.Normalize(reqModel)
	switch norm {
	case "doubao-seed-code", "seed-code", "doubao", "seed":
		return "Doubao-Seed-Code", "Doubao-Seed-Code__dev"
	case "deepseek-v4-1-flash", "deepseek-v4.1-flash", "deepseek-v4.1", "deepseek-v4-flash", "deepseek-v4", "deepseek-chat":
		return "DeepSeek-V4-Flash", "DeepSeek-V4-Flash__dev"
	case "deepseek-v4-pro", "deepseek-r1", "deepseek-reasoner":
		return "DeepSeek-V4-Pro", "DeepSeek-V4-Pro__dev"
	case "glm-5-3", "glm-5.3", "glm-5", "glm":
		return "glm-5.3", "glm-5.3__dev"
	case "glm-5-3-flash", "glm-5.3-flash":
		return "glm-5.2", "glm-5.2__dev"
	case "glm-5-2", "glm-5.2":
		return "glm-5.2", "glm-5.2__dev"
	case "glm-5-1", "glm-5.1":
		return "glm-5.1", "glm-5.1__dev"
	case "glm-4-7", "glm-4.7":
		return "glm-4.7", "glm-4.7__dev"
	case "kimi-k3", "kimi", "kimi-k2", "kimi-k2-8-preview", "kimi-k2.8-preview":
		return "kimi-k2", "kimi-k2__dev"
	case "minimax-m3", "minimax":
		return "minimax-m3", "minimax-m3__dev"
	case "minimax-m2-5", "minimax-m2.5":
		return "minimax-m2.5", "minimax-m2.5__dev"
	case "minimax-m2-1", "minimax-m2.1":
		return "minimax-m2.1", "minimax-m2.1__dev"
	case "qwen-3-7-plus", "qwen3.7-plus", "qwen", "qwen-3-8-flash", "qwen-3-8-max":
		return "qwen-3.7-plus", "qwen-3.7-plus__dev"
	case "qwen-3-5", "qwen3.5-plus":
		return "qwen-3.5", "qwen-3.5__dev"
	case "qwen3-coder", "qwen3-coder-next":
		return "qwen3-coder", "openai_qwen3-coder-plus__dev"
	case "gemini-3-1-pro", "gemini-3.1-pro":
		return "gemini-3.1-pro", "gemini-3.1-pro"
	case "gpt-5-mini":
		return "gpt-5-mini", "gpt-5-mini"
	case "seed-evolving", "seed-2-1-pro-0915", "seed-2-1-turbo":
		return "Doubao-Seed-Code", "Doubao-Seed-Code__dev"
	default:
		if strings.HasSuffix(reqModel, "__dev") {
			return strings.TrimSuffix(reqModel, "__dev"), reqModel
		}
		return reqModel, reqModel + "__dev"
	}
}

// extractPrompt converts incoming messages into a coherent user prompt string.
func extractPrompt(messages []Message) string {
	if len(messages) == 1 && messages[0].Role == "user" {
		return messages[0].Content
	}
	var sb strings.Builder
	for _, m := range messages {
		switch strings.ToLower(m.Role) {
		case "system":
			sb.WriteString("System: " + m.Content + "\n\n")
		case "assistant":
			sb.WriteString("Assistant: " + m.Content + "\n\n")
		case "user":
			sb.WriteString("User: " + m.Content + "\n\n")
		default:
			sb.WriteString(m.Role + ": " + m.Content + "\n\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

// buildAgentUserInput converts incoming messages into the agent_task user_input payload.
func buildAgentUserInput(messages []Message) (AgentUserInput, string) {
	prompt := extractPrompt(messages)
	return AgentUserInput{
		ID: randomHex(12), // 24-character hex ID
		Messages: []AgentUserMessageItem{
			{
				Type:        "text",
				TextContent: prompt,
			},
		},
	}, prompt
}

// extractUserID parses the unverified claims from a JWT token to extract the user_id.
func extractUserID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) >= 2 {
		payloadSegment := parts[1]
		if rem := len(payloadSegment) % 4; rem != 0 {
			payloadSegment += strings.Repeat("=", 4-rem)
		}
		data, err := base64.RawURLEncoding.DecodeString(payloadSegment)
		if err != nil {
			data, err = base64.URLEncoding.DecodeString(payloadSegment)
		}
		if err == nil {
			var claims struct {
				Data struct {
					ID interface{} `json:"id"`
				} `json:"data"`
				UserID interface{} `json:"user_id"`
			}
			if err := json.Unmarshal(data, &claims); err == nil {
				if claims.Data.ID != nil {
					return fmt.Sprintf("%v", claims.Data.ID)
				}
				if claims.UserID != nil {
					return fmt.Sprintf("%v", claims.UserID)
				}
			}
		}
	}
	return "4355622541471866" // fallback default user_id
}

// buildAgentTaskPayload constructs the JSON payload for create_agent_task.
func buildAgentTaskPayload(req *ChatCompletionRequest, token string, deviceID string) (*AgentTaskPayload, error) {
	cfgName, mdlName := ResolveAgentTaskConfig(req.ModelName)

	convID := req.ConversationID
	if len(convID) < 12 {
		convID = randomHex(12)
	}
	sessID := req.SessionID
	if len(sessID) < 12 {
		sessID = randomHex(12)
	}

	userID := extractUserID(token)
	if deviceID == "" {
		deviceID = "2262131830954826"
	}

	userInput, prompt := buildAgentUserInput(req.Messages)

	// Upstream prompt template engine fills the <user_input> tag from
	// variables.input / variables.raw_input. Missing or empty variables
	// cause the model thinking chain to report "user_input tag is empty".
	now := time.Now()
	varsMap := map[string]interface{}{
		"agent_name":                   "Builder",
		"agent_type":                   "builder_v3",
		"enable_parallel_tool_calling": false,
		"response_can_be_text":         true,
		"native_function_call":         true,
		"system_type":                  "Windows",
		"locale":                       "zh-cn",
		"language_settings":            "zh-cn",
		"date":                         now.Format("2006-01-02"),
		"current_time":                 now.Format("20060102 15:04:05"),
		"brand":                        "Trae",
		"raw_input":                    prompt,
		"input":                        prompt,
		"environment_context":          "",
	}
	varsBytes, err := json.Marshal(varsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal render_context variables: %w", err)
	}

	payload := &AgentTaskPayload{
		AgentID:           nil, // official captured trace has agent_id: null
		AgentType:         "builder_v3",
		ConfigName:        cfgName,
		ModelName:         mdlName,
		ConfigSource:      1,
		IDEVersion:        config.IDEVersion,
		UserID:            userID,
		DeviceID:          deviceID,
		ConversationID:    convID,
		SessionID:         sessID,
		UserInput:         userInput,
		HistoryIDList:     []string{},
		AvailableToolList: []interface{}{},
		RenderContext: AgentRenderContext{
			Variables:  string(varsBytes),
			References: map[string]interface{}{},
		},
		ExtraConfig: map[string]interface{}{
			"disable_parallel_agent": true,
			"enable_todo_list":       true,
			"enable_core_memory":     false,
		},
		AgentVersion: "v3",
		ModeType:     0,
		RequestSeq:   1,
	}

	return payload, nil
}

// doAgentTaskCompletion handles the dispatch to /api/agent/v3/create_agent_task,
// encrypts via masticate AES-256-GCM, and streams parsed events to handle.
func (p *TraeProxy) doAgentTaskCompletion(req *ChatCompletionRequest, token string, handle StreamHandler) (int, error) {
	release := p.limiter.Acquire()
	defer release()

	deviceID := ""
	if p.device != nil {
		deviceID = p.device.DeviceID
	}

	payload, err := buildAgentTaskPayload(req, token, deviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to build agent_task payload: %w", err)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal agent_task payload: %w", err)
	}

	// Masticate encryption (AES-256-GCM with pin XOR & timestamp AAD)
	encMsg, pin, requestAt, err := masticate(payloadBytes)
	if err != nil {
		return 0, fmt.Errorf("masticate encryption failed: %w", err)
	}

	endpoint := config.AgentDomain + config.EndpointAgentCreateTask
	httpReq, err := http.NewRequest("POST", endpoint, strings.NewReader(encMsg))
	if err != nil {
		return 0, fmt.Errorf("failed to create agent_task request: %w", err)
	}

	ids := NewRequestIDs()
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(config.HeaderIDEToken, token)
	httpReq.Header.Set("X-Request-Pin", pin)
	httpReq.Header.Set("X-Requested-At", strconv.FormatInt(requestAt, 10))
	httpReq.Header.Set("x-bridge-transport", "aha")
	httpReq.Header.Set("User-Agent", "TraeClient/TTNet")
	httpReq.Header.Set("app-version", config.IDEVersion)
	httpReq.Header.Set("x-ide-version", config.IDEVersion)
	httpReq.Header.Set("x-app-id", config.AppID)
	httpReq.Header.Set("package-type", "stable_cn")
	httpReq.Header.Set("x-request-id", ids.RequestID)
	httpReq.Header.Set("x-trae-request-id", ids.TraeRequestID)

	for k, v := range p.device.Headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("agent_task request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return resp.StatusCode, fmt.Errorf("agent_task API error %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return http.StatusOK, p.streamAgentTaskEvents(resp.Body, req.ModelName, handle)
}

// streamAgentTaskEvents reads the SSE event stream from create_agent_task,
// extracts thought, history, token_usage, turn_completion events, and emits
// normalized StreamEvents.
func (p *TraeProxy) streamAgentTaskEvents(body io.Reader, model string, handle StreamHandler) error {
	reader := sse.NewReader(body)
	handle, flush := WrapStreamHandler(handle)

	var finished bool

	for {
		evt, err := reader.ReadEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("SSE read error: %w", err)
		}
		if evt.Data == "" {
			continue
		}

		trimmedData := strings.TrimSpace(evt.Data)

		switch evt.Event {
		case "task_created":
			var tc AgentTaskCreatedEvent
			if err := json.Unmarshal([]byte(trimmedData), &tc); err == nil {
				if err := handle(&StreamEvent{Type: EventTaskCreated, TaskID: tc.TaskID}); err != nil {
					return err
				}
			}

		case "thought":
			var th AgentThoughtEvent
			if err := json.Unmarshal([]byte(trimmedData), &th); err == nil {
				if th.ReasoningContent != "" {
					if err := handle(&StreamEvent{Type: EventReasoning, Reasoning: th.ReasoningContent}); err != nil {
						return err
					}
				}
				if th.Thought != "" {
					if err := handle(&StreamEvent{Type: EventText, Text: th.Thought}); err != nil {
						return err
					}
				}
			}

		case "history", "model_config", "agent_status", "context_usage", "metadata", "timing_cost", "required_context":
			// Known AgentTask control/status events; do not treat as delta text.
			continue

		case "token_usage":
			var tu AgentTokenUsageEvent
			if err := json.Unmarshal([]byte(trimmedData), &tu); err == nil {
				usage := &Usage{
					PromptTokens:     tu.PromptTokens,
					CompletionTokens: tu.CompletionTokens,
					TotalTokens:      tu.TotalTokens,
				}
				if err := handle(&StreamEvent{Type: EventUsage, Usage: usage}); err != nil {
					return err
				}
			}

		case "turn_completion":
			finished = true
			if err := handle(&StreamEvent{Type: EventFinish, FinishReason: "stop"}); err != nil {
				return err
			}

		case "error":
			var e struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal([]byte(trimmedData), &e); err == nil && e.Message != "" {
				errType := "upstream_error"
				if e.Code != 0 {
					errType = fmt.Sprintf("upstream_code_%d", e.Code)
				}
				return handle(&StreamEvent{
					Type: EventError,
					Err:  &UpstreamError{Message: e.Message, Type: errType},
				})
			}

		default:
			// Fallback: parse OpenAI/Trae dialect data lines or [DONE] when event is untagged
			if evt.Event == "" {
				for _, se := range parseUpstreamData(trimmedData) {
					if se.Type == EventFinish {
						finished = true
					}
					if err := handle(se); err != nil {
						return err
					}
				}
			}
		}
	}

	if err := flush(); err != nil {
		return err
	}

	if !finished {
		if err := handle(&StreamEvent{Type: EventFinish, FinishReason: "stop"}); err != nil {
			return err
		}
	}

	return nil
}
