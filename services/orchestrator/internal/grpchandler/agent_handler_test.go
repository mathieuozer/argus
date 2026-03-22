package grpchandler

import (
	"context"
	"testing"

	agentv1 "github.com/argus-platform/argus/gen/go/agent"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newTestAgentHandler creates a handler backed by a fresh in-memory registry.
func newTestAgentHandler() (*AgentHandler, *registry.Registry) {
	reg := registry.New()
	return NewAgentHandler(reg), reg
}

// registerTestAgent is a helper that pre-registers an agent in the registry.
func registerTestAgent(reg *registry.Registry, tenantID, agentID, version, framework string, caps []string, nodeID string) *registry.Agent {
	return reg.Register(tenantID, &registry.RegisterRequest{
		AgentID:      agentID,
		Version:      version,
		Framework:    framework,
		Capabilities: caps,
		NodeID:       nodeID,
	})
}

func TestRegisterAgent(t *testing.T) {
	tests := []struct {
		name     string
		req      *agentv1.RegisterAgentRequest
		ctx      context.Context
		wantCode codes.Code
		wantID   string
	}{
		{
			name: "register with tenant_id in request",
			req: &agentv1.RegisterAgentRequest{
				TenantId:     "tenant-1",
				AgentId:      "agent-alpha",
				Version:      "1.0.0",
				Framework:    "langchain",
				Capabilities: []string{"read:db", "write:report"},
				NodeId:       "node-1",
			},
			ctx:    context.Background(),
			wantID: "agent-alpha",
		},
		{
			name: "register with tenant_id from context",
			req: &agentv1.RegisterAgentRequest{
				AgentId:      "agent-beta",
				Version:      "2.0.0",
				Framework:    "autogen",
				Capabilities: []string{"read:budget"},
				NodeId:       "node-2",
			},
			ctx:    tenancy.WithTenant(context.Background(), "tenant-ctx"),
			wantID: "agent-beta",
		},
		{
			name: "register with no tenant_id at all",
			req: &agentv1.RegisterAgentRequest{
				AgentId:   "agent-gamma",
				Version:   "1.0.0",
				Framework: "custom",
				NodeId:    "node-3",
			},
			ctx:      context.Background(),
			wantCode: codes.InvalidArgument,
		},
		{
			name: "register with empty capabilities",
			req: &agentv1.RegisterAgentRequest{
				TenantId:     "tenant-1",
				AgentId:      "agent-minimal",
				Version:      "0.1.0",
				Framework:    "custom",
				Capabilities: []string{},
				NodeId:       "node-1",
			},
			ctx:    context.Background(),
			wantID: "agent-minimal",
		},
		{
			name: "request tenant_id takes precedence over context",
			req: &agentv1.RegisterAgentRequest{
				TenantId:  "tenant-req",
				AgentId:   "agent-precedence",
				Version:   "1.0.0",
				Framework: "langchain",
				NodeId:    "node-1",
			},
			ctx:    tenancy.WithTenant(context.Background(), "tenant-ctx"),
			wantID: "agent-precedence",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestAgentHandler()

			resp, err := h.RegisterAgent(tc.ctx, tc.req)
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
			if resp.Agent == nil {
				t.Fatal("expected agent in response, got nil")
			}
			if resp.Agent.Id != tc.wantID {
				t.Errorf("expected agent ID %q, got %q", tc.wantID, resp.Agent.Id)
			}
			if resp.Agent.Status != agentv1.AgentStatus_AGENT_STATUS_DISCOVERED {
				t.Errorf("expected status DISCOVERED, got %v", resp.Agent.Status)
			}
			if resp.Agent.Version != tc.req.Version {
				t.Errorf("expected version %q, got %q", tc.req.Version, resp.Agent.Version)
			}
			if resp.Agent.Framework != tc.req.Framework {
				t.Errorf("expected framework %q, got %q", tc.req.Framework, resp.Agent.Framework)
			}
			if resp.Agent.NodeId != tc.req.NodeId {
				t.Errorf("expected node_id %q, got %q", tc.req.NodeId, resp.Agent.NodeId)
			}
		})
	}
}

