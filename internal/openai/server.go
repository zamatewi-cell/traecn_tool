package openai

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/db"
	"github.com/zamatewi-cell/traecn_tool/internal/models"
	"github.com/zamatewi-cell/traecn_tool/internal/openai/handlers"
	"github.com/zamatewi-cell/traecn_tool/internal/openai/middleware"
	"github.com/zamatewi-cell/traecn_tool/internal/protocol"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
	"github.com/zamatewi-cell/traecn_tool/internal/traeapi"
	"github.com/zamatewi-cell/traecn_tool/internal/version"
)

//go:embed dashboard.html
var dashboardHTML []byte

// Server is the OpenAI-compatible API server
type Server struct {
	proxy      *proxy.TraeProxy
	traeClient *traeapi.Client
	logger     *slog.Logger
	mux        *http.ServeMux
	apiKeys    []string
	handler    http.Handler
}

// ServerConfig holds server configuration
type ServerConfig struct {
	APIKeys []string
}

// NewServer creates an OpenAI API server
func NewServer(p *proxy.TraeProxy, logger *slog.Logger, config *ServerConfig) *Server {
	s := &Server{
		proxy:      p,
		traeClient: traeapi.NewClient(""),
		logger:     logger,
		mux:        http.NewServeMux(),
		apiKeys:    []string{},
	}

	if config != nil && len(config.APIKeys) > 0 {
		s.apiKeys = config.APIKeys
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// Create handlers
	chatHandler := handlers.NewChatHandler(s.proxy)
	completionsHandler := handlers.NewCompletionsHandler()
	anthropicHandler := protocol.NewAnthropicHandler(s.proxy)
	responsesHandler := protocol.NewResponsesHandler(s.proxy)

	// Create middlewares
	corsMiddleware := middleware.NewCORSMiddleware()
	loggerMiddleware := middleware.NewLoggerMiddleware(s.logger)

	// Create API key auth middleware (optional)
	var authMiddleware *middleware.APIKeyAuthMiddleware
	if len(s.apiKeys) > 0 {
		authMiddleware = middleware.NewAPIKeyAuthMiddleware(s.apiKeys)
	}

	// Register routes
	s.mux.HandleFunc("GET /v1/models", s.handleModels)
	s.mux.HandleFunc("POST /v1/chat/completions", chatHandler.HandleChatCompletions)
	s.mux.HandleFunc("POST /v1/completions", completionsHandler.HandleCompletions)
	s.mux.HandleFunc("POST /v1/messages", anthropicHandler.HandleMessages)
	s.mux.HandleFunc("POST /v1/responses", responsesHandler.HandleResponses)
	s.mux.HandleFunc("GET /v1/queue/status", s.handleQueueStatus)
	s.mux.HandleFunc("GET /v1/accounts", s.handleAccounts)
	s.mux.HandleFunc("GET /v1/trae/profile", s.handleTraeProfile)
	s.mux.HandleFunc("GET /v1/trae/billing_history", s.handleTraeBillingHistory)
	s.mux.HandleFunc("GET /v1/proxy/logs", s.handleProxyLogs)
	s.mux.HandleFunc("DELETE /v1/proxy/logs", s.handleClearProxyLogs)
	s.mux.HandleFunc("GET /v1/proxy/stats", s.handleProxyStats)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /dashboard", s.handleDashboard)
	s.mux.HandleFunc("GET /", s.handleRoot)

	// Apply middlewares
	var handler http.Handler = s.mux
	handler = corsMiddleware.Middleware(handler)
	handler = loggerMiddleware.Middleware(handler)
	if authMiddleware != nil {
		handler = authMiddleware.Middleware(handler)
	}

	// Store the wrapped handler
	s.handler = handler
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	type modelObj struct {
		ID            string `json:"id"`
		Object        string `json:"object"`
		Created       int64  `json:"created"`
		OwnedBy       string `json:"owned_by"`
		DisplayName   string `json:"display_name,omitempty"`
		MaxTokens     int    `json:"max_tokens,omitempty"`
		ContextWindow int    `json:"context_window,omitempty"`
	}

	var modelList []modelObj
	for _, m := range models.Default().List() {
		modelList = append(modelList, modelObj{
			ID:            m.ModelID,
			Object:        "model",
			Created:       time.Now().Unix(),
			OwnedBy:       m.Provider,
			DisplayName:   m.DisplayName,
			MaxTokens:     m.MaxTokens,
			ContextWindow: m.ContextWindow,
		})
	}

	resp := map[string]interface{}{
		"object": "list",
		"data":   modelList,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleQueueStatus(w http.ResponseWriter, r *http.Request) {
	model := r.URL.Query().Get("model")
	monitor := s.proxy.GetQueueMonitor()

	if model != "" {
		writeJSON(w, http.StatusOK, monitor.GetQueueStatus(model))
	} else {
		writeJSON(w, http.StatusOK, monitor.GetAllStatus())
	}
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"accounts": s.proxy.TokenPool().GetAccounts(),
	})
}

func (s *Server) handleTraeProfile(w http.ResponseWriter, r *http.Request) {
	token, _, err := s.proxy.TokenPool().GetToken()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "no_token", "未找到可用的 Trae 账号凭据: "+err.Error())
		return
	}

	profile, err := s.traeClient.GetFullProfile(r.Context(), token, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "trae_api_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handleTraeBillingHistory(w http.ResponseWriter, r *http.Request) {
	token, _, err := s.proxy.TokenPool().GetToken()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "no_token", "未找到可用的 Trae 账号凭据: "+err.Error())
		return
	}

	page := 1
	pageSize := 20
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	records, total, err := s.traeClient.GetUsageRecords(r.Context(), token, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "trae_api_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":   total,
		"page":    page,
		"records": records,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version.Version})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(dashboardHTML)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// 如果是浏览器页面访问，优先展示 WebUI 控制台仪表盘
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/html") || r.URL.Query().Get("format") == "html" {
		s.handleDashboard(w, r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"name":      "trae-proxy",
		"version":   version.Version,
		"dashboard": "/dashboard",
		"docs":      "https://github.com/zamatewi-cell/traecn_tool",
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, errType string, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{"type": errType, "message": message},
	})
}

