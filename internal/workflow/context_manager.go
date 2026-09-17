package workflow

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ContextScope defines context scope type
type ContextScope string

const (
	ScopeGlobal    ContextScope = "global"    // Global shared context
	ScopeAgent     ContextScope = "agent"     // Agent private context
	ScopeTask      ContextScope = "task"      // Task specific context
	ScopeTemporary ContextScope = "temporary" // Temporary context (auto cleanup)
)

// ContextPriority defines context priority
type ContextPriority int

const (
	PriorityVIP      ContextPriority = 100 // VIP priority (highest)
	PriorityCritical ContextPriority = 100 // Critical data, never auto cleanup
	PriorityHigh     ContextPriority = 10  // Important data
	PriorityNormal   ContextPriority = 5   // Normal data
	PriorityLow      ContextPriority = 1   // Can be cleaned when memory is low
)

// ContextItem represents a single context item
type ContextItem struct {
	Key        string
	Value      interface{}
	Scope      ContextScope
	Priority   ContextPriority
	AgentID    string // Owner Agent ID (if scope is agent/task)
	TaskID     string // Owner Task ID (if scope is task)
	CreatedAt  time.Time
	ExpiresAt  time.Time // Auto cleanup time (0 = never expire)
	AccessNum  int       // Access count (for LRU)
	LastAccess time.Time // Last access time
}

// ContextManager manages all context items
type ContextManager struct {
	mu      sync.RWMutex
	items   map[string]*ContextItem // key -> item
	maxSize int                     // Maximum items (default 10000)
}

// NewContextManager creates a context manager
func NewContextManager(maxSize int) *ContextManager {
	if maxSize <= 0 {
		maxSize = 10000
	}

	cm := &ContextManager{
		items:   make(map[string]*ContextItem),
		maxSize: maxSize,
	}

	// Start cleanup goroutine
	go cm.cleanup()

	return cm
}

// Add adds a context item
func (cm *ContextManager) Add(key string, value interface{}, scope ContextScope, priority ContextPriority, agentID, taskID string, expire time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if need to cleanup
	if len(cm.items) >= cm.maxSize {
		cm.cleanupLowPriority()
	}

	expiresAt := time.Time{}
	if expire > 0 {
		expiresAt = time.Now().Add(expire)
	}

	cm.items[key] = &ContextItem{
		Key:        key,
		Value:      value,
		Scope:      scope,
		Priority:   priority,
		AgentID:    agentID,
		TaskID:     taskID,
		CreatedAt:  time.Now(),
		ExpiresAt:  expiresAt,
		AccessNum:  0,
		LastAccess: time.Now(),
	}

	log.Printf("[Context] Add: key=%s, scope=%s, priority=%d", key, scope, priority)
}

// Get gets a context item
func (cm *ContextManager) Get(key string) (interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	item, exists := cm.items[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
		return nil, false
	}

	// Update access info
	cm.mu.RUnlock()
	cm.mu.Lock()
	item.AccessNum++
	item.LastAccess = time.Now()
	cm.mu.Unlock()
	cm.mu.RLock()

	return item.Value, true
}

// GetByAgent gets all context items for an agent
func (cm *ContextManager) GetByAgent(agentID string) map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[string]interface{})
	for key, item := range cm.items {
		if item.AgentID == agentID && item.Scope == ScopeAgent {
			// Check if not expired
			if item.ExpiresAt.IsZero() || time.Now().Before(item.ExpiresAt) {
				result[key] = item.Value
			}
		}
	}

	return result
}

// GetByTask gets all context items for a task
func (cm *ContextManager) GetByTask(taskID string) map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[string]interface{})
	for key, item := range cm.items {
		if item.TaskID == taskID && item.Scope == ScopeTask {
			// Check if not expired
			if item.ExpiresAt.IsZero() || time.Now().Before(item.ExpiresAt) {
				result[key] = item.Value
			}
		}
	}

	return result
}

// GetGlobal gets all global context items
func (cm *ContextManager) GetGlobal() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[string]interface{})
	for key, item := range cm.items {
		if item.Scope == ScopeGlobal {
			// Check if not expired
			if item.ExpiresAt.IsZero() || time.Now().Before(item.ExpiresAt) {
				result[key] = item.Value
			}
		}
	}

	return result
}

// Remove removes a context item
func (cm *ContextManager) Remove(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.items, key)
	log.Printf("[Context] Remove: key=%s", key)
}

// ClearAgent clears all context items for an agent
func (cm *ContextManager) ClearAgent(agentID string) int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	count := 0
	for key, item := range cm.items {
		if item.AgentID == agentID {
			delete(cm.items, key)
			count++
		}
	}

	log.Printf("[Context] ClearAgent: agentID=%s, count=%d", agentID, count)
	return count
}

// ClearTask clears all context items for a task
func (cm *ContextManager) ClearTask(taskID string) int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	count := 0
	for key, item := range cm.items {
		if item.TaskID == taskID {
			delete(cm.items, key)
			count++
		}
	}

	log.Printf("[Context] ClearTask: taskID=%s, count=%d", taskID, count)
	return count
}

// GetStats gets context statistics
func (cm *ContextManager) GetStats() ContextStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := ContextStats{
		Total: len(cm.items),
	}

	for _, item := range cm.items {
		switch item.Scope {
		case ScopeGlobal:
			stats.Global++
		case ScopeAgent:
			stats.ByAgent[item.AgentID]++
		case ScopeTask:
			stats.ByTask[item.TaskID]++
		}
	}

	return stats
}

