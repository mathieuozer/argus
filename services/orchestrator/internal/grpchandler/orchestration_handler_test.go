package grpchandler

import (
	"context"
	"testing"

	orchestrationv1 "github.com/argus-platform/argus/gen/go/orchestration"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
	"github.com/argus-platform/argus/services/orchestrator/internal/router"
	"github.com/argus-platform/argus/services/orchestrator/internal/statemachine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newTestOrchestrationHandler creates a handler backed by fresh in-memory
// registry, router, and state machine instances.
func newTestOrchestrationHandler() (*OrchestrationHandler, *registry.Registry, *statemachine.StateMachine) {
	reg := registry.New()
	r := router.New(reg)
	sm := statemachine.New()
	return NewOrchestrationHandler(sm, r), reg, sm
}

// setupHealthyAgent registers an agent and sets it to healthy status so
// the router can find it.
func setupHealthyAgent(reg *registry.Registry, tenantID, agentID string, caps []string) {
	reg.Register(tenantID, &registry.RegisterRequest{
		AgentID:      agentID,
		Version:      "1.0.0",
		Framework:    "langchain",
		Capabilities: caps,
		NodeID:       "node-1",
	})
	_ = reg.Heartbeat(tenantID, agentID, registry.StatusHealthy)
}

func TestSubmitTask(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*registry.Registry)
		req      *orchestrationv1.SubmitTaskRequest
		ctx      context.Context
		wantCode codes.Code
	}{
		{
			name: "submit task with matching agent",
			setup: func(reg *registry.Registry) {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db", "write:report"})
			},
			req: &orchestrationv1.SubmitTaskRequest{
				TenantId:             "tenant-1",
				RequiredCapabilities: []string{"read:db"},
				InputHash:            "abc123",
			},
			ctx: context.Background(),
		},
		{
			name: "submit task with tenant_id from context",
			setup: func(reg *registry.Registry) {
				setupHealthyAgent(reg, "tenant-ctx", "agent-ctx", []string{"read:db"})
			},
			req: &orchestrationv1.SubmitTaskRequest{
				RequiredCapabilities: []string{"read:db"},
			},
			ctx: tenancy.WithTenant(context.Background(), "tenant-ctx"),
		},
		{
			name:  "submit task with no tenant_id at all",
			setup: func(_ *registry.Registry) {},
			req: &orchestrationv1.SubmitTaskRequest{
				RequiredCapabilities: []string{"read:db"},
			},
			ctx:      context.Background(),
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "submit task with no matching agents",
			setup: func(_ *registry.Registry) {},
			req: &orchestrationv1.SubmitTaskRequest{
				TenantId:             "tenant-1",
				RequiredCapabilities: []string{"write:secret"},
			},
			ctx:      context.Background(),
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "submit task with preferred agent",
			setup: func(reg *registry.Registry) {
				setupHealthyAgent(reg, "tenant-1", "agent-preferred", []string{"read:db", "write:report"})
				setupHealthyAgent(reg, "tenant-1", "agent-other", []string{"read:db", "write:report"})
			},
			req: &orchestrationv1.SubmitTaskRequest{
				TenantId:             "tenant-1",
				RequiredCapabilities: []string{"read:db"},
				PreferredAgentId:     strPtr("agent-preferred"),
			},
			ctx: context.Background(),
		},
		{
			name: "submit task auto-generates input hash when empty",
			setup: func(reg *registry.Registry) {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
			},
			req: &orchestrationv1.SubmitTaskRequest{
				TenantId:             "tenant-1",
				RequiredCapabilities: []string{"read:db"},
				InputHash:            "",
			},
			ctx: context.Background(),
		},
		{
			name: "submit task with empty capabilities matches agents with any caps",
			setup: func(reg *registry.Registry) {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
			},
			req: &orchestrationv1.SubmitTaskRequest{
				TenantId:             "tenant-1",
				RequiredCapabilities: []string{},
			},
			ctx: context.Background(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, reg, _ := newTestOrchestrationHandler()
			tc.setup(reg)

			resp, err := h.SubmitTask(tc.ctx, tc.req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Task == nil {
				t.Fatal("expected task in response, got nil")
			}
			if resp.Task.Id == "" {
				t.Error("expected non-empty task ID")
			}
			if resp.Task.Status != orchestrationv1.TaskStatus_TASK_STATUS_PENDING {
				t.Errorf("expected status PENDING, got %v", resp.Task.Status)
			}
			if resp.Task.StartedAt == nil {
				t.Error("expected StartedAt to be set")
			}

			wantTenantID := tc.req.TenantId
			if wantTenantID == "" {
				wantTenantID, _ = tenancy.FromContext(tc.ctx)
			}
			if resp.Task.TenantId != wantTenantID {
				t.Errorf("expected tenant_id %q, got %q", wantTenantID, resp.Task.TenantId)
			}

			if tc.req.InputHash != "" && resp.Task.InputHash != tc.req.InputHash {
				t.Errorf("expected input_hash %q, got %q", tc.req.InputHash, resp.Task.InputHash)
			}
			if tc.req.InputHash == "" && resp.Task.InputHash == "" {
				t.Error("expected auto-generated input_hash when none provided")
			}
		})
	}
}

