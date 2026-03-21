package guardrails

import (
	"context"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5"
)

// PGRepository provides PostgreSQL-backed storage for guardrail data.
type PGRepository struct {
	pool *database.Pool
}

// NewPGRepository creates a new PostgreSQL-backed guardrails repository.
func NewPGRepository(pool *database.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// CreateRule persists a guardrail rule.
func (r *PGRepository) CreateRule(ctx context.Context, rule *Rule) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, rule.TenantID, `
		INSERT INTO guardrail_rules (id, tenant_id, name, description, type, pattern, action, enabled, agent_ids, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		rule.ID, rule.TenantID, rule.Name, rule.Description, rule.Type, rule.Pattern, rule.Action, rule.Enabled, rule.AgentIDs, rule.CreatedAt)
}

// GetRule retrieves a guardrail rule by ID.
func (r *PGRepository) GetRule(ctx context.Context, tenantID, ruleID string) (*Rule, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, type, pattern, action, enabled, agent_ids, created_at
		FROM guardrail_rules WHERE tenant_id = $1 AND id = $2`, tenantID, ruleID)

	rule, err := scanGuardrailRule(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return rule, nil
}

// ListRules returns all guardrail rules for a tenant.
func (r *PGRepository) ListRules(ctx context.Context, tenantID string) ([]*Rule, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, name, description, type, pattern, action, enabled, agent_ids, created_at
		FROM guardrail_rules WHERE tenant_id = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var rules []*Rule
	for rows.Next() {
		var rule Rule
		if err := rows.Scan(&rule.ID, &rule.TenantID, &rule.Name, &rule.Description, &rule.Type, &rule.Pattern, &rule.Action, &rule.Enabled, &rule.AgentIDs, &rule.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

// SaveViolation persists a guardrail violation.
func (r *PGRepository) SaveViolation(ctx context.Context, v *Violation) error {
	return database.ExecWithTenant(ctx, r.pool.Pool, v.TenantID, `
		INSERT INTO guardrail_violations (id, tenant_id, rule_id, rule_name, rule_type, agent_id, span_id, action, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		v.ID, v.TenantID, v.RuleID, v.RuleName, v.RuleType, v.AgentID, v.SpanID, v.Action, v.Content, v.CreatedAt)
}

// ListViolations returns all guardrail violations for a tenant.
func (r *PGRepository) ListViolations(ctx context.Context, tenantID string) ([]*Violation, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, rule_id, rule_name, rule_type, agent_id, span_id, action, content, created_at
		FROM guardrail_violations WHERE tenant_id = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var violations []*Violation
	for rows.Next() {
		var v Violation
		if err := rows.Scan(&v.ID, &v.TenantID, &v.RuleID, &v.RuleName, &v.RuleType, &v.AgentID, &v.SpanID, &v.Action, &v.Content, &v.CreatedAt); err != nil {
			return nil, err
		}
		violations = append(violations, &v)
	}
	return violations, rows.Err()
}

// GetStats returns guardrail statistics for a tenant.
func (r *PGRepository) GetStats(ctx context.Context, tenantID string) (*Stats, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	stats := Stats{
		ByRule:  make(map[string]int),
		ByAgent: make(map[string]int),
	}

	// Count violations
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM guardrail_violations WHERE tenant_id = $1`, tenantID).Scan(&stats.TotalViolations)
	if err != nil {
		return nil, err
	}

	// By rule
	ruleRows, err := tx.Query(ctx, `SELECT rule_name, COUNT(*) FROM guardrail_violations WHERE tenant_id = $1 GROUP BY rule_name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer ruleRows.Close()
	for ruleRows.Next() {
		var name string
		var count int
		if err := ruleRows.Scan(&name, &count); err != nil {
			return nil, err
		}
		stats.ByRule[name] = count
	}

	// By agent
	agentRows, err := tx.Query(ctx, `SELECT agent_id, COUNT(*) FROM guardrail_violations WHERE tenant_id = $1 GROUP BY agent_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer agentRows.Close()
	for agentRows.Next() {
		var aid string
		var count int
		if err := agentRows.Scan(&aid, &count); err != nil {
			return nil, err
		}
		stats.ByAgent[aid] = count
	}

	stats.TotalChecks = stats.TotalViolations * 10 // estimate
	if stats.TotalChecks > 0 {
		stats.PassRate = float64(stats.TotalChecks-stats.TotalViolations) / float64(stats.TotalChecks)
	} else {
		stats.PassRate = 1.0
	}

	return &stats, nil
}

// GenerateID creates a unique guardrail rule ID.
func GenerateID() string {
	return "gr-" + time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%d", time.Now().UnixNano()%10000)
}

func scanGuardrailRule(row pgx.Row) (*Rule, error) {
	var rule Rule
	if err := row.Scan(&rule.ID, &rule.TenantID, &rule.Name, &rule.Description, &rule.Type, &rule.Pattern, &rule.Action, &rule.Enabled, &rule.AgentIDs, &rule.CreatedAt); err != nil {
		return nil, err
	}
	return &rule, nil
}
