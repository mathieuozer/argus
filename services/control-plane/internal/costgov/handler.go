package costgov

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/tenancy"
)

// Response DTOs matching the dashboard frontend contract.

type breakdownItem struct {
	Group      string  `json:"group"`
	CostUSD    float64 `json:"cost_usd"`
	TokensUsed int64   `json:"tokens_used"`
	TaskCount  int     `json:"task_count"`
}

type trendItem struct {
	Timestamp  string  `json:"timestamp"`
	CostUSD    float64 `json:"cost_usd"`
	TokensUsed int64   `json:"tokens_used"`
}

type budgetItem struct {
	ID             string  `json:"id"`
	AgentID        string  `json:"agent_id"`
	BudgetUSD      float64 `json:"budget_usd"`
	SpentUSD       float64 `json:"spent_usd"`
	Period         string  `json:"period"`
	AlertThreshold float64 `json:"alert_threshold"`
	Enabled        bool    `json:"enabled"`
	CreatedAt      string  `json:"created_at"`
}

type anomalyItem struct {
	ID          string  `json:"id"`
	AgentID     string  `json:"agent_id"`
	ExpectedUSD float64 `json:"expected_usd"`
	ActualUSD   float64 `json:"actual_usd"`
	Ratio       float64 `json:"ratio"`
	DetectedAt  string  `json:"detected_at"`
	Status      string  `json:"status"`
}

func (h *Handler) toBudgetItem(b *Budget) budgetItem {
	status := h.repo.GetBudgetStatus(b.TenantID, b.ID)
	spent := 0.0
	if status != nil {
		spent = status.CurrentSpend
	}
	return budgetItem{
		ID:             b.ID,
		AgentID:        b.AgentID,
		BudgetUSD:      b.LimitUSD,
		SpentUSD:       spent,
		Period:         b.PeriodType,
		AlertThreshold: b.AlertThreshold,
		Enabled:        b.Enabled,
		CreatedAt:      b.CreatedAt.Format(time.RFC3339),
	}
}

// Handler provides REST handlers for cost governance.
type Handler struct {
	repo     *Repository
	detector *AnomalyDetector
}

// NewHandler creates a new cost governance handler.
func NewHandler(repo *Repository, detector *AnomalyDetector) *Handler {
	return &Handler{repo: repo, detector: detector}
}

// RegisterRoutes registers all cost governance API routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/costs/breakdown", h.handleBreakdown)
	mux.HandleFunc("/api/v1/costs/trends", h.handleTrends)
	mux.HandleFunc("/api/v1/costs/agents/", h.handleAgentCosts)
	mux.HandleFunc("/api/v1/costs/budgets", h.handleBudgets)
	mux.HandleFunc("/api/v1/costs/budgets/", h.handleBudgetByID)
	mux.HandleFunc("/api/v1/costs/anomalies", h.handleAnomalies)
	mux.HandleFunc("/api/v1/costs/record", h.handleRecord)
}

