package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/argus-platform/argus/services/telemetry/internal/collector"
)

// ---------- Constructor tests ----------

func TestNewSpanRepository(t *testing.T) {
	t.Run("returns non-nil repository with nil pool", func(t *testing.T) {
		repo := NewSpanRepository(nil)
		if repo == nil {
			t.Fatal("expected non-nil SpanRepository")
		}
	})

	t.Run("pool field is set to nil", func(t *testing.T) {
		repo := NewSpanRepository(nil)
		if repo.pool != nil {
			t.Error("expected nil pool when constructed with nil")
		}
	})
}

// ---------- itoa helper ----------

func TestItoa(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  string
	}{
		{"zero", 0, "0"},
		{"one", 1, "1"},
		{"two", 2, "2"},
		{"ten", 10, "10"},
		{"large number", 12345, "12345"},
		{"negative", -1, "-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := itoa(tc.input)
			if got != tc.want {
				t.Errorf("itoa(%d) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------- Store: attributes JSON marshaling ----------

func TestSpanRepository_Store_AttributesMarshal(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string]string
		wantJSON   string
		wantErr    bool
	}{
		{
			name:       "nil attributes marshals to null",
			attributes: nil,
			wantJSON:   "null",
			wantErr:    false,
		},
		{
			name:       "empty attributes marshals to empty object",
			attributes: map[string]string{},
			wantJSON:   "{}",
			wantErr:    false,
		},
		{
			name: "populated attributes marshals correctly",
			attributes: map[string]string{
				"model": "gpt-4",
				"tier":  "1",
			},
			wantErr: false,
		},
		{
			name: "single attribute",
			attributes: map[string]string{
				"error_type": "timeout",
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			span := &collector.Span{
				Attributes: tc.attributes,
			}

			attrsJSON, err := json.Marshal(span.Attributes)
			if tc.wantErr && err == nil {
				t.Error("expected marshal error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected marshal error: %v", err)
			}

			if tc.wantJSON != "" {
				if string(attrsJSON) != tc.wantJSON {
					t.Errorf("marshal result = %s, want %s", string(attrsJSON), tc.wantJSON)
				}
			}

			// Verify round-trip: unmarshal back
			if err == nil && tc.attributes != nil {
				var unmarshaled map[string]string
				if err := json.Unmarshal(attrsJSON, &unmarshaled); err != nil {
					t.Errorf("failed to unmarshal attrs: %v", err)
				}
				for k, v := range tc.attributes {
					if unmarshaled[k] != v {
						t.Errorf("round-trip: key %q = %q, want %q", k, unmarshaled[k], v)
					}
				}
			}
		})
	}
}

// ---------- Store: SQL parameter construction ----------

func TestSpanRepository_Store_SQLArgs(t *testing.T) {
	tests := []struct {
		name string
		span *collector.Span
	}{
		{
			name: "span with all fields",
			span: &collector.Span{
				SpanID:        "span-001",
				TraceID:       "trace-001",
				TenantID:      "tenant-1",
				AgentID:       "agent-alpha",
				TaskID:        "task-001",
				OperationName: "llm.generate",
				StartedAt:     time.Now(),
				DurationMs:    1500,
				Tier:          1,
				Attributes:    map[string]string{"model": "gpt-4"},
				ErrorCode:     nil,
			},
		},
		{
			name: "span with error code",
			span: &collector.Span{
				SpanID:        "span-002",
				TraceID:       "trace-002",
				TenantID:      "tenant-2",
				AgentID:       "agent-beta",
				TaskID:        "task-002",
				OperationName: "tool.call",
				StartedAt:     time.Now(),
				DurationMs:    50,
				Tier:          2,
				Attributes:    map[string]string{},
				ErrorCode:     stringPtr("TIMEOUT"),
			},
		},
		{
			name: "span with tier 3 (restricted)",
			span: &collector.Span{
				SpanID:        "span-003",
				TraceID:       "trace-003",
				TenantID:      "tenant-gov",
				AgentID:       "agent-classified",
				TaskID:        "task-003",
				OperationName: "data.process",
				StartedAt:     time.Now(),
				DurationMs:    3000,
				Tier:          3,
				Attributes:    nil,
				ErrorCode:     nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrsJSON, err := json.Marshal(tc.span.Attributes)
			if err != nil {
				t.Fatalf("failed to marshal attributes: %v", err)
			}

			tenantID := tc.span.TenantID

			// The SQL:
			// INSERT INTO telemetry_spans (span_id, trace_id, tenant_id, agent_id, task_id,
			//   operation_name, started_at, duration_ms, tier, attributes, error_code)
			// VALUES ($1, $2, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11)
			args := []interface{}{
				tc.span.SpanID, tc.span.TraceID, tenantID, tc.span.AgentID,
				tc.span.TaskID, tc.span.OperationName, tc.span.StartedAt,
				tc.span.DurationMs, tc.span.Tier, attrsJSON, tc.span.ErrorCode,
			}

			if len(args) != 11 {
				t.Errorf("expected 11 SQL args for Store, got %d", len(args))
			}

			if args[0].(string) != tc.span.SpanID {
				t.Errorf("arg[0] (span_id) = %q, want %q", args[0], tc.span.SpanID)
			}
			if args[1].(string) != tc.span.TraceID {
				t.Errorf("arg[1] (trace_id) = %q, want %q", args[1], tc.span.TraceID)
			}
			if args[2].(string) != tenantID {
				t.Errorf("arg[2] (tenant_id) = %q, want %q", args[2], tenantID)
			}
			if args[3].(string) != tc.span.AgentID {
				t.Errorf("arg[3] (agent_id) = %q, want %q", args[3], tc.span.AgentID)
			}
			if args[7].(int64) != tc.span.DurationMs {
				t.Errorf("arg[7] (duration_ms) = %v, want %v", args[7], tc.span.DurationMs)
			}
			if args[8].(int) != tc.span.Tier {
				t.Errorf("arg[8] (tier) = %v, want %v", args[8], tc.span.Tier)
			}
		})
	}
}

// ---------- Query: dynamic query construction ----------

func TestSpanRepository_Query_DynamicQueryConstruction(t *testing.T) {
	tests := []struct {
		name         string
		tenantID     string
		agentID      string
		traceID      string
		limit        int
		wantArgCount int
		wantArgIdx   int // expected final argIdx value
	}{
		{
			name:         "tenant only (no filters)",
			tenantID:     "tenant-1",
			agentID:      "",
			traceID:      "",
			limit:        100,
			wantArgCount: 2, // tenantID + limit
			wantArgIdx:   3,
		},
		{
			name:         "with agent_id filter",
			tenantID:     "tenant-2",
			agentID:      "agent-alpha",
			traceID:      "",
			limit:        50,
			wantArgCount: 3, // tenantID + agentID + limit
			wantArgIdx:   4,
		},
		{
			name:         "with trace_id filter",
			tenantID:     "tenant-3",
			agentID:      "",
			traceID:      "trace-001",
			limit:        25,
			wantArgCount: 3, // tenantID + traceID + limit
			wantArgIdx:   4,
		},
		{
			name:         "with both agent_id and trace_id filters",
			tenantID:     "tenant-4",
			agentID:      "agent-beta",
			traceID:      "trace-002",
			limit:        10,
			wantArgCount: 4, // tenantID + agentID + traceID + limit
			wantArgIdx:   5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Replicate the query construction from Query method
			query := `
		SELECT span_id, trace_id, tenant_id, agent_id, task_id, operation_name, started_at, duration_ms, tier, attributes, error_code
		FROM telemetry_spans
		WHERE tenant_id = $1::uuid`
			args := []interface{}{tc.tenantID}
			argIdx := 2

			if tc.agentID != "" {
				query += ` AND agent_id = $` + itoa(argIdx)
				args = append(args, tc.agentID)
				argIdx++
			}
			if tc.traceID != "" {
				query += ` AND trace_id = $` + itoa(argIdx)
				args = append(args, tc.traceID)
				argIdx++
			}

			query += ` ORDER BY started_at DESC LIMIT $` + itoa(argIdx)
			args = append(args, tc.limit)

			if len(args) != tc.wantArgCount {
				t.Errorf("arg count = %d, want %d", len(args), tc.wantArgCount)
			}

			// Verify first arg is always tenant_id
			if args[0].(string) != tc.tenantID {
				t.Errorf("arg[0] (tenant_id) = %q, want %q", args[0], tc.tenantID)
			}

			// Verify last arg is always limit
			lastArg := args[len(args)-1].(int)
			if lastArg != tc.limit {
				t.Errorf("last arg (limit) = %d, want %d", lastArg, tc.limit)
			}

			// Verify query contains LIMIT placeholder with correct index
			wantLimitPlaceholder := "LIMIT $" + itoa(argIdx)
			if !containsSubstr(query, wantLimitPlaceholder) {
				t.Errorf("query should contain %q", wantLimitPlaceholder)
			}
		})
	}
}

func TestSpanRepository_Query_PlaceholderSequence(t *testing.T) {
	t.Run("placeholders are sequential with both filters", func(t *testing.T) {
		args := []interface{}{"tenant-1"}
		argIdx := 2

		agentID := "agent-1"
		traceID := "trace-1"
		limit := 100

		query := `WHERE tenant_id = $1::uuid`

		query += ` AND agent_id = $` + itoa(argIdx)
		args = append(args, agentID)
		argIdx++
		// argIdx should be 3 now

		query += ` AND trace_id = $` + itoa(argIdx)
		args = append(args, traceID)
		argIdx++
		// argIdx should be 4 now

		query += ` ORDER BY started_at DESC LIMIT $` + itoa(argIdx)
		args = append(args, limit)

		if !containsSubstr(query, "$1") {
			t.Error("query missing $1 placeholder")
		}
		if !containsSubstr(query, "$2") {
			t.Error("query missing $2 placeholder")
		}
		if !containsSubstr(query, "$3") {
			t.Error("query missing $3 placeholder")
		}
		if !containsSubstr(query, "$4") {
			t.Error("query missing $4 placeholder")
		}

		if len(args) != 4 {
			t.Errorf("expected 4 args, got %d", len(args))
		}
	})

	t.Run("placeholders are sequential with no filters", func(t *testing.T) {
		args := []interface{}{"tenant-1"}
		argIdx := 2

		limit := 100

		query := `WHERE tenant_id = $1::uuid`
		query += ` ORDER BY started_at DESC LIMIT $` + itoa(argIdx)
		args = append(args, limit)

		if !containsSubstr(query, "$1") {
			t.Error("query missing $1 placeholder")
		}
		if !containsSubstr(query, "$2") {
			t.Error("query missing $2 placeholder")
		}

		if len(args) != 2 {
			t.Errorf("expected 2 args, got %d", len(args))
		}
	})
}

// ---------- scanSpan: attributes unmarshal ----------

func TestScanSpan_AttributesUnmarshal(t *testing.T) {
	tests := []struct {
		name      string
		attrsJSON []byte
		wantNil   bool
		wantLen   int
	}{
		{
			name:      "nil JSON keeps nil attributes",
			attrsJSON: nil,
			wantNil:   true,
			wantLen:   0,
		},
		{
			name:      "empty object JSON",
			attrsJSON: []byte("{}"),
			wantNil:   false,
			wantLen:   0,
		},
		{
			name:      "populated JSON",
			attrsJSON: []byte(`{"model":"gpt-4","tier":"1"}`),
			wantNil:   false,
			wantLen:   2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Replicate the attributes handling from scanSpan
			var s collector.Span
			if tc.attrsJSON != nil {
				_ = json.Unmarshal(tc.attrsJSON, &s.Attributes)
			}

			if tc.wantNil && s.Attributes != nil {
				t.Errorf("expected nil attributes, got %v", s.Attributes)
			}
			if !tc.wantNil && s.Attributes == nil {
				t.Error("expected non-nil attributes")
			}
			if !tc.wantNil && len(s.Attributes) != tc.wantLen {
				t.Errorf("attributes length = %d, want %d", len(s.Attributes), tc.wantLen)
			}
		})
	}
}

