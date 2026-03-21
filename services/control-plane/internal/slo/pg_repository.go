package slo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5"
)

// PGRepository provides PostgreSQL-backed storage for SLOs and measurements.
type PGRepository struct {
	pool *database.Pool
}

// NewPGRepository creates a new PostgreSQL-backed SLO repository.
func NewPGRepository(pool *database.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// CreateSLO creates a new service level objective.
func (r *PGRepository) CreateSLO(ctx context.Context, tenantID, name, description, agentID string, sloType SLOType, target float64, window string) (*SLO, error) {
	if window == "" {
		window = "30d"
	}
	now := time.Now()
	s := &SLO{
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		AgentID:     agentID,
		Type:        sloType,
		Target:      target,
		Window:      window,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO slos (tenant_id, name, description, agent_id, type, target, window, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		tenantID, name, description, agentID, string(sloType), target, window, true, now, now,
	).Scan(&id)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("insert SLO: %w", err)
	}

	s.ID = id
	return s, tx.Commit(ctx)
}

// ListSLOs returns all SLOs for a tenant, optionally filtered by agent.
func (r *PGRepository) ListSLOs(ctx context.Context, tenantID, agentID string) ([]*SLO, error) {
	query := `SELECT id, tenant_id, name, description, agent_id, type, target, window, enabled, created_at, updated_at
		FROM slos WHERE tenant_id = $1`
	args := []any{tenantID}

	if agentID != "" {
		query += " AND (agent_id = $2 OR agent_id = '')"
		args = append(args, agentID)
	}
	query += " ORDER BY created_at DESC"

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var slos []*SLO
	for rows.Next() {
		s, err := scanSLO(rows)
		if err != nil {
			return nil, err
		}
		slos = append(slos, s)
	}
	return slos, rows.Err()
}

// GetSLO returns a specific SLO by ID within a tenant.
func (r *PGRepository) GetSLO(ctx context.Context, tenantID, sloID string) (*SLO, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, agent_id, type, target, window, enabled, created_at, updated_at
		FROM slos WHERE tenant_id = $1 AND id = $2`, tenantID, sloID)

	s, err := scanSLORow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// UpdateSLO updates an SLO's fields.
func (r *PGRepository) UpdateSLO(ctx context.Context, tenantID, sloID, name, description string, target float64, enabled bool) (*SLO, error) {
	now := time.Now()
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{"updated_at = $3", "enabled = $4"}
	args := []any{tenantID, sloID, now, enabled}
	argIdx := 5

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
	if target > 0 {
		setClauses = append(setClauses, fmt.Sprintf("target = $%d", argIdx))
		args = append(args, target)
		argIdx++
	}
	_ = argIdx

	query := fmt.Sprintf(`UPDATE slos SET %s WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, name, description, agent_id, type, target, window, enabled, created_at, updated_at`,
		strings.Join(setClauses, ", "))

	row := tx.QueryRow(ctx, query, args...)
	s, err := scanSLORow(row)
	if err != nil {
		_ = tx.Rollback(ctx)
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("SLO %s not found", sloID)
		}
		return nil, err
	}

	return s, tx.Commit(ctx)
}

// DeleteSLO removes an SLO and its associated measurements.
func (r *PGRepository) DeleteSLO(ctx context.Context, tenantID, sloID string) error {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return err
	}

	// Delete associated measurements first
	_, err = tx.Exec(ctx, `DELETE FROM slo_measurements WHERE tenant_id = $1 AND slo_id = $2`, tenantID, sloID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("delete measurements: %w", err)
	}

	tag, err := tx.Exec(ctx, `DELETE FROM slos WHERE tenant_id = $1 AND id = $2`, tenantID, sloID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("delete SLO: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("SLO %s not found", sloID)
	}

	return tx.Commit(ctx)
}

// RecordMeasurement records a measurement for an SLO.
func (r *PGRepository) RecordMeasurement(ctx context.Context, tenantID, sloID, agentID string, value float64, good, total int64) (*Measurement, error) {
	now := time.Now()
	m := &Measurement{
		TenantID:  tenantID,
		SLOID:     sloID,
		AgentID:   agentID,
		Value:     value,
		Good:      good,
		Total:     total,
		Timestamp: now,
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO slo_measurements (tenant_id, slo_id, agent_id, value, good, total, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		tenantID, sloID, agentID, value, good, total, now,
	).Scan(&id)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("insert measurement: %w", err)
	}

	m.ID = id
	return m, tx.Commit(ctx)
}

// GetMeasurements returns measurements for an SLO within a time window.
func (r *PGRepository) GetMeasurements(ctx context.Context, tenantID, sloID string, since time.Time) ([]*Measurement, error) {
	query := `SELECT id, tenant_id, slo_id, agent_id, value, good, total, timestamp
		FROM slo_measurements WHERE tenant_id = $1 AND slo_id = $2`
	args := []any{tenantID, sloID}

	if !since.IsZero() {
		query += " AND timestamp >= $3"
		args = append(args, since)
	}
	query += " ORDER BY timestamp ASC"

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var measurements []*Measurement
	for rows.Next() {
		var m Measurement
		if err := rows.Scan(&m.ID, &m.TenantID, &m.SLOID, &m.AgentID, &m.Value, &m.Good, &m.Total, &m.Timestamp); err != nil {
			return nil, err
		}
		measurements = append(measurements, &m)
	}
	return measurements, rows.Err()
}

func scanSLO(rows pgx.Rows) (*SLO, error) {
	var s SLO
	var st string
	if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Description, &s.AgentID, &st, &s.Target, &s.Window, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	s.Type = SLOType(st)
	return &s, nil
}

func scanSLORow(row pgx.Row) (*SLO, error) {
	var s SLO
	var st string
	if err := row.Scan(&s.ID, &s.TenantID, &s.Name, &s.Description, &s.AgentID, &st, &s.Target, &s.Window, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	s.Type = SLOType(st)
	return &s, nil
}
