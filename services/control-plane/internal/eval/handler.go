package eval

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/tenancy"
)

// TestSuite defines a collection of test cases for evaluating agents.
type TestSuite struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	AgentID     string     `json:"agent_id"`
	TestCases   []TestCase `json:"test_cases"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TestCase is a single evaluation test.
type TestCase struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Input          string            `json:"input"`
	ExpectedOutput string            `json:"expected_output"`
	Criteria       map[string]string `json:"criteria"`
	MaxLatencyMs   int64             `json:"max_latency_ms"`
}

// EvalRun represents a single evaluation execution.
type EvalRun struct {
	ID          string       `json:"id"`
	TenantID    string       `json:"tenant_id"`
	SuiteID     string       `json:"suite_id"`
	SuiteName   string       `json:"suite_name"`
	AgentID     string       `json:"agent_id"`
	Status      string       `json:"status"` // running, completed, failed
	Score       float64      `json:"score"`
	TotalCases  int          `json:"total_cases"`
	PassedCases int          `json:"passed_cases"`
	FailedCases int          `json:"failed_cases"`
	Results     []CaseResult `json:"results"`
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt *time.Time   `json:"completed_at"`
}

// CaseResult is the result of a single test case execution.
type CaseResult struct {
	TestCaseID   string  `json:"test_case_id"`
	TestCaseName string  `json:"test_case_name"`
	Status       string  `json:"status"` // passed, failed, error
	ActualOutput string  `json:"actual_output"`
	LatencyMs    int64   `json:"latency_ms"`
	Score        float64 `json:"score"`
	Reason       string  `json:"reason"`
}

// Repository stores evaluation data.
type Repository struct {
	mu     sync.RWMutex
	suites map[string]*TestSuite
	runs   map[string]*EvalRun
}

// NewRepository creates a new in-memory eval repository.
func NewRepository() *Repository {
	return &Repository{
		suites: make(map[string]*TestSuite),
		runs:   make(map[string]*EvalRun),
	}
}

// Handler handles evaluation HTTP requests.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new eval handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes registers eval API routes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/evals/suites", h.CreateSuite)
	mux.HandleFunc("GET /api/v1/evals/suites", h.ListSuites)
	mux.HandleFunc("GET /api/v1/evals/suites/{id}", h.GetSuite)
	mux.HandleFunc("POST /api/v1/evals/suites/{id}/run", h.RunEval)
	mux.HandleFunc("GET /api/v1/evals/runs", h.ListRuns)
	mux.HandleFunc("GET /api/v1/evals/runs/{id}", h.GetRun)
}

func (h *Handler) CreateSuite(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}

	var suite TestSuite
	if err := json.NewDecoder(r.Body).Decode(&suite); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	suite.ID = generateID("suite")
	suite.TenantID = tenantID
	suite.CreatedAt = time.Now()
	suite.UpdatedAt = time.Now()

	h.repo.mu.Lock()
	h.repo.suites[suite.ID] = &suite
	h.repo.mu.Unlock()

	httputil.WriteJSON(w, http.StatusCreated, suite, "")
}

func (h *Handler) ListSuites(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}

	h.repo.mu.RLock()
	var suites []*TestSuite
	for _, s := range h.repo.suites {
		if s.TenantID == tenantID {
			suites = append(suites, s)
		}
	}
	h.repo.mu.RUnlock()

	if suites == nil {
		suites = []*TestSuite{}
	}

	httputil.WriteJSON(w, http.StatusOK, suites, "")
}

func (h *Handler) GetSuite(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}
	id := r.PathValue("id")

	h.repo.mu.RLock()
	suite, ok := h.repo.suites[id]
	h.repo.mu.RUnlock()

	if !ok || suite.TenantID != tenantID {
		httputil.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Test suite not found")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, suite, "")
}

func (h *Handler) RunEval(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}
	suiteID := r.PathValue("id")

	h.repo.mu.RLock()
	suite, ok := h.repo.suites[suiteID]
	h.repo.mu.RUnlock()

	if !ok || suite.TenantID != tenantID {
		httputil.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Test suite not found")
		return
	}

	run := &EvalRun{
		ID:         generateID("run"),
		TenantID:   tenantID,
		SuiteID:    suite.ID,
		SuiteName:  suite.Name,
		AgentID:    suite.AgentID,
		Status:     "completed",
		TotalCases: len(suite.TestCases),
		StartedAt:  time.Now(),
	}

	// Simulate running test cases
	var passed, failed int
	var totalScore float64
	for _, tc := range suite.TestCases {
		result := CaseResult{
			TestCaseID:   tc.ID,
			TestCaseName: tc.Name,
			Status:       "passed",
			ActualOutput: "Simulated response for: " + tc.Input,
			LatencyMs:    150 + int64(len(tc.Input)),
			Score:        0.85,
		}
		if tc.MaxLatencyMs > 0 && result.LatencyMs > tc.MaxLatencyMs {
			result.Status = "failed"
			result.Reason = "Exceeded max latency"
			failed++
		} else {
			passed++
		}
		totalScore += result.Score
		run.Results = append(run.Results, result)
	}

	run.PassedCases = passed
	run.FailedCases = failed
	if len(suite.TestCases) > 0 {
		run.Score = totalScore / float64(len(suite.TestCases))
	}
	now := time.Now()
	run.CompletedAt = &now

	h.repo.mu.Lock()
	h.repo.runs[run.ID] = run
	h.repo.mu.Unlock()

	httputil.WriteJSON(w, http.StatusOK, run, "")
}

func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}

	h.repo.mu.RLock()
	var runs []*EvalRun
	for _, run := range h.repo.runs {
		if run.TenantID == tenantID {
			runs = append(runs, run)
		}
	}
	h.repo.mu.RUnlock()

	if runs == nil {
		runs = []*EvalRun{}
	}

	httputil.WriteJSON(w, http.StatusOK, runs, "")
}

func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenancy.FromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID is required")
		return
	}
	id := r.PathValue("id")

	h.repo.mu.RLock()
	run, ok := h.repo.runs[id]
	h.repo.mu.RUnlock()

	if !ok || run.TenantID != tenantID {
		httputil.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Eval run not found")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, run, "")
}

func generateID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102150405") + "-" + randomSuffix()
}

func randomSuffix() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(1 * time.Nanosecond) // vary the nanosecond
	}
	return string(b)
}
