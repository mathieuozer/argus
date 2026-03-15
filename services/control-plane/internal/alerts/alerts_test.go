package alerts

import (
	"testing"
)

func TestFire(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		agentID  string
		severity Severity
		title    string
		message  string
	}{
		{
			name:     "fire critical alert",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			severity: SeverityCritical,
			title:    "Agent failure imminent",
			message:  "Token escalation detected, estimated TTF: 180s",
		},
		{
			name:     "fire warning alert",
			tenantID: "tenant-1",
			agentID:  "agent-2",
			severity: SeverityWarning,
			title:    "Elevated latency",
			message:  "p99 latency 3x above p50 for 45 seconds",
		},
		{
			name:     "fire info alert",
			tenantID: "tenant-2",
			agentID:  "agent-1",
			severity: SeverityInfo,
			title:    "Agent version updated",
			message:  "Agent updated from v1.0.0 to v1.1.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter()
			alert := router.Fire(tc.tenantID, tc.agentID, tc.severity, tc.title, tc.message)

			if alert == nil {
				t.Fatal("Fire() returned nil")
			}
			if alert.ID == "" {
				t.Error("expected non-empty alert ID")
			}
			if alert.TenantID != tc.tenantID {
				t.Errorf("TenantID = %q, want %q", alert.TenantID, tc.tenantID)
			}
			if alert.AgentID != tc.agentID {
				t.Errorf("AgentID = %q, want %q", alert.AgentID, tc.agentID)
			}
			if alert.Severity != tc.severity {
				t.Errorf("Severity = %q, want %q", alert.Severity, tc.severity)
			}
			if alert.Title != tc.title {
				t.Errorf("Title = %q, want %q", alert.Title, tc.title)
			}
			if alert.Message != tc.message {
				t.Errorf("Message = %q, want %q", alert.Message, tc.message)
			}
			if alert.Status != StatusOpen {
				t.Errorf("Status = %q, want %q", alert.Status, StatusOpen)
			}
			if alert.CreatedAt.IsZero() {
				t.Error("expected CreatedAt to be set")
			}
		})
	}

	t.Run("sequential fires produce unique IDs", func(t *testing.T) {
		router := NewRouter()
		alert1 := router.Fire("t1", "a1", SeverityInfo, "title1", "msg1")
		alert2 := router.Fire("t1", "a1", SeverityInfo, "title2", "msg2")

		if alert1.ID == alert2.ID {
			t.Errorf("expected unique IDs, got %q and %q", alert1.ID, alert2.ID)
		}
	})

	t.Run("sequential fires produce incrementing IDs", func(t *testing.T) {
		router := NewRouter()
		alert1 := router.Fire("t1", "a1", SeverityInfo, "title1", "msg1")
		alert2 := router.Fire("t1", "a1", SeverityInfo, "title2", "msg2")

		if alert1.ID != "alert-1" {
			t.Errorf("first alert ID = %q, want %q", alert1.ID, "alert-1")
		}
		if alert2.ID != "alert-2" {
			t.Errorf("second alert ID = %q, want %q", alert2.ID, "alert-2")
		}
	})
}

func TestAlertList(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		setup    func(*Router)
		wantLen  int
	}{
		{
			name:     "empty router returns empty list",
			tenantID: "tenant-1",
			setup:    func(r *Router) {},
			wantLen:  0,
		},
		{
			name:     "returns alerts for matching tenant",
			tenantID: "tenant-1",
			setup: func(r *Router) {
				r.Fire("tenant-1", "agent-1", SeverityCritical, "alert1", "msg1")
				r.Fire("tenant-1", "agent-2", SeverityWarning, "alert2", "msg2")
			},
			wantLen: 2,
		},
		{
			name:     "filters by tenant - isolates data",
			tenantID: "tenant-1",
			setup: func(r *Router) {
				r.Fire("tenant-1", "agent-1", SeverityCritical, "alert1", "msg1")
				r.Fire("tenant-2", "agent-1", SeverityWarning, "alert2", "msg2")
				r.Fire("tenant-1", "agent-2", SeverityInfo, "alert3", "msg3")
			},
			wantLen: 2,
		},
		{
			name:     "no alerts for nonexistent tenant",
			tenantID: "nonexistent",
			setup: func(r *Router) {
				r.Fire("tenant-1", "agent-1", SeverityInfo, "alert", "msg")
			},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter()
			tc.setup(router)

			alerts := router.List(tc.tenantID)
			if len(alerts) != tc.wantLen {
				t.Errorf("List(%q) returned %d alerts, want %d", tc.tenantID, len(alerts), tc.wantLen)
			}

			// Verify all returned alerts belong to the correct tenant
			for _, a := range alerts {
				if a.TenantID != tc.tenantID {
					t.Errorf("alert %q has TenantID %q, want %q", a.ID, a.TenantID, tc.tenantID)
				}
			}
		})
	}
}