func TestGetTask(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*registry.Registry, *statemachine.StateMachine) string // returns task ID
		req      func(taskID string) *orchestrationv1.GetTaskRequest
		wantCode codes.Code
	}{
		{
			name: "get existing task",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) string {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				task := sm.CreateTask("task-get-1", "tenant-1", "agent-1", "hash123")
				return task.ID
			},
			req: func(taskID string) *orchestrationv1.GetTaskRequest {
				return &orchestrationv1.GetTaskRequest{TenantId: "tenant-1", TaskId: taskID}
			},
		},
		{
			name: "get non-existent task",
			setup: func(_ *registry.Registry, _ *statemachine.StateMachine) string {
				return "task-nonexistent"
			},
			req: func(taskID string) *orchestrationv1.GetTaskRequest {
				return &orchestrationv1.GetTaskRequest{TenantId: "tenant-1", TaskId: taskID}
			},
			wantCode: codes.NotFound,
		},
		{
			name: "cross-tenant access denied",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) string {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				task := sm.CreateTask("task-cross-tenant", "tenant-1", "agent-1", "hash123")
				return task.ID
			},
			req: func(taskID string) *orchestrationv1.GetTaskRequest {
				return &orchestrationv1.GetTaskRequest{TenantId: "tenant-2", TaskId: taskID}
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name: "get task with tenant_id from context",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) string {
				setupHealthyAgent(reg, "tenant-ctx", "agent-1", []string{"read:db"})
				task := sm.CreateTask("task-ctx", "tenant-ctx", "agent-1", "hash123")
				return task.ID
			},
			req: func(taskID string) *orchestrationv1.GetTaskRequest {
				return &orchestrationv1.GetTaskRequest{TaskId: taskID}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, reg, sm := newTestOrchestrationHandler()
			taskID := tc.setup(reg, sm)
			req := tc.req(taskID)

			ctx := context.Background()
			if req.TenantId == "" {
				ctx = tenancy.WithTenant(ctx, "tenant-ctx")
			}

			resp, err := h.GetTask(ctx, req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Task == nil {
				t.Fatal("expected task in response, got nil")
			}
			if resp.Task.Id != taskID {
				t.Errorf("expected task ID %q, got %q", taskID, resp.Task.Id)
			}
		})
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*registry.Registry, *statemachine.StateMachine) string
		req      func(taskID string) *orchestrationv1.UpdateTaskStatusRequest
		wantCode codes.Code
		wantStatus orchestrationv1.TaskStatus
	}{
		{
			name: "transition pending to running",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) string {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				task := sm.CreateTask("task-1", "tenant-1", "agent-1", "hash123")
				return task.ID
			},
			req: func(taskID string) *orchestrationv1.UpdateTaskStatusRequest {
				return &orchestrationv1.UpdateTaskStatusRequest{
					TenantId: "tenant-1",
					TaskId:   taskID,
					Status:   orchestrationv1.TaskStatus_TASK_STATUS_RUNNING,
				}
			},
			wantStatus: orchestrationv1.TaskStatus_TASK_STATUS_RUNNING,
		},
		{
			name: "transition running to completed",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) string {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				task := sm.CreateTask("task-2", "tenant-1", "agent-1", "hash123")
				_ = sm.Transition(task.ID, statemachine.StatusRunning)
				return task.ID
			},
			req: func(taskID string) *orchestrationv1.UpdateTaskStatusRequest {
				return &orchestrationv1.UpdateTaskStatusRequest{
					TenantId: "tenant-1",
					TaskId:   taskID,
					Status:   orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED,
				}
			},
			wantStatus: orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED,
		},
		{
			name: "transition running to failed",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) string {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				task := sm.CreateTask("task-fail", "tenant-1", "agent-1", "hash123")
				_ = sm.Transition(task.ID, statemachine.StatusRunning)
				return task.ID
			},
			req: func(taskID string) *orchestrationv1.UpdateTaskStatusRequest {
				return &orchestrationv1.UpdateTaskStatusRequest{
					TenantId: "tenant-1",
					TaskId:   taskID,
					Status:   orchestrationv1.TaskStatus_TASK_STATUS_FAILED,
				}
			},
			wantStatus: orchestrationv1.TaskStatus_TASK_STATUS_FAILED,
		},
		{
			name: "invalid transition pending to completed",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) string {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				task := sm.CreateTask("task-invalid", "tenant-1", "agent-1", "hash123")
				return task.ID
			},
			req: func(taskID string) *orchestrationv1.UpdateTaskStatusRequest {
				return &orchestrationv1.UpdateTaskStatusRequest{
					TenantId: "tenant-1",
					TaskId:   taskID,
					Status:   orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED,
				}
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "update non-existent task",
			setup: func(_ *registry.Registry, _ *statemachine.StateMachine) string {
				return "task-nonexistent"
			},
			req: func(taskID string) *orchestrationv1.UpdateTaskStatusRequest {
				return &orchestrationv1.UpdateTaskStatusRequest{
					TenantId: "tenant-1",
					TaskId:   taskID,
					Status:   orchestrationv1.TaskStatus_TASK_STATUS_RUNNING,
				}
			},
			wantCode: codes.NotFound,
		},
		{
			name: "cross-tenant update denied",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) string {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				task := sm.CreateTask("task-cross", "tenant-1", "agent-1", "hash123")
				return task.ID
			},
			req: func(taskID string) *orchestrationv1.UpdateTaskStatusRequest {
				return &orchestrationv1.UpdateTaskStatusRequest{
					TenantId: "tenant-2",
					TaskId:   taskID,
					Status:   orchestrationv1.TaskStatus_TASK_STATUS_RUNNING,
				}
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name: "update with cost and tokens",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) string {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				task := sm.CreateTask("task-cost", "tenant-1", "agent-1", "hash123")
				return task.ID
			},
			req: func(taskID string) *orchestrationv1.UpdateTaskStatusRequest {
				cost := 0.05
				tokens := int64(1500)
				return &orchestrationv1.UpdateTaskStatusRequest{
					TenantId:   "tenant-1",
					TaskId:     taskID,
					Status:     orchestrationv1.TaskStatus_TASK_STATUS_RUNNING,
					CostUsd:    &cost,
					TokensUsed: &tokens,
				}
			},
			wantStatus: orchestrationv1.TaskStatus_TASK_STATUS_RUNNING,
		},
		{
			name: "transition running to awaiting_approval",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) string {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				task := sm.CreateTask("task-approval", "tenant-1", "agent-1", "hash123")
				_ = sm.Transition(task.ID, statemachine.StatusRunning)
				return task.ID
			},
			req: func(taskID string) *orchestrationv1.UpdateTaskStatusRequest {
				return &orchestrationv1.UpdateTaskStatusRequest{
					TenantId: "tenant-1",
					TaskId:   taskID,
					Status:   orchestrationv1.TaskStatus_TASK_STATUS_AWAITING_APPROVAL,
				}
			},
			wantStatus: orchestrationv1.TaskStatus_TASK_STATUS_AWAITING_APPROVAL,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, reg, sm := newTestOrchestrationHandler()
			taskID := tc.setup(reg, sm)
			req := tc.req(taskID)

			resp, err := h.UpdateTaskStatus(context.Background(), req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Task == nil {
				t.Fatal("expected task in response, got nil")
			}
			if resp.Task.Status != tc.wantStatus {
				t.Errorf("expected status %v, got %v", tc.wantStatus, resp.Task.Status)
			}
		})
	}
}

