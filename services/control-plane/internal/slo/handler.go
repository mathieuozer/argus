package slo

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/argus-platform/argus/pkg/tenancy"
)

// Handler provides REST handlers for SLO management.
type Handler struct {
	repo       *Repository
	calculator *Calculator
}

// NewHandler creates a new SLO handler.
func NewHandler(repo *Repository, calculator *Calculator) *Handler {
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
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		agentID := r.URL.Query().Get("agent_id")
		slos := h.repo.ListSLOs(tenantID, agentID)
		writeJSON(w, http.StatusOK, slos, tenantID)

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
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" || req.Type == "" || req.Target <= 0 {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name, type, and target (> 0) are required")
			return
		}
		slo := h.repo.CreateSLO(tenantID, req.Name, req.Description, req.AgentID, req.Type, req.Target, req.Window)
		writeJSON(w, http.StatusCreated, slo, tenantID)

	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleAllStatuses(w http.ResponseWriter, r *http.Request) {
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
	statuses := h.calculator.CalculateAllStatuses(tenantID, agentID)
	writeJSON(w, http.StatusOK, statuses, tenantID)
}

func (h *Handler) handleSLOByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
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
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "SLO ID required")
		return
	}
	sloID := parts[0]

	// Handle status sub-resource: /api/v1/slos/{id}/status
	if len(parts) > 1 && parts[1] == "status" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		status := h.calculator.CalculateStatus(tenantID, sloID)
		if status == nil {
			writeError(w, http.StatusNotFound, "SLO_NOT_FOUND", "SLO not found")
			return
		}
		writeJSON(w, http.StatusOK, status, tenantID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		slo := h.repo.GetSLO(tenantID, sloID)
		if slo == nil {
			writeError(w, http.StatusNotFound, "SLO_NOT_FOUND", "SLO not found")
			return
		}
		writeJSON(w, http.StatusOK, slo, tenantID)

	case http.MethodPut:
		var req struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Target      float64 `json:"target"`
			Enabled     bool    `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		updated, err := h.repo.UpdateSLO(tenantID, sloID, req.Name, req.Description, req.Target, req.Enabled)
		if err != nil {
			writeError(w, http.StatusNotFound, "SLO_NOT_FOUND", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated, tenantID)

	case http.MethodDelete:
		if err := h.repo.DeleteSLO(tenantID, sloID); err != nil {
			writeError(w, http.StatusNotFound, "SLO_NOT_FOUND", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)

	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
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