func (s *Server) handleProxyLogs(w http.ResponseWriter, r *http.Request) {
	store := db.GetGlobalStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage_disabled", "SQLite storage not initialized")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	model := r.URL.Query().Get("model")

	logs, total, err := store.QueryLogs(r.Context(), page, pageSize, model)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	if logs == nil {
		logs = []*db.LogRecord{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (s *Server) handleClearProxyLogs(w http.ResponseWriter, r *http.Request) {
	store := db.GetGlobalStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage_disabled", "SQLite storage not initialized")
		return
	}

	if err := store.ClearLogs(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Activity logs cleared",
	})
}

func (s *Server) handleProxyStats(w http.ResponseWriter, r *http.Request) {
	store := db.GetGlobalStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "storage_disabled", "SQLite storage not initialized")
		return
	}

	stats, err := store.GetStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe(addr string) error {
	s.logger.Info("starting OpenAI-compatible API server", "addr", addr)
	host := addr
	if strings.HasPrefix(addr, ":") {
		host = "localhost" + addr
	} else if strings.HasPrefix(addr, "0.0.0.0:") {
		host = "127.0.0.1" + strings.TrimPrefix(addr, "0.0.0.0")
	}
	baseURL := fmt.Sprintf("http://%s", host)
	s.logger.Info(fmt.Sprintf("Dashboard: %s/", baseURL))
	s.logger.Info(fmt.Sprintf("API Base:  %s/v1", baseURL))
	s.logger.Info(fmt.Sprintf("Models:    %s/v1/models", baseURL))
	s.logger.Info(fmt.Sprintf("Queue:     %s/v1/queue/status", baseURL))
	return http.ListenAndServe(addr, s)
}
