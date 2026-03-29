package guardrails

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/pkg/tenancy"
)

// Rule defines a guardrail rule (mirrors telemetry service definition).
type Rule struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Pattern     string    `json:"pattern"`
	Action      string    `json:"action"`
	Enabled     bool      `json:"enabled"`
	AgentIDs    []string  `json:"agent_ids"`
	CreatedAt   time.Time `json:"created_at"`
}

// Violation records a guardrail violation.
type Violation struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	RuleType  string    `json:"rule_type"`
	AgentID   string    `json:"agent_id"`
	SpanID    string    `json:"span_id"`
	Action    string    `json:"action"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Stats holds guardrail statistics.
type Stats struct {
	TotalChecks     int            `json:"total_checks"`
	TotalViolations int            `json:"total_violations"`
	PassRate        float64        `json:"pass_rate"`
	ByRule          map[string]int `json:"by_rule"`
	ByAgent         map[string]int `json:"by_agent"`
}

// Repository stores guardrail data.
type Repository struct {
	mu         sync.RWMutex
	rules      map[string]*Rule
	violations []*Violation
}

// NewRepository creates a new in-memory repository.
func NewRepository() *Repository {
	return &Repository{
		rules:      make(map[string]*Rule),
		violations: make([]*Violation, 0),
	}
}

// AddRule adds a rule directly (used for seeding).
func (r *Repository) AddRule(rule *Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[rule.ID] = rule
}

// AddViolation adds a violation directly (used for seeding).
func (r *Repository) AddViolation(v *Violation) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.violations = append(r.violations, v)
}

// Handler handles guardrail HTTP requests.
type Handler struct {
	repo   *Repository
	pgRepo *PGRepository
}

// NewHandler creates a new guardrails handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// SetPG attaches a PostgreSQL repository for dual-write persistence.
func (h *Handler) SetPG(pg *PGRepository) {
	h.pgRepo = pg
}

// RegisterRoutes registers guardrail API routes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/guardrails/rules", h.CreateRule)
	mux.HandleFunc("GET /api/v1/guardrails/rules", h.ListRules)
	mux.HandleFunc("POST /api/v1/guardrails/violations", h.CreateViolation)
	mux.HandleFunc("GET /api/v1/guardrails/violations", h.ListViolations)
	mux.HandleFunc("GET /api/v1/guardrails/stats", h.GetStats)
}

// CreateViolation handles POST /api/v1/guardrails/violations.
func (h *Handler) CreateViolation(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}

	var req struct {
		RuleID  string `json:"rule_id"`
		AgentID string `json:"agent_id"`
		SpanID  string `json:"span_id"`
		Action  string `json:"action"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	v := &Violation{
		ID:        "gv-" + time.Now().Format("20060102150405.000"),
		TenantID:  tenantID,
		RuleID:    req.RuleID,
		AgentID:   req.AgentID,
		SpanID:    req.SpanID,
		Action:    req.Action,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	// Look up rule name/type
	h.repo.mu.RLock()
	if rule, ok := h.repo.rules[req.RuleID]; ok {
		v.RuleName = rule.Name
		v.RuleType = rule.Type
	}
	h.repo.mu.RUnlock()

	h.repo.AddViolation(v)

	if h.pgRepo != nil {
		if err := h.pgRepo.SaveViolation(r.Context(), v); err != nil {
			logger.FromContext(r.Context()).Error("pg dual-write SaveViolation failed: " + err.Error())
		}
	}

	httputil.WriteJSON(w, http.StatusCreated, v, "")
}

// CreateRule handles POST /api/v1/guardrails/rules.
func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}

	var rule Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	rule.ID = "gr-" + time.Now().Format("20060102150405")
	rule.TenantID = tenantID
	rule.CreatedAt = time.Now()

	h.repo.mu.Lock()
	h.repo.rules[rule.ID] = &rule
	h.repo.mu.Unlock()

	if h.pgRepo != nil {
		if err := h.pgRepo.CreateRule(r.Context(), &rule); err != nil {
			logger.FromContext(r.Context()).Error("pg dual-write CreateRule failed: " + err.Error())
		}
	}

	httputil.WriteJSON(w, http.StatusCreated, rule, "")
}

// ListRules handles GET /api/v1/guardrails/rules.
func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}

	h.repo.mu.RLock()
	var rules []*Rule
	for _, rule := range h.repo.rules {
		if rule.TenantID == tenantID {
			rules = append(rules, rule)
		}
	}
	h.repo.mu.RUnlock()

	if rules == nil {
		rules = []*Rule{}
	}

	httputil.WriteJSON(w, http.StatusOK, rules, "")
}

// ListViolations handles GET /api/v1/guardrails/violations.
func (h *Handler) ListViolations(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}

	h.repo.mu.RLock()
	var violations []*Violation
	for _, v := range h.repo.violations {
		if v.TenantID == tenantID {
			violations = append(violations, v)
		}
	}
	h.repo.mu.RUnlock()

	if violations == nil {
		violations = []*Violation{}
	}

	httputil.WriteJSON(w, http.StatusOK, violations, "")
}

// GetStats handles GET /api/v1/guardrails/stats.
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}

	h.repo.mu.RLock()
	stats := Stats{
		ByRule:  make(map[string]int),
		ByAgent: make(map[string]int),
	}
	for _, v := range h.repo.violations {
		if v.TenantID == tenantID {
			stats.TotalViolations++
			stats.ByRule[v.RuleName]++
			stats.ByAgent[v.AgentID]++
		}
	}
	h.repo.mu.RUnlock()

	stats.TotalChecks = stats.TotalViolations * 10 // estimate
	if stats.TotalChecks > 0 {
		stats.PassRate = float64(stats.TotalChecks-stats.TotalViolations) / float64(stats.TotalChecks)
	} else {
		stats.PassRate = 1.0
	}

	httputil.WriteJSON(w, http.StatusOK, stats, "")
}
