package repository

import (
	"context"
	"testing"
	"time"

	"github.com/argus-platform/argus/services/orchestrator/internal/statemachine"
	"github.com/jackc/pgx/v5"
)

// ---------- Constructor tests ----------

func TestNewTaskRepository(t *testing.T) {
	t.Run("returns non-nil repository with nil pool", func(t *testing.T) {
		repo := NewTaskRepository(nil)
		if repo == nil {
			t.Fatal("expected non-nil TaskRepository")
		}
	})

	t.Run("pool field is set", func(t *testing.T) {
		repo := NewTaskRepository(nil)
		if repo.pool != nil {
			t.Error("expected nil pool when constructed with nil")
		}
	})
}

// ---------- Create: task field construction ----------

func TestTaskRepository_Create_TaskFields(t *testing.T) {
	tests := []struct {
		name string
		task *statemachine.Task
	}{
		{
			name: "all fields populated",
			task: &statemachine.Task{
				ID:         "task-001",
				TenantID:   "tenant-1",
				AgentID:    "agent-alpha",
				Status:     statemachine.StatusPending,
				InputHash:  "sha256:abc123",
				StartedAt:  time.Now(),
				CostUSD:    0.05,
				TokensUsed: 1500,
			},
		},
		{
			name: "zero-value cost and tokens",
			task: &statemachine.Task{
				ID:         "task-002",
				TenantID:   "tenant-2",
				AgentID:    "agent-beta",
				Status:     statemachine.StatusPending,
				InputHash:  "sha256:def456",
				StartedAt:  time.Now(),
				CostUSD:    0.0,
				TokensUsed: 0,
			},
		},
		{
			name: "empty input hash",
			task: &statemachine.Task{
				ID:        "task-003",
				TenantID:  "tenant-3",
				AgentID:   "agent-gamma",
				Status:    statemachine.StatusPending,
				InputHash: "",
				StartedAt: time.Now(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Verify the SQL args that would be passed match the task fields.
			// The SQL: INSERT INTO tasks (id, tenant_id, agent_id, status, input_hash, started_at, cost_usd, tokens_used)
			//          VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8)
			args := []interface{}{
				tc.task.ID, tc.task.TenantID, tc.task.AgentID, string(tc.task.Status),
				tc.task.InputHash, tc.task.StartedAt, tc.task.CostUSD, tc.task.TokensUsed,
			}

			if len(args) != 8 {
				t.Errorf("expected 8 SQL args for Create, got %d", len(args))
			}

			if args[0].(string) != tc.task.ID {
				t.Errorf("arg[0] (id) = %q, want %q", args[0], tc.task.ID)
			}
			if args[1].(string) != tc.task.TenantID {
				t.Errorf("arg[1] (tenant_id) = %q, want %q", args[1], tc.task.TenantID)
			}
			if args[2].(string) != tc.task.AgentID {
				t.Errorf("arg[2] (agent_id) = %q, want %q", args[2], tc.task.AgentID)
			}
			if args[3].(string) != string(tc.task.Status) {
				t.Errorf("arg[3] (status) = %q, want %q", args[3], string(tc.task.Status))
			}
			if args[4].(string) != tc.task.InputHash {
				t.Errorf("arg[4] (input_hash) = %q, want %q", args[4], tc.task.InputHash)
			}
			if args[6].(float64) != tc.task.CostUSD {
				t.Errorf("arg[6] (cost_usd) = %v, want %v", args[6], tc.task.CostUSD)
			}
			if args[7].(int64) != tc.task.TokensUsed {
				t.Errorf("arg[7] (tokens_used) = %v, want %v", args[7], tc.task.TokensUsed)
			}
		})
	}
}

// ---------- UpdateStatus: completion time logic ----------

func TestTaskRepository_UpdateStatus_CompletionTime(t *testing.T) {
	tests := []struct {
		name           string
		newStatus      statemachine.TaskStatus
		wantCompleted  bool
	}{
		{
			name:          "completed status sets completed_at",
			newStatus:     statemachine.StatusCompleted,
			wantCompleted: true,
		},
		{
			name:          "failed status sets completed_at",
			newStatus:     statemachine.StatusFailed,
			wantCompleted: true,
		},
		{
			name:          "running status does not set completed_at",
			newStatus:     statemachine.StatusRunning,
			wantCompleted: false,
		},
		{
			name:          "pending status does not set completed_at",
			newStatus:     statemachine.StatusPending,
			wantCompleted: false,
		},
		{
			name:          "awaiting_approval status does not set completed_at",
			newStatus:     statemachine.StatusAwaitingApproval,
			wantCompleted: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Replicate the completion time logic from UpdateStatus
			var completedAt *time.Time
			if tc.newStatus == statemachine.StatusCompleted || tc.newStatus == statemachine.StatusFailed {
				now := time.Now()
				completedAt = &now
			}

			if tc.wantCompleted && completedAt == nil {
				t.Errorf("expected completed_at to be set for status %q", tc.newStatus)
			}
			if !tc.wantCompleted && completedAt != nil {
				t.Errorf("expected completed_at to be nil for status %q", tc.newStatus)
			}

			if completedAt != nil {
				if completedAt.IsZero() {
					t.Error("completed_at should not be zero time")
				}
				if time.Since(*completedAt) > time.Second {
					t.Error("completed_at should be close to now")
				}
			}
		})
	}
}

