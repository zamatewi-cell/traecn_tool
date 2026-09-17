package workflow

import (
	"context"
)

// AgentState represents agent state
type AgentState string

const (
	StateIdle      AgentState = "idle"
	StateRunning   AgentState = "running"
	StatePaused    AgentState = "paused"
	StateCompleted AgentState = "completed"
	StateFailed    AgentState = "failed"
)

// Progress represents agent progress
type Progress struct {
	Percentage int    // 0-100
	Status     string // Status message
	Step       int    // Current step
	TotalSteps int    // Total steps
}

// Agent interface defines the contract for all agents
type Agent interface {
	Run(ctx context.Context, task *Task) error
	GetStatus() AgentState
	GetProgress() Progress
	GetName() string
	GetID() string
}
