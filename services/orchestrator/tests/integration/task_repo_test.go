package integration

import (
	"context"
	"testing"
	"time"

	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
	"github.com/argus-platform/argus/services/orchestrator/internal/repository"
	"github.com/argus-platform/argus/services/orchestrator/internal/statemachine"
)

func TestTaskRepository_CreateAndGet(t *testing.T) {
	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	tenantID := CreateTestTenant(t, pool, "test-task-tenant")
	agentRepo := repository.NewAgentRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)
	ctx := context.Background()

	// Register agent first (FK constraint)
	_, err := agentRepo.Register(ctx, tenantID, &registry.RegisterRequest{
		AgentID: "task-agent", Version: "1.0.0", Framework: "custom", NodeID: "node-1",
	})
	if err != nil {
		t.Fatalf("Register agent failed: %v", err)
	}

	task := &statemachine.Task{
		ID:        "task-001",
		TenantID:  tenantID,
		AgentID:   "task-agent",
		Status:    statemachine.StatusPending,
		InputHash: "abc123",
		StartedAt: time.Now(),
	}

	err = taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	got, err := taskRepo.Get(ctx, tenantID, "task-001")
	if err != nil {
		t.Fatalf("Get task failed: %v", err)
	}
	if got.AgentID != "task-agent" {
		t.Errorf("expected agent task-agent, got %s", got.AgentID)
	}
	if got.Status != statemachine.StatusPending {
		t.Errorf("expected status pending, got %s", got.Status)
	}
}

func TestTaskRepository_UpdateStatus(t *testing.T) {
	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	tenantID := CreateTestTenant(t, pool, "test-status-tenant")
	agentRepo := repository.NewAgentRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)
	ctx := context.Background()

	_, err := agentRepo.Register(ctx, tenantID, &registry.RegisterRequest{
		AgentID: "status-agent", Version: "1.0.0", Framework: "custom", NodeID: "node-1",
	})
	if err != nil {
		t.Fatalf("Register agent failed: %v", err)
	}

	task := &statemachine.Task{
		ID: "task-status-1", TenantID: tenantID, AgentID: "status-agent",
		Status: statemachine.StatusPending, InputHash: "def456", StartedAt: time.Now(),
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	err = taskRepo.UpdateStatus(ctx, tenantID, "task-status-1", statemachine.StatusRunning)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	got, err := taskRepo.Get(ctx, tenantID, "task-status-1")
	if err != nil {
		t.Fatalf("Get after status update failed: %v", err)
	}
	if got.Status != statemachine.StatusRunning {
		t.Errorf("expected status running, got %s", got.Status)
	}

	// Complete the task
	err = taskRepo.UpdateStatus(ctx, tenantID, "task-status-1", statemachine.StatusCompleted)
	if err != nil {
		t.Fatalf("UpdateStatus to completed failed: %v", err)
	}

	got, err = taskRepo.Get(ctx, tenantID, "task-status-1")
	if err != nil {
		t.Fatalf("Get after completion failed: %v", err)
	}
	if got.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestTaskRepository_UpdateCost(t *testing.T) {
	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	tenantID := CreateTestTenant(t, pool, "test-cost-tenant")
	agentRepo := repository.NewAgentRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)
	ctx := context.Background()

	_, err := agentRepo.Register(ctx, tenantID, &registry.RegisterRequest{
		AgentID: "cost-agent", Version: "1.0.0", Framework: "custom", NodeID: "node-1",
	})
	if err != nil {
		t.Fatalf("Register agent failed: %v", err)
	}

	task := &statemachine.Task{
		ID: "task-cost-1", TenantID: tenantID, AgentID: "cost-agent",
		Status: statemachine.StatusRunning, InputHash: "ghi789", StartedAt: time.Now(),
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	err = taskRepo.UpdateCost(ctx, tenantID, "task-cost-1", 0.05, 1500)
	if err != nil {
		t.Fatalf("UpdateCost failed: %v", err)
	}

	got, err := taskRepo.Get(ctx, tenantID, "task-cost-1")
	if err != nil {
		t.Fatalf("Get after cost update failed: %v", err)
	}
	if got.CostUSD < 0.04 || got.CostUSD > 0.06 {
		t.Errorf("expected cost ~0.05, got %f", got.CostUSD)
	}
	if got.TokensUsed != 1500 {
		t.Errorf("expected 1500 tokens, got %d", got.TokensUsed)
	}
}

func TestTaskRepository_ListByTenant(t *testing.T) {
	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	tenantID := CreateTestTenant(t, pool, "test-list-tasks-tenant")
	agentRepo := repository.NewAgentRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)
	ctx := context.Background()

	_, err := agentRepo.Register(ctx, tenantID, &registry.RegisterRequest{
		AgentID: "list-agent", Version: "1.0.0", Framework: "custom", NodeID: "node-1",
	})
	if err != nil {
		t.Fatalf("Register agent failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		task := &statemachine.Task{
			ID: "task-list-" + string(rune('a'+i)), TenantID: tenantID, AgentID: "list-agent",
			Status: statemachine.StatusPending, InputHash: "hash", StartedAt: time.Now(),
		}
		if err := taskRepo.Create(ctx, task); err != nil {
			t.Fatalf("Create task %d failed: %v", i, err)
		}
	}

	tasks, err := taskRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("ListByTenant failed: %v", err)
	}
	if len(tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(tasks))
	}
}
