package dashboard

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/control-plane/internal/alerts"
	"github.com/argus-platform/argus/services/control-plane/internal/audit"
)

// alertItem is the response DTO matching the dashboard PredictiveAlert contract.
type alertItem struct {
	ID            string   `json:"id"`
	TenantID      string   `json:"tenant_id"`
	AgentID       string   `json:"agent_id"`
	Probability   float64  `json:"probability"`
	EstimatedTTF  int      `json:"estimated_ttf"`
	PrecursorType string   `json:"precursor_type"`
	Evidence      []string `json:"evidence"`
	Status        string   `json:"status"`
	CreatedAt     string   `json:"created_at"`
	Severity      string   `json:"severity"`
	Title         string   `json:"title"`
}

func toAlertItem(a *alerts.Alert) alertItem {
	// Map severity to predictive failure probability and TTF
	prob := 0.3
	ttf := 1800
	precursor := "anomaly"
	switch a.Severity {
	case alerts.SeverityCritical:
		prob = 0.92
		ttf = 180
		precursor = "token_escalation"
	case alerts.SeverityWarning:
		prob = 0.65
		ttf = 600
		precursor = "latency_spike"
	case alerts.SeverityInfo:
		prob = 0.15
		ttf = 3600
		precursor = "info"
	}

	evidence := []string{}
	if a.Message != "" {
		evidence = append(evidence, a.Message)
	}

	return alertItem{
		ID:            a.ID,
		TenantID:      a.TenantID,
		AgentID:       a.AgentID,
		Probability:   prob,
		EstimatedTTF:  ttf,
		PrecursorType: precursor,
		Evidence:      evidence,
		Status:        string(a.Status),
		CreatedAt:     a.CreatedAt.Format(time.RFC3339),
		Severity:      string(a.Severity),
		Title:         a.Title,
	}
}

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
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		alertList := h.alerts.List(tenantID)
		items := make([]alertItem, 0, len(alertList))
		for _, a := range alertList {
			items = append(items, toAlertItem(a))
		}
		httputil.WriteJSON(w, http.StatusOK, items, tenantID)
	case http.MethodPost:
		var req struct {
			AgentID  string          `json:"agent_id"`
			Severity alerts.Severity `json:"severity"`
			Title    string          `json:"title"`
			Message  string          `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		alert := h.alerts.Fire(tenantID, req.AgentID, req.Severity, req.Title, req.Message)
		h.auditLog.Write(tenantID, "system", "create_alert", "alert/"+alert.ID, req.Title)
		httputil.WriteJSON(w, http.StatusCreated, toAlertItem(alert), tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleAlertByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	// Extract alert ID from path: /api/v1/alerts/{id}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "alert ID required")
		return
	}
	alertID := parts[0]

	switch r.Method {
	case http.MethodPut, http.MethodPatch:
		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		updated, err := h.alerts.UpdateStatus(tenantID, alertID, req.Status)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "ALERT_NOT_FOUND", err.Error())
			return
		}
		h.auditLog.Write(tenantID, "system", "update_alert", "alert/"+alertID, "status: "+req.Status)
		httputil.WriteJSON(w, http.StatusOK, toAlertItem(updated), tenantID)
	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	entries := h.auditLog.List(tenantID)
	httputil.WriteJSON(w, http.StatusOK, entries, tenantID)
}