func TestHeartbeat(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*registry.Registry)
		req      *agentv1.HeartbeatRequest
		wantCode codes.Code
		wantAck  bool
	}{
		{
			name: "successful heartbeat with healthy status",
			setup: func(reg *registry.Registry) {
				registerTestAgent(reg, "tenant-1", "agent-1", "1.0.0", "langchain", []string{"read:db"}, "node-1")
			},
			req: &agentv1.HeartbeatRequest{
				TenantId: "tenant-1",
				AgentId:  "agent-1",
				Status:   agentv1.AgentStatus_AGENT_STATUS_HEALTHY,
			},
			wantAck: true,
		},
		{
			name: "heartbeat with degraded status",
			setup: func(reg *registry.Registry) {
				registerTestAgent(reg, "tenant-1", "agent-1", "1.0.0", "langchain", nil, "node-1")
			},
			req: &agentv1.HeartbeatRequest{
				TenantId: "tenant-1",
				AgentId:  "agent-1",
				Status:   agentv1.AgentStatus_AGENT_STATUS_DEGRADED,
			},
			wantAck: true,
		},
		{
			name:  "heartbeat for non-existent agent",
			setup: func(_ *registry.Registry) {},
			req: &agentv1.HeartbeatRequest{
				TenantId: "tenant-1",
				AgentId:  "agent-nonexistent",
				Status:   agentv1.AgentStatus_AGENT_STATUS_HEALTHY,
			},
			wantCode: codes.NotFound,
		},
		{
			name: "heartbeat for agent in wrong tenant (tenant isolation)",
			setup: func(reg *registry.Registry) {
				registerTestAgent(reg, "tenant-1", "agent-1", "1.0.0", "langchain", nil, "node-1")
			},
			req: &agentv1.HeartbeatRequest{
				TenantId: "tenant-2",
				AgentId:  "agent-1",
				Status:   agentv1.AgentStatus_AGENT_STATUS_HEALTHY,
			},
			wantCode: codes.NotFound,
		},
		{
			name: "heartbeat with tenant_id from context",
			setup: func(reg *registry.Registry) {
				registerTestAgent(reg, "tenant-ctx", "agent-ctx", "1.0.0", "custom", nil, "node-1")
			},
			req: &agentv1.HeartbeatRequest{
				AgentId: "agent-ctx",
				Status:  agentv1.AgentStatus_AGENT_STATUS_HEALTHY,
			},
			wantAck: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, reg := newTestAgentHandler()
			tc.setup(reg)

			ctx := context.Background()
			// For the context-based tenant test, inject tenant via context
			if tc.req.TenantId == "" {
				ctx = tenancy.WithTenant(ctx, "tenant-ctx")
			}

			resp, err := h.Heartbeat(ctx, tc.req)
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
			if resp.Acknowledged != tc.wantAck {
				t.Errorf("expected acknowledged=%v, got %v", tc.wantAck, resp.Acknowledged)
			}
		})
	}
}

func TestListAgents(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*registry.Registry)
		req     *agentv1.ListAgentsRequest
		wantLen int
	}{
		{
			name: "list agents for tenant with multiple agents",
			setup: func(reg *registry.Registry) {
				registerTestAgent(reg, "tenant-1", "agent-1", "1.0.0", "langchain", nil, "node-1")
				registerTestAgent(reg, "tenant-1", "agent-2", "2.0.0", "autogen", nil, "node-1")
				registerTestAgent(reg, "tenant-1", "agent-3", "1.0.0", "custom", nil, "node-2")
			},
			req:     &agentv1.ListAgentsRequest{TenantId: "tenant-1"},
			wantLen: 3,
		},
		{
			name: "list agents enforces tenant isolation",
			setup: func(reg *registry.Registry) {
				registerTestAgent(reg, "tenant-1", "agent-1", "1.0.0", "langchain", nil, "node-1")
				registerTestAgent(reg, "tenant-2", "agent-2", "2.0.0", "autogen", nil, "node-1")
			},
			req:     &agentv1.ListAgentsRequest{TenantId: "tenant-1"},
			wantLen: 1,
		},
		{
			name:    "list agents for tenant with no agents",
			setup:   func(_ *registry.Registry) {},
			req:     &agentv1.ListAgentsRequest{TenantId: "tenant-empty"},
			wantLen: 0,
		},
		{
			name: "list agents with tenant_id from context",
			setup: func(reg *registry.Registry) {
				registerTestAgent(reg, "tenant-ctx", "agent-1", "1.0.0", "langchain", nil, "node-1")
			},
			req:     &agentv1.ListAgentsRequest{},
			wantLen: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, reg := newTestAgentHandler()
			tc.setup(reg)

			ctx := context.Background()
			if tc.req.TenantId == "" {
				ctx = tenancy.WithTenant(ctx, "tenant-ctx")
			}

			resp, err := h.ListAgents(ctx, tc.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(resp.Agents) != tc.wantLen {
				t.Errorf("expected %d agents, got %d", tc.wantLen, len(resp.Agents))
			}
		})
	}
}

