package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zamatewi-cell/traecn_tool/internal/workflow"
)

// AgentState represents agent state
type AgentState string

const (
	StateIdle      AgentState = "idle"      // Agent is idle
	StateRunning   AgentState = "running"   // Agent is running
	StatePaused    AgentState = "paused"    // Agent is paused
	StateCompleted AgentState = "completed" // Agent completed task
	StateFailed    AgentState = "failed"    // Agent failed
)

// BaseAgent provides base agent functionality
type BaseAgent struct {
	ID          string
	Name        string
	State       AgentState
	Progress    Progress
	ctxManager  *workflow.ContextManager
	taskManager *workflow.TaskManager
	eventBus    *workflow.EventBus
	mu          sync.RWMutex
}

// Progress represents task progress
type Progress struct {
	Percentage int    `json:"percentage"` // 0-100
	Status     string `json:"status"`     // Status description
	Step       int    `json:"step"`       // Current step
	Total      int    `json:"total"`      // Total steps
}

// NewBaseAgent creates a base agent
func NewBaseAgent(id, name string, ctxManager *workflow.ContextManager, taskManager *workflow.TaskManager, eventBus *workflow.EventBus) *BaseAgent {
	return &BaseAgent{
		ID:          id,
		Name:        name,
		State:       StateIdle,
		Progress:    Progress{Percentage: 0, Status: "Idle"},
		ctxManager:  ctxManager,
		taskManager: taskManager,
		eventBus:    eventBus,
	}
}

// GetStatus gets agent state
func (b *BaseAgent) GetStatus() AgentState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.State
}

// SetStatus sets agent state
func (b *BaseAgent) SetStatus(state AgentState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.State = state
}

// GetProgress gets agent progress
func (b *BaseAgent) GetProgress() Progress {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Progress
}

// SetProgress sets agent progress
func (b *BaseAgent) SetProgress(percentage int, status string, step, total int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Progress = Progress{
		Percentage: percentage,
		Status:     status,
		Step:       step,
		Total:      total,
	}
}

// GetName gets agent name
func (b *BaseAgent) GetName() string {
	return b.Name
}

// GetID gets agent ID
func (b *BaseAgent) GetID() string {
	return b.ID
}

// GetContextHelper gets context helper for this agent
func (b *BaseAgent) GetContextHelper() *workflow.AgentContextHelper {
	return workflow.NewAgentContextHelper(b.ctxManager, b.ID)
}

// GetTaskHelper gets task helper
func (b *BaseAgent) GetTaskHelper() *workflow.TaskHelper {
	return workflow.NewTaskHelper(b.taskManager)
}

// PublishEvent publishes an event
func (b *BaseAgent) PublishEvent(eventType workflow.EventType, data map[string]interface{}) {
	event := &workflow.Event{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      eventType,
		Source:    b.ID,
		Timestamp: time.Now(),
		Payload:   data,
	}

	b.eventBus.Publish(event)
}

// Run implements agent logic (to be overridden by subclasses)
func (b *BaseAgent) Run(ctx context.Context, task *workflow.Task) error {
	// Default implementation: just mark as completed
	b.SetStatus(StateCompleted)
	b.SetProgress(100, "Completed", 1, 1)
	return nil
}
