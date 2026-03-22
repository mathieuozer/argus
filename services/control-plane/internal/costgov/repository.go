package costgov

import (
	"fmt"
	"sync"
	"time"
)

// CostEntry represents a single cost record for an agent.
type CostEntry struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	AgentID    string    `json:"agent_id"`
	TaskID     string    `json:"task_id,omitempty"`
	CostUSD    float64   `json:"cost_usd"`
	TokensUsed int64     `json:"tokens_used"`
	Model      string    `json:"model,omitempty"`
	Category   string    `json:"category"` // "inference", "tool_call", "embedding", "other"
	Timestamp  time.Time `json:"timestamp"`
}

// CostBreakdown provides aggregated cost information.
type CostBreakdown struct {
	TenantID     string             `json:"tenant_id"`
	TotalCostUSD float64            `json:"total_cost_usd"`
	TotalTokens  int64              `json:"total_tokens"`
	ByAgent      map[string]float64 `json:"by_agent"`
	ByCategory   map[string]float64 `json:"by_category"`
	ByModel      map[string]float64 `json:"by_model"`
	EntryCount   int                `json:"entry_count"`
}

// CostTrend represents cost data over a time window.
type CostTrend struct {
	Period     string  `json:"period"` // "2024-01-15", "2024-01-15T14:00"
	CostUSD    float64 `json:"cost_usd"`
	TokensUsed int64   `json:"tokens_used"`
	EntryCount int     `json:"entry_count"`
}

// Budget defines spending limits for a tenant or agent.
type Budget struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	AgentID        string    `json:"agent_id,omitempty"` // empty means tenant-wide
	Name           string    `json:"name"`
	LimitUSD       float64   `json:"limit_usd"`
	PeriodType     string    `json:"period_type"` // "daily", "weekly", "monthly"
	CurrentSpend   float64   `json:"current_spend"`
	AlertThreshold float64   `json:"alert_threshold"` // 0.0 to 1.0, fraction of limit
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// BudgetStatus provides current budget utilization information.
type BudgetStatus struct {
	Budget       *Budget `json:"budget"`
	CurrentSpend float64 `json:"current_spend"`
	Remaining    float64 `json:"remaining"`
	Utilization  float64 `json:"utilization"` // 0.0 to 1.0
	IsOverBudget bool    `json:"is_over_budget"`
	IsAlert      bool    `json:"is_alert"` // over alert threshold but under limit
}

// Repository provides in-memory storage for cost data and budgets.
type Repository struct {
	mu        sync.RWMutex
	entries   []*CostEntry
	budgets   []*Budget
	entrySeq  int
	budgetSeq int
}

// NewRepository creates a new cost repository.
func NewRepository() *Repository {
	return &Repository{
		entries: make([]*CostEntry, 0),
		budgets: make([]*Budget, 0),
	}
}

// RecordCost records a cost entry.
func (r *Repository) RecordCost(tenantID, agentID, taskID string, costUSD float64, tokensUsed int64, model, category string) *CostEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entrySeq++
	entry := &CostEntry{
		ID:         fmt.Sprintf("cost-%d", r.entrySeq),
		TenantID:   tenantID,
		AgentID:    agentID,
		TaskID:     taskID,
		CostUSD:    costUSD,
		TokensUsed: tokensUsed,
		Model:      model,
		Category:   category,
		Timestamp:  time.Now(),
	}
	r.entries = append(r.entries, entry)
	return entry
}

// RecordCostAt records a cost entry at a specific time (for testing and historical import).
func (r *Repository) RecordCostAt(tenantID, agentID, taskID string, costUSD float64, tokensUsed int64, model, category string, ts time.Time) *CostEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entrySeq++
	entry := &CostEntry{
		ID:         fmt.Sprintf("cost-%d", r.entrySeq),
		TenantID:   tenantID,
		AgentID:    agentID,
		TaskID:     taskID,
		CostUSD:    costUSD,
		TokensUsed: tokensUsed,
		Model:      model,
		Category:   category,
		Timestamp:  ts,
	}
	r.entries = append(r.entries, entry)
	return entry
}

// GetBreakdown returns an aggregated cost breakdown for a tenant.
func (r *Repository) GetBreakdown(tenantID string, since time.Time) *CostBreakdown {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bd := &CostBreakdown{
		TenantID:   tenantID,
		ByAgent:    make(map[string]float64),
		ByCategory: make(map[string]float64),
		ByModel:    make(map[string]float64),
	}

	for _, e := range r.entries {
		if e.TenantID != tenantID {
			continue
		}
		if !since.IsZero() && e.Timestamp.Before(since) {
			continue
		}
		bd.TotalCostUSD += e.CostUSD
		bd.TotalTokens += e.TokensUsed
		bd.ByAgent[e.AgentID] += e.CostUSD
		bd.ByCategory[e.Category] += e.CostUSD
		if e.Model != "" {
			bd.ByModel[e.Model] += e.CostUSD
		}
		bd.EntryCount++
	}

	return bd
}

