package trace

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/tenancy"
)

// Handler provides REST handlers for distributed trace data.
type Handler struct {
	service *Service
}

// NewHandler creates a new trace handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{service: svc}
}

// RegisterRoutes registers all trace API routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/traces", h.handleTraces)
	mux.HandleFunc("/api/v1/traces/", h.handleTraceByID)
}

func (h *Handler) handleTraces(w http.ResponseWriter, r *http.Request) {
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
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	traces := h.service.ListTraces(tenantID, agentID, limit)
	httputil.WriteJSON(w, http.StatusOK, traces, tenantID)
}

func (h *Handler) handleTraceByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/traces/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "trace ID required")
		return
	}
	traceID := parts[0]

	// Handle flamegraph sub-resource: /api/v1/traces/{id}/flamegraph
	if len(parts) > 1 && parts[1] == "flamegraph" {
		fg := h.service.GetFlameGraph(tenantID, traceID)
		if fg == nil {
			httputil.WriteError(w, http.StatusNotFound, "TRACE_NOT_FOUND", "trace not found")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, fg, tenantID)
		return
	}

	detail := h.service.GetTrace(tenantID, traceID)
	if detail == nil {
		httputil.WriteError(w, http.StatusNotFound, "TRACE_NOT_FOUND", "trace not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, detail, tenantID)
}
