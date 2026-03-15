package router

import (
	"testing"

	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
)

func TestRoute(t *testing.T) {
	tests := []struct {
		name             string
		tenantID         string
		capabilities     []string
		preferredAgentID string
		setup            func(*registry.Registry)
		wantErr          bool
		wantAgentID      string
	}{
		{
			name:             "preferred agent is selected when healthy and has capabilities",
			tenantID:         "tenant-1",
			capabilities:     []string{"read:db"},
			preferredAgentID: "agent-preferred",
			setup: func(reg *registry.Registry) {
				reg.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-preferred",
					Version:      "1.0.0",
					Capabilities: []string{"read:db", "write:report"},
				})
				_ = reg.Heartbeat("tenant-1", "agent-preferred", registry.StatusHealthy)

				reg.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-other",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = reg.Heartbeat("tenant-1", "agent-other", registry.StatusHealthy)
			},
			wantErr:     false,
			wantAgentID: "agent-preferred",
		},
		{
			name:             "falls back to capability matching when preferred agent not healthy",
			tenantID:         "tenant-1",
			capabilities:     []string{"read:db"},
			preferredAgentID: "agent-preferred",
			setup: func(reg *registry.Registry) {
				reg.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-preferred",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = reg.Heartbeat("tenant-1", "agent-preferred", registry.StatusDegraded)

				reg.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-healthy",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = reg.Heartbeat("tenant-1", "agent-healthy", registry.StatusHealthy)
			},
			wantErr:     false,
			wantAgentID: "agent-healthy",
		},
		{
			name:             "capability matching without preferred agent",
			tenantID:         "tenant-1",
			capabilities:     []string{"write:report"},
			preferredAgentID: "",
			setup: func(reg *registry.Registry) {
				reg.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-1",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = reg.Heartbeat("tenant-1", "agent-1", registry.StatusHealthy)

				reg.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-2",
					Version:      "1.0.0",
					Capabilities: []string{"write:report"},
				})
				_ = reg.Heartbeat("tenant-1", "agent-2", registry.StatusHealthy)
			},
			wantErr:     false,
			wantAgentID: "agent-2",
		},
		{
			name:             "no matching agents returns error",
			tenantID:         "tenant-1",
			capabilities:     []string{"admin:all"},
			preferredAgentID: "",
			setup: func(reg *registry.Registry) {
				reg.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-1",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = reg.Heartbeat("tenant-1", "agent-1", registry.StatusHealthy)
			},
			wantErr: true,
		},
		{
			name:             "no agents at all returns error",
			tenantID:         "tenant-1",
			capabilities:     []string{"read:db"},
			preferredAgentID: "",
			setup:            func(reg *registry.Registry) {},
			wantErr:          true,
		},
		{
			name:             "preferred agent lacks capabilities, falls back to capability match",
			tenantID:         "tenant-1",
			capabilities:     []string{"write:report"},
			preferredAgentID: "agent-preferred",
			setup: func(reg *registry.Registry) {
				reg.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-preferred",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = reg.Heartbeat("tenant-1", "agent-preferred", registry.StatusHealthy)

				reg.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-writer",
					Version:      "1.0.0",
					Capabilities: []string{"write:report"},
				})
				_ = reg.Heartbeat("tenant-1", "agent-writer", registry.StatusHealthy)
			},
			wantErr:     false,
			wantAgentID: "agent-writer",
		},
		{
			name:             "returns discovered agent when no healthy agents available",
			tenantID:         "tenant-1",
			capabilities:     []string{"read:db"},
			preferredAgentID: "",
			setup: func(reg *registry.Registry) {
				reg.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-discovered",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				// Agent stays in discovered state (not healthy)
			},
			wantErr:     false,
			wantAgentID: "agent-discovered",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := registry.New()
			tc.setup(reg)

			r := New(reg)
			agent, err := r.Route(tc.tenantID, tc.capabilities, tc.preferredAgentID)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if agent.ID != tc.wantAgentID {
				t.Errorf("expected agent ID %q, got %q", tc.wantAgentID, agent.ID)
			}
		})
	}
}
