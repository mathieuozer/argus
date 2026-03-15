package identity

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewManager(t *testing.T) {
	logger := zap.NewNop()
	m := NewManager(logger)
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.logger != logger {
		t.Error("expected logger to be set")
	}
	if m.spiffeID != "" {
		t.Errorf("expected empty spiffeID on creation, got %s", m.spiffeID)
	}
	if m.certPEM != nil {
		t.Error("expected nil certPEM on creation")
	}
	if m.keyPEM != nil {
		t.Error("expected nil keyPEM on creation")
	}
}

func TestRequestSVID(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		agentID  string
		version  string
	}{
		{
			name:     "standard request",
			tenantID: "ministry-finance-tr",
			agentID:  "budget-reconciler",
			version:  "1.0.0",
		},
		{
			name:     "empty version",
			tenantID: "test-tenant",
			agentID:  "test-agent",
			version:  "",
		},
		{
			name:     "complex agent ID",
			tenantID: "eu-corp-alpha",
			agentID:  "data-pipeline-processor-v2",
			version:  "2.3.1-beta",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewManager(zap.NewNop())
			err := m.RequestSVID(tc.tenantID, tc.agentID, tc.version)
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
		})
	}
}

func TestRenewSVID(t *testing.T) {
	tests := []struct {
		name     string
		spiffeID string
	}{
		{
			name:     "renew with empty spiffe ID",
			spiffeID: "",
		},
		{
			name:     "renew with existing spiffe ID",
			spiffeID: "spiffe://argus.example.com/tenant/test-tenant/agent/test-agent/v1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewManager(zap.NewNop())
			m.spiffeID = tc.spiffeID
			err := m.RenewSVID()
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
		})
	}
}

func TestGetSpiffeID(t *testing.T) {
	tests := []struct {
		name     string
		spiffeID string
		expected string
	}{
		{
			name:     "empty on fresh manager",
			spiffeID: "",
			expected: "",
		},
		{
			name:     "returns set spiffe ID",
			spiffeID: "spiffe://argus.example.com/tenant/ministry-finance-tr/agent/budget-reconciler/v1",
			expected: "spiffe://argus.example.com/tenant/ministry-finance-tr/agent/budget-reconciler/v1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewManager(zap.NewNop())
			m.spiffeID = tc.spiffeID
			got := m.GetSpiffeID()
			if got != tc.expected {
				t.Errorf("expected spiffeID %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestGetSpiffeID_EmptyOnCreation(t *testing.T) {
	m := NewManager(zap.NewNop())
	if got := m.GetSpiffeID(); got != "" {
		t.Errorf("expected empty spiffeID on new manager, got %q", got)
	}
}