func TestGetAgent(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*registry.Registry)
		req      *agentv1.GetAgentRequest
		wantCode codes.Code
		wantID   string
	}{
		{
			name: "get existing agent",
			setup: func(reg *registry.Registry) {
				registerTestAgent(reg, "tenant-1", "agent-1", "1.0.0", "langchain", []string{"read:db"}, "node-1")
			},
			req:    &agentv1.GetAgentRequest{TenantId: "tenant-1", AgentId: "agent-1"},
			wantID: "agent-1",
		},
		{
			name:     "get non-existent agent",
			setup:    func(_ *registry.Registry) {},
			req:      &agentv1.GetAgentRequest{TenantId: "tenant-1", AgentId: "agent-missing"},
			wantCode: codes.NotFound,
		},
		{
			name: "get agent from wrong tenant (tenant isolation)",
			setup: func(reg *registry.Registry) {
				registerTestAgent(reg, "tenant-1", "agent-1", "1.0.0", "langchain", nil, "node-1")
			},
			req:      &agentv1.GetAgentRequest{TenantId: "tenant-2", AgentId: "agent-1"},
			wantCode: codes.NotFound,
		},
		{
			name: "get agent with tenant_id from context",
			setup: func(reg *registry.Registry) {
				registerTestAgent(reg, "tenant-ctx", "agent-ctx", "1.0.0", "custom", nil, "node-1")
			},
			req:    &agentv1.GetAgentRequest{AgentId: "agent-ctx"},
			wantID: "agent-ctx",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, reg := newTestAgentHandler()
			tc.setup(reg)

			ctx := context.Background()
			if tc.req.TenantId == "" {
				ctx = tenancy.WithTenant(ctx, "tenant-ctx")
			}

			resp, err := h.GetAgent(ctx, tc.req)
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
			if resp.Agent == nil {
				t.Fatal("expected agent in response, got nil")
			}
			if resp.Agent.Id != tc.wantID {
				t.Errorf("expected agent ID %q, got %q", tc.wantID, resp.Agent.Id)
			}
		})
	}
}

func TestAgentStatusConversion(t *testing.T) {
	tests := []struct {
		name           string
		registryStatus registry.AgentStatus
		protoStatus    agentv1.AgentStatus
	}{
		{"discovered", registry.StatusDiscovered, agentv1.AgentStatus_AGENT_STATUS_DISCOVERED},
		{"healthy", registry.StatusHealthy, agentv1.AgentStatus_AGENT_STATUS_HEALTHY},
		{"degraded", registry.StatusDegraded, agentv1.AgentStatus_AGENT_STATUS_DEGRADED},
		{"failed", registry.StatusFailed, agentv1.AgentStatus_AGENT_STATUS_FAILED},
		{"quarantined", registry.StatusQuarantined, agentv1.AgentStatus_AGENT_STATUS_QUARANTINED},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// registry -> proto
			gotProto := agentStatusToProto(tc.registryStatus)
			if gotProto != tc.protoStatus {
				t.Errorf("agentStatusToProto(%q) = %v, want %v", tc.registryStatus, gotProto, tc.protoStatus)
			}

			// proto -> registry
			gotRegistry := protoToAgentStatus(tc.protoStatus)
			if gotRegistry != tc.registryStatus {
				t.Errorf("protoToAgentStatus(%v) = %q, want %q", tc.protoStatus, gotRegistry, tc.registryStatus)
			}
		})
	}

	t.Run("unknown registry status maps to UNSPECIFIED", func(t *testing.T) {
		got := agentStatusToProto(registry.AgentStatus("unknown"))
		if got != agentv1.AgentStatus_AGENT_STATUS_UNSPECIFIED {
			t.Errorf("expected UNSPECIFIED for unknown status, got %v", got)
		}
	})

	t.Run("UNSPECIFIED proto status maps to discovered", func(t *testing.T) {
		got := protoToAgentStatus(agentv1.AgentStatus_AGENT_STATUS_UNSPECIFIED)
		if got != registry.StatusDiscovered {
			t.Errorf("expected discovered for UNSPECIFIED, got %q", got)
		}
	})
}