func TestUpdateTaskStatusWithCost(t *testing.T) {
	t.Run("cost and tokens are tracked on the task", func(t *testing.T) {
		h, reg, sm := newTestOrchestrationHandler()
		setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
		task := sm.CreateTask("task-cost-track", "tenant-1", "agent-1", "hash123")

		cost := 0.05
		tokens := int64(1500)
		resp, err := h.UpdateTaskStatus(context.Background(), &orchestrationv1.UpdateTaskStatusRequest{
			TenantId:   "tenant-1",
			TaskId:     task.ID,
			Status:     orchestrationv1.TaskStatus_TASK_STATUS_RUNNING,
			CostUsd:    &cost,
			TokensUsed: &tokens,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Task.CostUsd != 0.05 {
			t.Errorf("expected cost_usd 0.05, got %f", resp.Task.CostUsd)
		}
		if resp.Task.TokensUsed != 1500 {
			t.Errorf("expected tokens_used 1500, got %d", resp.Task.TokensUsed)
		}
	})

	t.Run("cost accumulates across updates", func(t *testing.T) {
		h, reg, sm := newTestOrchestrationHandler()
		setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
		task := sm.CreateTask("task-cost-accum", "tenant-1", "agent-1", "hash123")

		cost1 := 0.03
		tokens1 := int64(1000)
		_, err := h.UpdateTaskStatus(context.Background(), &orchestrationv1.UpdateTaskStatusRequest{
			TenantId:   "tenant-1",
			TaskId:     task.ID,
			Status:     orchestrationv1.TaskStatus_TASK_STATUS_RUNNING,
			CostUsd:    &cost1,
			TokensUsed: &tokens1,
		})
		if err != nil {
			t.Fatalf("first update: unexpected error: %v", err)
		}

		cost2 := 0.07
		tokens2 := int64(2000)
		resp, err := h.UpdateTaskStatus(context.Background(), &orchestrationv1.UpdateTaskStatusRequest{
			TenantId:   "tenant-1",
			TaskId:     task.ID,
			Status:     orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED,
			CostUsd:    &cost2,
			TokensUsed: &tokens2,
		})
		if err != nil {
			t.Fatalf("second update: unexpected error: %v", err)
		}

		// Cost should be accumulated (0.03 + 0.07 = 0.10)
		if resp.Task.CostUsd < 0.09 || resp.Task.CostUsd > 0.11 {
			t.Errorf("expected accumulated cost ~0.10, got %f", resp.Task.CostUsd)
		}
		if resp.Task.TokensUsed != 3000 {
			t.Errorf("expected accumulated tokens 3000, got %d", resp.Task.TokensUsed)
		}
	})
}

func TestListTasks(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*registry.Registry, *statemachine.StateMachine)
		req     *orchestrationv1.ListTasksRequest
		ctx     context.Context
		wantLen int
	}{
		{
			name: "list all tasks for tenant",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				sm.CreateTask("task-1", "tenant-1", "agent-1", "hash1")
				sm.CreateTask("task-2", "tenant-1", "agent-1", "hash2")
				sm.CreateTask("task-3", "tenant-1", "agent-1", "hash3")
			},
			req:     &orchestrationv1.ListTasksRequest{TenantId: "tenant-1"},
			ctx:     context.Background(),
			wantLen: 3,
		},
		{
			name: "list tasks enforces tenant isolation",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				setupHealthyAgent(reg, "tenant-2", "agent-2", []string{"read:db"})
				sm.CreateTask("task-1", "tenant-1", "agent-1", "hash1")
				sm.CreateTask("task-2", "tenant-2", "agent-2", "hash2")
			},
			req:     &orchestrationv1.ListTasksRequest{TenantId: "tenant-1"},
			ctx:     context.Background(),
			wantLen: 1,
		},
		{
			name: "list tasks filtered by agent_id",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				setupHealthyAgent(reg, "tenant-1", "agent-2", []string{"write:report"})
				sm.CreateTask("task-1", "tenant-1", "agent-1", "hash1")
				sm.CreateTask("task-2", "tenant-1", "agent-1", "hash2")
				sm.CreateTask("task-3", "tenant-1", "agent-2", "hash3")
			},
			req:     &orchestrationv1.ListTasksRequest{TenantId: "tenant-1", AgentId: strPtr("agent-1")},
			ctx:     context.Background(),
			wantLen: 2,
		},
		{
			name: "list tasks for non-existent tenant",
			setup: func(_ *registry.Registry, _ *statemachine.StateMachine) {
			},
			req:     &orchestrationv1.ListTasksRequest{TenantId: "tenant-empty"},
			ctx:     context.Background(),
			wantLen: 0,
		},
		{
			name: "list tasks with tenant_id from context",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) {
				setupHealthyAgent(reg, "tenant-ctx", "agent-1", []string{"read:db"})
				sm.CreateTask("task-ctx-1", "tenant-ctx", "agent-1", "hash1")
			},
			req:     &orchestrationv1.ListTasksRequest{},
			ctx:     tenancy.WithTenant(context.Background(), "tenant-ctx"),
			wantLen: 1,
		},
		{
			name: "list tasks by agent with no tasks for that agent",
			setup: func(reg *registry.Registry, sm *statemachine.StateMachine) {
				setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
				setupHealthyAgent(reg, "tenant-1", "agent-2", []string{"write:report"})
				sm.CreateTask("task-1", "tenant-1", "agent-1", "hash1")
			},
			req:     &orchestrationv1.ListTasksRequest{TenantId: "tenant-1", AgentId: strPtr("agent-2")},
			ctx:     context.Background(),
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, reg, sm := newTestOrchestrationHandler()
			tc.setup(reg, sm)

			resp, err := h.ListTasks(tc.ctx, tc.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(resp.Tasks) != tc.wantLen {
				t.Errorf("expected %d tasks, got %d", tc.wantLen, len(resp.Tasks))
			}
		})
	}
}

