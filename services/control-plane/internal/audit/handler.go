package audit

import (
	"net/http"
	"strconv"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/tenancy"
)

// Handler provides REST handlers for querying audit logs.
type Handler struct {
	store Store
}

// NewHandler creates a new audit handler backed by a Store.
func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes registers audit API routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/audit/logs", h.handleLogs)
	mux.HandleFunc("/api/v1/audit/search", h.handleSearch)
	mux.HandleFunc("/api/v1/audit/stats", h.handleStats)
}

func (h *Handler) handleLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	entries, err := h.store.List(r.Context(), tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list audit logs")
		return
	}

	// Apply optional pagination
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Apply offset
	if offset >= len(entries) {
		entries = nil
	} else if offset > 0 {
		entries = entries[offset:]
	}

	// Apply limit
	if entries != nil && len(entries) > limit {
		entries = entries[:limit]
	}

	httputil.WriteJSON(w, http.StatusOK, entries, tenantID)
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	actor := r.URL.Query().Get("actor")
	action := r.URL.Query().Get("action")
	resource := r.URL.Query().Get("resource")

	filtered, err := h.store.Search(r.Context(), tenantID, actor, action, resource)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to search audit logs")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, filtered, tenantID)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	allEntries, err := h.store.List(r.Context(), tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get audit stats")
		return
	}

	byAction := make(map[string]int)
	byActor := make(map[string]int)
	for _, e := range allEntries {
		byAction[e.Action]++
		byActor[e.Actor]++
	}

	stats := map[string]interface{}{
		"total_entries":  len(allEntries),
		"by_action":      byAction,
		"by_actor":       byActor,
		"unique_actions": len(byAction),
		"unique_actors":  len(byActor),
	}

	httputil.WriteJSON(w, http.StatusOK, stats, tenantID)
}