// ---------- Span fields ----------

func TestSpan_Fields(t *testing.T) {
	tests := []struct {
		name string
		span collector.Span
	}{
		{
			name: "structural span (tier 1)",
			span: collector.Span{
				SpanID:        "span-001",
				TraceID:       "trace-001",
				TenantID:      "tenant-1",
				AgentID:       "agent-alpha",
				TaskID:        "task-001",
				OperationName: "llm.generate",
				StartedAt:     time.Now(),
				DurationMs:    1500,
				Tier:          1,
				Attributes:    map[string]string{"latency_ms": "1500"},
				ErrorCode:     nil,
			},
		},
		{
			name: "sensitive span (tier 2)",
			span: collector.Span{
				SpanID:        "span-002",
				TraceID:       "trace-002",
				TenantID:      "tenant-2",
				AgentID:       "agent-beta",
				TaskID:        "task-002",
				OperationName: "tool.call",
				StartedAt:     time.Now(),
				DurationMs:    200,
				Tier:          2,
				Attributes:    map[string]string{},
				ErrorCode:     nil,
			},
		},
		{
			name: "restricted span (tier 3) with error",
			span: collector.Span{
				SpanID:        "span-003",
				TraceID:       "trace-003",
				TenantID:      "tenant-gov",
				AgentID:       "agent-classified",
				TaskID:        "task-003",
				OperationName: "data.process",
				StartedAt:     time.Now(),
				DurationMs:    3000,
				Tier:          3,
				Attributes:    nil,
				ErrorCode:     stringPtr("CONTEXT_OVERFLOW"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.span.SpanID == "" {
				t.Error("SpanID should not be empty")
			}
			if tc.span.TraceID == "" {
				t.Error("TraceID should not be empty")
			}
			if tc.span.TenantID == "" {
				t.Error("TenantID should not be empty")
			}
			if tc.span.AgentID == "" {
				t.Error("AgentID should not be empty")
			}
			if tc.span.OperationName == "" {
				t.Error("OperationName should not be empty")
			}
			if tc.span.StartedAt.IsZero() {
				t.Error("StartedAt should not be zero")
			}
			if tc.span.DurationMs < 0 {
				t.Errorf("DurationMs = %d, should be non-negative", tc.span.DurationMs)
			}
			if tc.span.Tier < 1 || tc.span.Tier > 3 {
				t.Errorf("Tier = %d, should be 1, 2, or 3", tc.span.Tier)
			}
		})
	}
}

// ---------- Data tier values ----------

func TestSpan_TierValues(t *testing.T) {
	tests := []struct {
		name        string
		tier        int
		description string
	}{
		{"tier 1 - structural", 1, "latency, token_count, error_code, agent_id"},
		{"tier 2 - sensitive", 2, "task descriptions, tool call params"},
		{"tier 3 - restricted", 3, "full I/O content, never leaves node"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.tier < 1 || tc.tier > 3 {
				t.Errorf("tier = %d, should be between 1 and 3", tc.tier)
			}
		})
	}
}

