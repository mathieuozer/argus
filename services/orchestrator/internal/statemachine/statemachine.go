package statemachine

import (
	"fmt"
	"sync"
	"time"
)

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	StatusPending          TaskStatus = "pending"
	StatusRunning          TaskStatus = "running"
	StatusCompleted        TaskStatus = "completed"
	StatusFailed           TaskStatus = "failed"
	StatusAwaitingApproval TaskStatus = "awaiting_approval"
)

// validTransitions defines the allowed state transitions.
var validTransitions = map[TaskStatus][]TaskStatus{
	StatusPending:          {StatusRunning, StatusFailed},
	StatusRunning:          {StatusCompleted, StatusFailed, StatusAwaitingApproval},
	StatusAwaitingApproval: {StatusRunning, StatusFailed},
}

// Task represents a task in the state machine.
type Task struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	AgentID     string      `json:"agent_id"`
	Status      TaskStatus  `json:"status"`
	InputHash   string      `json:"input_hash"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	CostUSD     float64     `json:"cost_usd"`
	TokensUsed  int64       `json:"tokens_used"`
}

// StateMachine manages task state transitions.
type StateMachine struct {
	mu    sync.RWMutex
	tasks map[string]*Task // task_id -> task
}

// New creates a new state machine.
func New() *StateMachine {
	return &StateMachine{
		tasks: make(map[string]*Task),
	}
}

// CreateTask creates a new task in pending state.
func (sm *StateMachine) CreateTask(id, tenantID, agentID, inputHash string) *Task {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	task := &Task{
		ID:        id,
		TenantID:  tenantID,
		AgentID:   agentID,
		Status:    StatusPending,
		InputHash: inputHash,
		StartedAt: time.Now(),
	}
	sm.tasks[id] = task
	return task
}

// Transition attempts to move a task to a new status.
func (sm *StateMachine) Transition(taskID string, newStatus TaskStatus) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	task, ok := sm.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	allowed, ok := validTransitions[task.Status]
	if !ok {
		return fmt.Errorf("no transitions from status %s", task.Status)
	}

	for _, s := range allowed {
		if s == newStatus {
			task.Status = newStatus
			if newStatus == StatusCompleted || newStatus == StatusFailed {
				now := time.Now()
				task.CompletedAt = &now
			}
			return nil
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", task.Status, newStatus)
}

// Get retrieves a task by ID.
func (sm *StateMachine) Get(taskID string) (*Task, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	task, ok := sm.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return task, nil
}

// ListByTenant returns all tasks for a given tenant.
func (sm *StateMachine) ListByTenant(tenantID string) []*Task {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []*Task
	for _, t := range sm.tasks {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	return result
}

// ListByAgent returns all tasks for a given agent within a tenant.
func (sm *StateMachine) ListByAgent(tenantID, agentID string) []*Task {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var result []*Task
	for _, t := range sm.tasks {
		if t.TenantID == tenantID && t.AgentID == agentID {
			result = append(result, t)
		}
	}
	return result
}

// UpdateCost updates the cost and token usage for a task.
func (sm *StateMachine) UpdateCost(taskID string, costUSD float64, tokensUsed int64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	task, ok := sm.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.CostUSD += costUSD
	task.TokensUsed += tokensUsed
	return nil
}
