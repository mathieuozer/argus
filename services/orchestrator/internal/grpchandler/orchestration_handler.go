package grpchandler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	orchestrationv1 "github.com/argus-platform/argus/gen/go/orchestration"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/orchestrator/internal/router"
	"github.com/argus-platform/argus/services/orchestrator/internal/statemachine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OrchestrationHandler implements the OrchestrationServiceServer gRPC interface.
type OrchestrationHandler struct {
	orchestrationv1.UnimplementedOrchestrationServiceServer
	sm     *statemachine.StateMachine
	router *router.Router
}

// NewOrchestrationHandler creates a new gRPC handler for orchestration operations.
func NewOrchestrationHandler(sm *statemachine.StateMachine, r *router.Router) *OrchestrationHandler {
	return &OrchestrationHandler{sm: sm, router: r}
}

func (h *OrchestrationHandler) SubmitTask(ctx context.Context, req *orchestrationv1.SubmitTaskRequest) (*orchestrationv1.SubmitTaskResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}
	if tenantID == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	preferredAgent := ""
	if req.PreferredAgentId != nil {
		preferredAgent = *req.PreferredAgentId
	}

	agent, err := h.router.Route(tenantID, req.RequiredCapabilities, preferredAgent)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	inputHash := req.InputHash
	if inputHash == "" {
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", tenantID, time.Now().UnixNano())))
		inputHash = hex.EncodeToString(hash[:])
	}

	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	task := h.sm.CreateTask(taskID, tenantID, agent.ID, inputHash)

	return &orchestrationv1.SubmitTaskResponse{Task: taskToProto(task)}, nil
}

func (h *OrchestrationHandler) GetTask(ctx context.Context, req *orchestrationv1.GetTaskRequest) (*orchestrationv1.GetTaskResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}

	task, err := h.sm.Get(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if task.TenantID != tenantID {
		return nil, status.Error(codes.PermissionDenied, "cross-tenant access denied")
	}

	return &orchestrationv1.GetTaskResponse{Task: taskToProto(task)}, nil
}

func (h *OrchestrationHandler) UpdateTaskStatus(ctx context.Context, req *orchestrationv1.UpdateTaskStatusRequest) (*orchestrationv1.UpdateTaskStatusResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}

	task, err := h.sm.Get(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if task.TenantID != tenantID {
		return nil, status.Error(codes.PermissionDenied, "cross-tenant access denied")
	}

	newStatus := protoToTaskStatus(req.Status)
	if err := h.sm.Transition(req.TaskId, newStatus); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.CostUsd != nil {
		tokensUsed := int64(0)
		if req.TokensUsed != nil {
			tokensUsed = *req.TokensUsed
		}
		_ = h.sm.UpdateCost(req.TaskId, *req.CostUsd, tokensUsed)
	}

	updated, _ := h.sm.Get(req.TaskId)
	return &orchestrationv1.UpdateTaskStatusResponse{Task: taskToProto(updated)}, nil
}

func (h *OrchestrationHandler) ListTasks(ctx context.Context, req *orchestrationv1.ListTasksRequest) (*orchestrationv1.ListTasksResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}

	var tasks []*statemachine.Task
	if req.AgentId != nil {
		tasks = h.sm.ListByAgent(tenantID, *req.AgentId)
	} else {
		tasks = h.sm.ListByTenant(tenantID)
	}

	protoTasks := make([]*orchestrationv1.Task, len(tasks))
	for i, t := range tasks {
		protoTasks[i] = taskToProto(t)
	}

	return &orchestrationv1.ListTasksResponse{Tasks: protoTasks}, nil
}

func taskToProto(t *statemachine.Task) *orchestrationv1.Task {
	pt := &orchestrationv1.Task{
		Id:         t.ID,
		TenantId:   t.TenantID,
		AgentId:    t.AgentID,
		Status:     taskStatusToProto(t.Status),
		InputHash:  t.InputHash,
		StartedAt:  timestamppb.New(t.StartedAt),
		CostUsd:    t.CostUSD,
		TokensUsed: t.TokensUsed,
	}
	if t.CompletedAt != nil {
		pt.CompletedAt = timestamppb.New(*t.CompletedAt)
	}
	return pt
}

func taskStatusToProto(s statemachine.TaskStatus) orchestrationv1.TaskStatus {
	switch s {
	case statemachine.StatusPending:
		return orchestrationv1.TaskStatus_TASK_STATUS_PENDING
	case statemachine.StatusRunning:
		return orchestrationv1.TaskStatus_TASK_STATUS_RUNNING
	case statemachine.StatusCompleted:
		return orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED
	case statemachine.StatusFailed:
		return orchestrationv1.TaskStatus_TASK_STATUS_FAILED
	case statemachine.StatusAwaitingApproval:
		return orchestrationv1.TaskStatus_TASK_STATUS_AWAITING_APPROVAL
	default:
		return orchestrationv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

func protoToTaskStatus(s orchestrationv1.TaskStatus) statemachine.TaskStatus {
	switch s {
	case orchestrationv1.TaskStatus_TASK_STATUS_PENDING:
		return statemachine.StatusPending
	case orchestrationv1.TaskStatus_TASK_STATUS_RUNNING:
		return statemachine.StatusRunning
	case orchestrationv1.TaskStatus_TASK_STATUS_COMPLETED:
		return statemachine.StatusCompleted
	case orchestrationv1.TaskStatus_TASK_STATUS_FAILED:
		return statemachine.StatusFailed
	case orchestrationv1.TaskStatus_TASK_STATUS_AWAITING_APPROVAL:
		return statemachine.StatusAwaitingApproval
	default:
		return statemachine.StatusPending
	}
}
