package repository

import (
	"context"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5"
)

// PredictiveAlert represents a pre-failure alert stored in PostgreSQL.
type PredictiveAlert struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	AgentID         string     `json:"agent_id"`
	Probability     float64    `json:"probability"`
	EstimatedTTFSec int        `json:"estimated_ttf_seconds"`
	PrecursorType   string     `json:"precursor_type"`
	Evidence        []string   `json:"evidence"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
}

// AlertRepository provides PostgreSQL-backed alert persistence.
type AlertRepository struct {
	pool *database.Pool
}

// NewAlertRepository creates a new PostgreSQL-backed alert repository.
func NewAlertRepository(pool *database.Pool) *AlertRepository {
	return &AlertRepository{pool: pool}
}

// Create persists a new predictive alert.
func (r *AlertRepository) Create(ctx context.Context, alert *PredictiveAlert) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, alert.TenantID, `
		INSERT INTO predictive_alerts (tenant_id, agent_id, probability, estimated_ttf_seconds, precursor_type, evidence, status)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)`,
		alert.TenantID, alert.AgentID, alert.Probability, alert.EstimatedTTFSec,
		alert.PrecursorType, alert.Evidence, alert.Status)
}

// ListByTenant returns alerts for a tenant.
func (r *AlertRepository) ListByTenant(ctx context.Context, tenantID string, status string) ([]*PredictiveAlert, error) {
	query := `
		SELECT id, tenant_id, agent_id, probability, estimated_ttf_seconds, precursor_type, evidence, status, created_at, resolved_at
		FROM predictive_alerts
		WHERE tenant_id = $1::uuid`
	args := []any{tenantID}

	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT 100`

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var alerts []*PredictiveAlert
	for rows.Next() {
		alert, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}

// UpdateStatus updates the status of an alert.
func (r *AlertRepository) UpdateStatus(ctx context.Context, tenantID, alertID, status string) error {
	var resolvedAt *time.Time
	if status == "resolved" || status == "false_positive" {
		now := time.Now()
		resolvedAt = &now
	}

	return database.ExecWithTenant(ctx, r.pool.Pool, tenantID, `
		UPDATE predictive_alerts SET status = $1, resolved_at = $2
		WHERE id = $3::uuid AND tenant_id = $4::uuid`,
		status, resolvedAt, alertID, tenantID)
}

func scanAlert(rows pgx.Rows) (*PredictiveAlert, error) {
	var a PredictiveAlert
	err := rows.Scan(&a.ID, &a.TenantID, &a.AgentID, &a.Probability, &a.EstimatedTTFSec,
		&a.PrecursorType, &a.Evidence, &a.Status, &a.CreatedAt, &a.ResolvedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
