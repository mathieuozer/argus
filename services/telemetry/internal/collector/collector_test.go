package collector

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestIngest(t *testing.T) {
	tests := []struct {
		name      string
		spans     []*Span
		wantCount int
	}{
		{
			name:      "ingest empty batch",
			spans:     []*Span{},
			wantCount: 0,
		},
		{
			name: "ingest single span",
			spans: []*Span{
				{
					SpanID:        "span-1",
					TraceID:       "trace-1",
					TenantID:      "tenant-1",
					AgentID:       "agent-1",
					TaskID:        "task-1",
					OperationName: "llm.call",
					StartedAt:     time.Now(),
					DurationMs:    150,
					Tier:          1,
				},
			},
			wantCount: 1,
		},
		{
			name: "ingest multiple spans",
			spans: []*Span{
				{SpanID: "span-1", TraceID: "trace-1", TenantID: "tenant-1", AgentID: "agent-1", TaskID: "task-1", OperationName: "llm.call", StartedAt: time.Now(), DurationMs: 100, Tier: 1},
				{SpanID: "span-2", TraceID: "trace-1", TenantID: "tenant-1", AgentID: "agent-1", TaskID: "task-1", OperationName: "tool.invoke", StartedAt: time.Now(), DurationMs: 200, Tier: 2},
				{SpanID: "span-3", TraceID: "trace-2", TenantID: "tenant-1", AgentID: "agent-2", TaskID: "task-2", OperationName: "llm.call", StartedAt: time.Now(), DurationMs: 300, Tier: 1},
			},
			wantCount: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := zap.NewNop()
			c := New(logger)

			count := c.Ingest(tc.spans)
			if count != tc.wantCount {
				t.Errorf("Ingest returned %d, want %d", count, tc.wantCount)
			}
		})
	}

	t.Run("accumulates spans across multiple ingests", func(t *testing.T) {
		logger := zap.NewNop()
		c := New(logger)

		c.Ingest([]*Span{
			{SpanID: "span-1", TenantID: "tenant-1", AgentID: "agent-1", TraceID: "trace-1"},
		})
		c.Ingest([]*Span{
			{SpanID: "span-2", TenantID: "tenant-1", AgentID: "agent-1", TraceID: "trace-1"},
			{SpanID: "span-3", TenantID: "tenant-1", AgentID: "agent-1", TraceID: "trace-1"},
		})

		result := c.Query("tenant-1", "", "", 0)
		if len(result) != 3 {
			t.Errorf("expected 3 accumulated spans, got %d", len(result))
		}
	})
}

func TestQuery(t *testing.T) {
	baseSpans := []*Span{
		{SpanID: "span-1", TraceID: "trace-1", TenantID: "tenant-1", AgentID: "agent-1", TaskID: "task-1"},
		{SpanID: "span-2", TraceID: "trace-1", TenantID: "tenant-1", AgentID: "agent-2", TaskID: "task-2"},
		{SpanID: "span-3", TraceID: "trace-2", TenantID: "tenant-1", AgentID: "agent-1", TaskID: "task-3"},
		{SpanID: "span-4", TraceID: "trace-3", TenantID: "tenant-2", AgentID: "agent-1", TaskID: "task-4"},
	}

	tests := []struct {
		name     string
		tenantID string
		agentID  string
		traceID  string
		limit    int
		wantLen  int
		wantIDs  []string
	}{
		{
			name:     "filter by tenant only",
			tenantID: "tenant-1",
			agentID:  "",
			traceID:  "",
			limit:    0,
			wantLen:  3,
		},
		{
			name:     "filter by tenant and agent",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			traceID:  "",
			limit:    0,
			wantLen:  2,
			wantIDs:  []string{"span-1", "span-3"},
		},
		{
			name:     "filter by tenant and trace",
			tenantID: "tenant-1",
			agentID:  "",
			traceID:  "trace-1",
			limit:    0,
			wantLen:  2,
			wantIDs:  []string{"span-1", "span-2"},
		},
		{
			name:     "filter by tenant, agent, and trace",
			tenantID: "tenant-1",
			agentID:  "agent-1",
			traceID:  "trace-1",
			limit:    0,
			wantLen:  1,
			wantIDs:  []string{"span-1"},
		},
		{
			name:     "respects limit",
			tenantID: "tenant-1",
			agentID:  "",
			traceID:  "",
			limit:    2,
			wantLen:  2,
		},
		{
			name:     "no results for unknown tenant",
			tenantID: "nonexistent",
			agentID:  "",
			traceID:  "",
			limit:    0,
			wantLen:  0,
		},
		{
			name:     "tenant isolation - tenant-2 only sees its spans",
			tenantID: "tenant-2",
			agentID:  "",
			traceID:  "",
			limit:    0,
			wantLen:  1,
			wantIDs:  []string{"span-4"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := zap.NewNop()
			c := New(logger)
			c.Ingest(baseSpans)

			result := c.Query(tc.tenantID, tc.agentID, tc.traceID, tc.limit)
			if len(result) != tc.wantLen {
				t.Fatalf("Query returned %d spans, want %d", len(result), tc.wantLen)
			}

			if tc.wantIDs != nil {
				gotIDs := make(map[string]bool)
				for _, s := range result {
					gotIDs[s.SpanID] = true
				}
				for _, wantID := range tc.wantIDs {
					if !gotIDs[wantID] {
						t.Errorf("expected span %q in results, but not found", wantID)
					}
				}
			}
		})
	}
}
