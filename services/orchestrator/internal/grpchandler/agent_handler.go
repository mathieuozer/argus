package grpchandler

import (
	"context"

	agentv1 "github.com/argus-platform/argus/gen/go/agent"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AgentHandler implements the AgentServiceServer gRPC interface.
type AgentHandler struct {
	agentv1.UnimplementedAgentServiceServer
	registry *registry.Registry
}

// NewAgentHandler creates a new gRPC handler for agent operations.
func NewAgentHandler(reg *registry.Registry) *AgentHandler {
	return &AgentHandler{registry: reg}
}

func (h *AgentHandler) RegisterAgent(ctx context.Context, req *agentv1.RegisterAgentRequest) (*agentv1.RegisterAgentResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}
	if tenantID == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	agent := h.registry.Register(tenantID, &registry.RegisterRequest{
		AgentID:      req.AgentId,
		Version:      req.Version,
		Framework:    req.Framework,
		Capabilities: req.Capabilities,
		NodeID:       req.NodeId,
	})

	return &agentv1.RegisterAgentResponse{
		Agent:   agentToProto(agent),
		SvidUri: agent.SVIDURI,
	}, nil
}

func (h *AgentHandler) Heartbeat(ctx context.Context, req *agentv1.HeartbeatRequest) (*agentv1.HeartbeatResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}

	agentStatus := protoToAgentStatus(req.Status)
	if err := h.registry.Heartbeat(tenantID, req.AgentId, agentStatus); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &agentv1.HeartbeatResponse{Acknowledged: true}, nil
}

func (h *AgentHandler) ListAgents(ctx context.Context, req *agentv1.ListAgentsRequest) (*agentv1.ListAgentsResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}

	agents := h.registry.List(tenantID)
	protoAgents := make([]*agentv1.Agent, len(agents))
	for i, a := range agents {
		protoAgents[i] = agentToProto(a)
	}

	return &agentv1.ListAgentsResponse{Agents: protoAgents}, nil
}

func (h *AgentHandler) GetAgent(ctx context.Context, req *agentv1.GetAgentRequest) (*agentv1.GetAgentResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}

	agent, err := h.registry.Get(tenantID, req.AgentId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &agentv1.GetAgentResponse{Agent: agentToProto(agent)}, nil
}

func agentToProto(a *registry.Agent) *agentv1.Agent {
	return &agentv1.Agent{
		Id:           a.ID,
		TenantId:     a.TenantID,
		Version:      a.Version,
		Framework:    a.Framework,
		Capabilities: a.Capabilities,
		Status:       agentStatusToProto(a.Status),
		SvidUri:      a.SVIDURI,
		LastSeen:     timestamppb.New(a.LastSeen),
		NodeId:       a.NodeID,
	}
}

func agentStatusToProto(s registry.AgentStatus) agentv1.AgentStatus {
	switch s {
	case registry.StatusDiscovered:
		return agentv1.AgentStatus_AGENT_STATUS_DISCOVERED
	case registry.StatusHealthy:
		return agentv1.AgentStatus_AGENT_STATUS_HEALTHY
	case registry.StatusDegraded:
		return agentv1.AgentStatus_AGENT_STATUS_DEGRADED
	case registry.StatusFailed:
		return agentv1.AgentStatus_AGENT_STATUS_FAILED
	case registry.StatusQuarantined:
		return agentv1.AgentStatus_AGENT_STATUS_QUARANTINED
	default:
		return agentv1.AgentStatus_AGENT_STATUS_UNSPECIFIED
	}
}

func protoToAgentStatus(s agentv1.AgentStatus) registry.AgentStatus {
	switch s {
	case agentv1.AgentStatus_AGENT_STATUS_DISCOVERED:
		return registry.StatusDiscovered
	case agentv1.AgentStatus_AGENT_STATUS_HEALTHY:
		return registry.StatusHealthy
	case agentv1.AgentStatus_AGENT_STATUS_DEGRADED:
		return registry.StatusDegraded
	case agentv1.AgentStatus_AGENT_STATUS_FAILED:
		return registry.StatusFailed
	case agentv1.AgentStatus_AGENT_STATUS_QUARANTINED:
		return registry.StatusQuarantined
	default:
		return registry.StatusDiscovered
	}
}
