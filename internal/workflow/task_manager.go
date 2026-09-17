package workflow

import (
	"fmt"
	"sync"
	"time"
)

// TaskStatus represents task status
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task represents a task
type Task struct {
	ID          string
	Type        string
	Title       string
	Description string
	Priority    ContextPriority
	Status      TaskStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AssignedTo  string // Agent ID
	Metadata    map[string]interface{}
}

// TaskManager manages tasks
type TaskManager struct {
	mu      sync.RWMutex
	tasks   map[string]*Task
	queue   []*Task
	workDir string
}

// NewTaskManager creates new task manager
func NewTaskManager(workDir string) *TaskManager {
	return &TaskManager{
		tasks:   make(map[string]*Task),
		queue:   make([]*Task, 0),
		workDir: workDir,
	}
}

// CreateTask creates a new task
func (tm *TaskManager) CreateTask(taskType, title, description string, priority ContextPriority) *Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task := &Task{
		ID:          fmt.Sprintf("TASK-%d", time.Now().UnixNano()),
		Type:        taskType,
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      TaskStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	tm.tasks[task.ID] = task
	tm.queue = append(tm.queue, task)

	fmt.Printf("[TaskManager] Created task: %s - %s\n", task.ID, task.Title)

	return task
}

// GetTask gets task by ID
func (tm *TaskManager) GetTask(taskID string) *Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.tasks[taskID]
}

// GetNextPendingTask gets next pending task sorted by priority
func (tm *TaskManager) GetNextPendingTask() *Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Sort queue by priority
	tm.sortByPriority()

	// Find first pending task
	for _, task := range tm.queue {
		if task.Status == TaskStatusPending {
			return task
		}
	}

	return nil
}

// AssignTask assigns task to agent
func (tm *TaskManager) AssignTask(taskID, agentID string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return false
	}

	task.AssignedTo = agentID
	task.Status = TaskStatusRunning
	task.UpdatedAt = time.Now()

	fmt.Printf("[TaskManager] Assigned task %s to agent %s\n", taskID, agentID)

	return true
}

// UpdateTaskStatus updates task status
func (tm *TaskManager) UpdateTaskStatus(taskID string, status TaskStatus) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return false
	}

	task.Status = status
	task.UpdatedAt = time.Now()

	fmt.Printf("[TaskManager] Updated task %s status to %s\n", taskID, status)

	return true
}

// CompleteTask marks task as completed
func (tm *TaskManager) CompleteTask(taskID string) bool {
	return tm.UpdateTaskStatus(taskID, TaskStatusCompleted)
}

// FailTask marks task as failed
func (tm *TaskManager) FailTask(taskID, reason string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return false
	}

	task.Status = TaskStatusFailed
	task.UpdatedAt = time.Now()
	task.Metadata["failure_reason"] = reason

	fmt.Printf("[TaskManager] Failed task %s: %s\n", taskID, reason)

	return true
}

// PauseTask pauses task
func (tm *TaskManager) PauseTask(taskID string) bool {
	return tm.UpdateTaskStatus(taskID, TaskStatusPaused)
}

// ResumeTask resumes paused task
func (tm *TaskManager) ResumeTask(taskID string) bool {
	return tm.UpdateTaskStatus(taskID, TaskStatusRunning)
}

// CancelTask cancels task
func (tm *TaskManager) CancelTask(taskID string) bool {
	return tm.UpdateTaskStatus(taskID, TaskStatusCancelled)
}

// GetTasksByStatus gets tasks by status
func (tm *TaskManager) GetTasksByStatus(status TaskStatus) []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make([]*Task, 0)
	for _, task := range tm.tasks {
		if task.Status == status {
			result = append(result, task)
		}
	}

	return result
}

// GetTasksByAgent gets tasks assigned to agent
func (tm *TaskManager) GetTasksByAgent(agentID string) []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make([]*Task, 0)
	for _, task := range tm.tasks {
		if task.AssignedTo == agentID {
			result = append(result, task)
		}
	}

	return result
}

// GetTaskCountByStatus gets count of tasks by status
func (tm *TaskManager) GetTaskCountByStatus(status TaskStatus) int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	count := 0
	for _, task := range tm.tasks {
		if task.Status == status {
			count++
		}
	}

	return count
}

// sortByPriority sorts task queue by priority (descending)
func (tm *TaskManager) sortByPriority() {
	for i := 0; i < len(tm.queue)-1; i++ {
		for j := i + 1; j < len(tm.queue); j++ {
			if tm.queue[i].Priority < tm.queue[j].Priority {
				tm.queue[i], tm.queue[j] = tm.queue[j], tm.queue[i]
			}
		}
	}
}

// GetStats gets task manager statistics
func (tm *TaskManager) GetStats() TaskStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return TaskStats{
		Total:     len(tm.tasks),
		Pending:   tm.getTaskCountByStatus(TaskStatusPending),
		Running:   tm.getTaskCountByStatus(TaskStatusRunning),
		Paused:    tm.getTaskCountByStatus(TaskStatusPaused),
		Completed: tm.getTaskCountByStatus(TaskStatusCompleted),
		Failed:    tm.getTaskCountByStatus(TaskStatusFailed),
		Cancelled: tm.getTaskCountByStatus(TaskStatusCancelled),
	}
}

// getTaskCountByStatus gets count of tasks by status (internal, no lock)
func (tm *TaskManager) getTaskCountByStatus(status TaskStatus) int {
	count := 0
	for _, task := range tm.tasks {
		if task.Status == status {
			count++
		}
	}
	return count
}

// TaskHelper provides task operations helper
type TaskHelper struct {
	tm *TaskManager
}

// NewTaskHelper creates task helper
func NewTaskHelper(tm *TaskManager) *TaskHelper {
	return &TaskHelper{tm: tm}
}

// CreateTask creates a task
func (th *TaskHelper) CreateTask(taskType, title, description string, priority ContextPriority) *Task {
	return th.tm.CreateTask(taskType, title, description, priority)
}

// GetTask gets a task by ID
func (th *TaskHelper) GetTask(taskID string) *Task {
	return th.tm.GetTask(taskID)
}

// UpdateTask updates task status
func (th *TaskHelper) UpdateTask(taskID string, status TaskStatus) {
	th.tm.UpdateTaskStatus(taskID, status)
}

// GetTasksByAgent gets tasks assigned to agent
func (th *TaskHelper) GetTasksByAgent(agentID string) []*Task {
	return th.tm.GetTasksByAgent(agentID)
}

// TaskStats represents task statistics
type TaskStats struct {
	Total     int
	Pending   int
	Running   int
	Paused    int
	Completed int
	Failed    int
	Cancelled int
}
