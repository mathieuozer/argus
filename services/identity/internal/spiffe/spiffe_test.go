package spiffe

import (
	"testing"
)

func TestAgentID(t *testing.T) {
	tests := []struct {
		name        string
		trustDomain string
		tenantID    string
		agentID     string
		version     string
		want        string
	}{
		{
			name:        "standard format",
			trustDomain: "argus.example.com",
			tenantID:    "ministry-finance",
			agentID:     "budget-reconciler",
			version:     "1.0.0",
			want:        "spiffe://argus.example.com/tenant/ministry-finance/agent/budget-reconciler/v1.0.0",
		},
		{
			name:        "different trust domain",
			trustDomain: "argus.gov.tr",
			tenantID:    "tenant-abc",
			agentID:     "agent-xyz",
			version:     "2.5.1",
			want:        "spiffe://argus.gov.tr/tenant/tenant-abc/agent/agent-xyz/v2.5.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gen := NewGenerator(tc.trustDomain)
			got := gen.AgentID(tc.tenantID, tc.agentID, tc.version)
			if got != tc.want {
				t.Errorf("AgentID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name          string
		trustDomain   string
		spiffeID      string
		wantTenantID  string
		wantAgentID   string
		wantVersion   string
		wantErr       bool
	}{
		{
			name:         "valid SPIFFE ID",
			trustDomain:  "argus.example.com",
			spiffeID:     "spiffe://argus.example.com/tenant/ministry-finance/agent/budget-reconciler/v1.0.0",
			wantTenantID: "ministry-finance",
			wantAgentID:  "budget-reconciler",
			wantVersion:  "1.0.0",
			wantErr:      false,
		},
		{
			name:         "valid SPIFFE ID with different domain",
			trustDomain:  "argus.gov.tr",
			spiffeID:     "spiffe://argus.gov.tr/tenant/tenant-1/agent/agent-1/v2.0.0",
			wantTenantID: "tenant-1",
			wantAgentID:  "agent-1",
			wantVersion:  "2.0.0",
			wantErr:      false,
		},
		{
			name:        "invalid - too short",
			trustDomain: "argus.example.com",
			spiffeID:    "spiffe://short",
			wantErr:     true,
		},
		{
			name:        "invalid - missing segments",
			trustDomain: "argus.example.com",
			spiffeID:    "spiffe://argus.example.com/tenant/only-tenant",
			wantErr:     true,
		},
		{
			name:        "invalid - empty string",
			trustDomain: "argus.example.com",
			spiffeID:    "",
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gen := NewGenerator(tc.trustDomain)
			tenantID, agentID, version, err := gen.Parse(tc.spiffeID)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tenantID != tc.wantTenantID {
				t.Errorf("tenantID = %q, want %q", tenantID, tc.wantTenantID)
			}
			if agentID != tc.wantAgentID {
				t.Errorf("agentID = %q, want %q", agentID, tc.wantAgentID)
			}
			if version != tc.wantVersion {
				t.Errorf("version = %q, want %q", version, tc.wantVersion)
			}
		})
	}

	t.Run("round trip - generate then parse", func(t *testing.T) {
		gen := NewGenerator("argus.test.local")
		spiffeID := gen.AgentID("tenant-42", "agent-alpha", "3.1.4")

		tenantID, agentID, version, err := gen.Parse(spiffeID)
		if err != nil {
			t.Fatalf("round trip parse failed: %v", err)
		}
		if tenantID != "tenant-42" {
			t.Errorf("tenantID = %q, want %q", tenantID, "tenant-42")
		}
		if agentID != "agent-alpha" {
			t.Errorf("agentID = %q, want %q", agentID, "agent-alpha")
		}
		if version != "3.1.4" {
			t.Errorf("version = %q, want %q", version, "3.1.4")
		}
	})
}
