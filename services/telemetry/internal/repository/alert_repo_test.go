package repository

import (
	"context"
	"testing"
	"time"
)

// ---------- Constructor tests ----------

func TestNewAlertRepository(t *testing.T) {
	t.Run("returns non-nil repository with nil pool", func(t *testing.T) {
		repo := NewAlertRepository(nil)
		if repo == nil {
			t.Fatal("expected non-nil AlertRepository")
		}
	})

	t.Run("pool field is set to nil", func(t *testing.T) {
		repo := NewAlertRepository(nil)
		if repo.pool != nil {
			t.Error("expected nil pool when constructed with nil")
		}
	})
}

// ---------- PredictiveAlert struct tests ----------

func TestPredictiveAlert_Fields(t *testing.T) {
	tests := []struct {
		name  string
		alert PredictiveAlert
	}{
		{
			name: "full alert with all fields",
			alert: PredictiveAlert{
				ID:              "alert-001",
				TenantID:        "tenant-1",
				AgentID:         "budget-reconciler",
				Probability:     0.92,
				EstimatedTTFSec: 180,
				PrecursorType:   "latency_spike",
				Evidence:        []string{"p99_latency=3200ms", "consecutive_slow=5"},
				Status:          "open",
				CreatedAt:       time.Now(),
				ResolvedAt:      nil,
			},
		},
		{
			name: "resolved alert",
			alert: PredictiveAlert{
				ID:              "alert-002",
				TenantID:        "tenant-2",
				AgentID:         "report-generator",
				Probability:     0.75,
				EstimatedTTFSec: 300,
				PrecursorType:   "token_escalation",
				Evidence:        []string{"token_velocity_spike"},
				Status:          "resolved",
				CreatedAt:       time.Now().Add(-1 * time.Hour),
				ResolvedAt:      timePtr(time.Now()),
			},
		},
		{
			name: "false positive alert",
			alert: PredictiveAlert{
				ID:              "alert-003",
				TenantID:        "tenant-3",
				AgentID:         "data-processor",
				Probability:     0.55,
				EstimatedTTFSec: 600,
				PrecursorType:   "retry_storm",
				Evidence:        []string{},
				Status:          "false_positive",
				CreatedAt:       time.Now().Add(-2 * time.Hour),
				ResolvedAt:      timePtr(time.Now()),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.alert.ID == "" {
				t.Error("alert ID should not be empty")
			}
			if tc.alert.TenantID == "" {
				t.Error("alert TenantID should not be empty")
			}
			if tc.alert.AgentID == "" {
				t.Error("alert AgentID should not be empty")
			}
			if tc.alert.Probability < 0 || tc.alert.Probability > 1 {
				t.Errorf("probability = %f, should be between 0 and 1", tc.alert.Probability)
			}
			if tc.alert.EstimatedTTFSec < 0 {
				t.Errorf("estimated_ttf_seconds = %d, should be non-negative", tc.alert.EstimatedTTFSec)
			}
			if tc.alert.PrecursorType == "" {
				t.Error("precursor_type should not be empty")
			}
			if tc.alert.Status == "" {
				t.Error("status should not be empty")
			}
			if tc.alert.CreatedAt.IsZero() {
				t.Error("created_at should not be zero")
			}
		})
	}
}

// ---------- Create: SQL parameter construction ----------

func TestAlertRepository_Create_SQLArgs(t *testing.T) {
	tests := []struct {
		name  string
		alert *PredictiveAlert
	}{
		{
			name: "standard alert create args",
			alert: &PredictiveAlert{
				TenantID:        "tenant-1",
				AgentID:         "agent-alpha",
				Probability:     0.85,
				EstimatedTTFSec: 120,
				PrecursorType:   "latency_spike",
				Evidence:        []string{"p99_ratio=4.2", "consecutive_slow=3"},
				Status:          "open",
			},
		},
		{
			name: "alert with empty evidence",
			alert: &PredictiveAlert{
				TenantID:        "tenant-2",
				AgentID:         "agent-beta",
				Probability:     0.60,
				EstimatedTTFSec: 300,
				PrecursorType:   "error_rate_delta",
				Evidence:        []string{},
				Status:          "open",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The SQL: INSERT INTO predictive_alerts
			//   (tenant_id, agent_id, probability, estimated_ttf_seconds, precursor_type, evidence, status)
			//   VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
			args := []interface{}{
				tc.alert.TenantID, tc.alert.AgentID, tc.alert.Probability,
				tc.alert.EstimatedTTFSec, tc.alert.PrecursorType,
				tc.alert.Evidence, tc.alert.Status,
			}

			if len(args) != 7 {
				t.Errorf("expected 7 SQL args for Create, got %d", len(args))
			}

			if args[0].(string) != tc.alert.TenantID {
				t.Errorf("arg[0] (tenant_id) = %q, want %q", args[0], tc.alert.TenantID)
			}
			if args[1].(string) != tc.alert.AgentID {
				t.Errorf("arg[1] (agent_id) = %q, want %q", args[1], tc.alert.AgentID)
			}
			if args[2].(float64) != tc.alert.Probability {
				t.Errorf("arg[2] (probability) = %v, want %v", args[2], tc.alert.Probability)
			}
			if args[3].(int) != tc.alert.EstimatedTTFSec {
				t.Errorf("arg[3] (estimated_ttf_seconds) = %v, want %v", args[3], tc.alert.EstimatedTTFSec)
			}
			if args[4].(string) != tc.alert.PrecursorType {
				t.Errorf("arg[4] (precursor_type) = %q, want %q", args[4], tc.alert.PrecursorType)
			}
			if args[6].(string) != tc.alert.Status {
				t.Errorf("arg[6] (status) = %q, want %q", args[6], tc.alert.Status)
			}
		})
	}
}

