package revocation

import (
	"testing"
)

func TestRevoke(t *testing.T) {
	tests := []struct {
		name     string
		spiffeID string
		reason   string
	}{
		{
			name:     "revoke with reason",
			spiffeID: "spiffe://argus.test/tenant/t1/agent/a1/v1.0.0",
			reason:   "compromised key",
		},
		{
			name:     "revoke with empty reason",
			spiffeID: "spiffe://argus.test/tenant/t2/agent/a2/v2.0.0",
			reason:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore()
			store.Revoke(tc.spiffeID, tc.reason)

			if !store.IsRevoked(tc.spiffeID) {
				t.Errorf("expected %q to be revoked", tc.spiffeID)
			}
		})
	}

	t.Run("revoke sets correct fields", func(t *testing.T) {
		store := NewStore()
		store.Revoke("spiffe://test/agent", "key compromised")

		entries := store.List()
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}

		entry := entries[0]
		if entry.SpiffeID != "spiffe://test/agent" {
			t.Errorf("expected SpiffeID %q, got %q", "spiffe://test/agent", entry.SpiffeID)
		}
		if entry.Reason != "key compromised" {
			t.Errorf("expected Reason %q, got %q", "key compromised", entry.Reason)
		}
		if entry.RevokedAt.IsZero() {
			t.Error("expected RevokedAt to be set")
		}
	})
}

func TestIsRevoked(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Store)
		spiffeID string
		want     bool
	}{
		{
			name: "revoked cert returns true",
			setup: func(s *Store) {
				s.Revoke("spiffe://test/revoked", "bad cert")
			},
			spiffeID: "spiffe://test/revoked",
			want:     true,
		},
		{
			name:     "non-revoked cert returns false",
			setup:    func(s *Store) {},
			spiffeID: "spiffe://test/valid",
			want:     false,
		},
		{
			name: "different cert not affected",
			setup: func(s *Store) {
				s.Revoke("spiffe://test/other", "revoked")
			},
			spiffeID: "spiffe://test/valid",
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore()
			tc.setup(store)

			got := store.IsRevoked(tc.spiffeID)
			if got != tc.want {
				t.Errorf("IsRevoked(%q) = %v, want %v", tc.spiffeID, got, tc.want)
			}
		})
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Store)
		wantLen int
	}{
		{
			name:    "empty store",
			setup:   func(s *Store) {},
			wantLen: 0,
		},
		{
			name: "single revoked cert",
			setup: func(s *Store) {
				s.Revoke("spiffe://test/a1", "reason1")
			},
			wantLen: 1,
		},
		{
			name: "multiple revoked certs",
			setup: func(s *Store) {
				s.Revoke("spiffe://test/a1", "reason1")
				s.Revoke("spiffe://test/a2", "reason2")
				s.Revoke("spiffe://test/a3", "reason3")
			},
			wantLen: 3,
		},
		{
			name: "revoking same cert twice does not duplicate",
			setup: func(s *Store) {
				s.Revoke("spiffe://test/a1", "reason1")
				s.Revoke("spiffe://test/a1", "reason2")
			},
			wantLen: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore()
			tc.setup(store)

			entries := store.List()
			if len(entries) != tc.wantLen {
				t.Errorf("List() returned %d entries, want %d", len(entries), tc.wantLen)
			}
		})
	}

	t.Run("list contains all revoked spiffe IDs", func(t *testing.T) {
		store := NewStore()
		ids := []string{
			"spiffe://test/agent-1",
			"spiffe://test/agent-2",
			"spiffe://test/agent-3",
		}
		for _, id := range ids {
			store.Revoke(id, "revoked")
		}

		entries := store.List()
		found := make(map[string]bool)
		for _, e := range entries {
			found[e.SpiffeID] = true
		}

		for _, id := range ids {
			if !found[id] {
				t.Errorf("expected %q in list, but not found", id)
			}
		}
	})
}
