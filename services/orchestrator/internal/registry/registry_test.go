package registry

import (
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	t.Run("creates agent with status discovered", func(t *testing.T) {
		reg := New()
		req := &RegisterRequest{
			AgentID:      "agent-1",
			Version:      "1.0.0",
			Framework:    "langchain",
			Capabilities: []string{"read:db", "write:report"},
			NodeID:       "node-1",
		}

		agent := reg.Register("tenant-1", req)

		if agent.ID != "agent-1" {
			t.Errorf("expected agent ID %q, got %q", "agent-1", agent.ID)
		}
		if agent.TenantID != "tenant-1" {
			t.Errorf("expected tenant ID %q, got %q", "tenant-1", agent.TenantID)
		}
		if agent.Version != "1.0.0" {
			t.Errorf("expected version %q, got %q", "1.0.0", agent.Version)
		}
		if agent.Framework != "langchain" {
			t.Errorf("expected framework %q, got %q", "langchain", agent.Framework)
		}
		if agent.Status != StatusDiscovered {
			t.Errorf("expected status %q, got %q", StatusDiscovered, agent.Status)
		}
		if agent.NodeID != "node-1" {
			t.Errorf("expected node ID %q, got %q", "node-1", agent.NodeID)
		}
		if len(agent.Capabilities) != 2 {
			t.Errorf("expected 2 capabilities, got %d", len(agent.Capabilities))
		}
		if agent.LastSeen.IsZero() {
			t.Error("expected LastSeen to be set")
		}
	})
}

func TestGet(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		agentID   string
		setup     func(*Registry)
		wantErr   bool
		wantAgent string
	}{
		{
			name:     "found",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
			},
			wantErr:   false,
			wantAgent: "agent-1",
		},
		{
			name:     "not found - no tenant",
			tenantID: "nonexistent",
			agentID:  "agent-1",
			setup:    func(r *Registry) {},
			wantErr:  true,
		},
		{
			name:     "not found - wrong agent ID",
			tenantID: "tenant-1",
			agentID:  "nonexistent",
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := New()
			tc.setup(reg)

			agent, err := reg.Get(tc.tenantID, tc.agentID)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if agent.ID != tc.wantAgent {
				t.Errorf("expected agent ID %q, got %q", tc.wantAgent, agent.ID)
			}
		})
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		setup    func(*Registry)
		wantLen  int
	}{
		{
			name:     "empty registry",
			tenantID: "tenant-1",
			setup:    func(r *Registry) {},
			wantLen:  0,
		},
		{
			name:     "multiple agents",
			tenantID: "tenant-1",
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-2", Version: "2.0.0"})
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-3", Version: "1.5.0"})
			},
			wantLen: 3,
		},
		{
			name:     "only returns agents for specified tenant",
			tenantID: "tenant-1",
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
				r.Register("tenant-2", &RegisterRequest{AgentID: "agent-2", Version: "2.0.0"})
			},
			wantLen: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := New()
			tc.setup(reg)

			agents := reg.List(tc.tenantID)
			if len(agents) != tc.wantLen {
				t.Errorf("expected %d agents, got %d", tc.wantLen, len(agents))
			}
		})
	}
}

func TestHeartbeat(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		agentID    string
		status     AgentStatus
		setup      func(*Registry)
		wantErr    bool
		wantStatus AgentStatus
	}{
		{
			name:     "updates status and last_seen",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			status:   StatusHealthy,
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
			},
			wantErr:    false,
			wantStatus: StatusHealthy,
		},
		{
			name:     "agent not found",
			tenantID: "tenant-1",
			agentID:  "nonexistent",
			status:   StatusHealthy,
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
			},
			wantErr: true,
		},
		{
			name:     "tenant not found",
			tenantID: "nonexistent",
			agentID:  "agent-1",
			status:   StatusHealthy,
			setup:    func(r *Registry) {},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := New()
			tc.setup(reg)

			beforeHeartbeat := time.Now()
			err := reg.Heartbeat(tc.tenantID, tc.agentID, tc.status)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			agent, err := reg.Get(tc.tenantID, tc.agentID)
			if err != nil {
				t.Fatalf("failed to get agent after heartbeat: %v", err)
			}
			if agent.Status != tc.wantStatus {
				t.Errorf("expected status %q, got %q", tc.wantStatus, agent.Status)
			}
			if agent.LastSeen.Before(beforeHeartbeat) {
				t.Error("expected LastSeen to be updated to current time")
			}
		})
	}
}