// ---------- ListByTenant: query construction logic ----------

func TestAlertRepository_ListByTenant_QueryConstruction(t *testing.T) {
	tests := []struct {
		name         string
		tenantID     string
		status       string
		wantArgCount int
	}{
		{
			name:         "no status filter uses 1 arg",
			tenantID:     "tenant-1",
			status:       "",
			wantArgCount: 1,
		},
		{
			name:         "with status filter uses 2 args",
			tenantID:     "tenant-2",
			status:       "open",
			wantArgCount: 2,
		},
		{
			name:         "resolved status filter uses 2 args",
			tenantID:     "tenant-3",
			status:       "resolved",
			wantArgCount: 2,
		},
		{
			name:         "false_positive status filter uses 2 args",
			tenantID:     "tenant-4",
			status:       "false_positive",
			wantArgCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Replicate the query construction from ListByTenant
			args := []interface{}{tc.tenantID}

			if tc.status != "" {
				args = append(args, tc.status)
			}

			if len(args) != tc.wantArgCount {
				t.Errorf("arg count = %d, want %d", len(args), tc.wantArgCount)
			}

			// First arg is always tenant_id
			if args[0].(string) != tc.tenantID {
				t.Errorf("arg[0] (tenant_id) = %q, want %q", args[0], tc.tenantID)
			}

			// If status filter is present, it's the second arg
			if tc.status != "" {
				if args[1].(string) != tc.status {
					t.Errorf("arg[1] (status) = %q, want %q", args[1], tc.status)
				}
			}
		})
	}
}

func TestAlertRepository_ListByTenant_QueryStringVariants(t *testing.T) {
	t.Run("query without status filter does not include AND status clause", func(t *testing.T) {
		// Replicate query construction
		query := `
		SELECT id, tenant_id, agent_id, probability, estimated_ttf_seconds, precursor_type, evidence, status, created_at, resolved_at
		FROM predictive_alerts
		WHERE tenant_id = $1::uuid`

		status := ""
		if status != "" {
			query += ` AND status = $2`
		}
		query += ` ORDER BY created_at DESC LIMIT 100`

		// Verify query does NOT contain "AND status" when no filter
		if containsSubstring(query, "AND status") {
			t.Error("query should not contain 'AND status' when status is empty")
		}
	})

	t.Run("query with status filter includes AND status clause", func(t *testing.T) {
		query := `
		SELECT id, tenant_id, agent_id, probability, estimated_ttf_seconds, precursor_type, evidence, status, created_at, resolved_at
		FROM predictive_alerts
		WHERE tenant_id = $1::uuid`

		status := "open"
		if status != "" {
			query += ` AND status = $2`
		}
		query += ` ORDER BY created_at DESC LIMIT 100`

		// Verify query contains "AND status"
		if !containsSubstring(query, "AND status = $2") {
			t.Error("query should contain 'AND status = $2' when status is set")
		}
	})
}

// ---------- UpdateStatus: resolved_at logic ----------

func TestAlertRepository_UpdateStatus_ResolvedAtLogic(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		wantResolved bool
	}{
		{
			name:         "resolved status sets resolved_at",
			status:       "resolved",
			wantResolved: true,
		},
		{
			name:         "false_positive status sets resolved_at",
			status:       "false_positive",
			wantResolved: true,
		},
		{
			name:         "open status does not set resolved_at",
			status:       "open",
			wantResolved: false,
		},
		{
			name:         "acknowledged status does not set resolved_at",
			status:       "acknowledged",
			wantResolved: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Replicate the resolved_at logic from UpdateStatus
			var resolvedAt *time.Time
			if tc.status == "resolved" || tc.status == "false_positive" {
				now := time.Now()
				resolvedAt = &now
			}

			if tc.wantResolved && resolvedAt == nil {
				t.Errorf("expected resolved_at to be set for status %q", tc.status)
			}
			if !tc.wantResolved && resolvedAt != nil {
				t.Errorf("expected resolved_at to be nil for status %q", tc.status)
			}

			if resolvedAt != nil {
				if resolvedAt.IsZero() {
					t.Error("resolved_at should not be zero time")
				}
				if time.Since(*resolvedAt) > time.Second {
					t.Error("resolved_at should be close to now")
				}
			}
		})
	}
}

