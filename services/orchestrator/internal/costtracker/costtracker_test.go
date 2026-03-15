package costtracker

import (
	"math"
	"testing"
)

func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestRecord(t *testing.T) {
	tests := []struct {
		name     string
		records  []struct{ tenantID, agentID string; cost float64 }
		checkTID string
		checkAID string
		wantCost float64
	}{
		{
			name: "single record",
			records: []struct{ tenantID, agentID string; cost float64 }{
				{"tenant-1", "agent-1", 1.50},
			},
			checkTID: "tenant-1",
			checkAID: "agent-1",
			wantCost: 1.50,
		},
		{
			name: "accumulates costs for same agent",
			records: []struct{ tenantID, agentID string; cost float64 }{
				{"tenant-1", "agent-1", 1.50},
				{"tenant-1", "agent-1", 2.50},
				{"tenant-1", "agent-1", 0.75},
			},
			checkTID: "tenant-1",
			checkAID: "agent-1",
			wantCost: 4.75,
		},
		{
			name: "separate agents tracked independently",
			records: []struct{ tenantID, agentID string; cost float64 }{
				{"tenant-1", "agent-1", 3.00},
				{"tenant-1", "agent-2", 5.00},
			},
			checkTID: "tenant-1",
			checkAID: "agent-1",
			wantCost: 3.00,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tracker := New()
			for _, r := range tc.records {
				tracker.Record(r.tenantID, r.agentID, r.cost)
			}

			got := tracker.GetAgentCost(tc.checkTID, tc.checkAID)
			if !floatEquals(got, tc.wantCost) {
				t.Errorf("expected agent cost %f, got %f", tc.wantCost, got)
			}
		})
	}
}

func TestGetAgentCost(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		agentID  string
		setup    func(*Tracker)
		wantCost float64
	}{
		{
			name:     "returns 0 for unknown tenant",
			tenantID: "nonexistent",
			agentID:  "agent-1",
			setup:    func(tr *Tracker) {},
			wantCost: 0,
		},
		{
			name:     "returns 0 for unknown agent",
			tenantID: "tenant-1",
			agentID:  "nonexistent",
			setup: func(tr *Tracker) {
				tr.Record("tenant-1", "agent-1", 5.00)
			},
			wantCost: 0,
		},
		{
			name:     "returns correct cost",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(tr *Tracker) {
				tr.Record("tenant-1", "agent-1", 2.50)
				tr.Record("tenant-1", "agent-1", 3.50)
			},
			wantCost: 6.00,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tracker := New()
			tc.setup(tracker)

			got := tracker.GetAgentCost(tc.tenantID, tc.agentID)
			if !floatEquals(got, tc.wantCost) {
				t.Errorf("expected %f, got %f", tc.wantCost, got)
			}
		})
	}
}

func TestGetTenantCost(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		setup    func(*Tracker)
		wantCost float64
	}{
		{
			name:     "returns 0 for unknown tenant",
			tenantID: "nonexistent",
			setup:    func(tr *Tracker) {},
			wantCost: 0,
		},
		{
			name:     "sums costs across all agents",
			tenantID: "tenant-1",
			setup: func(tr *Tracker) {
				tr.Record("tenant-1", "agent-1", 2.00)
				tr.Record("tenant-1", "agent-2", 3.00)
				tr.Record("tenant-1", "agent-1", 1.00)
			},
			wantCost: 6.00,
		},
		{
			name:     "does not include costs from other tenants",
			tenantID: "tenant-1",
			setup: func(tr *Tracker) {
				tr.Record("tenant-1", "agent-1", 2.00)
				tr.Record("tenant-2", "agent-2", 10.00)
			},
			wantCost: 2.00,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tracker := New()
			tc.setup(tracker)

			got := tracker.GetTenantCost(tc.tenantID)
			if !floatEquals(got, tc.wantCost) {
				t.Errorf("expected %f, got %f", tc.wantCost, got)
			}
		})
	}
}
