package grpchandler

import (
	"context"
	"testing"
	"time"

	telemetryv1 "github.com/argus-platform/argus/gen/go/telemetry"
	"github.com/argus-platform/argus/services/telemetry/internal/collector"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestIngestSpans(t *testing.T) {
	tests := []struct {
		name         string
		req          *telemetryv1.IngestSpansRequest
		wantAccepted int32
		wantCode     codes.Code
	}{
		{
			name:         "empty batch",
			req:          &telemetryv1.IngestSpansRequest{Spans: []*telemetryv1.Span{}},
			wantAccepted: 0,
		},
		{
			name: "single span with tenant",
			req: &telemetryv1.IngestSpansRequest{
				Spans: []*telemetryv1.Span{
					{
						SpanId:        "span-1",
						TraceId:       "trace-1",
						TenantId:      "tenant-1",
						AgentId:       "agent-1",
						TaskId:        "task-1",
						OperationName: "llm.call",
						StartedAt:     timestamppb.New(time.Now()),
						DurationMs:    150,
						Tier:          telemetryv1.DataTier_DATA_TIER_STRUCTURAL,
					},
				},
			},
			wantAccepted: 1,
		},
		{
			name: "multiple spans",
			req: &telemetryv1.IngestSpansRequest{
				Spans: []*telemetryv1.Span{
					{SpanId: "span-1", TenantId: "tenant-1", AgentId: "agent-1", OperationName: "llm.call"},
					{SpanId: "span-2", TenantId: "tenant-1", AgentId: "agent-2", OperationName: "tool.invoke"},
					{SpanId: "span-3", TenantId: "tenant-1", AgentId: "agent-1", OperationName: "llm.embed"},
				},
			},
			wantAccepted: 3,
		},
		{
			name: "missing tenant_id",
			req: &telemetryv1.IngestSpansRequest{
				Spans: []*telemetryv1.Span{
					{SpanId: "span-1", AgentId: "agent-1", OperationName: "llm.call"},
				},
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := collector.New(zap.NewNop())
			h := NewTelemetryHandler(c)

			resp, err := h.IngestSpans(context.Background(), tc.req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Accepted != tc.wantAccepted {
				t.Errorf("expected %d accepted, got %d", tc.wantAccepted, resp.Accepted)
			}
		})
	}
}

func TestIngestMetrics(t *testing.T) {
	tests := []struct {
		name         string
		req          *telemetryv1.IngestMetricsRequest
		wantAccepted int32
		wantCode     codes.Code
	}{
		{
			name:         "empty batch",
			req:          &telemetryv1.IngestMetricsRequest{Metrics: []*telemetryv1.Metric{}},
			wantAccepted: 0,
		},
		{
			name: "single metric",
			req: &telemetryv1.IngestMetricsRequest{
				Metrics: []*telemetryv1.Metric{
					{TenantId: "tenant-1", AgentId: "agent-1", Name: "latency_ms", Value: 42.5},
				},
			},
			wantAccepted: 1,
		},
		{
			name: "missing tenant_id",
			req: &telemetryv1.IngestMetricsRequest{
				Metrics: []*telemetryv1.Metric{
					{AgentId: "agent-1", Name: "latency_ms", Value: 42.5},
				},
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := collector.New(zap.NewNop())
			h := NewTelemetryHandler(c)

			resp, err := h.IngestMetrics(context.Background(), tc.req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Accepted != tc.wantAccepted {
				t.Errorf("expected %d accepted, got %d", tc.wantAccepted, resp.Accepted)
			}
		})
	}
}

func TestIngestEvents(t *testing.T) {
	tests := []struct {
		name         string
		req          *telemetryv1.IngestEventsRequest
		wantAccepted int32
		wantCode     codes.Code
	}{
		{
			name:         "empty batch",
			req:          &telemetryv1.IngestEventsRequest{Events: []*telemetryv1.Event{}},
			wantAccepted: 0,
		},
		{
			name: "single event",
			req: &telemetryv1.IngestEventsRequest{
				Events: []*telemetryv1.Event{
					{TenantId: "tenant-1", AgentId: "agent-1", EventType: "tool.call", Payload: "{}"},
				},
			},
			wantAccepted: 1,
		},
		{
			name: "missing tenant_id",
			req: &telemetryv1.IngestEventsRequest{
				Events: []*telemetryv1.Event{
					{AgentId: "agent-1", EventType: "tool.call"},
				},
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := collector.New(zap.NewNop())
			h := NewTelemetryHandler(c)

			resp, err := h.IngestEvents(context.Background(), tc.req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Accepted != tc.wantAccepted {
				t.Errorf("expected %d accepted, got %d", tc.wantAccepted, resp.Accepted)
			}
		})
	}
}

func TestQuerySpans(t *testing.T) {
	tests := []struct {
		name     string
		setup    []*collector.Span
		req      *telemetryv1.QuerySpansRequest
		wantLen  int
		wantCode codes.Code
	}{
		{
			name: "query by tenant",
			setup: []*collector.Span{
				{SpanID: "span-1", TenantID: "tenant-1", AgentID: "agent-1", TraceID: "trace-1"},
				{SpanID: "span-2", TenantID: "tenant-1", AgentID: "agent-2", TraceID: "trace-1"},
				{SpanID: "span-3", TenantID: "tenant-2", AgentID: "agent-1", TraceID: "trace-2"},
			},
			req:     &telemetryv1.QuerySpansRequest{TenantId: "tenant-1"},
			wantLen: 2,
		},
		{
			name: "query by tenant and agent",
			setup: []*collector.Span{
				{SpanID: "span-1", TenantID: "tenant-1", AgentID: "agent-1", TraceID: "trace-1"},
				{SpanID: "span-2", TenantID: "tenant-1", AgentID: "agent-2", TraceID: "trace-1"},
			},
			req:     &telemetryv1.QuerySpansRequest{TenantId: "tenant-1", AgentId: "agent-1"},
			wantLen: 1,
		},
		{
			name: "query with limit",
			setup: []*collector.Span{
				{SpanID: "span-1", TenantID: "tenant-1", AgentID: "agent-1", TraceID: "trace-1"},
				{SpanID: "span-2", TenantID: "tenant-1", AgentID: "agent-2", TraceID: "trace-1"},
				{SpanID: "span-3", TenantID: "tenant-1", AgentID: "agent-1", TraceID: "trace-2"},
			},
			req:     &telemetryv1.QuerySpansRequest{TenantId: "tenant-1", Limit: 2},
			wantLen: 2,
		},
		{
			name:     "missing tenant_id",
			req:      &telemetryv1.QuerySpansRequest{},
			wantCode: codes.InvalidArgument,
		},
		{
			name:    "tenant isolation",
			setup: []*collector.Span{
				{SpanID: "span-1", TenantID: "tenant-1", AgentID: "agent-1", TraceID: "trace-1"},
			},
			req:     &telemetryv1.QuerySpansRequest{TenantId: "tenant-2"},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := collector.New(zap.NewNop())
			if tc.setup != nil {
				c.Ingest(tc.setup)
			}
			h := NewTelemetryHandler(c)

			resp, err := h.QuerySpans(context.Background(), tc.req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(resp.Spans) != tc.wantLen {
				t.Errorf("expected %d spans, got %d", tc.wantLen, len(resp.Spans))
			}
		})
	}
}

func TestSpanConversion(t *testing.T) {
	t.Run("round-trip proto to span to proto", func(t *testing.T) {
		now := time.Now().Truncate(time.Microsecond) // protobuf timestamps have microsecond precision
		errorCode := "ERR_TIMEOUT"
		original := &telemetryv1.Span{
			SpanId:        "span-1",
			TraceId:       "trace-1",
			TenantId:      "tenant-1",
			AgentId:       "agent-1",
			TaskId:        "task-1",
			OperationName: "llm.call",
			StartedAt:     timestamppb.New(now),
			DurationMs:    150,
			Tier:          telemetryv1.DataTier_DATA_TIER_SENSITIVE,
			Attributes:    map[string]string{"model": "gpt-4"},
			ErrorCode:     &errorCode,
		}

		span := protoToSpan(original)
		if span.SpanID != original.SpanId {
			t.Errorf("SpanID mismatch: %s != %s", span.SpanID, original.SpanId)
		}
		if span.Tier != 2 {
			t.Errorf("Tier mismatch: %d != 2", span.Tier)
		}
		if span.ErrorCode == nil || *span.ErrorCode != errorCode {
			t.Errorf("ErrorCode mismatch")
		}

		result := spanToProto(span)
		if result.SpanId != original.SpanId {
			t.Errorf("Round-trip SpanId mismatch: %s != %s", result.SpanId, original.SpanId)
		}
		if result.Tier != original.Tier {
			t.Errorf("Round-trip Tier mismatch: %v != %v", result.Tier, original.Tier)
		}
	})
}