func TestAgentToProto(t *testing.T) {
	t.Run("full agent conversion preserves all fields", func(t *testing.T) {
		h, reg := newTestAgentHandler()
		agent := registerTestAgent(reg, "tenant-1", "agent-1", "1.0.0", "langchain",
			[]string{"read:db", "write:report"}, "node-1")

		resp, err := h.GetAgent(context.Background(), &agentv1.GetAgentRequest{
			TenantId: "tenant-1",
			AgentId:  "agent-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		pa := resp.Agent
		if pa.Id != agent.ID {
			t.Errorf("ID mismatch: %q != %q", pa.Id, agent.ID)
		}
		if pa.TenantId != agent.TenantID {
			t.Errorf("TenantId mismatch: %q != %q", pa.TenantId, agent.TenantID)
		}
		if pa.Version != agent.Version {
			t.Errorf("Version mismatch: %q != %q", pa.Version, agent.Version)
		}
		if pa.Framework != agent.Framework {
			t.Errorf("Framework mismatch: %q != %q", pa.Framework, agent.Framework)
		}
		if pa.NodeId != agent.NodeID {
			t.Errorf("NodeId mismatch: %q != %q", pa.NodeId, agent.NodeID)
		}
		if len(pa.Capabilities) != len(agent.Capabilities) {
			t.Errorf("Capabilities length mismatch: %d != %d", len(pa.Capabilities), len(agent.Capabilities))
		}
		if pa.LastSeen == nil {
			t.Error("expected LastSeen to be set")
		}
	})
}

func TestRegisterAndGetRoundTrip(t *testing.T) {
	t.Run("register then get returns same agent", func(t *testing.T) {
		h, _ := newTestAgentHandler()

		registerResp, err := h.RegisterAgent(context.Background(), &agentv1.RegisterAgentRequest{
			TenantId:     "tenant-1",
			AgentId:      "agent-roundtrip",
			Version:      "3.2.1",
			Framework:    "crewai",
			Capabilities: []string{"read:budget_db", "write:report_store"},
			NodeId:       "node-42",
		})
		if err != nil {
			t.Fatalf("RegisterAgent: unexpected error: %v", err)
		}

		getResp, err := h.GetAgent(context.Background(), &agentv1.GetAgentRequest{
			TenantId: "tenant-1",
			AgentId:  "agent-roundtrip",
		})
		if err != nil {
			t.Fatalf("GetAgent: unexpected error: %v", err)
		}

		if registerResp.Agent.Id != getResp.Agent.Id {
			t.Errorf("agent ID mismatch after round-trip: %q != %q", registerResp.Agent.Id, getResp.Agent.Id)
		}
		if registerResp.Agent.Version != getResp.Agent.Version {
			t.Errorf("version mismatch after round-trip: %q != %q", registerResp.Agent.Version, getResp.Agent.Version)
		}
	})
}

func TestHeartbeatUpdatesStatus(t *testing.T) {
	t.Run("heartbeat changes agent status", func(t *testing.T) {
		h, reg := newTestAgentHandler()
		registerTestAgent(reg, "tenant-1", "agent-1", "1.0.0", "langchain", nil, "node-1")

		// Verify initial status is discovered
		getResp, err := h.GetAgent(context.Background(), &agentv1.GetAgentRequest{
			TenantId: "tenant-1",
			AgentId:  "agent-1",
		})
		if err != nil {
			t.Fatalf("GetAgent: unexpected error: %v", err)
		}
		if getResp.Agent.Status != agentv1.AgentStatus_AGENT_STATUS_DISCOVERED {
			t.Errorf("expected initial status DISCOVERED, got %v", getResp.Agent.Status)
		}

		// Send heartbeat with healthy status
		_, err = h.Heartbeat(context.Background(), &agentv1.HeartbeatRequest{
			TenantId: "tenant-1",
			AgentId:  "agent-1",
			Status:   agentv1.AgentStatus_AGENT_STATUS_HEALTHY,
		})
		if err != nil {
			t.Fatalf("Heartbeat: unexpected error: %v", err)
		}

		// Verify status changed
		getResp, err = h.GetAgent(context.Background(), &agentv1.GetAgentRequest{
			TenantId: "tenant-1",
			AgentId:  "agent-1",
		})
		if err != nil {
			t.Fatalf("GetAgent after heartbeat: unexpected error: %v", err)
		}
		if getResp.Agent.Status != agentv1.AgentStatus_AGENT_STATUS_HEALTHY {
			t.Errorf("expected status HEALTHY after heartbeat, got %v", getResp.Agent.Status)
		}
	})
}

func TestListAgentsTenantIsolation(t *testing.T) {
	t.Run("listing agents never leaks across tenants", func(t *testing.T) {
		h, reg := newTestAgentHandler()
		registerTestAgent(reg, "tenant-alpha", "agent-1", "1.0.0", "langchain", nil, "node-1")
		registerTestAgent(reg, "tenant-alpha", "agent-2", "1.0.0", "autogen", nil, "node-1")
		registerTestAgent(reg, "tenant-beta", "agent-3", "1.0.0", "custom", nil, "node-2")
		registerTestAgent(reg, "tenant-gamma", "agent-4", "1.0.0", "langchain", nil, "node-3")

		// tenant-alpha should see exactly 2 agents
		resp, err := h.ListAgents(context.Background(), &agentv1.ListAgentsRequest{TenantId: "tenant-alpha"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Agents) != 2 {
			t.Errorf("tenant-alpha: expected 2 agents, got %d", len(resp.Agents))
		}
		for _, a := range resp.Agents {
			if a.TenantId != "tenant-alpha" {
				t.Errorf("tenant-alpha list leaked agent with tenant_id=%q", a.TenantId)
			}
		}

		// tenant-beta should see exactly 1 agent
		resp, err = h.ListAgents(context.Background(), &agentv1.ListAgentsRequest{TenantId: "tenant-beta"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Agents) != 1 {
			t.Errorf("tenant-beta: expected 1 agent, got %d", len(resp.Agents))
		}

		// non-existent tenant should see 0 agents
		resp, err = h.ListAgents(context.Background(), &agentv1.ListAgentsRequest{TenantId: "tenant-nonexistent"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Agents) != 0 {
			t.Errorf("non-existent tenant: expected 0 agents, got %d", len(resp.Agents))
		}
	})
}

func TestRegisterAgentOverwrite(t *testing.T) {
	t.Run("re-registering same agent_id updates the agent", func(t *testing.T) {
		h, _ := newTestAgentHandler()

		_, err := h.RegisterAgent(context.Background(), &agentv1.RegisterAgentRequest{
			TenantId:  "tenant-1",
			AgentId:   "agent-1",
			Version:   "1.0.0",
			Framework: "langchain",
			NodeId:    "node-1",
		})
		if err != nil {
			t.Fatalf("first register: unexpected error: %v", err)
		}

		resp, err := h.RegisterAgent(context.Background(), &agentv1.RegisterAgentRequest{
			TenantId:  "tenant-1",
			AgentId:   "agent-1",
			Version:   "2.0.0",
			Framework: "autogen",
			NodeId:    "node-2",
		})
		if err != nil {
			t.Fatalf("second register: unexpected error: %v", err)
		}
		if resp.Agent.Version != "2.0.0" {
			t.Errorf("expected version 2.0.0 after re-register, got %q", resp.Agent.Version)
		}
		if resp.Agent.Framework != "autogen" {
			t.Errorf("expected framework autogen after re-register, got %q", resp.Agent.Framework)
		}

		// List should still show only 1 agent
		listResp, err := h.ListAgents(context.Background(), &agentv1.ListAgentsRequest{TenantId: "tenant-1"})
		if err != nil {
			t.Fatalf("ListAgents: unexpected error: %v", err)
		}
		if len(listResp.Agents) != 1 {
			t.Errorf("expected 1 agent after re-register, got %d", len(listResp.Agents))
		}
	})
}
