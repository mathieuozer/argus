package audit

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/argus-platform/argus/pkg/tenancy"
)

// Handler provides REST handlers for querying audit logs.
type Handler struct {
	writer *Writer
}

// NewHandler creates a new audit handler.
func NewHandler(writer *Writer) *Handler {
	return &Handler{writer: writer}
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
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	entries := h.writer.List(tenantID)

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

	writeJSON(w, http.StatusOK, entries, tenantID)
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	// Filter parameters
	actor := r.URL.Query().Get("actor")
	action := r.URL.Query().Get("action")
	resource := r.URL.Query().Get("resource")

	allEntries := h.writer.List(tenantID)
	var filtered []*Entry

	for _, e := range allEntries {
		if actor != "" && e.Actor != actor {
			continue
		}
		if action != "" && e.Action != action {
			continue
		}
		if resource != "" && e.Resource != resource {
			continue
		}
		filtered = append(filtered, e)
	}

	writeJSON(w, http.StatusOK, filtered, tenantID)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	allEntries := h.writer.List(tenantID)

	// Aggregate statistics
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

	writeJSON(w, http.StatusOK, stats, tenantID)
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
