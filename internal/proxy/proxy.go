package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/auth"
	"github.com/zamatewi-cell/traecn_tool/internal/config"
	"github.com/zamatewi-cell/traecn_tool/internal/device"
	"github.com/zamatewi-cell/traecn_tool/internal/models"
	"github.com/zamatewi-cell/traecn_tool/internal/protect"
	"github.com/zamatewi-cell/traecn_tool/internal/queue"
	"github.com/zamatewi-cell/traecn_tool/internal/sse"
)

// TraeProxy is the core forwarding client: it converts client requests
// into the upstream Trae CN session payload, injects the real client
// fingerprint headers, and normalizes the SSE event stream.
type TraeProxy struct {
	client      *http.Client
	tokens      *auth.Pool
	device      *device.DeviceInfo
	queue       *queue.Monitor
	logger      *slog.Logger
	projector   *protect.Projector
	filter      *protect.Filter
	limiter     *protect.Limiter
}

// NewTraeProxy creates a new proxy instance
func NewTraeProxy(tokens *auth.Pool, logger *slog.Logger) *TraeProxy {
	return &TraeProxy{
		client:      &http.Client{Timeout: 5 * time.Minute},
		tokens:      tokens,
		device:      device.NewDeviceInfo(),
		queue:       queue.NewMonitor(),
		logger:      logger,
	}
}

// SetProtection installs the anti-abuse layer (context projection,
// sensitive-word filter, rate limiting) from config.
func (p *TraeProxy) SetProtection(cfg protect.Config) {
	p.projector = protect.NewProjector(cfg.MaxPayloadBytes, cfg.MaxMessageBytes)
	p.filter = protect.NewFilter(cfg.FilterEnabled, cfg.FilterReplacements)
	p.limiter = protect.NewLimiter(cfg.MaxConcurrent, time.Duration(cfg.MinIntervalMs)*time.Millisecond)
}

// SetRequestTimeout configures the upstream HTTP client timeout.
func (p *TraeProxy) SetRequestTimeout(d time.Duration) {
	if d > 0 && p.client != nil {
		p.client.Timeout = d
	}
}

// RequestTimeout returns the current HTTP client timeout.
func (p *TraeProxy) RequestTimeout() time.Duration {
	if p.client != nil {
		return p.client.Timeout
	}
	return 0
}

// applyProtection compresses and sanitizes the outbound message list.
func (p *TraeProxy) applyProtection(req *ChatCompletionRequest) {
	if p.projector != nil {
		views := make([]protect.MessageView, len(req.Messages))
		for i, m := range req.Messages {
			views[i] = protect.MessageView{Role: m.Role, Content: m.Content}
		}
		projected := p.projector.Project(views)
		for i := range projected {
			req.Messages[i].Content = projected[i].Content
		}
	}
	if p.filter != nil {
		for i := range req.Messages {
			req.Messages[i].Content = p.filter.Apply(req.Messages[i].Content)
		}
	}
}

// GetQueueMonitor returns the queue monitor
func (p *TraeProxy) GetQueueMonitor() *queue.Monitor {
	return p.queue
}

// TokenPool exposes the credential pool (account status endpoint).
func (p *TraeProxy) TokenPool() *auth.Pool {
	return p.tokens
}

// ChatCompletion forwards a chat request upstream and streams normalized
// events to handle. On upstream auth failure (401/403) the offending
// account is reported stale and the request is retried once with the next
// account from the pool.
func (p *TraeProxy) ChatCompletion(req *ChatCompletionRequest, handle StreamHandler) error {
	p.applyProtection(req)

	channel := models.ResolveChannel(req.ModelName)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		token, accountName, err := p.tokens.GetToken()
		if err != nil {
			return fmt.Errorf("failed to get token: %w", err)
		}
		if attempt == 0 {
			p.logger.Info("using account", "account", accountName, "model", req.ModelName, "channel", channel)
		}

		var status int
		if channel == models.ChannelAgentTask {
			status, err = p.doAgentTaskCompletion(req, token, handle)
		} else {
			status, err = p.doChatCompletion(req, token, handle)
		}
		if err == nil {
			p.tokens.ReportSuccess(accountName)
			return nil
		}
		lastErr = err

		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			p.logger.Warn("upstream auth rejected, rotating account", "account", accountName, "status", status)
			p.tokens.ReportFailure(accountName)
			continue
		}
		return err
	}
	return lastErr
}

// doChatCompletion performs a single upstream attempt; the returned status
// is the upstream HTTP status (0 when no response was received).
func (p *TraeProxy) doChatCompletion(req *ChatCompletionRequest, token string, handle StreamHandler) (int, error) {
	release := p.limiter.Acquire()
	defer release()

	ids := NewRequestIDs()
	if req.SessionID == "" {
		req.SessionID = ids.SessionID
	}
	if req.ConversationID == "" {
		req.ConversationID = ids.ConversationID
	}
	if req.TaskID == "" {
		req.TaskID = ids.TaskID
	}

	// Resolve the upstream model_name and encrypt the conversation into
	// the message field (shape verified against the live backend: preset
	// models answer 200 + SSE; anything else is 4023 MODEL_NOT_EXISTED).
	modelName := models.Default().UpstreamFor(req.ModelName)
	body, pin, requestAt, err := buildChatBody(req, modelName)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := config.AgentDomain + config.EndpointLLMRawChat
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	p.setHeadersWithIDs(httpReq, token, ids)
	// get-svc plus the pin/at pair correlated with the encrypted message;
	// x-requested-at must equal the GCM AAD timestamp exactly.
	httpReq.Header.Set("get-svc", "1")
	httpReq.Header.Set("x-request-pin", pin)
	httpReq.Header.Set("x-requested-at", strconv.FormatInt(requestAt, 10))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return resp.StatusCode, fmt.Errorf("API error %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	contentType := resp.Header.Get("Content-Type")
	if req.Stream || strings.Contains(contentType, "text/event-stream") {
		return http.StatusOK, p.streamEvents(resp.Body, req.ModelName, handle)
	}

	// Non-streaming JSON body: parse as a single response payload.
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusOK, fmt.Errorf("failed to read response: %w", err)
	}
	if isSSEResponse(contentType, strings.TrimSpace(string(data[:min(len(data), 16)]))) {
		return http.StatusOK, p.streamEvents(bytes.NewReader(data), req.ModelName, handle)
	}
	wrapped, flush := WrapStreamHandler(handle)
	for _, evt := range parseUpstreamData(string(data)) {
		if err := wrapped(evt); err != nil {
			return http.StatusOK, err
		}
	}
	if err := flush(); err != nil {
		return http.StatusOK, err
	}
	return http.StatusOK, nil
}

