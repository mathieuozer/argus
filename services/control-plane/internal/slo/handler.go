package slo

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/tenancy"
)

// Handler provides REST handlers for SLO management.
type Handler struct {
	repo       Store
	calculator *Calculator
}

// NewHandler creates a new SLO handler.
func NewHandler(repo Store, calculator *Calculator) *Handler {
	return &Handler{repo: repo, calculator: calculator}
}

// RegisterRoutes registers all SLO API routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/slos", h.handleSLOs)
	mux.HandleFunc("/api/v1/slos/", h.handleSLOByID)
	mux.HandleFunc("/api/v1/slos/status", h.handleAllStatuses)
}

func (h *Handler) handleSLOs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		agentID := r.URL.Query().Get("agent_id")
		slos, err := h.repo.ListSLOs(r.Context(), tenantID, agentID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list SLOs")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, slos, tenantID)

	case http.MethodPost:
		var req struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			AgentID     string  `json:"agent_id"`
			Type        SLOType `json:"type"`
			Target      float64 `json:"target"`
			Window      string  `json:"window"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" || req.Type == "" || req.Target <= 0 {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name, type, and target (> 0) are required")
			return
		}
		slo, err := h.repo.CreateSLO(r.Context(), tenantID, req.Name, req.Description, req.AgentID, req.Type, req.Target, req.Window)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create SLO")
			return
		}
		httputil.WriteJSON(w, http.StatusCreated, slo, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleAllStatuses(w http.ResponseWriter, r *http.Request) {
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
	statuses, err := h.calculator.CalculateAllStatuses(r.Context(), tenantID, agentID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to calculate SLO statuses")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, statuses, tenantID)
}

func (h *Handler) handleSLOByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/slos/")
	// Avoid routing "status" as an SLO ID — that's handled by handleAllStatuses
	if path == "status" {
		h.handleAllStatuses(w, r)
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "SLO ID required")
		return
	}
	sloID := parts[0]

	// Handle status sub-resource: /api/v1/slos/{id}/status
	if len(parts) > 1 && parts[1] == "status" {
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		status, err := h.calculator.CalculateStatus(r.Context(), tenantID, sloID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to calculate SLO status")
			return
		}
		if status == nil {
			httputil.WriteError(w, http.StatusNotFound, "SLO_NOT_FOUND", "SLO not found")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, status, tenantID)
		return
	}

	// Handle budget sub-resource: /api/v1/slos/{id}/budget
	if len(parts) > 1 && parts[1] == "budget" {
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		sloObj, err := h.repo.GetSLO(r.Context(), tenantID, sloID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get SLO")
			return
		}
		if sloObj == nil {
			httputil.WriteError(w, http.StatusNotFound, "SLO_NOT_FOUND", "SLO not found")
			return
		}
		budget, err := h.calculator.CalculateErrorBudget(r.Context(), tenantID, sloID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to calculate error budget")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, budget, tenantID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		slo, err := h.repo.GetSLO(r.Context(), tenantID, sloID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get SLO")
			return
		}
		if slo == nil {
			httputil.WriteError(w, http.StatusNotFound, "SLO_NOT_FOUND", "SLO not found")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, slo, tenantID)

	case http.MethodPut:
		var req struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Target      float64 `json:"target"`
			Enabled     bool    `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		updated, err := h.repo.UpdateSLO(r.Context(), tenantID, sloID, req.Name, req.Description, req.Target, req.Enabled)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "SLO_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, updated, tenantID)

	case http.MethodDelete:
		if err := h.repo.DeleteSLO(r.Context(), tenantID, sloID); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "SLO_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}
