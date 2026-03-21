package catalog

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/argus-platform/argus/pkg/tenancy"
)

// Handler provides REST handlers for the data catalog.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new catalog handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes registers all catalog API routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/catalog/sources", h.handleSources)
	mux.HandleFunc("/api/v1/catalog/sources/", h.handleSourceByID)
	mux.HandleFunc("/api/v1/catalog/lineage", h.handleLineageEdges)
}

func (h *Handler) handleSources(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		srcType := SourceType(r.URL.Query().Get("type"))
		tag := r.URL.Query().Get("tag")
		sources := h.repo.ListSources(tenantID, srcType, tag)
		writeJSON(w, http.StatusOK, sources, tenantID)

	case http.MethodPost:
		var req struct {
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Type        SourceType        `json:"type"`
			Owner       string            `json:"owner"`
			AgentID     string            `json:"agent_id"`
			Tags        []string          `json:"tags"`
			Schema      map[string]string `json:"schema"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" || req.Type == "" {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and type are required")
			return
		}
		src := h.repo.CreateSource(tenantID, req.Name, req.Description, req.Type, req.Owner, req.AgentID, req.Tags, req.Schema)
		writeJSON(w, http.StatusCreated, src, tenantID)

	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleSourceByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/catalog/sources/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "source ID required")
		return
	}
	sourceID := parts[0]

	// Handle lineage sub-resource: /api/v1/catalog/sources/{id}/lineage
	if len(parts) > 1 && parts[1] == "lineage" {
		h.handleSourceLineage(w, r, tenantID, sourceID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		src := h.repo.GetSource(tenantID, sourceID)
		if src == nil {
			writeError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", "source not found")
			return
		}
		writeJSON(w, http.StatusOK, src, tenantID)

	case http.MethodPut:
		var req struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Owner       string   `json:"owner"`
			Tags        []string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		updated, err := h.repo.UpdateSource(tenantID, sourceID, req.Name, req.Description, req.Owner, req.Tags)
		if err != nil {
			writeError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, updated, tenantID)

	case http.MethodDelete:
		if err := h.repo.DeleteSource(tenantID, sourceID); err != nil {
			writeError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)

	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleSourceLineage(w http.ResponseWriter, r *http.Request, tenantID, sourceID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	graph := h.repo.GetLineage(tenantID, sourceID)
	if graph == nil {
		writeError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", "source not found")
		return
	}
	writeJSON(w, http.StatusOK, graph, tenantID)
}

func (h *Handler) handleLineageEdges(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		edges := h.repo.ListEdges(tenantID)
		writeJSON(w, http.StatusOK, edges, tenantID)

	case http.MethodPost:
		var req struct {
			SourceID      string `json:"source_id"`
			TargetID      string `json:"target_id"`
			TransformType string `json:"transform_type"`
			AgentID       string `json:"agent_id"`
			Description   string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.SourceID == "" || req.TargetID == "" {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "source_id and target_id are required")
			return
		}
		if req.TransformType == "" {
			req.TransformType = "copy"
		}
		edge, err := h.repo.AddLineageEdge(tenantID, req.SourceID, req.TargetID, req.TransformType, req.AgentID, req.Description)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, edge, tenantID)

	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
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
