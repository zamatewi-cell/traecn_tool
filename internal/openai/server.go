package openai

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/models"
	"github.com/zamatewi-cell/traecn_tool/internal/openai/handlers"
	"github.com/zamatewi-cell/traecn_tool/internal/openai/middleware"
	"github.com/zamatewi-cell/traecn_tool/internal/protocol"
	"github.com/zamatewi-cell/traecn_tool/internal/proxy"
)

// Server is the OpenAI-compatible API server
type Server struct {
	proxy   *proxy.TraeProxy
	logger  *slog.Logger
	mux     *http.ServeMux
	apiKeys []string
	handler http.Handler
}

// ServerConfig holds server configuration
type ServerConfig struct {
	APIKeys []string
}

// NewServer creates an OpenAI API server
func NewServer(p *proxy.TraeProxy, logger *slog.Logger, config *ServerConfig) *Server {
	s := &Server{
		proxy:   p,
		logger:  logger,
		mux:     http.NewServeMux(),
		apiKeys: []string{},
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
	s.mux.HandleFunc("GET /health", s.handleHealth)
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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "0.1.0"})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"name":    "trae-proxy",
		"version": "0.1.0",
		"docs":    "https://github.com/zamatewi-cell/traecn_tool",
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

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe(addr string) error {
	s.logger.Info("starting OpenAI-compatible API server", "addr", addr)
	s.logger.Info(fmt.Sprintf("API: http://localhost%s/v1/chat/completions", addr))
	s.logger.Info(fmt.Sprintf("Models: http://localhost%s/v1/models", addr))
	s.logger.Info(fmt.Sprintf("Queue: http://localhost%s/v1/queue/status", addr))
	return http.ListenAndServe(addr, s)
}
