package agent

import (
	"context"
	"fmt"

	"github.com/zamatewi-cell/traecn_tool/internal/workflow"
)

// APIAgent develops OpenAI-compatible API layer
type APIAgent struct {
	*BaseAgent
	ServerInstance interface{}
	Routes         []RouteInfo
	Middleware     []MiddlewareInfo
}

// RouteInfo represents HTTP route information
type RouteInfo struct {
	Method      string
	Path        string
	Handler     string
	Description string
}

// MiddlewareInfo represents middleware information
type MiddlewareInfo struct {
	Name        string
	Description string
}

// NewAPIAgent creates an API agent
func NewAPIAgent(ctxManager *workflow.ContextManager, taskManager *workflow.TaskManager, eventBus *workflow.EventBus) *APIAgent {
	return &APIAgent{
		BaseAgent:  NewBaseAgent("api-agent", "API Agent", ctxManager, taskManager, eventBus),
		Routes:     make([]RouteInfo, 0),
		Middleware: make([]MiddlewareInfo, 0),
	}
}

// Run starts API development
func (a *APIAgent) Run(ctx context.Context, task *workflow.Task) error {
	a.SetStatus(StateRunning)

	defer func() {
		if r := recover(); r != nil {
			a.SetStatus(StateFailed)
			a.PublishEvent(workflow.EventError, map[string]interface{}{
				"error": fmt.Sprintf("%v", r),
			})
		}
	}()

	// Execute task based on type
	switch task.Type {
	case "api_server":
		return a.implementServer(ctx, task)
	case "api_routes":
		return a.implementRoutes(ctx, task)
	case "api_middleware":
		return a.implementMiddleware(ctx, task)
	default:
		return a.implementRoutes(ctx, task)
	}
}

// implementServer implements HTTP server
func (a *APIAgent) implementServer(ctx context.Context, task *workflow.Task) error {
	a.SetProgress(20, "Setting up Gin framework", 1, 4)
	a.SetProgress(50, "Configuring routes", 2, 4)
	a.SetProgress(80, "Starting server", 3, 4)
	a.SetProgress(100, "Server implementation completed", 4, 4)

	// TODO: Implement Gin server setup

	a.SetStatus(StateCompleted)

	a.PublishEvent(workflow.EventTaskCompleted, map[string]interface{}{
		"task_id": task.ID,
		"server":  "Gin",
	})

	return nil
}

// implementRoutes implements API routes
func (a *APIAgent) implementRoutes(ctx context.Context, task *workflow.Task) error {
	a.SetProgress(30, "Implementing /v1/chat/completions", 1, 3)

	// Add route info
	a.Routes = append(a.Routes, RouteInfo{
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Handler:     "ChatCompletionsHandler",
		Description: "OpenAI-compatible chat completions endpoint",
	})

	a.SetProgress(60, "Implementing /v1/models", 2, 3)

	a.Routes = append(a.Routes, RouteInfo{
		Method:      "GET",
		Path:        "/v1/models",
		Handler:     "ListModelsHandler",
		Description: "List available models",
	})

	a.SetProgress(100, "Routes implementation completed", 3, 3)

	a.SetStatus(StateCompleted)

	a.PublishEvent(workflow.EventTaskCompleted, map[string]interface{}{
		"task_id":      task.ID,
		"routes_count": len(a.Routes),
	})

	return nil
}

// implementMiddleware implements middleware
func (a *APIAgent) implementMiddleware(ctx context.Context, task *workflow.Task) error {
	a.SetProgress(30, "Implementing authentication middleware", 1, 3)

	a.Middleware = append(a.Middleware, MiddlewareInfo{
		Name:        "AuthMiddleware",
		Description: "Validates API key and Token",
	})

	a.SetProgress(60, "Implementing rate limiting middleware", 2, 3)

	a.Middleware = append(a.Middleware, MiddlewareInfo{
		Name:        "RateLimitMiddleware",
		Description: "Rate limiting per API key",
	})

	a.SetProgress(100, "Middleware implementation completed", 3, 3)

	a.SetStatus(StateCompleted)

	a.PublishEvent(workflow.EventTaskCompleted, map[string]interface{}{
		"task_id":          task.ID,
		"middleware_count": len(a.Middleware),
	})

	return nil
}

// GetRoutes gets implemented routes
func (a *APIAgent) GetRoutes() []RouteInfo {
	return a.Routes
}

// GetMiddleware gets implemented middleware
func (a *APIAgent) GetMiddleware() []MiddlewareInfo {
	return a.Middleware
}