func (h *Handler) handleRecord(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req struct {
		AgentID    string  `json:"agent_id"`
		TaskID     string  `json:"task_id"`
		Model      string  `json:"model"`
		CostUSD    float64 `json:"cost_usd"`
		TokensUsed int64   `json:"tokens_used"`
		Category   string  `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	entry := h.repo.RecordCost(tenantID, req.AgentID, req.TaskID, req.CostUSD, req.TokensUsed, req.Model, req.Category)
	httputil.WriteJSON(w, http.StatusCreated, entry, tenantID)
}

func (h *Handler) handleBreakdown(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var since time.Time
	if s := r.URL.Query().Get("since"); s != "" {
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			since = parsed
		}
	}

	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "agent"
	}

	// Aggregate entries by group
	type groupAgg struct {
		costUSD    float64
		tokensUsed int64
		taskIDs    map[string]bool
	}
	groups := make(map[string]*groupAgg)

	h.repo.mu.RLock()
	for _, e := range h.repo.entries {
		if e.TenantID != tenantID {
			continue
		}
		if !since.IsZero() && e.Timestamp.Before(since) {
			continue
		}
		var key string
		switch groupBy {
		case "agent":
			key = e.AgentID
		case "operation", "category":
			key = e.Category
		case "model":
			key = e.Model
		case "day":
			key = e.Timestamp.Format("2006-01-02")
		default:
			key = e.AgentID
		}
		agg, ok := groups[key]
		if !ok {
			agg = &groupAgg{taskIDs: make(map[string]bool)}
			groups[key] = agg
		}
		agg.costUSD += e.CostUSD
		agg.tokensUsed += e.TokensUsed
		if e.TaskID != "" {
			agg.taskIDs[e.TaskID] = true
		}
	}
	h.repo.mu.RUnlock()

	items := make([]breakdownItem, 0, len(groups))
	for key, agg := range groups {
		items = append(items, breakdownItem{
			Group:      key,
			CostUSD:    agg.costUSD,
			TokensUsed: agg.tokensUsed,
			TaskCount:  len(agg.taskIDs),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CostUSD > items[j].CostUSD
	})

	httputil.WriteJSON(w, http.StatusOK, items, tenantID)
}

func (h *Handler) handleTrends(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	trends := h.repo.GetTrends(tenantID, days)

	// Transform to frontend format (period → timestamp)
	items := make([]trendItem, 0, len(trends))
	for _, t := range trends {
		items = append(items, trendItem{
			Timestamp:  t.Period,
			CostUSD:    t.CostUSD,
			TokensUsed: t.TokensUsed,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, items, tenantID)
}

func (h *Handler) handleAgentCosts(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/costs/agents/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "agent ID required")
		return
	}
	agentID := parts[0]

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	costs := h.repo.GetAgentCosts(tenantID, agentID, limit)
	httputil.WriteJSON(w, http.StatusOK, costs, tenantID)
}

func (h *Handler) handleBudgets(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		budgets := h.repo.ListBudgets(tenantID)
		items := make([]budgetItem, 0, len(budgets))
		for _, b := range budgets {
			items = append(items, h.toBudgetItem(b))
		}
		httputil.WriteJSON(w, http.StatusOK, items, tenantID)

	case http.MethodPost:
		var req struct {
			AgentID        string  `json:"agent_id"`
			Name           string  `json:"name"`
			BudgetUSD      float64 `json:"budget_usd"`
			LimitUSD       float64 `json:"limit_usd"`
			Period         string  `json:"period"`
			PeriodType     string  `json:"period_type"`
			AlertThreshold float64 `json:"alert_threshold"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		limit := req.BudgetUSD
		if limit == 0 {
			limit = req.LimitUSD
		}
		period := req.Period
		if period == "" {
			period = req.PeriodType
		}
		if req.Name == "" || limit <= 0 {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and budget_usd (> 0) are required")
			return
		}
		if period == "" {
			period = "monthly"
		}
		budget := h.repo.CreateBudget(tenantID, req.AgentID, req.Name, limit, period, req.AlertThreshold)
		httputil.WriteJSON(w, http.StatusCreated, h.toBudgetItem(budget), tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleBudgetByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/costs/budgets/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "budget ID required")
		return
	}
	budgetID := parts[0]

	// Handle status sub-resource: /api/v1/costs/budgets/{id}/status
	if len(parts) > 1 && parts[1] == "status" {
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		status := h.repo.GetBudgetStatus(tenantID, budgetID)
		if status == nil {
			httputil.WriteError(w, http.StatusNotFound, "BUDGET_NOT_FOUND", "budget not found")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, status, tenantID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		budget := h.repo.GetBudget(tenantID, budgetID)
		if budget == nil {
			httputil.WriteError(w, http.StatusNotFound, "BUDGET_NOT_FOUND", "budget not found")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, h.toBudgetItem(budget), tenantID)

	case http.MethodPut:
		var req struct {
			Name           string  `json:"name"`
			BudgetUSD      float64 `json:"budget_usd"`
			LimitUSD       float64 `json:"limit_usd"`
			AlertThreshold float64 `json:"alert_threshold"`
			Enabled        bool    `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		limit := req.BudgetUSD
		if limit == 0 {
			limit = req.LimitUSD
		}
		updated, err := h.repo.UpdateBudget(tenantID, budgetID, req.Name, limit, req.AlertThreshold, req.Enabled)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "BUDGET_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, h.toBudgetItem(updated), tenantID)

	case http.MethodDelete:
		if err := h.repo.DeleteBudget(tenantID, budgetID); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "BUDGET_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	anomalies := h.detector.DetectAnomalies(h.repo, tenantID)

	// Transform to frontend format
	items := make([]anomalyItem, 0, len(anomalies))
	for _, a := range anomalies {
		items = append(items, anomalyItem{
			ID:          a.ID,
			AgentID:     a.AgentID,
			ExpectedUSD: a.ExpectedCost,
			ActualUSD:   a.ActualCost,
			Ratio:       a.Deviation / 100.0,
			DetectedAt:  a.DetectedAt.Format(time.RFC3339),
			Status:      "open",
		})
	}
	httputil.WriteJSON(w, http.StatusOK, items, tenantID)
}
