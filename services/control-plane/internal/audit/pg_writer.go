package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/database"
)

// PGWriter writes audit log entries to PostgreSQL.
type PGWriter struct {
	pool *database.Pool
}

// NewPGWriter creates a new PostgreSQL-backed audit log writer.
func NewPGWriter(pool *database.Pool) *PGWriter {
	return &PGWriter{pool: pool}
}

// Write appends an entry to the audit log.
func (w *PGWriter) Write(ctx context.Context, tenantID, actor, action, resource, details string) (*Entry, error) {
	now := time.Now()
	entry := &Entry{
		TenantID:  tenantID,
		Actor:     actor,
		Action:    action,
		Resource:  resource,
		Details:   details,
		Timestamp: now,
	}

	tx, err := database.WithTenantTx(ctx, w.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	// The audit_logs table stores details as JSONB; wrap plain-string details
	detailsJSON := fmt.Sprintf(`{"message": %q}`, details)
	if details == "" {
		detailsJSON = "{}"
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO audit_logs (tenant_id, actor, action, resource, details, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		RETURNING id`,
		tenantID, actor, action, resource, detailsJSON, now,
	).Scan(&id)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("insert audit entry: %w", err)
	}

	entry.ID = id
	return entry, tx.Commit(ctx)
}

// List returns all audit entries for a tenant.
func (w *PGWriter) List(ctx context.Context, tenantID string) ([]*Entry, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, w.pool.Pool, tenantID, `
		SELECT id, tenant_id, actor, action, resource, details, created_at
		FROM audit_logs WHERE tenant_id = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var entries []*Entry
	for rows.Next() {
		var e Entry
		var detailsJSON []byte
		if err := rows.Scan(&e.ID, &e.TenantID, &e.Actor, &e.Action, &e.Resource, &detailsJSON, &e.Timestamp); err != nil {
			return nil, err
		}
		e.Details = string(detailsJSON)
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

// Search returns filtered audit entries for a tenant.
func (w *PGWriter) Search(ctx context.Context, tenantID, actor, action, resource string) ([]*Entry, error) {
	query := `SELECT id, tenant_id, actor, action, resource, details, created_at
		FROM audit_logs WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if actor != "" {
		query += fmt.Sprintf(" AND actor = $%d", argIdx)
		args = append(args, actor)
		argIdx++
	}
	if action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, action)
		argIdx++
	}
	if resource != "" {
		query += fmt.Sprintf(" AND resource = $%d", argIdx)
		args = append(args, resource)
		argIdx++
	}
	_ = argIdx
	query += " ORDER BY created_at DESC"

	rows, cleanup, err := database.QueryWithTenant(ctx, w.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var entries []*Entry
	for rows.Next() {
		var e Entry
		var detailsJSON []byte
		if err := rows.Scan(&e.ID, &e.TenantID, &e.Actor, &e.Action, &e.Resource, &detailsJSON, &e.Timestamp); err != nil {
			return nil, err
		}
		e.Details = string(detailsJSON)
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}