func TestTaskStatusConversion(t *testing.T) {
	tests := []struct {
		name       string
		smStatus   statemachine.TaskStatus
		protoStatus orchestrationv1.TaskStatus
	}{
		{"pending", statemachine.StatusPending, orchestrationv1.TaskStatus_TASK_STATUS_PENDING},
		{"running", statemachine.StatusRunning, orchestrationv1.TaskStatus_TASK_STATUS_RUNNING},
		{"completed", statemachine.StatusCompleted, orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED},
		{"failed", statemachine.StatusFailed, orchestrationv1.TaskStatus_TASK_STATUS_FAILED},
		{"awaiting_approval", statemachine.StatusAwaitingApproval, orchestrationv1.TaskStatus_TASK_STATUS_AWAITING_APPROVAL},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// statemachine -> proto
			gotProto := taskStatusToProto(tc.smStatus)
			if gotProto != tc.protoStatus {
				t.Errorf("taskStatusToProto(%q) = %v, want %v", tc.smStatus, gotProto, tc.protoStatus)
			}

			// proto -> statemachine
			gotSM := protoToTaskStatus(tc.protoStatus)
			if gotSM != tc.smStatus {
				t.Errorf("protoToTaskStatus(%v) = %q, want %q", tc.protoStatus, gotSM, tc.smStatus)
			}
		})
	}

	t.Run("unknown statemachine status maps to UNSPECIFIED", func(t *testing.T) {
		got := taskStatusToProto(statemachine.TaskStatus("unknown"))
		if got != orchestrationv1.TaskStatus_TASK_STATUS_UNSPECIFIED {
			t.Errorf("expected UNSPECIFIED for unknown status, got %v", got)
		}
	})

	t.Run("UNSPECIFIED proto status maps to pending", func(t *testing.T) {
		got := protoToTaskStatus(orchestrationv1.TaskStatus_TASK_STATUS_UNSPECIFIED)
		if got != statemachine.StatusPending {
			t.Errorf("expected pending for UNSPECIFIED, got %q", got)
		}
	})
}

