package governance

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/tenancy"
)

// Handler provides REST handlers for data governance.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new governance handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes registers all governance API routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/governance/summary", h.handleSummary)
	mux.HandleFunc("/api/v1/governance/classification-policies", h.handleClassPolicies)
	mux.HandleFunc("/api/v1/governance/classification-policies/", h.handleClassPolicyByID)
	mux.HandleFunc("/api/v1/governance/retention-policies", h.handleRetPolicies)
	mux.HandleFunc("/api/v1/governance/retention-policies/", h.handleRetPolicyByID)
	mux.HandleFunc("/api/v1/governance/access-logs", h.handleAccessLogs)
	mux.HandleFunc("/api/v1/governance/pii-scans", h.handlePIIScans)
	mux.HandleFunc("/api/v1/governance/pii-scans/", h.handlePIIScanByID)
	mux.HandleFunc("/api/v1/governance/compliance-mappings", h.handleComplianceMappings)
	mux.HandleFunc("/api/v1/governance/compliance-mappings/", h.handleComplianceMappingByID)
	mux.HandleFunc("/api/v1/governance/stewards", h.handleStewards)
	mux.HandleFunc("/api/v1/governance/stewards/", h.handleStewardByID)
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

func (h *Handler) handleClassPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		httputil.WriteJSON(w, http.StatusOK, h.repo.ListClassificationPolicies(tenantID), tenantID)
	case http.MethodPost:
		var req struct {
			Name           string `json:"name"`
			Description    string `json:"description"`
			MatchPattern   string `json:"match_pattern"`
			MatchType      string `json:"match_type"`
			Classification string `json:"classification"`
			AutoApply      bool   `json:"auto_apply"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" || req.Classification == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and classification are required")
			return
		}
		p := h.repo.CreateClassificationPolicy(tenantID, req.Name, req.Description, req.MatchPattern, req.MatchType, req.Classification, req.AutoApply)
		httputil.WriteJSON(w, http.StatusCreated, p, tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleClassPolicyByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/governance/classification-policies/")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "policy ID required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		p := h.repo.GetClassificationPolicy(tenantID, id)
		if p == nil {
			httputil.WriteError(w, http.StatusNotFound, "POLICY_NOT_FOUND", "policy not found")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, p, tenantID)
	case http.MethodDelete:
		if err := h.repo.DeleteClassificationPolicy(tenantID, id); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "POLICY_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleRetPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		httputil.WriteJSON(w, http.StatusOK, h.repo.ListRetentionPolicies(tenantID), tenantID)
	case http.MethodPost:
		var req struct {
			Name           string `json:"name"`
			Description    string `json:"description"`
			Classification string `json:"classification"`
			RetentionDays  int    `json:"retention_days"`
			Action         string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" || req.Classification == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and classification are required")
			return
		}
		if req.Action == "" {
			req.Action = "archive"
		}
		p := h.repo.CreateRetentionPolicy(tenantID, req.Name, req.Description, req.Classification, req.RetentionDays, req.Action)
		httputil.WriteJSON(w, http.StatusCreated, p, tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleRetPolicyByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/governance/retention-policies/")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "policy ID required")
		return
	}
	switch r.Method {
	case http.MethodDelete:
		if err := h.repo.DeleteRetentionPolicy(tenantID, id); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "POLICY_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleAccessLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	sourceID := r.URL.Query().Get("source_id")
	agentID := r.URL.Query().Get("agent_id")
	httputil.WriteJSON(w, http.StatusOK, h.repo.ListAccessLogs(tenantID, sourceID, agentID), tenantID)
}

func (h *Handler) handlePIIScans(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		httputil.WriteJSON(w, http.StatusOK, h.repo.ListPIIScans(tenantID), tenantID)
	case http.MethodPost:
		var req struct {
			SourceID    string      `json:"source_id"`
			SourceName  string      `json:"source_name"`
			PIIFields   []*PIIField `json:"pii_fields"`
			TotalFields int         `json:"total_fields"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		scan := h.repo.RecordPIIScan(tenantID, req.SourceID, req.SourceName, req.PIIFields, req.TotalFields)
		httputil.WriteJSON(w, http.StatusCreated, scan, tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handlePIIScanByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/governance/pii-scans/")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "scan ID required")
		return
	}
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	scan := h.repo.GetPIIScan(tenantID, id)
	if scan == nil {
		httputil.WriteError(w, http.StatusNotFound, "SCAN_NOT_FOUND", "PII scan not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, scan, tenantID)
}

func (h *Handler) handleComplianceMappings(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		framework := r.URL.Query().Get("framework")
		status := r.URL.Query().Get("status")
		httputil.WriteJSON(w, http.StatusOK, h.repo.ListComplianceMappings(tenantID, framework, status), tenantID)
	case http.MethodPost:
		var req struct {
			SourceID    string   `json:"source_id"`
			SourceName  string   `json:"source_name"`
			Framework   string   `json:"framework"`
			ArticleRef  string   `json:"article_ref"`
			Requirement string   `json:"requirement"`
			Status      string   `json:"status"`
			Evidence    []string `json:"evidence"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Framework == "" || req.ArticleRef == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "framework and article_ref are required")
			return
		}
		if req.Status == "" {
			req.Status = "not_assessed"
		}
		cm := h.repo.CreateComplianceMapping(tenantID, req.SourceID, req.SourceName, req.Framework, req.ArticleRef, req.Requirement, req.Status, req.Evidence)
		httputil.WriteJSON(w, http.StatusCreated, cm, tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleComplianceMappingByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/governance/compliance-mappings/")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "mapping ID required")
		return
	}
	if r.Method != http.MethodPut {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	var req struct {
		Status   string   `json:"status"`
		Evidence []string `json:"evidence"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}
	cm, err := h.repo.UpdateComplianceMappingStatus(tenantID, id, req.Status, req.Evidence)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "MAPPING_NOT_FOUND", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, cm, tenantID)
}

func (h *Handler) handleStewards(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		httputil.WriteJSON(w, http.StatusOK, h.repo.ListStewards(tenantID), tenantID)
	case http.MethodPost:
		var req struct {
			UserID    string   `json:"user_id"`
			Name      string   `json:"name"`
			Email     string   `json:"email"`
			Domains   []string `json:"domains"`
			SourceIDs []string `json:"source_ids"`
			Role      string   `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" || req.Email == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and email are required")
			return
		}
		if req.Role == "" {
			req.Role = "steward"
		}
		ds := h.repo.CreateSteward(tenantID, req.UserID, req.Name, req.Email, req.Domains, req.SourceIDs, req.Role)
		httputil.WriteJSON(w, http.StatusCreated, ds, tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleStewardByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/governance/stewards/")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "steward ID required")
		return
	}
	if r.Method != http.MethodDelete {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	if err := h.repo.DeleteSteward(tenantID, id); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "STEWARD_NOT_FOUND", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)
}
