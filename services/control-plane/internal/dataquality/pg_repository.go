package dataquality

import (
	"context"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5"
)

// PGRepository provides PostgreSQL-backed storage for data quality rules, scores, and violations.
type PGRepository struct {
	pool *database.Pool
}

// NewPGRepository creates a new PostgreSQL-backed data quality repository.
func NewPGRepository(pool *database.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// CreateRule adds a new data quality rule.
func (r *PGRepository) CreateRule(ctx context.Context, tenantID, name, description string, ruleType RuleType, agentID, field, operator, threshold string, severity Severity) (*Rule, error) {
	now := time.Now()
	rule := &Rule{
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Type:        ruleType,
		AgentID:     agentID,
		Field:       field,
		Operator:    operator,
		Threshold:   threshold,
		Severity:    severity,
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
		INSERT INTO dq_rules (tenant_id, name, description, type, agent_id, field, operator, threshold, severity, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		tenantID, name, description, string(ruleType), agentID, field, operator, threshold, string(severity), true, now, now,
	).Scan(&id)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("insert rule: %w", err)
	}

	rule.ID = id
	return rule, tx.Commit(ctx)
}

// ListRules returns all rules for a tenant, optionally filtered by agent ID.
func (r *PGRepository) ListRules(ctx context.Context, tenantID, agentID string) ([]*Rule, error) {
	query := `SELECT id, tenant_id, name, description, type, agent_id, field, operator, threshold, severity, enabled, created_at, updated_at
		FROM dq_rules WHERE tenant_id = $1`
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

	var rules []*Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// GetRule returns a specific rule by ID within a tenant.
func (r *PGRepository) GetRule(ctx context.Context, tenantID, ruleID string) (*Rule, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, type, agent_id, field, operator, threshold, severity, enabled, created_at, updated_at
		FROM dq_rules WHERE tenant_id = $1 AND id = $2`, tenantID, ruleID)

	rule, err := scanRuleRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return rule, nil
}

// UpdateRule updates a rule's fields.
func (r *PGRepository) UpdateRule(ctx context.Context, tenantID, ruleID string, name, description string, enabled bool) (*Rule, error) {
	now := time.Now()
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	setClauses := "updated_at = $3, enabled = $4"
	args := []any{tenantID, ruleID, now, enabled}
	argIdx := 5

	if name != "" {
		setClauses += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, name)
		argIdx++
	}
	if description != "" {
		setClauses += fmt.Sprintf(", description = $%d", argIdx)
		args = append(args, description)
		argIdx++
	}
	_ = argIdx

	query := fmt.Sprintf(`UPDATE dq_rules SET %s WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, name, description, type, agent_id, field, operator, threshold, severity, enabled, created_at, updated_at`, setClauses)

	row := tx.QueryRow(ctx, query, args...)
	rule, err := scanRuleRow(row)
	if err != nil {
		_ = tx.Rollback(ctx)
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("rule %s not found", ruleID)
		}
		return nil, err
	}

	return rule, tx.Commit(ctx)
}

// DeleteRule removes a rule.
func (r *PGRepository) DeleteRule(ctx context.Context, tenantID, ruleID string) error {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM dq_rules WHERE tenant_id = $1 AND id = $2`, tenantID, ruleID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("delete rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("rule %s not found", ruleID)
	}

	return tx.Commit(ctx)
}

// RecordScore records a data quality score for an agent.
func (r *PGRepository) RecordScore(ctx context.Context, tenantID, agentID string, overall, completeness, accuracy, consistency, timeliness float64, total, passed, failed int) (*Score, error) {
	now := time.Now()
	score := &Score{
		TenantID:          tenantID,
		AgentID:           agentID,
		OverallScore:      overall,
		CompletenessScore: completeness,
		AccuracyScore:     accuracy,
		ConsistencyScore:  consistency,
		TimelinessScore:   timeliness,
		TotalChecks:       total,
		PassedChecks:      passed,
		FailedChecks:      failed,
		EvaluatedAt:       now,
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO dq_scores (tenant_id, agent_id, overall_score, completeness_score, accuracy_score, consistency_score, timeliness_score, total_checks, passed_checks, failed_checks, evaluated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		tenantID, agentID, overall, completeness, accuracy, consistency, timeliness, total, passed, failed, now,
	).Scan(&id)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("insert score: %w", err)
	}

	score.ID = id
	return score, tx.Commit(ctx)
}

// ListScores returns data quality scores for a tenant, optionally filtered by agent.
func (r *PGRepository) ListScores(ctx context.Context, tenantID, agentID string) ([]*Score, error) {
	query := `SELECT id, tenant_id, agent_id, overall_score, completeness_score, accuracy_score, consistency_score, timeliness_score, total_checks, passed_checks, failed_checks, evaluated_at
		FROM dq_scores WHERE tenant_id = $1`
	args := []any{tenantID}

	if agentID != "" {
		query += " AND agent_id = $2"
		args = append(args, agentID)
	}
	query += " ORDER BY evaluated_at DESC"

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var scores []*Score
	for rows.Next() {
		s, err := scanScore(rows)
		if err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}

// GetLatestScore returns the most recent score for an agent in a tenant.
func (r *PGRepository) GetLatestScore(ctx context.Context, tenantID, agentID string) (*Score, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, agent_id, overall_score, completeness_score, accuracy_score, consistency_score, timeliness_score, total_checks, passed_checks, failed_checks, evaluated_at
		FROM dq_scores WHERE tenant_id = $1 AND agent_id = $2
		ORDER BY evaluated_at DESC LIMIT 1`, tenantID, agentID)

	s, err := scanScoreRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

// RecordViolation records a data quality violation.
func (r *PGRepository) RecordViolation(ctx context.Context, tenantID, ruleID, ruleName, agentID, field, value, expected string, severity Severity, message string) (*Violation, error) {
	now := time.Now()
	v := &Violation{
		TenantID:   tenantID,
		RuleID:     ruleID,
		RuleName:   ruleName,
		AgentID:    agentID,
		Field:      field,
		Value:      value,
		Expected:   expected,
		Severity:   severity,
		Message:    message,
		OccurredAt: now,
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO dq_violations (tenant_id, rule_id, rule_name, agent_id, field, value, expected, severity, message, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		tenantID, ruleID, ruleName, agentID, field, value, expected, string(severity), message, now,
	).Scan(&id)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("insert violation: %w", err)
	}

	v.ID = id
	return v, tx.Commit(ctx)
}

// ListViolations returns violations for a tenant, optionally filtered by agent and/or rule.
func (r *PGRepository) ListViolations(ctx context.Context, tenantID, agentID, ruleID string) ([]*Violation, error) {
	query := `SELECT id, tenant_id, rule_id, rule_name, agent_id, field, value, expected, severity, message, occurred_at
		FROM dq_violations WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if agentID != "" {
		query += fmt.Sprintf(" AND agent_id = $%d", argIdx)
		args = append(args, agentID)
		argIdx++
	}
	if ruleID != "" {
		query += fmt.Sprintf(" AND rule_id = $%d", argIdx)
		args = append(args, ruleID)
		argIdx++
	}
	_ = argIdx
	query += " ORDER BY occurred_at DESC"

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, query, args...)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var violations []*Violation
	for rows.Next() {
		var v Violation
		var sev string
		if err := rows.Scan(&v.ID, &v.TenantID, &v.RuleID, &v.RuleName, &v.AgentID, &v.Field, &v.Value, &v.Expected, &sev, &v.Message, &v.OccurredAt); err != nil {
			return nil, err
		}
		v.Severity = Severity(sev)
		violations = append(violations, &v)
	}
	return violations, rows.Err()
}

func scanRule(rows pgx.Rows) (*Rule, error) {
	var r Rule
	var rt, sev string
	if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.Description, &rt, &r.AgentID, &r.Field, &r.Operator, &r.Threshold, &sev, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	r.Type = RuleType(rt)
	r.Severity = Severity(sev)
	return &r, nil
}

func scanRuleRow(row pgx.Row) (*Rule, error) {
	var r Rule
	var rt, sev string
	if err := row.Scan(&r.ID, &r.TenantID, &r.Name, &r.Description, &rt, &r.AgentID, &r.Field, &r.Operator, &r.Threshold, &sev, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	r.Type = RuleType(rt)
	r.Severity = Severity(sev)
	return &r, nil
}

func scanScore(rows pgx.Rows) (*Score, error) {
	var s Score
	if err := rows.Scan(&s.ID, &s.TenantID, &s.AgentID, &s.OverallScore, &s.CompletenessScore, &s.AccuracyScore, &s.ConsistencyScore, &s.TimelinessScore, &s.TotalChecks, &s.PassedChecks, &s.FailedChecks, &s.EvaluatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func scanScoreRow(row pgx.Row) (*Score, error) {
	var s Score
	if err := row.Scan(&s.ID, &s.TenantID, &s.AgentID, &s.OverallScore, &s.CompletenessScore, &s.AccuracyScore, &s.ConsistencyScore, &s.TimelinessScore, &s.TotalChecks, &s.PassedChecks, &s.FailedChecks, &s.EvaluatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}
