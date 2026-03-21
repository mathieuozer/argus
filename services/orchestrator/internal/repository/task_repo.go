package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/argus-platform/argus/services/orchestrator/internal/statemachine"
	"github.com/jackc/pgx/v5"
)

// TaskRepository provides PostgreSQL-backed task persistence.
type TaskRepository struct {
	pool *database.Pool
}

// NewTaskRepository creates a new PostgreSQL-backed task repository.
func NewTaskRepository(pool *database.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

// Create persists a new task.
func (r *TaskRepository) Create(ctx context.Context, task *statemachine.Task) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, task.TenantID, `
		INSERT INTO tasks (id, tenant_id, agent_id, status, input_hash, started_at, cost_usd, tokens_used)
		VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8)`,
		task.ID, task.TenantID, task.AgentID, string(task.Status),
		task.InputHash, task.StartedAt, task.CostUSD, task.TokensUsed)
}

// Get retrieves a task by ID within a tenant context.
func (r *TaskRepository) Get(ctx context.Context, tenantID, taskID string) (*statemachine.Task, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, agent_id, status, input_hash, started_at, completed_at, cost_usd, tokens_used
		FROM tasks WHERE id = $1 AND tenant_id = $2::uuid`, taskID, tenantID)

	task, err := scanTask(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, err
	}
	return task, nil
}

// ListByTenant returns all tasks for a tenant.
func (r *TaskRepository) ListByTenant(ctx context.Context, tenantID string) ([]*statemachine.Task, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, agent_id, status, input_hash, started_at, completed_at, cost_usd, tokens_used
		FROM tasks WHERE tenant_id = $1::uuid
		ORDER BY started_at DESC
		LIMIT 1000`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var tasks []*statemachine.Task
	for rows.Next() {
		task, err := scanTaskRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// UpdateStatus transitions a task to a new status.
func (r *TaskRepository) UpdateStatus(ctx context.Context, tenantID, taskID string, newStatus statemachine.TaskStatus) error {
	var completedAt *time.Time
	if newStatus == statemachine.StatusCompleted || newStatus == statemachine.StatusFailed {
		now := time.Now()
		completedAt = &now
	}

	return database.ExecWithTenant(ctx, r.pool.Pool, tenantID, `
		UPDATE tasks SET status = $1, completed_at = $2
		WHERE id = $3 AND tenant_id = $4::uuid`,
		string(newStatus), completedAt, taskID, tenantID)
}

// UpdateCost updates cost and token usage for a task.
func (r *TaskRepository) UpdateCost(ctx context.Context, tenantID, taskID string, costUSD float64, tokensUsed int64) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, tenantID, `
		UPDATE tasks SET cost_usd = cost_usd + $1, tokens_used = tokens_used + $2
		WHERE id = $3 AND tenant_id = $4::uuid`,
		costUSD, tokensUsed, taskID, tenantID)
}

func scanTask(row pgx.Row) (*statemachine.Task, error) {
	var t statemachine.Task
	err := row.Scan(&t.ID, &t.TenantID, &t.AgentID, &t.Status,
		&t.InputHash, &t.StartedAt, &t.CompletedAt, &t.CostUSD, &t.TokensUsed)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func scanTaskRows(rows pgx.Rows) (*statemachine.Task, error) {
	var t statemachine.Task
	err := rows.Scan(&t.ID, &t.TenantID, &t.AgentID, &t.Status,
		&t.InputHash, &t.StartedAt, &t.CompletedAt, &t.CostUSD, &t.TokensUsed)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
