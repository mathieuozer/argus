package versioning

import (
	"testing"
)

func TestSet(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		agentID  string
		version  string
		isCanary bool
	}{
		{
			name:     "set stable version",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			version:  "1.0.0",
			isCanary: false,
		},
		{
			name:     "set canary version",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			version:  "2.0.0-beta",
			isCanary: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tracker := New()
			tracker.Set(tc.tenantID, tc.agentID, tc.version, tc.isCanary)

			v, err := tracker.Get(tc.tenantID, tc.agentID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.AgentID != tc.agentID {
				t.Errorf("expected agent ID %q, got %q", tc.agentID, v.AgentID)
			}
			if v.Version != tc.version {
				t.Errorf("expected version %q, got %q", tc.version, v.Version)
			}
			if v.IsCanary != tc.isCanary {
				t.Errorf("expected isCanary %v, got %v", tc.isCanary, v.IsCanary)
			}
		})
	}

	t.Run("overwrites existing version", func(t *testing.T) {
		tracker := New()
		tracker.Set("tenant-1", "agent-1", "1.0.0", false)
		tracker.Set("tenant-1", "agent-1", "2.0.0", true)

		v, err := tracker.Get("tenant-1", "agent-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.Version != "2.0.0" {
			t.Errorf("expected version %q, got %q", "2.0.0", v.Version)
		}
		if !v.IsCanary {
			t.Error("expected isCanary to be true after overwrite")
		}
	})
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		agentID  string
		setup    func(*Tracker)
		wantErr  bool
		wantVer  string
	}{
		{
			name:     "found",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(tr *Tracker) {
				tr.Set("tenant-1", "agent-1", "3.2.1", false)
			},
			wantErr: false,
			wantVer: "3.2.1",
		},
		{
			name:     "not found - unknown tenant",
			tenantID: "nonexistent",
			agentID:  "agent-1",
			setup:    func(tr *Tracker) {},
			wantErr:  true,
		},
		{
			name:     "not found - unknown agent",
			tenantID: "tenant-1",
			agentID:  "nonexistent",
			setup: func(tr *Tracker) {
				tr.Set("tenant-1", "agent-1", "1.0.0", false)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tracker := New()
			tc.setup(tracker)

			v, err := tracker.Get(tc.tenantID, tc.agentID)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.Version != tc.wantVer {
				t.Errorf("expected version %q, got %q", tc.wantVer, v.Version)
			}
		})
	}
}
