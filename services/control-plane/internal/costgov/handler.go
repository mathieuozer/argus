package costgov

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/argus-platform/argus/pkg/tenancy"
)

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
}

func (h *Handler) handleBreakdown(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var since time.Time
	if s := r.URL.Query().Get("since"); s != "" {
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			since = parsed
		}
	}

	breakdown := h.repo.GetBreakdown(tenantID, since)
	writeJSON(w, http.StatusOK, breakdown, tenantID)
}

func (h *Handler) handleTrends(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	trends := h.repo.GetTrends(tenantID, days)
	writeJSON(w, http.StatusOK, trends, tenantID)
}

func (h *Handler) handleAgentCosts(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/costs/agents/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "agent ID required")
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
	writeJSON(w, http.StatusOK, costs, tenantID)
}

func (h *Handler) handleBudgets(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		budgets := h.repo.ListBudgets(tenantID)
		writeJSON(w, http.StatusOK, budgets, tenantID)

	case http.MethodPost:
		var req struct {
			AgentID        string  `json:"agent_id"`
			Name           string  `json:"name"`
			LimitUSD       float64 `json:"limit_usd"`
			PeriodType     string  `json:"period_type"`
			AlertThreshold float64 `json:"alert_threshold"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" || req.LimitUSD <= 0 {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and limit_usd (> 0) are required")
			return
		}
		if req.PeriodType == "" {
			req.PeriodType = "monthly"
		}
		budget := h.repo.CreateBudget(tenantID, req.AgentID, req.Name, req.LimitUSD, req.PeriodType, req.AlertThreshold)
		writeJSON(w, http.StatusCreated, budget, tenantID)

	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleBudgetByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/costs/budgets/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "budget ID required")
		return
	}
	budgetID := parts[0]

	// Handle status sub-resource: /api/v1/costs/budgets/{id}/status
	if len(parts) > 1 && parts[1] == "status" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		status := h.repo.GetBudgetStatus(tenantID, budgetID)
		if status == nil {
			writeError(w, http.StatusNotFound, "BUDGET_NOT_FOUND", "budget not found")
			return
		}
		writeJSON(w, http.StatusOK, status, tenantID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		budget := h.repo.GetBudget(tenantID, budgetID)
		if budget == nil {
			writeError(w, http.StatusNotFound, "BUDGET_NOT_FOUND", "budget not found")
			return
		}
		writeJSON(w, http.StatusOK, budget, tenantID)

	case http.MethodPut:
		var req struct {
			Name           string  `json:"name"`
			LimitUSD       float64 `json:"limit_usd"`
			AlertThreshold float64 `json:"alert_threshold"`
			Enabled        bool    `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		updated, err := h.repo.UpdateBudget(tenantID, budgetID, req.Name, req.LimitUSD, req.AlertThreshold, req.Enabled)
		if err != nil {
			writeError(w, http.StatusNotFound, "BUDGET_NOT_FOUND", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated, tenantID)

	case http.MethodDelete:
		if err := h.repo.DeleteBudget(tenantID, budgetID); err != nil {
			writeError(w, http.StatusNotFound, "BUDGET_NOT_FOUND", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)

	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	anomalies := h.detector.DetectAnomalies(h.repo, tenantID)
	writeJSON(w, http.StatusOK, anomalies, tenantID)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}, tenantID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": ensureNotNil(data),
		"meta": map[string]string{"tenant_id": tenantID},
	})
}

func ensureNotNil(v interface{}) interface{} {
	if v == nil {
		return []interface{}{}
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		return []interface{}{}
	}
	return v
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