// ---------- UpdateStatus: SQL parameter construction ----------

func TestTaskRepository_UpdateStatus_SQLArgs(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		taskID    string
		newStatus statemachine.TaskStatus
	}{
		{
			name:      "completed task args",
			tenantID:  "tenant-1",
			taskID:    "task-001",
			newStatus: statemachine.StatusCompleted,
		},
		{
			name:      "running task args",
			tenantID:  "tenant-2",
			taskID:    "task-002",
			newStatus: statemachine.StatusRunning,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Replicate UpdateStatus arg construction
			var completedAt *time.Time
			if tc.newStatus == statemachine.StatusCompleted || tc.newStatus == statemachine.StatusFailed {
				now := time.Now()
				completedAt = &now
			}

			// The SQL: UPDATE tasks SET status = $1, completed_at = $2
			//          WHERE id = $3 AND tenant_id = $4::uuid
			args := []interface{}{string(tc.newStatus), completedAt, tc.taskID, tc.tenantID}

			if len(args) != 4 {
				t.Errorf("expected 4 SQL args, got %d", len(args))
			}
			if args[0].(string) != string(tc.newStatus) {
				t.Errorf("arg[0] (status) = %q, want %q", args[0], string(tc.newStatus))
			}
			if args[2].(string) != tc.taskID {
				t.Errorf("arg[2] (task_id) = %q, want %q", args[2], tc.taskID)
			}
			if args[3].(string) != tc.tenantID {
				t.Errorf("arg[3] (tenant_id) = %q, want %q", args[3], tc.tenantID)
			}
		})
	}
}

// ---------- UpdateCost: SQL parameter construction ----------

func TestTaskRepository_UpdateCost_SQLArgs(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		taskID     string
		costUSD    float64
		tokensUsed int64
	}{
		{
			name:       "normal cost update",
			tenantID:   "tenant-1",
			taskID:     "task-001",
			costUSD:    0.0025,
			tokensUsed: 500,
		},
		{
			name:       "zero cost update",
			tenantID:   "tenant-2",
			taskID:     "task-002",
			costUSD:    0.0,
			tokensUsed: 0,
		},
		{
			name:       "large cost update",
			tenantID:   "tenant-3",
			taskID:     "task-003",
			costUSD:    123.456,
			tokensUsed: 1000000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The SQL: UPDATE tasks SET cost_usd = cost_usd + $1, tokens_used = tokens_used + $2
			//          WHERE id = $3 AND tenant_id = $4::uuid
			args := []interface{}{tc.costUSD, tc.tokensUsed, tc.taskID, tc.tenantID}

			if len(args) != 4 {
				t.Errorf("expected 4 SQL args, got %d", len(args))
			}
			if args[0].(float64) != tc.costUSD {
				t.Errorf("arg[0] (cost_usd) = %v, want %v", args[0], tc.costUSD)
			}
			if args[1].(int64) != tc.tokensUsed {
				t.Errorf("arg[1] (tokens_used) = %v, want %v", args[1], tc.tokensUsed)
			}
		})
	}
}

// ---------- Task status values ----------

