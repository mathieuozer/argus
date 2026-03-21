package costgov

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5"
)

// PGRepository provides PostgreSQL-backed storage for cost data and budgets.
type PGRepository struct {
	pool *database.Pool
}

// NewPGRepository creates a new PostgreSQL-backed cost repository.
func NewPGRepository(pool *database.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// RecordCost records a cost entry.
func (r *PGRepository) RecordCost(ctx context.Context, tenantID, agentID, taskID string, costUSD float64, tokensUsed int64, model, category string) (*CostEntry, error) {
	now := time.Now()
	entry := &CostEntry{
		TenantID:   tenantID,
		AgentID:    agentID,
		TaskID:     taskID,
		CostUSD:    costUSD,
		TokensUsed: tokensUsed,
		Model:      model,
		Category:   category,
		Timestamp:  now,
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO cost_entries (tenant_id, agent_id, task_id, cost_usd, tokens_used, model, category, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		tenantID, agentID, taskID, costUSD, tokensUsed, model, category, now,
	).Scan(&id)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("insert cost entry: %w", err)
	}

	entry.ID = id
	return entry, tx.Commit(ctx)
}

// GetBreakdown returns an aggregated cost breakdown for a tenant.
func (r *PGRepository) GetBreakdown(ctx context.Context, tenantID string, since time.Time) (*CostBreakdown, error) {
	bd := &CostBreakdown{
		TenantID:   tenantID,
		ByAgent:    make(map[string]float64),
		ByCategory: make(map[string]float64),
		ByModel:    make(map[string]float64),
	}

	query := `SELECT COALESCE(SUM(cost_usd), 0), COALESCE(SUM(tokens_used), 0), COUNT(*)
		FROM cost_entries WHERE tenant_id = $1`
	args := []any{tenantID}
	if !since.IsZero() {
		query += " AND timestamp >= $2"
		args = append(args, since)
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	err = tx.QueryRow(ctx, query, args...).Scan(&bd.TotalCostUSD, &bd.TotalTokens, &bd.EntryCount)
	if err != nil {
		return nil, err
	}

	// By agent
	agentQuery := `SELECT agent_id, COALESCE(SUM(cost_usd), 0) FROM cost_entries WHERE tenant_id = $1`
	agentArgs := []any{tenantID}
	if !since.IsZero() {
		agentQuery += " AND timestamp >= $2"
		agentArgs = append(agentArgs, since)
	}
	agentQuery += " GROUP BY agent_id"

	agentRows, err := tx.Query(ctx, agentQuery, agentArgs...)
	if err != nil {
		return nil, err
	}
	defer agentRows.Close()
	for agentRows.Next() {
		var aid string
		var cost float64
		if err := agentRows.Scan(&aid, &cost); err != nil {
			return nil, err
		}
		bd.ByAgent[aid] = cost
	}

	// By category
	catQuery := `SELECT category, COALESCE(SUM(cost_usd), 0) FROM cost_entries WHERE tenant_id = $1`
	catArgs := []any{tenantID}
	if !since.IsZero() {
		catQuery += " AND timestamp >= $2"
		catArgs = append(catArgs, since)
	}
	catQuery += " GROUP BY category"

	catRows, err := tx.Query(ctx, catQuery, catArgs...)
	if err != nil {
		return nil, err
	}
	defer catRows.Close()
	for catRows.Next() {
		var cat string
		var cost float64
		if err := catRows.Scan(&cat, &cost); err != nil {
			return nil, err
		}
		bd.ByCategory[cat] = cost
	}

	// By model
	modelQuery := `SELECT model, COALESCE(SUM(cost_usd), 0) FROM cost_entries WHERE tenant_id = $1 AND model != ''`
	modelArgs := []any{tenantID}
	if !since.IsZero() {
		modelQuery += " AND timestamp >= $2"
		modelArgs = append(modelArgs, since)
	}
	modelQuery += " GROUP BY model"

	modelRows, err := tx.Query(ctx, modelQuery, modelArgs...)
	if err != nil {
		return nil, err
	}
	defer modelRows.Close()
	for modelRows.Next() {
		var m string
		var cost float64
		if err := modelRows.Scan(&m, &cost); err != nil {
			return nil, err
		}
		bd.ByModel[m] = cost
	}

	return bd, nil
}

// GetTrends returns cost data aggregated by day for a tenant.
func (r *PGRepository) GetTrends(ctx context.Context, tenantID string, days int) ([]CostTrend, error) {
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT DATE(timestamp) AS day, COALESCE(SUM(cost_usd), 0), COALESCE(SUM(tokens_used), 0), COUNT(*)
		FROM cost_entries
		WHERE tenant_id = $1 AND timestamp >= $2
		GROUP BY DATE(timestamp)
		ORDER BY day ASC`, tenantID, cutoff)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var trends []CostTrend
	for rows.Next() {
		var t CostTrend
		var day time.Time
		if err := rows.Scan(&day, &t.CostUSD, &t.TokensUsed, &t.EntryCount); err != nil {
			return nil, err
		}
		t.Period = day.Format("2006-01-02")
		trends = append(trends, t)
	}
	return trends, rows.Err()
}

// GetAgentCosts returns cost entries for a specific agent in a tenant.
func (r *PGRepository) GetAgentCosts(ctx context.Context, tenantID, agentID string, limit int) ([]*CostEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, agent_id, task_id, cost_usd, tokens_used, model, category, timestamp
		FROM cost_entries
		WHERE tenant_id = $1 AND agent_id = $2
		ORDER BY timestamp DESC
		LIMIT $3`, tenantID, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var result []*CostEntry
	for rows.Next() {
		e, err := scanCostEntry(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// CreateBudget creates a new spending budget.
func (r *PGRepository) CreateBudget(ctx context.Context, tenantID, agentID, name string, limitUSD float64, periodType string, alertThreshold float64) (*Budget, error) {
	if alertThreshold <= 0 {
		alertThreshold = 0.8
	}
	now := time.Now()
	budget := &Budget{
		TenantID:       tenantID,
		AgentID:        agentID,
		Name:           name,
		LimitUSD:       limitUSD,
		PeriodType:     periodType,
		AlertThreshold: alertThreshold,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO cost_budgets (tenant_id, agent_id, name, limit_usd, period_type, alert_threshold, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		tenantID, agentID, name, limitUSD, periodType, alertThreshold, true, now, now,
	).Scan(&id)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("insert budget: %w", err)
	}

	budget.ID = id
	return budget, tx.Commit(ctx)
}

// ListBudgets returns all budgets for a tenant.
func (r *PGRepository) ListBudgets(ctx context.Context, tenantID string) ([]*Budget, error) {
	rows, cleanup, err := database.QueryWithTenant(ctx, r.pool.Pool, tenantID, `
		SELECT id, tenant_id, agent_id, name, limit_usd, period_type, current_spend, alert_threshold, enabled, created_at, updated_at
		FROM cost_budgets WHERE tenant_id = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var budgets []*Budget
	for rows.Next() {
		b, err := scanBudget(rows)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}
	return budgets, rows.Err()
}

// GetBudget returns a specific budget by ID.
func (r *PGRepository) GetBudget(ctx context.Context, tenantID, budgetID string) (*Budget, error) {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, tenant_id, agent_id, name, limit_usd, period_type, current_spend, alert_threshold, enabled, created_at, updated_at
		FROM cost_budgets WHERE tenant_id = $1 AND id = $2`, tenantID, budgetID)

	b, err := scanBudgetRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

// UpdateBudget updates budget fields.
func (r *PGRepository) UpdateBudget(ctx context.Context, tenantID, budgetID string, name string, limitUSD float64, alertThreshold float64, enabled bool) (*Budget, error) {
	now := time.Now()

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{"updated_at = $3", "enabled = $4"}
	args := []any{tenantID, budgetID, now, enabled}
	argIdx := 5

	if name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, name)
		argIdx++
	}
	if limitUSD > 0 {
		setClauses = append(setClauses, fmt.Sprintf("limit_usd = $%d", argIdx))
		args = append(args, limitUSD)
		argIdx++
	}
	if alertThreshold > 0 {
		setClauses = append(setClauses, fmt.Sprintf("alert_threshold = $%d", argIdx))
		args = append(args, alertThreshold)
		argIdx++
	}
	_ = argIdx

	query := fmt.Sprintf(`UPDATE cost_budgets SET %s WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, agent_id, name, limit_usd, period_type, current_spend, alert_threshold, enabled, created_at, updated_at`,
		strings.Join(setClauses, ", "))

	row := tx.QueryRow(ctx, query, args...)
	b, err := scanBudgetRow(row)
	if err != nil {
		_ = tx.Rollback(ctx)
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("budget %s not found", budgetID)
		}
		return nil, err
	}

	return b, tx.Commit(ctx)
}

// DeleteBudget removes a budget.
func (r *PGRepository) DeleteBudget(ctx context.Context, tenantID, budgetID string) error {
	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM cost_budgets WHERE tenant_id = $1 AND id = $2`, tenantID, budgetID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("delete budget: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("budget %s not found", budgetID)
	}

	return tx.Commit(ctx)
}

// GetBudgetStatus calculates current budget utilization.
func (r *PGRepository) GetBudgetStatus(ctx context.Context, tenantID, budgetID string) (*BudgetStatus, error) {
	budget, err := r.GetBudget(ctx, tenantID, budgetID)
	if err != nil {
		return nil, err
	}
	if budget == nil {
		return nil, nil
	}

	// Calculate current spend for the period
	var since time.Time
	now := time.Now()
	switch budget.PeriodType {
	case "daily":
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "weekly":
		weekday := int(now.Weekday())
		since = time.Date(now.Year(), now.Month(), now.Day()-weekday, 0, 0, 0, 0, now.Location())
	case "monthly":
		since = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		since = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	tx, err := database.WithTenantTx(ctx, r.pool.Pool, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Commit(ctx) }()

	query := `SELECT COALESCE(SUM(cost_usd), 0) FROM cost_entries WHERE tenant_id = $1 AND timestamp >= $2`
	args := []any{tenantID, since}
	if budget.AgentID != "" {
		query += " AND agent_id = $3"
		args = append(args, budget.AgentID)
	}

	var currentSpend float64
	if err := tx.QueryRow(ctx, query, args...).Scan(&currentSpend); err != nil {
		return nil, err
	}

	utilization := 0.0
	if budget.LimitUSD > 0 {
		utilization = currentSpend / budget.LimitUSD
	}

	return &BudgetStatus{
		Budget:       budget,
		CurrentSpend: currentSpend,
		Remaining:    budget.LimitUSD - currentSpend,
		Utilization:  utilization,
		IsOverBudget: currentSpend > budget.LimitUSD,
		IsAlert:      utilization >= budget.AlertThreshold && currentSpend <= budget.LimitUSD,
	}, nil
}

func scanCostEntry(rows pgx.Rows) (*CostEntry, error) {
	var e CostEntry
	if err := rows.Scan(&e.ID, &e.TenantID, &e.AgentID, &e.TaskID, &e.CostUSD, &e.TokensUsed, &e.Model, &e.Category, &e.Timestamp); err != nil {
		return nil, err
	}
	return &e, nil
}

func scanBudget(rows pgx.Rows) (*Budget, error) {
	var b Budget
	if err := rows.Scan(&b.ID, &b.TenantID, &b.AgentID, &b.Name, &b.LimitUSD, &b.PeriodType, &b.CurrentSpend, &b.AlertThreshold, &b.Enabled, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

func scanBudgetRow(row pgx.Row) (*Budget, error) {
	var b Budget
	if err := row.Scan(&b.ID, &b.TenantID, &b.AgentID, &b.Name, &b.LimitUSD, &b.PeriodType, &b.CurrentSpend, &b.AlertThreshold, &b.Enabled, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}
	return &b, nil
}
