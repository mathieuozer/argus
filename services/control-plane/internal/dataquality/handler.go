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
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	for _, prefix := range []string{"/api/v1/dataquality", "/api/v1/data-quality"} {
		mux.HandleFunc(prefix+"/rules", h.handleRules)
		mux.HandleFunc(prefix+"/rules/", h.handleRuleByID)
		mux.HandleFunc(prefix+"/scores", h.handleScores)
		mux.HandleFunc(prefix+"/violations", h.handleViolations)
		mux.HandleFunc(prefix+"/drift/", h.handleDrift)
		mux.HandleFunc(prefix+"/summary", h.handleSummary)
		mux.HandleFunc(prefix+"/profiles", h.handleProfiles)
		mux.HandleFunc(prefix+"/profiles/", h.handleProfileByAgent)
		mux.HandleFunc(prefix+"/contracts", h.handleContracts)
		mux.HandleFunc(prefix+"/contracts/", h.handleContractByID)
		mux.HandleFunc(prefix+"/trends/", h.handleTrends)
		mux.HandleFunc(prefix+"/incidents", h.handleIncidents)
		mux.HandleFunc(prefix+"/incidents/", h.handleIncidentByID)
		mux.HandleFunc(prefix+"/anomalies", h.handleAnomalies)
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
		httputil.WriteJSON(w, http.StatusOK, h.repo.ListRules(tenantID, agentID), tenantID)
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
	// Handle both path prefixes
	path := r.URL.Path
	for _, prefix := range []string{"/api/v1/dataquality/rules/", "/api/v1/data-quality/rules/"} {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}
	ruleID := strings.Split(path, "/")[0]
	if ruleID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "rule ID required")
		return
	}
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
				httputil.WriteError(w, http.StatusNotFound, "SCORE_NOT_FOUND", "no score found")
				return
			}
			httputil.WriteJSON(w, http.StatusOK, score, tenantID)
			return
		}
		httputil.WriteJSON(w, http.StatusOK, h.repo.ListScores(tenantID, ""), tenantID)
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
	httputil.WriteJSON(w, http.StatusOK, h.repo.ListViolations(tenantID, agentID, ruleID), tenantID)
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
	// Extract agent ID from path
	path := r.URL.Path
	for _, prefix := range []string{"/api/v1/dataquality/drift/", "/api/v1/data-quality/drift/"} {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}
	agentID := strings.Split(path, "/")[0]
	points := h.repo.GetDrift(tenantID, agentID)
	httputil.WriteJSON(w, http.StatusOK, points, tenantID)
}

func (h *Handler) handleSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, h.repo.GetSummary(tenantID), tenantID)
}

func (h *Handler) handleProfiles(w http.ResponseWriter, r *http.Request) {
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
	httputil.WriteJSON(w, http.StatusOK, h.repo.ListProfiles(tenantID, agentID), tenantID)
}

func (h *Handler) handleProfileByAgent(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	path := r.URL.Path
	for _, prefix := range []string{"/api/v1/dataquality/profiles/", "/api/v1/data-quality/profiles/"} {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}
	agentID := strings.Split(path, "/")[0]
	profile := h.repo.GetLatestProfile(tenantID, agentID)
	if profile == nil {
		httputil.WriteJSON(w, http.StatusOK, []*DataProfile{}, tenantID)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, profile, tenantID)
}

func (h *Handler) handleContracts(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		agentID := r.URL.Query().Get("agent_id")
		httputil.WriteJSON(w, http.StatusOK, h.repo.ListContracts(tenantID, agentID), tenantID)
	case http.MethodPost:
		var req struct {
			Name           string            `json:"name"`
			Description    string            `json:"description"`
			ProducerAgent  string            `json:"producer_agent"`
			ConsumerAgents []string          `json:"consumer_agents"`
			SourceID       string            `json:"source_id"`
			SchemaSpec     map[string]string `json:"schema_spec"`
			FreshnessSpec  *FreshnessSpec    `json:"freshness_spec"`
			QualitySpec    *QualitySpec      `json:"quality_spec"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
			return
		}
		c := h.repo.CreateContract(tenantID, req.Name, req.Description, req.ProducerAgent, req.ConsumerAgents, req.SourceID, req.SchemaSpec, req.FreshnessSpec, req.QualitySpec)
		httputil.WriteJSON(w, http.StatusCreated, c, tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleContractByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	path := r.URL.Path
	for _, prefix := range []string{"/api/v1/dataquality/contracts/", "/api/v1/data-quality/contracts/"} {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}
	contractID := strings.Split(path, "/")[0]
	switch r.Method {
	case http.MethodGet:
		c := h.repo.GetContract(tenantID, contractID)
		if c == nil {
			httputil.WriteError(w, http.StatusNotFound, "CONTRACT_NOT_FOUND", "contract not found")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, c, tenantID)
	case http.MethodPut:
		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		c, err := h.repo.UpdateContractStatus(tenantID, contractID, req.Status)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "CONTRACT_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, c, tenantID)
	case http.MethodDelete:
		if err := h.repo.DeleteContract(tenantID, contractID); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "CONTRACT_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
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
	path := r.URL.Path
	for _, prefix := range []string{"/api/v1/dataquality/trends/", "/api/v1/data-quality/trends/"} {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}
	agentID := strings.Split(path, "/")[0]
	httputil.WriteJSON(w, http.StatusOK, h.repo.GetQualityTrend(tenantID, agentID, 30), tenantID)
}

func (h *Handler) handleIncidents(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		httputil.WriteJSON(w, http.StatusOK, h.repo.ListIncidents(tenantID, status), tenantID)
	case http.MethodPost:
		var req struct {
			AgentID      string   `json:"agent_id"`
			ContractID   string   `json:"contract_id"`
			Title        string   `json:"title"`
			Description  string   `json:"description"`
			Severity     Severity `json:"severity"`
			ViolationIDs []string `json:"violation_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Title == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "title is required")
			return
		}
		if req.Severity == "" {
			req.Severity = SeverityWarning
		}
		inc := h.repo.RecordIncident(tenantID, req.AgentID, req.ContractID, req.Title, req.Description, req.Severity, req.ViolationIDs)
		httputil.WriteJSON(w, http.StatusCreated, inc, tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleIncidentByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	if r.Method != http.MethodPut {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	path := r.URL.Path
	for _, prefix := range []string{"/api/v1/dataquality/incidents/", "/api/v1/data-quality/incidents/"} {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}
	incidentID := strings.Split(path, "/")[0]
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	inc, err := h.repo.UpdateIncidentStatus(tenantID, incidentID, req.Status)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "INCIDENT_NOT_FOUND", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, inc, tenantID)
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
	agentID := r.URL.Query().Get("agent_id")
	httputil.WriteJSON(w, http.StatusOK, h.repo.ListAnomalies(tenantID, agentID), tenantID)
}
