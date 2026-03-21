package repository

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/argus-platform/argus/services/telemetry/internal/collector"
	"github.com/jackc/pgx/v5"
)

// SpanRepository provides PostgreSQL-backed span persistence.
type SpanRepository struct {
	pool *database.Pool
}

// NewSpanRepository creates a new PostgreSQL-backed span repository.
func NewSpanRepository(pool *database.Pool) *SpanRepository {
	return &SpanRepository{pool: pool}
}

// Store persists a batch of spans.
func (r *SpanRepository) Store(ctx context.Context, tenantID string, spans []*collector.Span) error {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return err
	}

	for _, span := range spans {
		attrsJSON, err := json.Marshal(span.Attributes)
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO telemetry_spans (span_id, trace_id, tenant_id, agent_id, task_id, operation_name, started_at, duration_ms, tier, attributes, error_code)
			VALUES ($1, $2, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (tenant_id, span_id) DO NOTHING`,
			span.SpanID, span.TraceID, tenantID, span.AgentID, span.TaskID,
			span.OperationName, span.StartedAt, span.DurationMs, span.Tier,
			attrsJSON, span.ErrorCode)
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
	}

	return tx.Commit(ctx)
}

// Query retrieves spans matching the given filters.
func (r *SpanRepository) Query(ctx context.Context, tenantID, agentID, traceID string, limit int) ([]*collector.Span, error) {
	query := `
		SELECT span_id, trace_id, tenant_id, agent_id, task_id, operation_name, started_at, duration_ms, tier, attributes, error_code
		FROM telemetry_spans
		WHERE tenant_id = $1::uuid`
	args := []any{tenantID}
	argIdx := 2

	if agentID != "" {
		query += ` AND agent_id = $` + itoa(argIdx)
		args = append(args, agentID)
		argIdx++
	}
	if traceID != "" {
		query += ` AND trace_id = $` + itoa(argIdx)
		args = append(args, traceID)
		argIdx++
	}

	query += ` ORDER BY started_at DESC LIMIT $` + itoa(argIdx)
	args = append(args, limit)

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var spans []*collector.Span
	for rows.Next() {
		span, err := scanSpan(rows)
		if err != nil {
			return nil, err
		}
		spans = append(spans, span)
	}
	return spans, rows.Err()
}

func scanSpan(rows pgx.Rows) (*collector.Span, error) {
	var s collector.Span
	var attrsJSON []byte
	err := rows.Scan(&s.SpanID, &s.TraceID, &s.TenantID, &s.AgentID, &s.TaskID,
		&s.OperationName, &s.StartedAt, &s.DurationMs, &s.Tier, &attrsJSON, &s.ErrorCode)
	if err != nil {
		return nil, err
	}
	if attrsJSON != nil {
		_ = json.Unmarshal(attrsJSON, &s.Attributes)
	}
	return &s, nil
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
