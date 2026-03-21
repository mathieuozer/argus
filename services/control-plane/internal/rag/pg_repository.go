package rag

import (
	"context"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/database"
)

// PGRepository provides PostgreSQL-backed storage for RAG data.
type PGRepository struct {
	pool *database.Pool
}

// NewPGRepository creates a new PostgreSQL-backed RAG repository.
func NewPGRepository(pool *database.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// SaveRetrieval persists a RAG retrieval record.
func (r *PGRepository) SaveRetrieval(ctx context.Context, ret *Retrieval) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, ret.TenantID, `
		INSERT INTO rag_retrievals (id, tenant_id, agent_id, span_id, query, num_chunks, avg_relevance, latency_ms, source_ids, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		ret.ID, ret.TenantID, ret.AgentID, ret.SpanID, ret.Query, ret.NumChunks, ret.AvgRelevance, ret.LatencyMs, ret.SourceIDs, ret.CreatedAt)
}

// ListRetrievals returns RAG retrievals for a tenant, optionally filtered by agent.
func (r *PGRepository) ListRetrievals(ctx context.Context, tenantID, agentID string) ([]*Retrieval, error) {
	query := `SELECT id, tenant_id, agent_id, span_id, query, num_chunks, avg_relevance, latency_ms, source_ids, created_at
		FROM rag_retrievals WHERE tenant_id = $1`
	args := []any{tenantID}

	if agentID != "" {
		query += " AND agent_id = $2"
		args = append(args, agentID)
	}
	query += " ORDER BY created_at DESC"

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var retrievals []*Retrieval
	for rows.Next() {
		var ret Retrieval
		if err := rows.Scan(&ret.ID, &ret.TenantID, &ret.AgentID, &ret.SpanID, &ret.Query, &ret.NumChunks, &ret.AvgRelevance, &ret.LatencyMs, &ret.SourceIDs, &ret.CreatedAt); err != nil {
			return nil, err
		}
		retrievals = append(retrievals, &ret)
	}
	return retrievals, rows.Err()
}

// SaveSource persists a RAG source.
func (r *PGRepository) SaveSource(ctx context.Context, src *Source) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, src.TenantID, `
		INSERT INTO rag_sources (id, tenant_id, name, type, total_chunks, avg_relevance, usage_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, id) DO UPDATE SET
			name = EXCLUDED.name,
			total_chunks = EXCLUDED.total_chunks,
			avg_relevance = EXCLUDED.avg_relevance,
			usage_count = EXCLUDED.usage_count`,
		src.ID, src.TenantID, src.Name, src.Type, src.TotalChunks, src.AvgRelevance, src.UsageCount)
}

// ListSources returns all RAG sources for a tenant.
func (r *PGRepository) ListSources(ctx context.Context, tenantID string) ([]*Source, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, name, type, total_chunks, avg_relevance, usage_count
		FROM rag_sources WHERE tenant_id = $1
		ORDER BY usage_count DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var sources []*Source
	for rows.Next() {
		var s Source
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Type, &s.TotalChunks, &s.AvgRelevance, &s.UsageCount); err != nil {
			return nil, err
		}
		sources = append(sources, &s)
	}
	return sources, rows.Err()
}

// GetQuality returns quality trend data.
func (r *PGRepository) GetQuality(ctx context.Context, tenantID string) ([]QualityTrend, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT DATE(created_at) AS day,
			AVG(avg_relevance) AS avg_relevance,
			AVG(latency_ms) AS avg_latency_ms,
			COUNT(*) AS total_queries
		FROM rag_retrievals WHERE tenant_id = $1
		GROUP BY DATE(created_at)
		ORDER BY day DESC
		LIMIT 30`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var trends []QualityTrend
	for rows.Next() {
		var t QualityTrend
		var day time.Time
		if err := rows.Scan(&day, &t.AvgRelevance, &t.AvgLatencyMs, &t.TotalQueries); err != nil {
			return nil, err
		}
		t.Timestamp = day
		trends = append(trends, t)
	}
	return trends, rows.Err()
}

// GenerateID creates a unique RAG ID.
func GenerateID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%d", time.Now().UnixNano()%10000)
}