func TestTaskToProto(t *testing.T) {
	t.Run("task conversion preserves all fields", func(t *testing.T) {
		sm := statemachine.New()
		task := sm.CreateTask("task-conv", "tenant-1", "agent-1", "hash-abc")

		pt := taskToProto(task)
		if pt.Id != task.ID {
			t.Errorf("ID mismatch: %q != %q", pt.Id, task.ID)
		}
		if pt.TenantId != task.TenantID {
			t.Errorf("TenantId mismatch: %q != %q", pt.TenantId, task.TenantID)
		}
		if pt.AgentId != task.AgentID {
			t.Errorf("AgentId mismatch: %q != %q", pt.AgentId, task.AgentID)
		}
		if pt.InputHash != task.InputHash {
			t.Errorf("InputHash mismatch: %q != %q", pt.InputHash, task.InputHash)
		}
		if pt.Status != orchestrationv1.TaskStatus_TASK_STATUS_PENDING {
			t.Errorf("expected PENDING status, got %v", pt.Status)
		}
		if pt.StartedAt == nil {
			t.Error("expected StartedAt to be set")
		}
		if pt.CompletedAt != nil {
			t.Error("expected CompletedAt to be nil for pending task")
		}
	})

	t.Run("completed task has CompletedAt set", func(t *testing.T) {
		sm := statemachine.New()
		task := sm.CreateTask("task-done", "tenant-1", "agent-1", "hash-abc")
		_ = sm.Transition(task.ID, statemachine.StatusRunning)
		_ = sm.Transition(task.ID, statemachine.StatusCompleted)

		updated, _ := sm.Get(task.ID)
		pt := taskToProto(updated)
		if pt.CompletedAt == nil {
			t.Error("expected CompletedAt to be set for completed task")
		}
		if pt.Status != orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED {
			t.Errorf("expected COMPLETED status, got %v", pt.Status)
		}
	})
}