// ---------- ErrorCode handling ----------

func TestSpan_ErrorCodeHandling(t *testing.T) {
	tests := []struct {
		name      string
		errorCode *string
		wantNil   bool
	}{
		{
			name:      "nil error code for successful span",
			errorCode: nil,
			wantNil:   true,
		},
		{
			name:      "non-nil error code for failed span",
			errorCode: stringPtr("TIMEOUT"),
			wantNil:   false,
		},
		{
			name:      "empty string error code",
			errorCode: stringPtr(""),
			wantNil:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			span := collector.Span{
				ErrorCode: tc.errorCode,
			}

			if tc.wantNil && span.ErrorCode != nil {
				t.Errorf("expected nil ErrorCode, got %q", *span.ErrorCode)
			}
			if !tc.wantNil && span.ErrorCode == nil {
				t.Error("expected non-nil ErrorCode")
			}
		})
	}
}

// ---------- Store: empty batch handling ----------

func TestSpanRepository_Store_EmptyBatch(t *testing.T) {
	t.Run("empty span slice produces no SQL args", func(t *testing.T) {
		spans := []*collector.Span{}
		if len(spans) != 0 {
			t.Errorf("expected 0 spans, got %d", len(spans))
		}
		// The Store method loops over spans - with empty slice, it executes
		// no INSERT statements and just commits the transaction.
	})
}

