package dataquality

import (
	"encoding/json"
	"net/http"
	"strings"

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
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/dataquality/rules", h.handleRules)
	mux.HandleFunc("/api/v1/dataquality/rules/", h.handleRuleByID)
	mux.HandleFunc("/api/v1/dataquality/scores", h.handleScores)
	mux.HandleFunc("/api/v1/dataquality/violations", h.handleViolations)
}

func (h *Handler) handleRules(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		agentID := r.URL.Query().Get("agent_id")
		rules := h.repo.ListRules(tenantID, agentID)
		writeJSON(w, http.StatusOK, rules, tenantID)

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
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" || req.Field == "" || req.Operator == "" {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name, field, and operator are required")
			return
		}
		if req.Type == "" {
			req.Type = RuleTypeCompleteness
		}
		if req.Severity == "" {
			req.Severity = SeverityWarning
		}
		rule := h.repo.CreateRule(tenantID, req.Name, req.Description, req.Type, req.AgentID, req.Field, req.Operator, req.Threshold, req.Severity)
		writeJSON(w, http.StatusCreated, rule, tenantID)

	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleRuleByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/dataquality/rules/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "rule ID required")
		return
	}
	ruleID := parts[0]

	switch r.Method {
	case http.MethodGet:
		rule := h.repo.GetRule(tenantID, ruleID)
		if rule == nil {
			writeError(w, http.StatusNotFound, "RULE_NOT_FOUND", "rule not found")
			return
		}
		writeJSON(w, http.StatusOK, rule, tenantID)

	case http.MethodPut:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Enabled     bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		updated, err := h.repo.UpdateRule(tenantID, ruleID, req.Name, req.Description, req.Enabled)
		if err != nil {
			writeError(w, http.StatusNotFound, "RULE_NOT_FOUND", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated, tenantID)

	case http.MethodDelete:
		if err := h.repo.DeleteRule(tenantID, ruleID); err != nil {
			writeError(w, http.StatusNotFound, "RULE_NOT_FOUND", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)

	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleScores(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID != "" {
		score := h.repo.GetLatestScore(tenantID, agentID)
		if score == nil {
			writeError(w, http.StatusNotFound, "SCORE_NOT_FOUND", "no score found for agent")
			return
		}
		writeJSON(w, http.StatusOK, score, tenantID)
		return
	}

	scores := h.repo.ListScores(tenantID, "")
	writeJSON(w, http.StatusOK, scores, tenantID)
}

func (h *Handler) handleViolations(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	ruleID := r.URL.Query().Get("rule_id")
	violations := h.repo.ListViolations(tenantID, agentID, ruleID)
	writeJSON(w, http.StatusOK, violations, tenantID)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}, tenantID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": data,
		"meta": map[string]string{"tenant_id": tenantID},
	})
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
