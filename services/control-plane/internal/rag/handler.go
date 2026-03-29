package rag

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/pkg/tenancy"
)

type Retrieval struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	AgentID      string    `json:"agent_id"`
	SpanID       string    `json:"span_id"`
	Query        string    `json:"query"`
	NumChunks    int       `json:"num_chunks"`
	AvgRelevance float64   `json:"avg_relevance"`
	LatencyMs    int64     `json:"latency_ms"`
	SourceIDs    []string  `json:"source_ids"`
	CreatedAt    time.Time `json:"created_at"`
}

type Source struct {
	ID           string  `json:"id"`
	TenantID     string  `json:"tenant_id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"` // document, database, api
	TotalChunks  int     `json:"total_chunks"`
	AvgRelevance float64 `json:"avg_relevance"`
	UsageCount   int     `json:"usage_count"`
}

type QualityTrend struct {
	Timestamp    time.Time `json:"timestamp"`
	AvgRelevance float64   `json:"avg_relevance"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	TotalQueries int       `json:"total_queries"`
}

type Repository struct {
	mu         sync.RWMutex
	retrievals []*Retrieval
	sources    map[string]*Source
}

func NewRepository() *Repository {
	return &Repository{
		retrievals: make([]*Retrieval, 0),
		sources:    make(map[string]*Source),
	}
}

// AddRetrieval adds a retrieval record (used for seeding).
func (r *Repository) AddRetrieval(ret *Retrieval) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retrievals = append(r.retrievals, ret)
}

// AddSource adds a source record (used for seeding).
func (r *Repository) AddSource(src *Source) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[src.ID] = src
}

type Handler struct {
	repo   *Repository
	pgRepo *PGRepository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// SetPG attaches a PostgreSQL repository for dual-write persistence.
func (h *Handler) SetPG(pg *PGRepository) {
	h.pgRepo = pg
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/rag/retrievals", h.ListRetrievals)
	mux.HandleFunc("POST /api/v1/rag/retrievals", h.CreateRetrieval)
	mux.HandleFunc("GET /api/v1/rag/sources", h.ListSources)
	mux.HandleFunc("POST /api/v1/rag/sources", h.CreateSource)
	mux.HandleFunc("GET /api/v1/rag/quality", h.GetQuality)
}

func (h *Handler) CreateRetrieval(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}

	var req struct {
		AgentID      string   `json:"agent_id"`
		SpanID       string   `json:"span_id"`
		Query        string   `json:"query"`
		NumChunks    int      `json:"num_chunks"`
		AvgRelevance float64  `json:"avg_relevance"`
		LatencyMs    int64    `json:"latency_ms"`
		SourceIDs    []string `json:"source_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	ret := &Retrieval{
		ID:           fmt.Sprintf("ret-%d", time.Now().UnixNano()),
		TenantID:     tenantID,
		AgentID:      req.AgentID,
		SpanID:       req.SpanID,
		Query:        req.Query,
		NumChunks:    req.NumChunks,
		AvgRelevance: req.AvgRelevance,
		LatencyMs:    req.LatencyMs,
		SourceIDs:    req.SourceIDs,
		CreatedAt:    time.Now(),
	}
	h.repo.AddRetrieval(ret)

	if h.pgRepo != nil {
		if err := h.pgRepo.SaveRetrieval(r.Context(), ret); err != nil {
			logger.FromContext(r.Context()).Error("pg dual-write SaveRetrieval failed: " + err.Error())
		}
	}

	httputil.WriteJSON(w, http.StatusCreated, ret, "")
}

func (h *Handler) CreateSource(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}

	var req struct {
		Name         string  `json:"name"`
		Type         string  `json:"type"`
		TotalChunks  int     `json:"total_chunks"`
		AvgRelevance float64 `json:"avg_relevance"`
		UsageCount   int     `json:"usage_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	src := &Source{
		ID:           fmt.Sprintf("src-%d", time.Now().UnixNano()),
		TenantID:     tenantID,
		Name:         req.Name,
		Type:         req.Type,
		TotalChunks:  req.TotalChunks,
		AvgRelevance: req.AvgRelevance,
		UsageCount:   req.UsageCount,
	}
	h.repo.AddSource(src)

	if h.pgRepo != nil {
		if err := h.pgRepo.SaveSource(r.Context(), src); err != nil {
			logger.FromContext(r.Context()).Error("pg dual-write SaveSource failed: " + err.Error())
		}
	}

	httputil.WriteJSON(w, http.StatusCreated, src, "")
}

func (h *Handler) ListRetrievals(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}
	agentID := r.URL.Query().Get("agent_id")

	h.repo.mu.RLock()
	var retrievals []*Retrieval
	for _, ret := range h.repo.retrievals {
		if ret.TenantID == tenantID {
			if agentID == "" || ret.AgentID == agentID {
				retrievals = append(retrievals, ret)
			}
		}
	}
	h.repo.mu.RUnlock()

	if retrievals == nil {
		retrievals = []*Retrieval{}
	}
	httputil.WriteJSON(w, http.StatusOK, retrievals, "")
}

func (h *Handler) ListSources(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}

	h.repo.mu.RLock()
	var sources []*Source
	for _, s := range h.repo.sources {
		if s.TenantID == tenantID {
			sources = append(sources, s)
		}
	}
	h.repo.mu.RUnlock()

	if sources == nil {
		sources = []*Source{}
	}
	httputil.WriteJSON(w, http.StatusOK, sources, "")
}

func (h *Handler) GetQuality(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}

	h.repo.mu.RLock()
	var trends []QualityTrend
	// Generate trend data from retrievals
	relevanceSum := 0.0
	latencySum := int64(0)
	count := 0
	for _, ret := range h.repo.retrievals {
		if ret.TenantID == tenantID {
			relevanceSum += ret.AvgRelevance
			latencySum += ret.LatencyMs
			count++
		}
	}
	h.repo.mu.RUnlock()

	if count > 0 {
		trends = append(trends, QualityTrend{
			Timestamp:    time.Now(),
			AvgRelevance: relevanceSum / float64(count),
			AvgLatencyMs: float64(latencySum) / float64(count),
			TotalQueries: count,
		})
	}

	if trends == nil {
		trends = []QualityTrend{}
	}
	httputil.WriteJSON(w, http.StatusOK, trends, "")
}