// streamEvents reads the upstream SSE stream line by line (tolerating TCP
// fragmentation via the buffered sse.Reader) and emits normalized events.
// Inline <think>...</think> blocks inside text deltas are split out into
// EventReasoning here so every protocol adapter gets clean channels.
func (p *TraeProxy) streamEvents(body io.Reader, model string, handle StreamHandler) error {
	reader := sse.NewReader(body)
	handle, flush := WrapStreamHandler(handle)

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

		for _, se := range parseUpstreamEvent(evt.Event, evt.Data) {
			switch se.Type {
			case EventQueue:
				se.QueueMessage = fmt.Sprintf("Queue position: %d", se.QueuePosition)
				p.queue.SetQueueStatus(model, se.QueuePosition, se.QueueMessage)
				p.logger.Info("queue status", "model", model, "position", se.QueuePosition)
			case EventFinish:
				p.queue.SetQueueStatus(model, 0, "")
			}
			if err := handle(se); err != nil {
				return err
			}
		}
	}

	// Stream ended without an explicit finish: flush any held-back tail.
	if err := flush(); err != nil {
		return err
	}

	p.queue.SetQueueStatus(model, 0, "")
	return nil
}

// FetchModels fetches the raw llm_raw_chat model list from the Trae backend.
func (p *TraeProxy) FetchModels() (json.RawMessage, error) {
	return p.postAuthenticated(config.EndpointModelList, []byte(`{"type":"llm_raw_chat"}`), map[string]string{"get-svc": "1"})
}

// RefreshModelRegistry pulls the live preset model list from upstream
// (model_list type=llm_raw_chat) and merges it into the shared registry.
// Only is_preset entries are usable through llm_raw_chat; custom/connect
// models answer 4023 and are skipped during the merge.
func (p *TraeProxy) RefreshModelRegistry() {
	body, err := p.FetchModels()
	if err != nil {
		p.logger.Debug("model_list refresh failed", "error", err)
		return
	}
	added, updated, err := models.Default().RefreshFromModelList(body)
	if err != nil {
		p.logger.Debug("model_list parse failed", "error", err)
		return
	}
	p.logger.Info("model registry refreshed (model_list)", "added", added, "updated", updated)
}

// postAuthenticated POSTs a JSON body to an upstream endpoint with full
// fingerprint headers plus any endpoint-specific extras, and returns the
// raw response body.
func (p *TraeProxy) postAuthenticated(endpoint string, body []byte, extra map[string]string) (json.RawMessage, error) {
	token, accountName, err := p.tokens.GetToken()
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", config.AgentDomain+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	p.setHeaders(httpReq, token)
	for k, v := range extra {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		p.tokens.ReportFailure(accountName)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return io.ReadAll(resp.Body)
}

// setHeaders injects the full client fingerprint with a fresh id set.
func (p *TraeProxy) setHeaders(req *http.Request, token string) {
	p.setHeadersWithIDs(req, token, NewRequestIDs())
}

// setHeadersWithIDs injects fingerprint headers aligned with the captured
// real Trae client traffic (scripts/captured/agent_task_req_headers.json).
func (p *TraeProxy) setHeadersWithIDs(req *http.Request, token string, ids RequestIDs) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(config.HeaderIDEToken, token)
	req.Header.Set("User-Agent", "TraeClient/TTNet")
	req.Header.Set("app-version", config.IDEVersion)
	req.Header.Set("package-type", "stable_cn")
	req.Header.Set("x-request-id", ids.RequestID)
	req.Header.Set("x-trae-request-id", ids.TraeRequestID)
	req.Header.Set("x-requested-at", strconv.FormatInt(time.Now().Unix(), 10))

	for k, v := range p.device.Headers() {
		req.Header.Set(k, v)
	}
}

// detectQueueStatus inspects a raw SSE event for queue position updates
// (kept for tests; the streaming path uses typed EventQueue events).
func (p *TraeProxy) detectQueueStatus(evt *sse.Event, model string) {
	if evt.Data == "" {
		return
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(evt.Data), &data); err != nil {
		return
	}
	if pos, ok := data["queue_position"]; ok {
		if posNum, ok := pos.(float64); ok && posNum > 0 {
			msg := fmt.Sprintf("Queue position: %d", int(posNum))
			p.queue.SetQueueStatus(model, int(posNum), msg)
			p.logger.Info("queue status", "model", model, "position", int(posNum))
		}
	}
}