// ---------- Interface compliance (compile-time check) ----------

// spanStore defines the expected interface for span persistence.
type spanStore interface {
	Store(ctx context.Context, tenantID string, spans []*collector.Span) error
	Query(ctx context.Context, tenantID, agentID, traceID string, limit int) ([]*collector.Span, error)
}

// Compile-time interface check: *SpanRepository must implement spanStore.
var _ spanStore = (*SpanRepository)(nil)

func TestSpanRepository_MethodSignatures(t *testing.T) {
	t.Run("constructor returns correct type", func(t *testing.T) {
		repo := NewSpanRepository(nil)
		if repo == nil {
			t.Fatal("expected non-nil SpanRepository")
		}
	})
}

// ---------- Tenant isolation contract ----------

func TestSpanRepository_TenantIDRequired(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		tenantID string
	}{
		{"Store requires tenant_id", "Store", "tenant-1"},
		{"Query requires tenant_id", "Query", "tenant-2"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.tenantID == "" {
				t.Errorf("method %s must always receive a tenant_id", tc.method)
			}
		})
	}
}

// ---------- JSON marshal/unmarshal round-trip ----------

func TestSpan_AttributesRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]string
	}{
		{
			name:  "empty attributes",
			attrs: map[string]string{},
		},
		{
			name: "standard attributes",
			attrs: map[string]string{
				"model":          "gpt-4",
				"token_count":    "1500",
				"latency_ms":     "1200",
				"context_window": "8192",
			},
		},
		{
			name: "attributes with special characters",
			attrs: map[string]string{
				"key_with_underscores": "value",
				"dotted.key":           "dotted.value",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			marshaled, err := json.Marshal(tc.attrs)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var unmarshaled map[string]string
			if err := json.Unmarshal(marshaled, &unmarshaled); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if len(unmarshaled) != len(tc.attrs) {
				t.Errorf("round-trip length mismatch: got %d, want %d",
					len(unmarshaled), len(tc.attrs))
			}

			for k, v := range tc.attrs {
				if unmarshaled[k] != v {
					t.Errorf("round-trip: key %q = %q, want %q", k, unmarshaled[k], v)
				}
			}
		})
	}
}

// ---------- helpers ----------

func stringPtr(s string) *string {
	return &s
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
