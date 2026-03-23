package dataquality

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/tenancy"
)

// Handler provides REST handlers for data quality management.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new data quality handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes registers all data quality API routes on the mux.
// Both /dataquality/ and /data-quality/ paths are registered for dashboard compatibility.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	for _, prefix := range []string{"/api/v1/dataquality", "/api/v1/data-quality"} {
		mux.HandleFunc(prefix+"/rules", h.handleRules)
		mux.HandleFunc(prefix+"/rules/", h.handleRuleByID)
		mux.HandleFunc(prefix+"/scores", h.handleScores)
		mux.HandleFunc(prefix+"/violations", h.handleViolations)
		mux.HandleFunc(prefix+"/drift/", h.handleDrift)
	}
}

func (h *Handler) handleRules(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		agentID := r.URL.Query().Get("agent_id")
		rules := h.repo.ListRules(tenantID, agentID)
		httputil.WriteJSON(w, http.StatusOK, rules, tenantID)

	case http.MethodPost:
		var req struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Type        RuleType `json:"type"`
			AgentID     string   `json:"agent_id"`
			Field       string   `json:"field"`
			Operator    string   `json:"operator"`
			Threshold   string   `json:"threshold"`
			Severity    Severity `json:"severity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" || req.Field == "" || req.Operator == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name, field, and operator are required")
			return
		}
		if req.Type == "" {
			req.Type = RuleTypeCompleteness
		}
		if req.Severity == "" {
			req.Severity = SeverityWarning
		}
		rule := h.repo.CreateRule(tenantID, req.Name, req.Description, req.Type, req.AgentID, req.Field, req.Operator, req.Threshold, req.Severity)
		httputil.WriteJSON(w, http.StatusCreated, rule, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleRuleByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/dataquality/rules/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "rule ID required")
		return
	}
	ruleID := parts[0]

	switch r.Method {
	case http.MethodGet:
		rule := h.repo.GetRule(tenantID, ruleID)
		if rule == nil {
			httputil.WriteError(w, http.StatusNotFound, "RULE_NOT_FOUND", "rule not found")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, rule, tenantID)

	case http.MethodPut:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Enabled     bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		updated, err := h.repo.UpdateRule(tenantID, ruleID, req.Name, req.Description, req.Enabled)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "RULE_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, updated, tenantID)

	case http.MethodDelete:
		if err := h.repo.DeleteRule(tenantID, ruleID); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "RULE_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleScores(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		agentID := r.URL.Query().Get("agent_id")
		if agentID != "" {
			score := h.repo.GetLatestScore(tenantID, agentID)
			if score == nil {
				httputil.WriteError(w, http.StatusNotFound, "SCORE_NOT_FOUND", "no score found for agent")
				return
			}
			httputil.WriteJSON(w, http.StatusOK, score, tenantID)
			return
		}
		scores := h.repo.ListScores(tenantID, "")
		httputil.WriteJSON(w, http.StatusOK, scores, tenantID)

	case http.MethodPost:
		var req struct {
			AgentID      string  `json:"agent_id"`
			Completeness float64 `json:"completeness"`
			Consistency  float64 `json:"consistency"`
			Timeliness   float64 `json:"timeliness"`
			Validity     float64 `json:"validity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		overall := (req.Completeness + req.Consistency + req.Timeliness + req.Validity) / 4.0 * 100
		h.repo.RecordScore(tenantID, req.AgentID, overall, req.Completeness*100, req.Consistency*100, req.Timeliness*100, req.Validity*100, 100, 0, 0)
		httputil.WriteJSON(w, http.StatusCreated, map[string]string{"status": "recorded"}, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleViolations(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	ruleID := r.URL.Query().Get("rule_id")
	violations := h.repo.ListViolations(tenantID, agentID, ruleID)
	httputil.WriteJSON(w, http.StatusOK, violations, tenantID)
}

func (h *Handler) handleDrift(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, []interface{}{}, tenantID)
}
