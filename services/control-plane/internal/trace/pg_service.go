package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5"
)

// PGService manages trace data in PostgreSQL.
type PGService struct {
	pool *database.Pool
}

// NewPGService creates a new PostgreSQL-backed trace service.
func NewPGService(pool *database.Pool) *PGService {
	return &PGService{pool: pool}
}

// IngestSpan adds a span to the trace store.
func (s *PGService) IngestSpan(ctx context.Context, span *Span) error {
	return database.ExecWithTenant(ctx, s.pool.Pool, span.TenantID, `
		INSERT INTO trace_spans (span_id, trace_id, parent_span_id, tenant_id, agent_id, task_id, operation_name, started_at, duration_ms, attributes, error_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id, span_id) DO NOTHING`,
		span.SpanID, span.TraceID, span.ParentSpanID, span.TenantID, span.AgentID, span.TaskID,
		span.OperationName, span.StartedAt, span.DurationMs, span.Attributes, span.ErrorCode)
}

// ListTraces returns trace summaries for a tenant, optionally filtered by agent ID.
func (s *PGService) ListTraces(ctx context.Context, tenantID, agentID string, limit int) ([]TraceSummary, error) {
	query := `
		SELECT trace_id,
			MIN(operation_name) FILTER (WHERE parent_span_id = '' OR parent_span_id IS NULL) AS root_op,
			MIN(agent_id) FILTER (WHERE parent_span_id = '' OR parent_span_id IS NULL) AS root_agent,
			COUNT(*) AS total_spans,
			MAX(duration_ms) AS total_duration_ms,
			BOOL_OR(error_code IS NOT NULL) AS has_errors,
			MIN(started_at) AS started_at
		FROM trace_spans
		WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if agentID != "" {
		query += fmt.Sprintf(` AND trace_id IN (SELECT DISTINCT trace_id FROM trace_spans WHERE tenant_id = $1 AND agent_id = $%d)`, argIdx)
		args = append(args, agentID)
		argIdx++
	}
	_ = argIdx

	query += ` GROUP BY trace_id ORDER BY MIN(started_at) DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, cleanup, err := database.QueryWithTenant(ctx, s.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var summaries []TraceSummary
	for rows.Next() {
		var ts TraceSummary
		var rootOp, rootAgent *string
		if err := rows.Scan(&ts.TraceID, &rootOp, &rootAgent, &ts.TotalSpans, &ts.TotalDurationMs, &ts.HasErrors, &ts.StartedAt); err != nil {
			return nil, err
		}
		if rootOp != nil {
			ts.RootOperation = *rootOp
		}
		if rootAgent != nil {
			ts.AgentID = *rootAgent
		}
		summaries = append(summaries, ts)
	}
	return summaries, rows.Err()
}

// GetTrace returns the full trace tree for a given trace ID.
func (s *PGService) GetTrace(ctx context.Context, tenantID, traceID string) (*TraceDetail, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, s.pool.Pool, tenantID, `
		SELECT span_id, trace_id, parent_span_id, tenant_id, agent_id, task_id, operation_name, started_at, duration_ms, attributes, error_code
		FROM trace_spans
		WHERE tenant_id = $1 AND trace_id = $2
		ORDER BY started_at ASC`, tenantID, traceID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var spans []*Span
	for rows.Next() {
		sp, err := scanSpan(rows)
		if err != nil {
			return nil, err
		}
		spans = append(spans, sp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(spans) == 0 {
		return nil, nil
	}

	return buildTree(traceID, spans), nil
}

// GetFlameGraph returns a flame graph representation for a trace.
func (s *PGService) GetFlameGraph(ctx context.Context, tenantID, traceID string) (*FlameGraphNode, error) {
	detail, err := s.GetTrace(ctx, tenantID, traceID)
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.RootSpan == nil {
		return nil, nil
	}
	return spanToFlame(detail.RootSpan), nil
}

func buildTree(traceID string, spans []*Span) *TraceDetail {
	nodeMap := make(map[string]*SpanNode)
	for _, sp := range spans {
		nodeMap[sp.SpanID] = &SpanNode{
			SpanID:        sp.SpanID,
			OperationName: sp.OperationName,
			DurationMs:    sp.DurationMs,
			StartedAt:     sp.StartedAt,
			AgentID:       sp.AgentID,
			Attributes:    sp.Attributes,
			ErrorCode:     sp.ErrorCode,
			Children:      make([]*SpanNode, 0),
		}
	}

	var root *SpanNode
	hasErrors := false

	for _, sp := range spans {
		if sp.ErrorCode != nil {
			hasErrors = true
		}
		node := nodeMap[sp.SpanID]
		if sp.ParentSpanID == "" {
			root = node
		} else if parent, ok := nodeMap[sp.ParentSpanID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	if root == nil && len(spans) > 0 {
		root = nodeMap[spans[0].SpanID]
	}

	var totalDuration int64
	if root != nil {
		totalDuration = root.DurationMs
	}

	return &TraceDetail{
		TraceID:         traceID,
		RootSpan:        root,
		TotalSpans:      len(spans),
		TotalDurationMs: totalDuration,
		HasErrors:       hasErrors,
	}
}

func scanSpan(rows pgx.Rows) (*Span, error) {
	var sp Span
	var parentSpanID *string
	var attributes map[string]string
	var errorCode *string
	var startedAt time.Time

	if err := rows.Scan(&sp.SpanID, &sp.TraceID, &parentSpanID, &sp.TenantID, &sp.AgentID, &sp.TaskID,
		&sp.OperationName, &startedAt, &sp.DurationMs, &attributes, &errorCode); err != nil {
		return nil, err
	}

	sp.StartedAt = startedAt
	if parentSpanID != nil {
		sp.ParentSpanID = *parentSpanID
	}
	if attributes != nil {
		sp.Attributes = attributes
	}
	sp.ErrorCode = errorCode

	return &sp, nil
}
