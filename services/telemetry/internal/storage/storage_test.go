package storage

import (
	"testing"

	"github.com/argus-platform/argus/services/telemetry/internal/classifier"
)

func TestInMemoryBackendStore(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		tier     classifier.DataTier
		data     map[string]string
		wantErr  bool
	}{
		{
			name:     "store structural data",
			tenantID: "tenant-1",
			tier:     classifier.TierStructural,
			data:     map[string]string{"latency_ms": "42", "error_code": "NONE"},
			wantErr:  false,
		},
		{
			name:     "store sensitive data",
			tenantID: "tenant-1",
			tier:     classifier.TierSensitive,
			data:     map[string]string{"task_desc": "summarize document"},
			wantErr:  false,
		},
		{
			name:     "store restricted data",
			tenantID: "tenant-1",
			tier:     classifier.TierRestricted,
			data:     map[string]string{"full_input": "classified content"},
			wantErr:  false,
		},
		{
			name:     "store empty data",
			tenantID: "tenant-1",
			tier:     classifier.TierStructural,
			data:     map[string]string{},
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			backend := NewInMemoryBackend()

			err := backend.Store(tc.tenantID, tc.tier, tc.data)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}

	t.Run("stores multiple records per tenant and tier", func(t *testing.T) {
		backend := NewInMemoryBackend()

		err := backend.Store("tenant-1", classifier.TierStructural, map[string]string{"latency_ms": "100"})
		if err != nil {
			t.Fatalf("unexpected error on first store: %v", err)
		}

		err = backend.Store("tenant-1", classifier.TierStructural, map[string]string{"latency_ms": "200"})
		if err != nil {
			t.Fatalf("unexpected error on second store: %v", err)
		}

		// Verify both records are stored
		records := backend.data["tenant-1"][classifier.TierStructural]
		if len(records) != 2 {
			t.Errorf("expected 2 records, got %d", len(records))
		}
	})

	t.Run("isolates data between tenants", func(t *testing.T) {
		backend := NewInMemoryBackend()

		err := backend.Store("tenant-1", classifier.TierStructural, map[string]string{"latency_ms": "100"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = backend.Store("tenant-2", classifier.TierStructural, map[string]string{"latency_ms": "200"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tenant1Records := backend.data["tenant-1"][classifier.TierStructural]
		tenant2Records := backend.data["tenant-2"][classifier.TierStructural]

		if len(tenant1Records) != 1 {
			t.Errorf("expected 1 record for tenant-1, got %d", len(tenant1Records))
		}
		if len(tenant2Records) != 1 {
			t.Errorf("expected 1 record for tenant-2, got %d", len(tenant2Records))
		}
	})

	t.Run("isolates data between tiers", func(t *testing.T) {
		backend := NewInMemoryBackend()

		err := backend.Store("tenant-1", classifier.TierStructural, map[string]string{"latency_ms": "100"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = backend.Store("tenant-1", classifier.TierSensitive, map[string]string{"task_desc": "summarize"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		structRecords := backend.data["tenant-1"][classifier.TierStructural]
		sensRecords := backend.data["tenant-1"][classifier.TierSensitive]

		if len(structRecords) != 1 {
			t.Errorf("expected 1 structural record, got %d", len(structRecords))
		}
		if len(sensRecords) != 1 {
			t.Errorf("expected 1 sensitive record, got %d", len(sensRecords))
		}
	})

	t.Run("implements Backend interface", func(t *testing.T) {
		var _ Backend = NewInMemoryBackend()
	})
}
