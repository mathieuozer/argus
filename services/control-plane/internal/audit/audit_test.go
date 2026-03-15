package audit

import (
	"testing"
)

func TestWrite(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		actor    string
		action   string
		resource string
		details  string
	}{
		{
			name:     "write standard audit entry",
			tenantID: "tenant-1",
			actor:    "admin@example.com",
			action:   "agent.register",
			resource: "agent/budget-reconciler",
			details:  "Registered new agent version 1.0.0",
		},
		{
			name:     "write with empty details",
			tenantID: "tenant-2",
			actor:    "system",
			action:   "cert.rotate",
			resource: "identity/ca",
			details:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			writer := NewWriter()
			entry := writer.Write(tc.tenantID, tc.actor, tc.action, tc.resource, tc.details)

			if entry == nil {
				t.Fatal("Write() returned nil")
			}
			if entry.ID == "" {
				t.Error("expected non-empty entry ID")
			}
			if entry.TenantID != tc.tenantID {
				t.Errorf("TenantID = %q, want %q", entry.TenantID, tc.tenantID)
			}
			if entry.Actor != tc.actor {
				t.Errorf("Actor = %q, want %q", entry.Actor, tc.actor)
			}
			if entry.Action != tc.action {
				t.Errorf("Action = %q, want %q", entry.Action, tc.action)
			}
			if entry.Resource != tc.resource {
				t.Errorf("Resource = %q, want %q", entry.Resource, tc.resource)
			}
			if entry.Details != tc.details {
				t.Errorf("Details = %q, want %q", entry.Details, tc.details)
			}
			if entry.Timestamp.IsZero() {
				t.Error("expected Timestamp to be set")
			}
		})
	}

	t.Run("sequential writes produce unique IDs", func(t *testing.T) {
		writer := NewWriter()
		entry1 := writer.Write("t1", "actor", "action", "resource", "")
		entry2 := writer.Write("t1", "actor", "action", "resource", "")

		if entry1.ID == entry2.ID {
			t.Errorf("expected unique IDs, got %q and %q", entry1.ID, entry2.ID)
		}
	})

	t.Run("sequential writes produce incrementing IDs", func(t *testing.T) {
		writer := NewWriter()
		entry1 := writer.Write("t1", "actor", "action", "resource", "")
		entry2 := writer.Write("t1", "actor", "action", "resource", "")

		if entry1.ID != "audit-1" {
			t.Errorf("first entry ID = %q, want %q", entry1.ID, "audit-1")
		}
		if entry2.ID != "audit-2" {
			t.Errorf("second entry ID = %q, want %q", entry2.ID, "audit-2")
		}
	})
}

func TestList(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		setup    func(*Writer)
		wantLen  int
	}{
		{
			name:     "empty writer returns empty list",
			tenantID: "tenant-1",
			setup:    func(w *Writer) {},
			wantLen:  0,
		},
		{
			name:     "returns entries for matching tenant",
			tenantID: "tenant-1",
			setup: func(w *Writer) {
				w.Write("tenant-1", "actor1", "action1", "resource1", "details1")
				w.Write("tenant-1", "actor2", "action2", "resource2", "details2")
			},
			wantLen: 2,
		},
		{
			name:     "filters by tenant - isolates data",
			tenantID: "tenant-1",
			setup: func(w *Writer) {
				w.Write("tenant-1", "actor1", "action1", "resource1", "")
				w.Write("tenant-2", "actor2", "action2", "resource2", "")
				w.Write("tenant-1", "actor3", "action3", "resource3", "")
			},
			wantLen: 2,
		},
		{
			name:     "no entries for nonexistent tenant",
			tenantID: "nonexistent",
			setup: func(w *Writer) {
				w.Write("tenant-1", "actor", "action", "resource", "")
			},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			writer := NewWriter()
			tc.setup(writer)

			entries := writer.List(tc.tenantID)
			if len(entries) != tc.wantLen {
				t.Errorf("List(%q) returned %d entries, want %d", tc.tenantID, len(entries), tc.wantLen)
			}

			// Verify all returned entries belong to the correct tenant
			for _, e := range entries {
				if e.TenantID != tc.tenantID {
					t.Errorf("entry %q has TenantID %q, want %q", e.ID, e.TenantID, tc.tenantID)
				}
			}
		})
	}
}