// GetTrends returns cost data aggregated by day for a tenant.
func (r *Repository) GetTrends(tenantID string, days int) []CostTrend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	// Aggregate by day
	dayMap := make(map[string]*CostTrend)
	for _, e := range r.entries {
		if e.TenantID != tenantID || e.Timestamp.Before(cutoff) {
			continue
		}
		day := e.Timestamp.Format("2006-01-02")
		if _, ok := dayMap[day]; !ok {
			dayMap[day] = &CostTrend{Period: day}
		}
		dayMap[day].CostUSD += e.CostUSD
		dayMap[day].TokensUsed += e.TokensUsed
		dayMap[day].EntryCount++
	}

	// Convert to sorted slice
	trends := make([]CostTrend, 0, len(dayMap))
	for _, t := range dayMap {
		trends = append(trends, *t)
	}

	// Sort by period (ascending)
	for i := 0; i < len(trends); i++ {
		for j := i + 1; j < len(trends); j++ {
			if trends[i].Period > trends[j].Period {
				trends[i], trends[j] = trends[j], trends[i]
			}
		}
	}

	return trends
}

// GetAgentCosts returns cost entries for a specific agent in a tenant.
func (r *Repository) GetAgentCosts(tenantID, agentID string, limit int) []*CostEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*CostEntry
	// Iterate in reverse for most recent first
	for i := len(r.entries) - 1; i >= 0; i-- {
		e := r.entries[i]
		if e.TenantID == tenantID && e.AgentID == agentID {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result
}

// CreateBudget creates a new spending budget.
func (r *Repository) CreateBudget(tenantID, agentID, name string, limitUSD float64, periodType string, alertThreshold float64) *Budget {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.budgetSeq++
	now := time.Now()
	if alertThreshold <= 0 {
		alertThreshold = 0.8
	}
	budget := &Budget{
		ID:             fmt.Sprintf("budget-%d", r.budgetSeq),
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
	r.budgets = append(r.budgets, budget)
	return budget
}

// ListBudgets returns all budgets for a tenant.
func (r *Repository) ListBudgets(tenantID string) []*Budget {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Budget
	for _, b := range r.budgets {
		if b.TenantID == tenantID {
			result = append(result, b)
		}
	}
	return result
}

// GetBudget returns a specific budget by ID.
func (r *Repository) GetBudget(tenantID, budgetID string) *Budget {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, b := range r.budgets {
		if b.TenantID == tenantID && b.ID == budgetID {
			return b
		}
	}
	return nil
}

// UpdateBudget updates budget fields.
func (r *Repository) UpdateBudget(tenantID, budgetID string, name string, limitUSD float64, alertThreshold float64, enabled bool) (*Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, b := range r.budgets {
		if b.TenantID == tenantID && b.ID == budgetID {
			if name != "" {
				b.Name = name
			}
			if limitUSD > 0 {
				b.LimitUSD = limitUSD
			}
			if alertThreshold > 0 {
				b.AlertThreshold = alertThreshold
			}
			b.Enabled = enabled
			b.UpdatedAt = time.Now()
			return b, nil
		}
	}
	return nil, fmt.Errorf("budget %s not found", budgetID)
}

// DeleteBudget removes a budget.
func (r *Repository) DeleteBudget(tenantID, budgetID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, b := range r.budgets {
		if b.TenantID == tenantID && b.ID == budgetID {
			r.budgets = append(r.budgets[:i], r.budgets[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("budget %s not found", budgetID)
}

// GetBudgetStatus calculates current budget utilization.
func (r *Repository) GetBudgetStatus(tenantID, budgetID string) *BudgetStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var budget *Budget
	for _, b := range r.budgets {
		if b.TenantID == tenantID && b.ID == budgetID {
			budget = b
			break
		}
	}
	if budget == nil {
		return nil
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

	var currentSpend float64
	for _, e := range r.entries {
		if e.TenantID != tenantID || e.Timestamp.Before(since) {
			continue
		}
		if budget.AgentID != "" && e.AgentID != budget.AgentID {
			continue
		}
		currentSpend += e.CostUSD
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
	}
}
