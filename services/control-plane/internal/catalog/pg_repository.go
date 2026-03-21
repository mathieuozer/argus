package catalog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5"
)

// PGRepository provides PostgreSQL-backed storage for catalog sources and lineage.
type PGRepository struct {
	pool *database.Pool
}

// NewPGRepository creates a new PostgreSQL-backed catalog repository.
func NewPGRepository(pool *database.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// CreateSource registers a new data source in the catalog.
func (r *PGRepository) CreateSource(ctx context.Context, tenantID, name, description string, srcType SourceType, owner, agentID string, tags []string, schema map[string]string) (*Source, error) {
	if tags == nil {
		tags = []string{}
	}

	now := time.Now()
	src := &Source{
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Type:        srcType,
		Owner:       owner,
		AgentID:     agentID,
		Tags:        tags,
		Schema:      schema,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO catalog_sources (tenant_id, name, description, type, owner, agent_id, tags, schema, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		tenantID, name, description, string(srcType), owner, agentID, tags, schemaToHstore(schema), now, now,
	).Scan(&id)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("insert source: %w", err)
	}

	src.ID = id
	return src, tx.Commit(ctx)
}

// ListSources returns all catalog sources for a tenant, optionally filtered by type or tag.
func (r *PGRepository) ListSources(ctx context.Context, tenantID string, srcType SourceType, tag string) ([]*Source, error) {
	query := `SELECT id, tenant_id, name, description, type, owner, agent_id, tags, created_at, updated_at
		FROM catalog_sources WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if srcType != "" {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, string(srcType))
		argIdx++
	}
	if tag != "" {
		query += fmt.Sprintf(" AND $%d = ANY(tags)", argIdx)
		args = append(args, tag)
		argIdx++
	}
	_ = argIdx

	query += " ORDER BY created_at DESC"

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var sources []*Source
	for rows.Next() {
		src, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}
	return sources, rows.Err()
}

// GetSource returns a specific source by ID within a tenant.
func (r *PGRepository) GetSource(ctx context.Context, tenantID, sourceID string) (*Source, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, type, owner, agent_id, tags, created_at, updated_at
		FROM catalog_sources WHERE tenant_id = $1 AND id = $2`, tenantID, sourceID)

	src, err := scanSourceRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return src, nil
}

// UpdateSource updates a source's metadata.
func (r *PGRepository) UpdateSource(ctx context.Context, tenantID, sourceID, name, description, owner string, tags []string) (*Source, error) {
	now := time.Now()

	setClauses := []string{"updated_at = $3"}
	args := []any{tenantID, sourceID, now}
	argIdx := 4

	if name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, name)
		argIdx++
	}
	if description != "" {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, description)
		argIdx++
	}
	if owner != "" {
		setClauses = append(setClauses, fmt.Sprintf("owner = $%d", argIdx))
		args = append(args, owner)
		argIdx++
	}
	if tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, tags)
		argIdx++
	}
	_ = argIdx

	query := fmt.Sprintf(`UPDATE catalog_sources SET %s WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, name, description, type, owner, agent_id, tags, created_at, updated_at`,
		strings.Join(setClauses, ", "))

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, query, args...)
	src, err := scanSourceRow(row)
	if err != nil {
		_ = tx.Rollback(ctx)
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("source %s not found", sourceID)
		}
		return nil, err
	}

	return src, tx.Commit(ctx)
}

// DeleteSource removes a source and its associated lineage edges.
func (r *PGRepository) DeleteSource(ctx context.Context, tenantID, sourceID string) error {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return err
	}

	// Delete associated edges
	_, err = tx.Exec(ctx, `DELETE FROM catalog_lineage_edges WHERE tenant_id = $1 AND (source_id = $2 OR target_id = $2)`,
		tenantID, sourceID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("delete edges: %w", err)
	}

	tag, err := tx.Exec(ctx, `DELETE FROM catalog_sources WHERE tenant_id = $1 AND id = $2`, tenantID, sourceID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("delete source: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("source %s not found", sourceID)
	}

	return tx.Commit(ctx)
}

