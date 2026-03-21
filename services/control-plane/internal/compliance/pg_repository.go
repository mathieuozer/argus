package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5"
)

// PGReportRepository provides PostgreSQL-backed storage for compliance reports.
type PGReportRepository struct {
	pool *database.Pool
}

// NewPGReportRepository creates a new PostgreSQL-backed report repository.
func NewPGReportRepository(pool *database.Pool) *PGReportRepository {
	return &PGReportRepository{pool: pool}
}

// Save persists a compliance report.
func (r *PGReportRepository) Save(ctx context.Context, report *Report) error {
	sectionsJSON, err := json.Marshal(report.Sections)
	if err != nil {
		return fmt.Errorf("marshal sections: %w", err)
	}

	return database.ExecWithTenant(ctx, r.pool.Pool, report.TenantID, `
		INSERT INTO compliance_reports (id, tenant_id, profile_id, profile_name, title, status, format, sections, generated_at, period_start, period_end)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		report.ID, report.TenantID, report.ProfileID, report.ProfileName, report.Title,
		report.Status, report.Format, sectionsJSON, report.GeneratedAt, report.PeriodStart, report.PeriodEnd)
}

// Get retrieves a compliance report by ID.
func (r *PGReportRepository) Get(ctx context.Context, tenantID, reportID string) (*Report, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, profile_id, profile_name, title, status, format, sections, generated_at, period_start, period_end
		FROM compliance_reports WHERE tenant_id = $1 AND id = $2`, tenantID, reportID)

	report, err := scanReport(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return report, nil
}

// List returns all reports for a tenant.
func (r *PGReportRepository) List(ctx context.Context, tenantID string) ([]*Report, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, profile_id, profile_name, title, status, format, sections, generated_at, period_start, period_end
		FROM compliance_reports WHERE tenant_id = $1
		ORDER BY generated_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var reports []*Report
	for rows.Next() {
		report, err := scanReportRows(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

// GenerateReportID creates a unique report ID.
func GenerateReportID() string {
	return "rpt-" + time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%d", time.Now().UnixNano()%10000)
}

func scanReport(row pgx.Row) (*Report, error) {
	var r Report
	var sectionsJSON []byte
	if err := row.Scan(&r.ID, &r.TenantID, &r.ProfileID, &r.ProfileName, &r.Title, &r.Status, &r.Format, &sectionsJSON, &r.GeneratedAt, &r.PeriodStart, &r.PeriodEnd); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(sectionsJSON, &r.Sections); err != nil {
		return nil, fmt.Errorf("unmarshal sections: %w", err)
	}
	return &r, nil
}

func scanReportRows(rows pgx.Rows) (*Report, error) {
	var r Report
	var sectionsJSON []byte
	if err := rows.Scan(&r.ID, &r.TenantID, &r.ProfileID, &r.ProfileName, &r.Title, &r.Status, &r.Format, &sectionsJSON, &r.GeneratedAt, &r.PeriodStart, &r.PeriodEnd); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(sectionsJSON, &r.Sections); err != nil {
		return nil, fmt.Errorf("unmarshal sections: %w", err)
	}
	return &r, nil
}
