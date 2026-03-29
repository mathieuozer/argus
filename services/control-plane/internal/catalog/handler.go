package catalog

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/tenancy"
)

// Handler provides REST handlers for the data catalog.
type Handler struct {
	repo Store
}

// NewHandler creates a new catalog handler.
func NewHandler(store Store) *Handler {
	return &Handler{repo: store}
}

// RegisterRoutes registers all catalog API routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/catalog/sources", h.handleSources)
	mux.HandleFunc("/api/v1/catalog/sources/", h.handleSourceByID)
	mux.HandleFunc("/api/v1/catalog/lineage", h.handleLineageEdges)
	mux.HandleFunc("/api/v1/catalog/lineage/graph", h.handleLineageEdges)
	mux.HandleFunc("/api/v1/catalog/graph", h.handleFullGraph)
	mux.HandleFunc("/api/v1/catalog/tools", h.handleTools)
	mux.HandleFunc("/api/v1/catalog/search", h.handleSearch)
	mux.HandleFunc("/api/v1/catalog/stats", h.handleStats)
	mux.HandleFunc("/api/v1/catalog/glossary", h.handleGlossary)
	mux.HandleFunc("/api/v1/catalog/glossary/", h.handleGlossaryByID)
}

func (h *Handler) handleSources(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		srcType := SourceType(r.URL.Query().Get("type"))
		domain := r.URL.Query().Get("domain")
		status := r.URL.Query().Get("status")
		classification := r.URL.Query().Get("classification")
		tag := r.URL.Query().Get("tag")

		if domain != "" || status != "" || classification != "" {
			sources, err := h.repo.ListSourcesFiltered(r.Context(), tenantID, srcType, domain, status, classification)
			if err != nil {
				httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
				return
			}
			httputil.WriteJSON(w, http.StatusOK, sources, tenantID)
		} else {
			sources, err := h.repo.ListSources(r.Context(), tenantID, srcType, tag)
			if err != nil {
				httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
				return
			}
			httputil.WriteJSON(w, http.StatusOK, sources, tenantID)
		}

	case http.MethodPost:
		var req struct {
			Name           string            `json:"name"`
			Description    string            `json:"description"`
			Type           SourceType        `json:"type"`
			Owner          string            `json:"owner"`
			AgentID        string            `json:"agent_id"`
			Tags           []string          `json:"tags"`
			Schema         map[string]string `json:"schema"`
			Classification string            `json:"classification"`
			Domain         string            `json:"domain"`
			Status         string            `json:"status"`
			Steward        string            `json:"steward"`
			QualityScore   float64           `json:"quality_score"`
			Columns        []*Column         `json:"columns"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Name == "" || req.Type == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and type are required")
			return
		}
		src, err := h.repo.CreateSourceFull(r.Context(), tenantID, req.Name, req.Description, req.Type, req.Owner, req.AgentID, req.Tags, req.Schema, req.Classification, req.Domain, req.Status, req.Steward, req.QualityScore, nil, nil, req.Columns)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusCreated, src, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleSourceByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/catalog/sources/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "source ID required")
		return
	}
	sourceID := parts[0]

	// Sub-resources
	if len(parts) > 1 {
		switch parts[1] {
		case "lineage":
			h.handleSourceLineage(w, r, tenantID, sourceID)
			return
		case "impact":
			h.handleImpact(w, r, tenantID, sourceID)
			return
		case "columns":
			h.handleColumns(w, r, tenantID, sourceID)
			return
		case "profile":
			h.handleProfile(w, r, tenantID, sourceID)
			return
		case "column-lineage":
			h.handleColumnLineage(w, r, tenantID, sourceID)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		src, err := h.repo.GetSource(r.Context(), tenantID, sourceID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		if src == nil {
			httputil.WriteError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", "source not found")
			return
		}
		h.repo.TrackPopularity(r.Context(), tenantID, sourceID, "view")
		httputil.WriteJSON(w, http.StatusOK, src, tenantID)

	case http.MethodPut:
		var req struct {
			Name           string   `json:"name"`
			Description    string   `json:"description"`
			Owner          string   `json:"owner"`
			Tags           []string `json:"tags"`
			Classification string   `json:"classification"`
			Domain         string   `json:"domain"`
			Status         string   `json:"status"`
			Steward        string   `json:"steward"`
			QualityScore   float64  `json:"quality_score"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		updated, err := h.repo.UpdateSource(r.Context(), tenantID, sourceID, req.Name, req.Description, req.Owner, req.Tags)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", err.Error())
			return
		}
		if req.Classification != "" || req.Domain != "" || req.Status != "" || req.Steward != "" || req.QualityScore > 0 {
			updated, _ = h.repo.UpdateSourceExtended(r.Context(), tenantID, sourceID, req.Classification, req.Domain, req.Status, req.Steward, req.QualityScore)
		}
		httputil.WriteJSON(w, http.StatusOK, updated, tenantID)

	case http.MethodDelete:
		if err := h.repo.DeleteSource(r.Context(), tenantID, sourceID); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleSourceLineage(w http.ResponseWriter, r *http.Request, tenantID, sourceID string) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	graph, err := h.repo.GetLineage(r.Context(), tenantID, sourceID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if graph == nil {
		httputil.WriteError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", "source not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, graph, tenantID)
}

func (h *Handler) handleImpact(w http.ResponseWriter, r *http.Request, tenantID, sourceID string) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	ia, err := h.repo.GetImpactAnalysis(r.Context(), tenantID, sourceID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if ia == nil {
		httputil.WriteError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", "source not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, ia, tenantID)
}

func (h *Handler) handleColumns(w http.ResponseWriter, r *http.Request, tenantID, sourceID string) {
	switch r.Method {
	case http.MethodGet:
		src, err := h.repo.GetSource(r.Context(), tenantID, sourceID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		if src == nil {
			httputil.WriteError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", "source not found")
			return
		}
		cols := src.Columns
		if cols == nil {
			cols = []*Column{}
		}
		httputil.WriteJSON(w, http.StatusOK, cols, tenantID)

	case http.MethodPost:
		var req struct {
			Columns []*Column `json:"columns"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if err := h.repo.SetSourceColumns(r.Context(), tenantID, sourceID, req.Columns); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"}, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleProfile(w http.ResponseWriter, r *http.Request, tenantID, sourceID string) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	src, err := h.repo.GetSource(r.Context(), tenantID, sourceID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if src == nil {
		httputil.WriteError(w, http.StatusNotFound, "SOURCE_NOT_FOUND", "source not found")
		return
	}
	if src.Profile == nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{}, tenantID)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, src.Profile, tenantID)
}

func (h *Handler) handleColumnLineage(w http.ResponseWriter, r *http.Request, tenantID, sourceID string) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	columnName := r.URL.Query().Get("column")
	entries, err := h.repo.GetColumnLineage(r.Context(), tenantID, sourceID, columnName)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if entries == nil {
		entries = []*ColumnLineageEntry{}
	}
	httputil.WriteJSON(w, http.StatusOK, entries, tenantID)
}

func (h *Handler) handleFullGraph(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	graph, err := h.repo.GetFullLineageGraph(r.Context(), tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, graph, tenantID)
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
	q := r.URL.Query().Get("q")
	if q == "" {
		httputil.WriteJSON(w, http.StatusOK, []*SearchResult{}, tenantID)
		return
	}
	filters := map[string]string{
		"type":           r.URL.Query().Get("type"),
		"domain":         r.URL.Query().Get("domain"),
		"classification": r.URL.Query().Get("classification"),
	}
	results, err := h.repo.SearchCatalog(r.Context(), tenantID, q, filters)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if results == nil {
		results = []*SearchResult{}
	}
	httputil.WriteJSON(w, http.StatusOK, results, tenantID)
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
	stats, err := h.repo.GetCatalogStats(r.Context(), tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, stats, tenantID)
}

func (h *Handler) handleGlossary(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		domain := r.URL.Query().Get("domain")
		terms, err := h.repo.ListGlossaryTerms(r.Context(), tenantID, domain)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, terms, tenantID)

	case http.MethodPost:
		var req struct {
			Term         string   `json:"term"`
			Definition   string   `json:"definition"`
			Domain       string   `json:"domain"`
			Owner        string   `json:"owner"`
			RelatedTerms []string `json:"related_terms"`
			LinkedAssets []string `json:"linked_assets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.Term == "" || req.Definition == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "term and definition are required")
			return
		}
		gt, err := h.repo.CreateGlossaryTerm(r.Context(), tenantID, req.Term, req.Definition, req.Domain, req.Owner, req.RelatedTerms, req.LinkedAssets)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusCreated, gt, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleGlossaryByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	termID := strings.TrimPrefix(r.URL.Path, "/api/v1/catalog/glossary/")
	if termID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "term ID required")
		return
	}
	switch r.Method {
	case http.MethodPut:
		var req struct {
			Term       string `json:"term"`
			Definition string `json:"definition"`
			Domain     string `json:"domain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		gt, err := h.repo.UpdateGlossaryTerm(r.Context(), tenantID, termID, req.Term, req.Definition, req.Domain)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "TERM_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, gt, tenantID)

	case http.MethodDelete:
		if err := h.repo.DeleteGlossaryTerm(r.Context(), tenantID, termID); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "TERM_NOT_FOUND", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"}, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleLineageEdges(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		edges, err := h.repo.ListEdges(r.Context(), tenantID)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusOK, edges, tenantID)

	case http.MethodPost:
		var req struct {
			SourceID      string `json:"source_id"`
			TargetID      string `json:"target_id"`
			TransformType string `json:"transform_type"`
			AgentID       string `json:"agent_id"`
			Description   string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}
		if req.SourceID == "" || req.TargetID == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "source_id and target_id are required")
			return
		}
		if req.TransformType == "" {
			req.TransformType = "copy"
		}
		edge, err := h.repo.AddLineageEdge(r.Context(), tenantID, req.SourceID, req.TargetID, req.TransformType, req.AgentID, req.Description)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		httputil.WriteJSON(w, http.StatusCreated, edge, tenantID)

	default:
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) handleTools(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant context required")
		return
	}
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, []interface{}{}, tenantID)
}
