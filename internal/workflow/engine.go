package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkflowEngine orchestrates all agents and manages task execution
type WorkflowEngine struct {
	ctx         context.Context
	cancel      context.CancelFunc
	ctxManager  *ContextManager
	taskManager *TaskManager
	eventBus    *EventBus
	agents      map[string]Agent
	agentQueue  chan Agent
	wg          sync.WaitGroup
	mu          sync.RWMutex
	isRunning   bool
	config      EngineConfig
}

// EngineConfig holds engine configuration
type EngineConfig struct {
	MaxConcurrentAgents int
	TaskPollInterval    time.Duration
	EnableAutoScaling   bool
}

// DefaultEngineConfig returns default configuration
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxConcurrentAgents: 3,
		TaskPollInterval:    time.Second * 5,
		EnableAutoScaling:   true,
	}
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(ctxManager *ContextManager, taskManager *TaskManager, eventBus *EventBus) *WorkflowEngine {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &WorkflowEngine{
		ctx:         ctx,
		cancel:      cancel,
		ctxManager:  ctxManager,
		taskManager: taskManager,
		eventBus:    eventBus,
		agents:      make(map[string]Agent),
		agentQueue:  make(chan Agent, 10),
		config:      DefaultEngineConfig(),
	}

	// Subscribe to events
	eventBus.Subscribe(EventTaskCreated, engine.handleTaskCreated)
	eventBus.Subscribe(EventTaskCompleted, engine.handleTaskCompleted)
	eventBus.Subscribe(EventProgressUpdate, engine.handleProgressUpdate)

	return engine
}

// RegisterAgent registers an agent with the engine
func (e *WorkflowEngine) RegisterAgent(a Agent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.agents[a.GetID()] = a
	fmt.Printf("[WorkflowEngine] Registered agent: %s\n", a.GetID())
}

// Start starts the workflow engine
func (e *WorkflowEngine) Start() error {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return fmt.Errorf("engine already running")
	}
	e.isRunning = true
	e.mu.Unlock()

	fmt.Println("[WorkflowEngine] Starting workflow engine...")

	// Start agent workers
	for i := 0; i < e.config.MaxConcurrentAgents; i++ {
		e.wg.Add(1)
		go e.agentWorker(i)
	}

	// Start task scheduler
	e.wg.Add(1)
	go e.taskScheduler()

	fmt.Println("[WorkflowEngine] Workflow engine started")

	return nil
}

// Stop stops the workflow engine
func (e *WorkflowEngine) Stop() error {
	e.mu.Lock()
	if !e.isRunning {
		e.mu.Unlock()
		return fmt.Errorf("engine not running")
	}
	e.isRunning = false
	e.mu.Unlock()

	fmt.Println("[WorkflowEngine] Stopping workflow engine...")

	e.cancel()
	e.wg.Wait()

	fmt.Println("[WorkflowEngine] Workflow engine stopped")

	return nil
}

// agentWorker runs agent tasks
func (e *WorkflowEngine) agentWorker(workerID int) {
	defer e.wg.Done()

	fmt.Printf("[WorkflowEngine] Agent worker %d started\n", workerID)

	for {
		select {
		case <-e.ctx.Done():
			fmt.Printf("[WorkflowEngine] Agent worker %d stopped\n", workerID)
			return
		case a := <-e.agentQueue:
			e.executeAgent(a, workerID)
		}
	}
}

// executeAgent executes an agent's task
func (e *WorkflowEngine) executeAgent(a Agent, workerID int) {
	fmt.Printf("[WorkflowEngine] Worker %d executing agent: %s\n", workerID, a.GetID())

	// Get next pending task
	task := e.taskManager.GetNextPendingTask()
	if task == nil {
		fmt.Println("[WorkflowEngine] No pending tasks")
		return
	}

	// Assign task to agent
	e.taskManager.AssignTask(task.ID, a.GetID())

	// Execute agent
	err := a.Run(e.ctx, task)
	if err != nil {
		fmt.Printf("[WorkflowEngine] Agent %s failed: %v\n", a.GetID(), err)
		e.taskManager.FailTask(task.ID, err.Error())
		return
	}

	fmt.Printf("[WorkflowEngine] Agent %s completed successfully\n", a.GetID())
}

// taskScheduler schedules tasks to agents
func (e *WorkflowEngine) taskScheduler() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.TaskPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.schedulePendingTasks()
		}
	}
}

// schedulePendingTasks schedules pending tasks to available agents
func (e *WorkflowEngine) schedulePendingTasks() {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.isRunning {
		return
	}

	// Get all pending tasks
	tasks := e.taskManager.GetTasksByStatus(TaskStatusPending)

	for _, task := range tasks {
		// Find suitable agent for task
		agent := e.findSuitableAgent(task)
		if agent != nil {
			select {
			case e.agentQueue <- agent:
				fmt.Printf("[WorkflowEngine] Scheduled task %s to agent %s\n", task.ID, agent.GetID())
			default:
				fmt.Println("[WorkflowEngine] Agent queue full, task waiting")
			}
		}
	}
}

// findSuitableAgent finds the most suitable agent for a task
func (e *WorkflowEngine) findSuitableAgent(task *Task) Agent {
	// Simple round-robin for now
	// TODO: Implement intelligent agent selection based on:
	// - Agent capabilities
	// - Current load
	// - Task priority
	// - Agent specialization

	for _, a := range e.agents {
		if a.GetStatus() == StateIdle {
			return a
		}
	}

	return nil
}

// handleTaskCreated handles task created events
func (e *WorkflowEngine) handleTaskCreated(event *Event) {
	fmt.Printf("[WorkflowEngine] Task created: %v\n", event.Payload)
	// Trigger immediate scheduling
	go e.schedulePendingTasks()
}

// handleTaskCompleted handles task completed events
func (e *WorkflowEngine) handleTaskCompleted(event *Event) {
	fmt.Printf("[WorkflowEngine] Task completed: %v\n", event.Payload)
	// Schedule next tasks
	go e.schedulePendingTasks()
}

// handleProgressUpdate handles progress update events
func (e *WorkflowEngine) handleProgressUpdate(event *Event) {
	// Log progress updates
	fmt.Printf("[WorkflowEngine] Progress update: %v\n", event.Payload)
}

// GetAgentStatus gets status of all agents
func (e *WorkflowEngine) GetAgentStatus() map[string]AgentState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := make(map[string]AgentState)
	for id, a := range e.agents {
		status[id] = a.GetStatus()
	}

	return status
}

// GetStats gets engine statistics
func (e *WorkflowEngine) GetStats() EngineStats {
	return EngineStats{
		TotalAgents:    len(e.agents),
		ActiveAgents:   e.countActiveAgents(),
		PendingTasks:   e.taskManager.GetTaskCountByStatus(TaskStatusPending),
		RunningTasks:   e.taskManager.GetTaskCountByStatus(TaskStatusRunning),
		CompletedTasks: e.taskManager.GetTaskCountByStatus(TaskStatusCompleted),
		IsRunning:      e.isRunning,
	}
}

// countActiveAgents counts active agents
func (e *WorkflowEngine) countActiveAgents() int {
	count := 0
	for _, a := range e.agents {
		if a.GetStatus() == StateRunning {
			count++
		}
	}
	return count
}

// EngineStats represents engine statistics
type EngineStats struct {
	TotalAgents    int
	ActiveAgents   int
	PendingTasks   int
	RunningTasks   int
	CompletedTasks int
	IsRunning      bool
}