// AddLineageEdge creates a data flow relationship between two sources.
func (r *PGRepository) AddLineageEdge(ctx context.Context, tenantID, sourceID, targetID, transformType, agentID, description string) (*LineageEdge, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	// Verify both sources exist
	var srcCount, tgtCount int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM catalog_sources WHERE tenant_id = $1 AND id = $2`, tenantID, sourceID).Scan(&srcCount)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	if srcCount == 0 {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("source %s not found", sourceID)
	}

	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM catalog_sources WHERE tenant_id = $1 AND id = $2`, tenantID, targetID).Scan(&tgtCount)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	if tgtCount == 0 {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("target %s not found", targetID)
	}

	now := time.Now()
	edge := &LineageEdge{
		TenantID:      tenantID,
		SourceID:      sourceID,
		TargetID:      targetID,
		TransformType: transformType,
		AgentID:       agentID,
		Description:   description,
		CreatedAt:     now,
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO catalog_lineage_edges (tenant_id, source_id, target_id, transform_type, agent_id, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		tenantID, sourceID, targetID, transformType, agentID, description, now,
	).Scan(&id)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("insert edge: %w", err)
	}

	edge.ID = id
	return edge, tx.Commit(ctx)
}

// GetLineage returns the lineage graph (upstream and downstream) for a source.
func (r *PGRepository) GetLineage(ctx context.Context, tenantID, sourceID string) (*LineageGraph, error) {
	// Verify source exists
	src, err := r.GetSource(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, nil
	}

	graph := &LineageGraph{
		SourceID:   sourceID,
		Upstream:   make([]*LineageNode, 0),
		Downstream: make([]*LineageNode, 0),
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	// Get upstream (sources that feed into this source)
	upRows, err := tx.Query(ctx, `
		SELECT e.source_id, s.name, e.transform_type, e.agent_id
		FROM catalog_lineage_edges e
		LEFT JOIN catalog_sources s ON s.id = e.source_id AND s.tenant_id = e.tenant_id
		WHERE e.tenant_id = $1 AND e.target_id = $2`, tenantID, sourceID)
	if err != nil {
		return nil, err
	}
	defer upRows.Close()

	for upRows.Next() {
		var node LineageNode
		var srcName *string
		if err := upRows.Scan(&node.SourceID, &srcName, &node.TransformType, &node.AgentID); err != nil {
			return nil, err
		}
		if srcName != nil {
			node.SourceName = *srcName
		}
		node.Depth = 1
		graph.Upstream = append(graph.Upstream, &node)
	}

	// Get downstream (sources that this source feeds)
	downRows, err := tx.Query(ctx, `
		SELECT e.target_id, s.name, e.transform_type, e.agent_id
		FROM catalog_lineage_edges e
		LEFT JOIN catalog_sources s ON s.id = e.target_id AND s.tenant_id = e.tenant_id
		WHERE e.tenant_id = $1 AND e.source_id = $2`, tenantID, sourceID)
	if err != nil {
		return nil, err
	}
	defer downRows.Close()

	for downRows.Next() {
		var node LineageNode
		var srcName *string
		if err := downRows.Scan(&node.SourceID, &srcName, &node.TransformType, &node.AgentID); err != nil {
			return nil, err
		}
		if srcName != nil {
			node.SourceName = *srcName
		}
		node.Depth = 1
		graph.Downstream = append(graph.Downstream, &node)
	}

	return graph, nil
}

// ListEdges returns all lineage edges for a tenant.
func (r *PGRepository) ListEdges(ctx context.Context, tenantID string) ([]*LineageEdge, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, source_id, target_id, transform_type, agent_id, description, created_at
		FROM catalog_lineage_edges WHERE tenant_id = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var edges []*LineageEdge
	for rows.Next() {
		var e LineageEdge
		if err := rows.Scan(&e.ID, &e.TenantID, &e.SourceID, &e.TargetID, &e.TransformType, &e.AgentID, &e.Description, &e.CreatedAt); err != nil {
			return nil, err
		}
		edges = append(edges, &e)
	}
	return edges, rows.Err()
}

func scanSource(rows pgx.Rows) (*Source, error) {
	var s Source
	var srcType string
	if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Description, &srcType, &s.Owner, &s.AgentID, &s.Tags, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	s.Type = SourceType(srcType)
	return &s, nil
}

func scanSourceRow(row pgx.Row) (*Source, error) {
	var s Source
	var srcType string
	if err := row.Scan(&s.ID, &s.TenantID, &s.Name, &s.Description, &srcType, &s.Owner, &s.AgentID, &s.Tags, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	s.Type = SourceType(srcType)
	return &s, nil
}

func schemaToHstore(schema map[string]string) map[string]string {
	if schema == nil {
		return map[string]string{}
	}
	return schema
}