func TestSubmitTaskAndGetRoundTrip(t *testing.T) {
	t.Run("submit then get returns consistent task", func(t *testing.T) {
		h, reg, _ := newTestOrchestrationHandler()
		setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})

		submitResp, err := h.SubmitTask(context.Background(), &orchestrationv1.SubmitTaskRequest{
			TenantId:             "tenant-1",
			RequiredCapabilities: []string{"read:db"},
			InputHash:            "input-hash-round-trip",
		})
		if err != nil {
			t.Fatalf("SubmitTask: unexpected error: %v", err)
		}

		getResp, err := h.GetTask(context.Background(), &orchestrationv1.GetTaskRequest{
			TenantId: "tenant-1",
			TaskId:   submitResp.Task.Id,
		})
		if err != nil {
			t.Fatalf("GetTask: unexpected error: %v", err)
		}

		if submitResp.Task.Id != getResp.Task.Id {
			t.Errorf("task ID mismatch: %q != %q", submitResp.Task.Id, getResp.Task.Id)
		}
		if getResp.Task.InputHash != "input-hash-round-trip" {
			t.Errorf("expected input_hash %q, got %q", "input-hash-round-trip", getResp.Task.InputHash)
		}
	})
}

func TestTaskLifecycle(t *testing.T) {
	t.Run("full task lifecycle: submit -> running -> completed", func(t *testing.T) {
		h, reg, _ := newTestOrchestrationHandler()
		setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})

		// Submit
		submitResp, err := h.SubmitTask(context.Background(), &orchestrationv1.SubmitTaskRequest{
			TenantId:             "tenant-1",
			RequiredCapabilities: []string{"read:db"},
			InputHash:            "lifecycle-hash",
		})
		if err != nil {
			t.Fatalf("SubmitTask: unexpected error: %v", err)
		}
		taskID := submitResp.Task.Id
		if submitResp.Task.Status != orchestrationv1.TaskStatus_TASK_STATUS_PENDING {
			t.Errorf("expected PENDING after submit, got %v", submitResp.Task.Status)
		}

		// Transition to running
		runResp, err := h.UpdateTaskStatus(context.Background(), &orchestrationv1.UpdateTaskStatusRequest{
			TenantId: "tenant-1",
			TaskId:   taskID,
			Status:   orchestrationv1.TaskStatus_TASK_STATUS_RUNNING,
		})
		if err != nil {
			t.Fatalf("UpdateTaskStatus to RUNNING: unexpected error: %v", err)
		}
		if runResp.Task.Status != orchestrationv1.TaskStatus_TASK_STATUS_RUNNING {
			t.Errorf("expected RUNNING, got %v", runResp.Task.Status)
		}

		// Complete with cost
		cost := 0.12
		tokens := int64(5000)
		doneResp, err := h.UpdateTaskStatus(context.Background(), &orchestrationv1.UpdateTaskStatusRequest{
			TenantId:   "tenant-1",
			TaskId:     taskID,
			Status:     orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED,
			CostUsd:    &cost,
			TokensUsed: &tokens,
		})
		if err != nil {
			t.Fatalf("UpdateTaskStatus to COMPLETED: unexpected error: %v", err)
		}
		if doneResp.Task.Status != orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED {
			t.Errorf("expected COMPLETED, got %v", doneResp.Task.Status)
		}
		if doneResp.Task.CompletedAt == nil {
			t.Error("expected CompletedAt to be set after completion")
		}
		if doneResp.Task.CostUsd != 0.12 {
			t.Errorf("expected cost_usd 0.12, got %f", doneResp.Task.CostUsd)
		}
		if doneResp.Task.TokensUsed != 5000 {
			t.Errorf("expected tokens_used 5000, got %d", doneResp.Task.TokensUsed)
		}

		// Verify via list
		listResp, err := h.ListTasks(context.Background(), &orchestrationv1.ListTasksRequest{TenantId: "tenant-1"})
		if err != nil {
			t.Fatalf("ListTasks: unexpected error: %v", err)
		}
		if len(listResp.Tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(listResp.Tasks))
		}
		if listResp.Tasks[0].Status != orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED {
			t.Errorf("listed task should be COMPLETED, got %v", listResp.Tasks[0].Status)
		}
	})
}

