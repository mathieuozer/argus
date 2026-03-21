package prompts

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/argus-platform/argus/pkg/tenancy"
)

type Prompt struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	AgentID       string    `json:"agent_id"`
	ActiveVersion int       `json:"active_version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PromptVersion struct {
	ID        string          `json:"id"`
	PromptID  string          `json:"prompt_id"`
	TenantID  string          `json:"tenant_id"`
	Version   int             `json:"version"`
	Content   string          `json:"content"`
	ChangeLog string          `json:"change_log"`
	Metrics   *VersionMetrics `json:"metrics,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type VersionMetrics struct {
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	SuccessRate  float64 `json:"success_rate"`
	TokensUsed   int64   `json:"tokens_used"`
	Invocations  int64   `json:"invocations"`
}

type Repository struct {
	mu       sync.RWMutex
	prompts  map[string]*Prompt
	versions map[string][]*PromptVersion
}

func NewRepository() *Repository {
	return &Repository{
		prompts:  make(map[string]*Prompt),
		versions: make(map[string][]*PromptVersion),
	}
}

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/prompts", h.CreatePrompt)
	mux.HandleFunc("GET /api/v1/prompts", h.ListPrompts)
	mux.HandleFunc("GET /api/v1/prompts/{id}", h.GetPrompt)
	mux.HandleFunc("POST /api/v1/prompts/{id}/versions", h.CreateVersion)
	mux.HandleFunc("GET /api/v1/prompts/{id}/versions", h.ListVersions)
	mux.HandleFunc("PUT /api/v1/prompts/{id}/active", h.SetActiveVersion)
	mux.HandleFunc("GET /api/v1/prompts/{id}/metrics", h.GetMetrics)
}

func (h *Handler) CreatePrompt(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}

	var prompt Prompt
	if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	prompt.ID = "prompt-" + time.Now().Format("20060102150405")
	prompt.TenantID = tenantID
	prompt.ActiveVersion = 1
	prompt.CreatedAt = time.Now()
	prompt.UpdatedAt = time.Now()

	h.repo.mu.Lock()
	h.repo.prompts[prompt.ID] = &prompt
	h.repo.mu.Unlock()

	writeJSON(w, http.StatusCreated, prompt)
}

func (h *Handler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}

	h.repo.mu.RLock()
	var prompts []*Prompt
	for _, p := range h.repo.prompts {
		if p.TenantID == tenantID {
			prompts = append(prompts, p)
		}
	}
	h.repo.mu.RUnlock()

	if prompts == nil {
		prompts = []*Prompt{}
	}
	writeJSON(w, http.StatusOK, prompts)
}

func (h *Handler) GetPrompt(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}
	id := r.PathValue("id")

	h.repo.mu.RLock()
	prompt, ok := h.repo.prompts[id]
	h.repo.mu.RUnlock()

	if !ok || prompt.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Prompt not found")
		return
	}
	writeJSON(w, http.StatusOK, prompt)
}

func (h *Handler) CreateVersion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}
	promptID := r.PathValue("id")

	h.repo.mu.Lock()
	defer h.repo.mu.Unlock()

	prompt, ok := h.repo.prompts[promptID]
	if !ok || prompt.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Prompt not found")
		return
	}

	var version PromptVersion
	if err := json.NewDecoder(r.Body).Decode(&version); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	version.ID = "pv-" + time.Now().Format("20060102150405")
	version.PromptID = promptID
	version.TenantID = tenantID
	version.Version = len(h.repo.versions[promptID]) + 1
	version.CreatedAt = time.Now()

	h.repo.versions[promptID] = append(h.repo.versions[promptID], &version)

	writeJSON(w, http.StatusCreated, version)
}

func (h *Handler) ListVersions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}
	promptID := r.PathValue("id")

	h.repo.mu.RLock()
	prompt, ok := h.repo.prompts[promptID]
	if !ok || prompt.TenantID != tenantID {
		h.repo.mu.RUnlock()
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Prompt not found")
		return
	}
	versions := h.repo.versions[promptID]
	h.repo.mu.RUnlock()

	if versions == nil {
		versions = []*PromptVersion{}
	}
	writeJSON(w, http.StatusOK, versions)
}

func (h *Handler) SetActiveVersion(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}
	promptID := r.PathValue("id")

	var body struct {
		Version int `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	h.repo.mu.Lock()
	prompt, ok := h.repo.prompts[promptID]
	if !ok || prompt.TenantID != tenantID {
		h.repo.mu.Unlock()
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Prompt not found")
		return
	}
	prompt.ActiveVersion = body.Version
	prompt.UpdatedAt = time.Now()
	h.repo.mu.Unlock()

	writeJSON(w, http.StatusOK, prompt)
}

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Tenant not found in context")
		return
	}
	promptID := r.PathValue("id")

	h.repo.mu.RLock()
	prompt, ok := h.repo.prompts[promptID]
	if !ok || prompt.TenantID != tenantID {
		h.repo.mu.RUnlock()
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Prompt not found")
		return
	}
	versions := h.repo.versions[promptID]
	h.repo.mu.RUnlock()

	// Return versions with mock metrics
	for _, v := range versions {
		v.Metrics = &VersionMetrics{
			AvgLatencyMs: 150.5,
			SuccessRate:  0.95,
			TokensUsed:   int64(v.Version) * 1000,
			Invocations:  int64(v.Version) * 50,
		}
	}

	_ = prompt // used for tenant check above
	writeJSON(w, http.StatusOK, versions)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "meta": map[string]any{}})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": code, "message": message}})
}
