package dashboard

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/control-plane/internal/alerts"
	"github.com/argus-platform/argus/services/control-plane/internal/audit"
)

// Handler provides REST handlers for the dashboard frontend.
type Handler struct {
	alerts   *alerts.Router
	auditLog *audit.Writer
}

// New creates a new dashboard handler.
func New(alertRouter *alerts.Router, auditLog *audit.Writer) *Handler {
	return &Handler{
		alerts:   alertRouter,
		auditLog: auditLog,
	}
}

// RegisterRoutes registers all dashboard API routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/alerts", h.handleAlerts)
	mux.HandleFunc("/api/v1/alerts/", h.handleAlertByID)
	mux.HandleFunc("/api/v1/audit", h.handleAuditLogs)
}

func (h *Handler) handleAlerts(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		alertList := h.alerts.List(tenantID)
		writeJSON(w, http.StatusOK, alertList, tenantID)
	case http.MethodPost:
		var req struct {
			AgentID  string          `json:"agent_id"`
			Severity alerts.Severity `json:"severity"`
			Title    string          `json:"title"`
			Message  string          `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		alert := h.alerts.Fire(tenantID, req.AgentID, req.Severity, req.Title, req.Message)
		h.auditLog.Write(tenantID, "system", "create_alert", "alert/"+alert.ID, req.Title)
		writeJSON(w, http.StatusCreated, alert, tenantID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleAlertByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	// Extract alert ID from path: /api/v1/alerts/{id}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "alert ID required")
		return
	}
	alertID := parts[0]

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		updated, err := h.alerts.UpdateStatus(tenantID, alertID, req.Status)
		if err != nil {
			writeError(w, http.StatusNotFound, "ALERT_NOT_FOUND", err.Error())
			return
		}
		h.auditLog.Write(tenantID, "system", "update_alert", "alert/"+alertID, "status: "+req.Status)
		writeJSON(w, http.StatusOK, updated, tenantID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	entries := h.auditLog.List(tenantID)
	writeJSON(w, http.StatusOK, entries, tenantID)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}, tenantID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": data,
		"meta": map[string]string{"tenant_id": tenantID},
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