// ContextStats represents context statistics
type ContextStats struct {
	Total   int
	Global  int
	ByAgent map[string]int
	ByTask  map[string]int
}

// cleanupLowPriority cleans up low priority items
func (cm *ContextManager) cleanupLowPriority() {
	// Find 10% of lowest priority items
	toRemove := cm.maxSize / 10
	if toRemove < 1 {
		toRemove = 1
	}

	// Sort by priority and access count
	type kv struct {
		Key   string
		Item  *ContextItem
		Score int // Priority * AccessNum (lower = remove first)
	}

	var items []kv
	for key, item := range cm.items {
		if item.Priority == PriorityCritical {
			continue // Never cleanup critical items
		}
		score := int(item.Priority) * (item.AccessNum + 1)
		items = append(items, kv{key, item, score})
	}

	// Sort by score (ascending)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].Score > items[j].Score {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Remove lowest score items
	removeCount := 0
	for _, item := range items {
		if removeCount >= toRemove {
			break
		}
		delete(cm.items, item.Key)
		removeCount++
	}

	log.Printf("[Context] Cleanup: removed %d low priority items", removeCount)
}

// cleanup periodically cleans up expired items
func (cm *ContextManager) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cm.mu.Lock()
		now := time.Now()
		for key, item := range cm.items {
			if !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
				delete(cm.items, key)
				log.Printf("[Context] Expired: key=%s", key)
			}
		}
		cm.mu.Unlock()
	}
}

// AgentContextHelper provides agent-specific context helper
type AgentContextHelper struct {
	cm      *ContextManager
	agentID string
}

// NewAgentContextHelper creates an agent context helper
func NewAgentContextHelper(cm *ContextManager, agentID string) *AgentContextHelper {
	return &AgentContextHelper{
		cm:      cm,
		agentID: agentID,
	}
}

// Set sets agent context
func (h *AgentContextHelper) Set(key string, value interface{}, priority ContextPriority, expire time.Duration) {
	fullKey := fmt.Sprintf("agent:%s:%s", h.agentID, key)
	h.cm.Add(fullKey, value, ScopeAgent, priority, h.agentID, "", expire)
}

// Get gets agent context
func (h *AgentContextHelper) Get(key string) (interface{}, bool) {
	fullKey := fmt.Sprintf("agent:%s:%s", h.agentID, key)
	return h.cm.Get(fullKey)
}

// GetAll gets all agent context
func (h *AgentContextHelper) GetAll() map[string]interface{} {
	return h.cm.GetByAgent(h.agentID)
}

// Remove removes agent context
func (h *AgentContextHelper) Remove(key string) {
	fullKey := fmt.Sprintf("agent:%s:%s", h.agentID, key)
	h.cm.Remove(fullKey)
}

// Clear clears all agent context
func (h *AgentContextHelper) Clear() int {
	return h.cm.ClearAgent(h.agentID)
}

// TaskContextHelper provides task-specific context helper
type TaskContextHelper struct {
	cm     *ContextManager
	taskID string
}

// NewTaskContextHelper creates a task context helper
func NewTaskContextHelper(cm *ContextManager, taskID string) *TaskContextHelper {
	return &TaskContextHelper{
		cm:     cm,
		taskID: taskID,
	}
}

// Set sets task context
func (h *TaskContextHelper) Set(key string, value interface{}, priority ContextPriority, expire time.Duration) {
	fullKey := fmt.Sprintf("task:%s:%s", h.taskID, key)
	h.cm.Add(fullKey, value, ScopeTask, PriorityNormal, "", h.taskID, expire)
}

// Get gets task context
func (h *TaskContextHelper) Get(key string) (interface{}, bool) {
	fullKey := fmt.Sprintf("task:%s:%s", h.taskID, key)
	return h.cm.Get(fullKey)
}

// GetAll gets all task context
func (h *TaskContextHelper) GetAll() map[string]interface{} {
	return h.cm.GetByTask(h.taskID)
}

// Clear clears all task context
func (h *TaskContextHelper) Clear() int {
	return h.cm.ClearTask(h.taskID)
}

// GlobalContextHelper provides global context helper
type GlobalContextHelper struct {
	cm *ContextManager
}

// NewGlobalContextHelper creates a global context helper
func NewGlobalContextHelper(cm *ContextManager) *GlobalContextHelper {
	return &GlobalContextHelper{cm: cm}
}

// Set sets global context
func (h *GlobalContextHelper) Set(key string, value interface{}, priority ContextPriority, expire time.Duration) {
	h.cm.Add(key, value, ScopeGlobal, priority, "", "", expire)
}

// Get gets global context
func (h *GlobalContextHelper) Get(key string) (interface{}, bool) {
	return h.cm.Get(key)
}

// GetAll gets all global context
func (h *GlobalContextHelper) GetAll() map[string]interface{} {
	return h.cm.GetGlobal()
}

// Remove removes global context
func (h *GlobalContextHelper) Remove(key string) {
	h.cm.Remove(key)
}

// Share shares context to an agent
func (h *GlobalContextHelper) Share(key string, agent Agent) bool {
	value, exists := h.Get(key)
	if !exists {
		return false
	}

	// Agent can access via global context or copy to agent context
	_ = value
	return true
}