func TestCrossTenantTaskAccess(t *testing.T) {
	t.Run("GetTask denies cross-tenant access", func(t *testing.T) {
		h, reg, sm := newTestOrchestrationHandler()
		setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
		task := sm.CreateTask("task-isolated", "tenant-1", "agent-1", "hash123")

		_, err := h.GetTask(context.Background(), &orchestrationv1.GetTaskRequest{
			TenantId: "tenant-2",
			TaskId:   task.ID,
		})
		if err == nil {
			t.Fatal("expected PermissionDenied error, got nil")
		}
		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got: %v", err)
		}
		if st.Code() != codes.PermissionDenied {
			t.Errorf("expected PermissionDenied, got %v", st.Code())
		}
	})

	t.Run("UpdateTaskStatus denies cross-tenant access", func(t *testing.T) {
		h, reg, sm := newTestOrchestrationHandler()
		setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
		task := sm.CreateTask("task-isolated-update", "tenant-1", "agent-1", "hash123")

		_, err := h.UpdateTaskStatus(context.Background(), &orchestrationv1.UpdateTaskStatusRequest{
			TenantId: "tenant-2",
			TaskId:   task.ID,
			Status:   orchestrationv1.TaskStatus_TASK_STATUS_RUNNING,
		})
		if err == nil {
			t.Fatal("expected PermissionDenied error, got nil")
		}
		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got: %v", err)
		}
		if st.Code() != codes.PermissionDenied {
			t.Errorf("expected PermissionDenied, got %v", st.Code())
		}
	})

	t.Run("ListTasks only returns own tenant tasks", func(t *testing.T) {
		h, reg, sm := newTestOrchestrationHandler()
		setupHealthyAgent(reg, "tenant-1", "agent-1", []string{"read:db"})
		setupHealthyAgent(reg, "tenant-2", "agent-2", []string{"read:db"})
		sm.CreateTask("task-t1", "tenant-1", "agent-1", "hash1")
		sm.CreateTask("task-t2", "tenant-2", "agent-2", "hash2")

		resp, err := h.ListTasks(context.Background(), &orchestrationv1.ListTasksRequest{TenantId: "tenant-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, task := range resp.Tasks {
			if task.TenantId != "tenant-1" {
				t.Errorf("list leaked task from tenant %q", task.TenantId)
			}
		}
		if len(resp.Tasks) != 1 {
			t.Errorf("expected 1 task for tenant-1, got %d", len(resp.Tasks))
		}
	})
}

func TestSubmitTaskWithPreferredAgent(t *testing.T) {
	t.Run("preferred agent is used when available and capable", func(t *testing.T) {
		h, reg, _ := newTestOrchestrationHandler()
		setupHealthyAgent(reg, "tenant-1", "agent-preferred", []string{"read:db", "write:report"})
		setupHealthyAgent(reg, "tenant-1", "agent-fallback", []string{"read:db", "write:report"})

		resp, err := h.SubmitTask(context.Background(), &orchestrationv1.SubmitTaskRequest{
			TenantId:             "tenant-1",
			RequiredCapabilities: []string{"read:db"},
			PreferredAgentId:     strPtr("agent-preferred"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Task.AgentId != "agent-preferred" {
			t.Errorf("expected preferred agent %q, got %q", "agent-preferred", resp.Task.AgentId)
		}
	})
}

// strPtr is a helper to create a pointer to a string.
func strPtr(s string) *string {
	return &s
}