func TestQuarantineAgent(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		agentID  string
		setup    func(*Registry)
		wantErr  bool
	}{
		{
			name:     "successfully quarantines a healthy agent",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
				_ = r.Heartbeat("tenant-1", "agent-1", StatusHealthy)
			},
			wantErr: false,
		},
		{
			name:     "successfully quarantines a degraded agent",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
				_ = r.Heartbeat("tenant-1", "agent-1", StatusDegraded)
			},
			wantErr: false,
		},
		{
			name:     "quarantining already quarantined agent is idempotent",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
				_ = r.QuarantineAgent("tenant-1", "agent-1")
			},
			wantErr: false,
		},
		{
			name:     "agent not found",
			tenantID: "tenant-1",
			agentID:  "nonexistent",
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
			},
			wantErr: true,
		},
		{
			name:     "tenant not found",
			tenantID: "nonexistent",
			agentID:  "agent-1",
			setup:    func(r *Registry) {},
			wantErr:  true,
		},
		{
			name:     "cross-tenant quarantine fails",
			tenantID: "tenant-2",
			agentID:  "agent-1",
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{AgentID: "agent-1", Version: "1.0.0"})
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := New()
			tc.setup(reg)

			beforeQuarantine := time.Now()
			err := reg.QuarantineAgent(tc.tenantID, tc.agentID)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			agent, err := reg.Get(tc.tenantID, tc.agentID)
			if err != nil {
				t.Fatalf("failed to get agent after quarantine: %v", err)
			}
			if agent.Status != StatusQuarantined {
				t.Errorf("expected status %q, got %q", StatusQuarantined, agent.Status)
			}
			if agent.LastSeen.Before(beforeQuarantine) {
				t.Error("expected LastSeen to be updated to current time")
			}
		})
	}
}

func TestFindByCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		tenantID     string
		capabilities []string
		setup        func(*Registry)
		wantLen      int
		wantIDs      []string
	}{
		{
			name:         "matches agents with required capabilities",
			tenantID:     "tenant-1",
			capabilities: []string{"read:db"},
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{
					AgentID:      "agent-1",
					Version:      "1.0.0",
					Capabilities: []string{"read:db", "write:report"},
				})
				// Set agent-1 to healthy so it qualifies
				_ = r.Heartbeat("tenant-1", "agent-1", StatusHealthy)
			},
			wantLen: 1,
			wantIDs: []string{"agent-1"},
		},
		{
			name:         "no match when capability not present",
			tenantID:     "tenant-1",
			capabilities: []string{"admin:all"},
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{
					AgentID:      "agent-1",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = r.Heartbeat("tenant-1", "agent-1", StatusHealthy)
			},
			wantLen: 0,
		},
		{
			name:         "skips quarantined agents",
			tenantID:     "tenant-1",
			capabilities: []string{"read:db"},
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{
					AgentID:      "agent-1",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = r.Heartbeat("tenant-1", "agent-1", StatusQuarantined)

				r.Register("tenant-1", &RegisterRequest{
					AgentID:      "agent-2",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = r.Heartbeat("tenant-1", "agent-2", StatusHealthy)
			},
			wantLen: 1,
			wantIDs: []string{"agent-2"},
		},
		{
			name:         "skips failed agents",
			tenantID:     "tenant-1",
			capabilities: []string{"read:db"},
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{
					AgentID:      "agent-1",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = r.Heartbeat("tenant-1", "agent-1", StatusFailed)
			},
			wantLen: 0,
		},
		{
			name:         "matches multiple capabilities",
			tenantID:     "tenant-1",
			capabilities: []string{"read:db", "write:report"},
			setup: func(r *Registry) {
				r.Register("tenant-1", &RegisterRequest{
					AgentID:      "agent-1",
					Version:      "1.0.0",
					Capabilities: []string{"read:db", "write:report", "admin:config"},
				})
				_ = r.Heartbeat("tenant-1", "agent-1", StatusHealthy)

				r.Register("tenant-1", &RegisterRequest{
					AgentID:      "agent-2",
					Version:      "1.0.0",
					Capabilities: []string{"read:db"},
				})
				_ = r.Heartbeat("tenant-1", "agent-2", StatusHealthy)
			},
			wantLen: 1,
			wantIDs: []string{"agent-1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := New()
			tc.setup(reg)

			agents := reg.FindByCapabilities(tc.tenantID, tc.capabilities)
			if len(agents) != tc.wantLen {
				t.Fatalf("expected %d agents, got %d", tc.wantLen, len(agents))
			}

			if tc.wantIDs != nil {
				foundIDs := make(map[string]bool)
				for _, a := range agents {
					foundIDs[a.ID] = true
				}
				for _, wantID := range tc.wantIDs {
					if !foundIDs[wantID] {
						t.Errorf("expected agent %q in results, but not found", wantID)
					}
				}
			}
		})
	}
}