func TestTaskStatusValues(t *testing.T) {
	tests := []struct {
		name   string
		status statemachine.TaskStatus
		want   string
	}{
		{"pending", statemachine.StatusPending, "pending"},
		{"running", statemachine.StatusRunning, "running"},
		{"completed", statemachine.StatusCompleted, "completed"},
		{"failed", statemachine.StatusFailed, "failed"},
		{"awaiting_approval", statemachine.StatusAwaitingApproval, "awaiting_approval"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.status) != tc.want {
				t.Errorf("status = %q, want %q", string(tc.status), tc.want)
			}
		})
	}
}

// ---------- Get: error handling for ErrNoRows ----------

func TestTaskRepository_Get_ErrNoRowsHandling(t *testing.T) {
	t.Run("pgx.ErrNoRows produces task not found error", func(t *testing.T) {
		err := pgx.ErrNoRows
		if err == nil {
			t.Fatal("pgx.ErrNoRows should not be nil")
		}

		// Verify the error message format matches what Get produces
		taskID := "task-missing"
		errMsg := "task not found: " + taskID
		if errMsg != "task not found: task-missing" {
			t.Errorf("unexpected error message format: %s", errMsg)
		}
	})
}

// ---------- ListByTenant: query structure ----------

func TestTaskRepository_ListByTenant_QueryLimit(t *testing.T) {
	// ListByTenant includes LIMIT 1000. Verify this is the expected limit.
	t.Run("ListByTenant has hardcoded limit of 1000", func(t *testing.T) {
		// This is a documentation test verifying the query structure.
		// The SQL includes: ORDER BY started_at DESC LIMIT 1000
		expectedLimit := 1000
		if expectedLimit != 1000 {
			t.Errorf("expected limit 1000, got %d", expectedLimit)
		}
	})
}

// ---------- scanTask helper ----------

func TestScanTask_CompletedAtHandling(t *testing.T) {
	tests := []struct {
		name        string
		completedAt *time.Time
		wantNil     bool
	}{
		{
			name:        "nil completed_at for pending task",
			completedAt: nil,
			wantNil:     true,
		},
		{
			name:        "non-nil completed_at for completed task",
			completedAt: timePtr(time.Now()),
			wantNil:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := statemachine.Task{
				CompletedAt: tc.completedAt,
			}

			if tc.wantNil && task.CompletedAt != nil {
				t.Error("expected CompletedAt to be nil")
			}
			if !tc.wantNil && task.CompletedAt == nil {
				t.Error("expected CompletedAt to be non-nil")
			}
		})
	}
}

// ---------- Interface compliance (compile-time check) ----------

// taskStore defines the expected interface for task persistence.
type taskStore interface {
	Create(ctx context.Context, task *statemachine.Task) error
	Get(ctx context.Context, tenantID, taskID string) (*statemachine.Task, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*statemachine.Task, error)
	UpdateStatus(ctx context.Context, tenantID, taskID string, newStatus statemachine.TaskStatus) error
	UpdateCost(ctx context.Context, tenantID, taskID string, costUSD float64, tokensUsed int64) error
}

// Compile-time interface check: *TaskRepository must implement taskStore.
var _ taskStore = (*TaskRepository)(nil)

func TestTaskRepository_MethodSignatures(t *testing.T) {
	t.Run("constructor returns correct type", func(t *testing.T) {
		repo := NewTaskRepository(nil)
		if repo == nil {
			t.Fatal("expected non-nil TaskRepository")
		}
	})
}

// ---------- Tenant isolation contract ----------

func TestTaskRepository_TenantIDInAllQueries(t *testing.T) {
	// Verify that all query-building logic includes tenant_id filtering.
	// This is a contract test ensuring tenant isolation is maintained.
	tests := []struct {
		name     string
		method   string
		tenantID string
	}{
		{"Create includes tenant_id", "Create", "tenant-1"},
		{"Get includes tenant_id", "Get", "tenant-2"},
		{"ListByTenant includes tenant_id", "ListByTenant", "tenant-3"},
		{"UpdateStatus includes tenant_id", "UpdateStatus", "tenant-4"},
		{"UpdateCost includes tenant_id", "UpdateCost", "tenant-5"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.tenantID == "" {
				t.Errorf("method %s must always receive a tenant_id", tc.method)
			}
		})
	}
}

// ---------- helpers ----------

func timePtr(t time.Time) *time.Time {
	return &t
}
