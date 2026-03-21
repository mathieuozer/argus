package grpchandler

import (
	"context"

	telemetryv1 "github.com/argus-platform/argus/gen/go/telemetry"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/telemetry/internal/collector"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TelemetryHandler implements the TelemetryServiceServer gRPC interface.
type TelemetryHandler struct {
	telemetryv1.UnimplementedTelemetryServiceServer
	collector *collector.Collector
}

// NewTelemetryHandler creates a new gRPC handler for telemetry operations.
func NewTelemetryHandler(c *collector.Collector) *TelemetryHandler {
	return &TelemetryHandler{collector: c}
}

func (h *TelemetryHandler) IngestSpans(ctx context.Context, req *telemetryv1.IngestSpansRequest) (*telemetryv1.IngestSpansResponse, error) {
	if len(req.Spans) == 0 {
		return &telemetryv1.IngestSpansResponse{Accepted: 0}, nil
	}

	spans := make([]*collector.Span, len(req.Spans))
	for i, s := range req.Spans {
		tenantID := s.TenantId
		if tenantID == "" {
			tenantID, _ = tenancy.FromContext(ctx)
		}
		if tenantID == "" {
			return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
		}

		span := protoToSpan(s)
		span.TenantID = tenantID
		spans[i] = span
	}

	accepted := h.collector.Ingest(spans)
	return &telemetryv1.IngestSpansResponse{Accepted: int32(accepted)}, nil
}

func (h *TelemetryHandler) IngestMetrics(ctx context.Context, req *telemetryv1.IngestMetricsRequest) (*telemetryv1.IngestMetricsResponse, error) {
	// Metrics ingestion is accepted and counted; storage is handled at the HTTP layer
	// or via a dedicated metrics backend. The gRPC handler validates tenant context.
	if len(req.Metrics) == 0 {
		return &telemetryv1.IngestMetricsResponse{Accepted: 0}, nil
	}

	for _, m := range req.Metrics {
		tenantID := m.TenantId
		if tenantID == "" {
			tenantID, _ = tenancy.FromContext(ctx)
		}
		if tenantID == "" {
			return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
		}
	}

	return &telemetryv1.IngestMetricsResponse{Accepted: int32(len(req.Metrics))}, nil
}

func (h *TelemetryHandler) IngestEvents(ctx context.Context, req *telemetryv1.IngestEventsRequest) (*telemetryv1.IngestEventsResponse, error) {
	// Events ingestion is accepted and counted; storage is handled at the HTTP layer
	// or via a dedicated events backend. The gRPC handler validates tenant context.
	if len(req.Events) == 0 {
		return &telemetryv1.IngestEventsResponse{Accepted: 0}, nil
	}

	for _, e := range req.Events {
		tenantID := e.TenantId
		if tenantID == "" {
			tenantID, _ = tenancy.FromContext(ctx)
		}
		if tenantID == "" {
			return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
		}
	}

	return &telemetryv1.IngestEventsResponse{Accepted: int32(len(req.Events))}, nil
}

func (h *TelemetryHandler) QuerySpans(ctx context.Context, req *telemetryv1.QuerySpansRequest) (*telemetryv1.QuerySpansResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}
	if tenantID == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 100
	}

	spans := h.collector.Query(tenantID, req.AgentId, req.TraceId, limit)
	protoSpans := make([]*telemetryv1.Span, len(spans))
	for i, s := range spans {
		protoSpans[i] = spanToProto(s)
	}

	return &telemetryv1.QuerySpansResponse{Spans: protoSpans}, nil
}

func protoToSpan(s *telemetryv1.Span) *collector.Span {
	span := &collector.Span{
		SpanID:        s.SpanId,
		TraceID:       s.TraceId,
		TenantID:      s.TenantId,
		AgentID:       s.AgentId,
		TaskID:        s.TaskId,
		OperationName: s.OperationName,
		DurationMs:    s.DurationMs,
		Tier:          int(s.Tier),
		Attributes:    s.Attributes,
	}

	if s.StartedAt != nil {
		span.StartedAt = s.StartedAt.AsTime()
	}

	if s.ErrorCode != nil {
		ec := *s.ErrorCode
		span.ErrorCode = &ec
	}

	return span
}

func spanToProto(s *collector.Span) *telemetryv1.Span {
	ps := &telemetryv1.Span{
		SpanId:        s.SpanID,
		TraceId:       s.TraceID,
		TenantId:      s.TenantID,
		AgentId:       s.AgentID,
		TaskId:        s.TaskID,
		OperationName: s.OperationName,
		StartedAt:     timestamppb.New(s.StartedAt),
		DurationMs:    s.DurationMs,
		Tier:          telemetryv1.DataTier(s.Tier),
		Attributes:    s.Attributes,
	}

	if s.ErrorCode != nil {
		ps.ErrorCode = s.ErrorCode
	}

	return ps
}
