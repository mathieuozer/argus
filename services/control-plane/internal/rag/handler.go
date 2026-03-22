package rag

import (
	"net/http"
	"sync"
	"time"

	"github.com/argus-platform/argus/pkg/httputil"
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
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/rag/retrievals", h.ListRetrievals)
	mux.HandleFunc("GET /api/v1/rag/sources", h.ListSources)
	mux.HandleFunc("GET /api/v1/rag/quality", h.GetQuality)
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
