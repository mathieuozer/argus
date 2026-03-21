package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5"
)

// PGRepository provides PostgreSQL-backed storage for eval data.
type PGRepository struct {
	pool *database.Pool
}

// NewPGRepository creates a new PostgreSQL-backed eval repository.
func NewPGRepository(pool *database.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// CreateSuite persists a test suite.
func (r *PGRepository) CreateSuite(ctx context.Context, suite *TestSuite) error {
	testCasesJSON, err := json.Marshal(suite.TestCases)
	if err != nil {
		return fmt.Errorf("marshal test cases: %w", err)
	}

	return database.ExecWithTenant(ctx, r.pool.Pool, suite.TenantID, `
		INSERT INTO eval_suites (id, tenant_id, name, description, agent_id, test_cases, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		suite.ID, suite.TenantID, suite.Name, suite.Description, suite.AgentID, testCasesJSON, suite.CreatedAt, suite.UpdatedAt)
}

// GetSuite retrieves a test suite by ID.
func (r *PGRepository) GetSuite(ctx context.Context, tenantID, suiteID string) (*TestSuite, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, agent_id, test_cases, created_at, updated_at
		FROM eval_suites WHERE tenant_id = $1 AND id = $2`, tenantID, suiteID)

	suite, err := scanSuite(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return suite, nil
}

// ListSuites returns all test suites for a tenant.
func (r *PGRepository) ListSuites(ctx context.Context, tenantID string) ([]*TestSuite, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, name, description, agent_id, test_cases, created_at, updated_at
		FROM eval_suites WHERE tenant_id = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var suites []*TestSuite
	for rows.Next() {
		suite, err := scanSuiteRows(rows)
		if err != nil {
			return nil, err
		}
		suites = append(suites, suite)
	}
	return suites, rows.Err()
}

// SaveRun persists an eval run.
func (r *PGRepository) SaveRun(ctx context.Context, run *EvalRun) error {
	resultsJSON, err := json.Marshal(run.Results)
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	return database.ExecWithTenant(ctx, r.pool.Pool, run.TenantID, `
		INSERT INTO eval_runs (id, tenant_id, suite_id, suite_name, agent_id, status, score, total_cases, passed_cases, failed_cases, results, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		run.ID, run.TenantID, run.SuiteID, run.SuiteName, run.AgentID, run.Status, run.Score,
		run.TotalCases, run.PassedCases, run.FailedCases, resultsJSON, run.StartedAt, run.CompletedAt)
}

// GetRun retrieves an eval run by ID.
func (r *PGRepository) GetRun(ctx context.Context, tenantID, runID string) (*EvalRun, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, suite_id, suite_name, agent_id, status, score, total_cases, passed_cases, failed_cases, results, started_at, completed_at
		FROM eval_runs WHERE tenant_id = $1 AND id = $2`, tenantID, runID)

	run, err := scanRun(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return run, nil
}

// ListRuns returns all eval runs for a tenant.
func (r *PGRepository) ListRuns(ctx context.Context, tenantID string) ([]*EvalRun, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, suite_id, suite_name, agent_id, status, score, total_cases, passed_cases, failed_cases, results, started_at, completed_at
		FROM eval_runs WHERE tenant_id = $1
		ORDER BY started_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var runs []*EvalRun
	for rows.Next() {
		run, err := scanRunRows(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func scanSuite(row pgx.Row) (*TestSuite, error) {
	var s TestSuite
	var testCasesJSON []byte
	if err := row.Scan(&s.ID, &s.TenantID, &s.Name, &s.Description, &s.AgentID, &testCasesJSON, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(testCasesJSON, &s.TestCases); err != nil {
		return nil, fmt.Errorf("unmarshal test cases: %w", err)
	}
	return &s, nil
}

func scanSuiteRows(rows pgx.Rows) (*TestSuite, error) {
	var s TestSuite
	var testCasesJSON []byte
	if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Description, &s.AgentID, &testCasesJSON, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(testCasesJSON, &s.TestCases); err != nil {
		return nil, fmt.Errorf("unmarshal test cases: %w", err)
	}
	return &s, nil
}

func scanRun(row pgx.Row) (*EvalRun, error) {
	var r EvalRun
	var resultsJSON []byte
	var completedAt *time.Time
	if err := row.Scan(&r.ID, &r.TenantID, &r.SuiteID, &r.SuiteName, &r.AgentID, &r.Status, &r.Score,
		&r.TotalCases, &r.PassedCases, &r.FailedCases, &resultsJSON, &r.StartedAt, &completedAt); err != nil {
		return nil, err
	}
	r.CompletedAt = completedAt
	if err := json.Unmarshal(resultsJSON, &r.Results); err != nil {
		return nil, fmt.Errorf("unmarshal results: %w", err)
	}
	return &r, nil
}

func scanRunRows(rows pgx.Rows) (*EvalRun, error) {
	var r EvalRun
	var resultsJSON []byte
	var completedAt *time.Time
	if err := rows.Scan(&r.ID, &r.TenantID, &r.SuiteID, &r.SuiteName, &r.AgentID, &r.Status, &r.Score,
		&r.TotalCases, &r.PassedCases, &r.FailedCases, &resultsJSON, &r.StartedAt, &completedAt); err != nil {
		return nil, err
	}
	r.CompletedAt = completedAt
	if err := json.Unmarshal(resultsJSON, &r.Results); err != nil {
		return nil, fmt.Errorf("unmarshal results: %w", err)
	}
	return &r, nil
}