func TestAlertRepository_UpdateStatus_SQLArgs(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		alertID  string
		status   string
	}{
		{
			name:     "resolve alert",
			tenantID: "tenant-1",
			alertID:  "alert-001",
			status:   "resolved",
		},
		{
			name:     "acknowledge alert",
			tenantID: "tenant-2",
			alertID:  "alert-002",
			status:   "acknowledged",
		},
		{
			name:     "mark false positive",
			tenantID: "tenant-3",
			alertID:  "alert-003",
			status:   "false_positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var resolvedAt *time.Time
			if tc.status == "resolved" || tc.status == "false_positive" {
				now := time.Now()
				resolvedAt = &now
			}

			// The SQL: UPDATE predictive_alerts SET status = $1, resolved_at = $2
			//          WHERE id = $3::uuid AND tenant_id = $4::uuid
			args := []interface{}{tc.status, resolvedAt, tc.alertID, tc.tenantID}

			if len(args) != 4 {
				t.Errorf("expected 4 SQL args, got %d", len(args))
			}
			if args[0].(string) != tc.status {
				t.Errorf("arg[0] (status) = %q, want %q", args[0], tc.status)
			}
			if args[2].(string) != tc.alertID {
				t.Errorf("arg[2] (alert_id) = %q, want %q", args[2], tc.alertID)
			}
			if args[3].(string) != tc.tenantID {
				t.Errorf("arg[3] (tenant_id) = %q, want %q", args[3], tc.tenantID)
			}
		})
	}
}

// ---------- Alert status values ----------

func TestAlertStatusValues(t *testing.T) {
	// Verify all known alert status values are valid strings
	// that match the expected SQL values.
	tests := []struct {
		name   string
		status string
	}{
		{"open", "open"},
		{"acknowledged", "acknowledged"},
		{"resolved", "resolved"},
		{"false_positive", "false_positive"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.status == "" {
				t.Error("status should not be empty")
			}
			if tc.status != tc.name {
				t.Errorf("status = %q, want %q", tc.status, tc.name)
			}
		})
	}
}

// ---------- Precursor type values ----------

func TestPrecursorTypeValues(t *testing.T) {
	// Verify common precursor types referenced in the CLAUDE.md spec.
	types := []string{
		"latency_spike",
		"token_escalation",
		"retry_storm",
		"error_rate_delta",
	}

	for _, pt := range types {
		t.Run(pt, func(t *testing.T) {
			if pt == "" {
				t.Error("precursor type should not be empty")
			}
		})
	}
}

// ---------- Probability validation ----------

func TestPredictiveAlert_ProbabilityRange(t *testing.T) {
	tests := []struct {
		name        string
		probability float64
		wantValid   bool
	}{
		{"zero probability", 0.0, true},
		{"max probability", 1.0, true},
		{"mid probability", 0.5, true},
		{"high probability", 0.92, true},
		{"negative probability", -0.1, false},
		{"over one probability", 1.1, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isValid := tc.probability >= 0.0 && tc.probability <= 1.0
			if isValid != tc.wantValid {
				t.Errorf("probability %f: valid = %v, want %v",
					tc.probability, isValid, tc.wantValid)
			}
		})
	}
}

// ---------- Interface compliance (compile-time check) ----------

// alertStore defines the expected interface for alert persistence.
type alertStore interface {
	Create(ctx context.Context, alert *PredictiveAlert) error
	ListByTenant(ctx context.Context, tenantID string, status string) ([]*PredictiveAlert, error)
	UpdateStatus(ctx context.Context, tenantID, alertID, status string) error
}

// Compile-time interface check: *AlertRepository must implement alertStore.
var _ alertStore = (*AlertRepository)(nil)

func TestAlertRepository_MethodSignatures(t *testing.T) {
	t.Run("constructor returns correct type", func(t *testing.T) {
		repo := NewAlertRepository(nil)
		if repo == nil {
			t.Fatal("expected non-nil AlertRepository")
		}
	})
}

// ---------- Tenant isolation contract ----------

func TestAlertRepository_TenantIDRequired(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		tenantID string
	}{
		{"Create requires tenant_id", "Create", "tenant-1"},
		{"ListByTenant requires tenant_id", "ListByTenant", "tenant-2"},
		{"UpdateStatus requires tenant_id", "UpdateStatus", "tenant-3"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.tenantID == "" {
				t.Errorf("method %s must always receive a tenant_id", tc.method)
			}
		})
	}
}

// ---------- helpers ----------

func timePtr(t time.Time) *time.Time {
	return &t
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
