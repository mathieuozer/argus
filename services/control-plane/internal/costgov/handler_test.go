package costgov

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/argus-platform/argus/pkg/tenancy"
)

// ---------- helpers ----------

func requestWithTenant(method, path, tenantID string, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	ctx := tenancy.WithTenant(r.Context(), tenantID)
	return r.WithContext(ctx)
}

func requestWithoutTenant(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

func responseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return body
}

func newTestHandler() (*Handler, *Repository) {
	repo := NewRepository()
	detector := NewAnomalyDetector()
	h := NewHandler(repo, detector)
	return h, repo
}

func serveMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

// ---------- Breakdown ----------

func TestHandleBreakdown_GET(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		seed           func(*Repository)
		wantStatus     int
		wantTotal      float64
		wantGroupCount int
	}{
		{
			name:           "empty breakdown when no costs exist",
			tenantID:       "tenant-a",
			seed:           func(r *Repository) {},
			wantStatus:     http.StatusOK,
			wantTotal:      0,
			wantGroupCount: 0,
		},
		{
			name:     "returns aggregated breakdown by agent",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.RecordCost("tenant-a", "agent-1", "task-1", 0.50, 1000, "gpt-4", "inference")
				r.RecordCost("tenant-a", "agent-2", "task-2", 0.25, 500, "gpt-3.5", "inference")
			},
			wantStatus:     http.StatusOK,
			wantTotal:      0.75,
			wantGroupCount: 2,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.RecordCost("tenant-a", "agent-1", "task-1", 1.00, 2000, "gpt-4", "inference")
				r.RecordCost("tenant-b", "agent-2", "task-2", 5.00, 10000, "gpt-4", "inference")
			},
			wantStatus:     http.StatusOK,
			wantTotal:      1.00,
			wantGroupCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/costs/breakdown?group_by=agent", tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := responseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantGroupCount != 0 {
					t.Errorf("data is nil, want %d groups", tc.wantGroupCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantGroupCount {
				t.Errorf("group count = %d, want %d", len(arr), tc.wantGroupCount)
			}
			// Verify total cost across all groups
			var total float64
			for _, item := range arr {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				if cost, ok := m["cost_usd"].(float64); ok {
					total += cost
				}
			}
			if total != tc.wantTotal {
				t.Errorf("total cost = %f, want %f", total, tc.wantTotal)
			}
		})
	}
}

func TestHandleBreakdown_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/costs/breakdown")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleBreakdown_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodPost, "/api/v1/costs/breakdown", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ---------- Trends ----------

func TestHandleTrends_GET(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		queryParams    string
		seed           func(*Repository)
		wantStatus     int
		wantTrendCount int
	}{
		{
			name:           "empty trends when no costs exist",
			tenantID:       "tenant-a",
			seed:           func(r *Repository) {},
			wantStatus:     http.StatusOK,
			wantTrendCount: 0,
		},
		{
			name:     "returns daily trends",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				now := time.Now()
				r.RecordCostAt("tenant-a", "agent-1", "task-1", 0.50, 1000, "gpt-4", "inference", now)
				r.RecordCostAt("tenant-a", "agent-1", "task-2", 0.30, 600, "gpt-4", "inference", now.AddDate(0, 0, -1))
			},
			wantStatus:     http.StatusOK,
			wantTrendCount: 2,
		},
		{
			name:     "tenant isolation in trends",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				now := time.Now()
				r.RecordCostAt("tenant-a", "agent-1", "task-1", 0.50, 1000, "gpt-4", "inference", now)
				r.RecordCostAt("tenant-b", "agent-2", "task-2", 0.30, 600, "gpt-4", "inference", now)
			},
			wantStatus:     http.StatusOK,
			wantTrendCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			path := "/api/v1/costs/trends"
			if tc.queryParams != "" {
				path += "?" + tc.queryParams
			}
			req := requestWithTenant(http.MethodGet, path, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := responseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantTrendCount != 0 {
					t.Errorf("data is nil, want %d trends", tc.wantTrendCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantTrendCount {
				t.Errorf("trend count = %d, want %d", len(arr), tc.wantTrendCount)
			}
		})
	}
}

func TestHandleTrends_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/costs/trends")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- Agent costs ----------

func TestHandleAgentCosts_GET(t *testing.T) {
	tests := []struct {
		name          string
		tenantID      string
		agentID       string
		seed          func(*Repository)
		wantStatus    int
		wantCostCount int
	}{
		{
			name:          "empty list when no costs exist for agent",
			tenantID:      "tenant-a",
			agentID:       "agent-1",
			seed:          func(r *Repository) {},
			wantStatus:    http.StatusOK,
			wantCostCount: 0,
		},
		{
			name:     "returns costs for agent",
			tenantID: "tenant-a",
			agentID:  "agent-1",
			seed: func(r *Repository) {
				r.RecordCost("tenant-a", "agent-1", "task-1", 0.50, 1000, "gpt-4", "inference")
				r.RecordCost("tenant-a", "agent-1", "task-2", 0.25, 500, "gpt-4", "inference")
				r.RecordCost("tenant-a", "agent-2", "task-3", 1.00, 2000, "gpt-4", "inference")
			},
			wantStatus:    http.StatusOK,
			wantCostCount: 2,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			agentID:  "agent-1",
			seed: func(r *Repository) {
				r.RecordCost("tenant-a", "agent-1", "task-1", 0.50, 1000, "gpt-4", "inference")
				r.RecordCost("tenant-b", "agent-1", "task-2", 0.25, 500, "gpt-4", "inference")
			},
			wantStatus:    http.StatusOK,
			wantCostCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/costs/agents/"+tc.agentID, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := responseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantCostCount != 0 {
					t.Errorf("data is nil, want %d costs", tc.wantCostCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantCostCount {
				t.Errorf("cost count = %d, want %d", len(arr), tc.wantCostCount)
			}
		})
	}
}

func TestHandleAgentCosts_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/costs/agents/agent-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- Budgets ----------

func TestHandleBudgets_GET(t *testing.T) {
	tests := []struct {
		name            string
		tenantID        string
		seed            func(*Repository)
		wantStatus      int
		wantBudgetCount int
	}{
		{
			name:            "empty list when no budgets exist",
			tenantID:        "tenant-a",
			seed:            func(r *Repository) {},
			wantStatus:      http.StatusOK,
			wantBudgetCount: 0,
		},
		{
			name:     "returns budgets for tenant",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateBudget("tenant-a", "", "Monthly total", 1000.0, "monthly", 0.8)
				r.CreateBudget("tenant-a", "agent-1", "Agent limit", 100.0, "daily", 0.9)
			},
			wantStatus:      http.StatusOK,
			wantBudgetCount: 2,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateBudget("tenant-a", "", "mine", 1000.0, "monthly", 0.8)
				r.CreateBudget("tenant-b", "", "theirs", 5000.0, "monthly", 0.8)
			},
			wantStatus:      http.StatusOK,
			wantBudgetCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/costs/budgets", tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := responseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantBudgetCount != 0 {
					t.Errorf("data is nil, want %d budgets", tc.wantBudgetCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantBudgetCount {
				t.Errorf("budget count = %d, want %d", len(arr), tc.wantBudgetCount)
			}
		})
	}
}

func TestHandleBudgets_POST(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		body       string
		wantStatus int
	}{
		{
			name:       "create budget succeeds",
			tenantID:   "tenant-1",
			body:       `{"name":"Monthly total","limit_usd":1000,"period_type":"monthly","alert_threshold":0.8}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "create budget with defaults",
			tenantID:   "tenant-1",
			body:       `{"name":"Simple budget","limit_usd":500}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing required fields returns 400",
			tenantID:   "tenant-1",
			body:       `{"description":"no name"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero limit returns 400",
			tenantID:   "tenant-1",
			body:       `{"name":"Bad budget","limit_usd":0}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON returns 400",
			tenantID:   "tenant-1",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestHandler()
			mux := serveMux(h)

			req := requestWithTenant(http.MethodPost, "/api/v1/costs/budgets", tc.tenantID, tc.body)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			ct := w.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}
		})
	}
}

func TestHandleBudgets_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/costs/budgets")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- Budget by ID ----------

func TestHandleBudgetByID_GET(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		budgetID   string
		wantStatus int
	}{
		{
			name:       "returns existing budget",
			tenantID:   "tenant-a",
			budgetID:   "budget-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found for missing budget",
			tenantID:   "tenant-a",
			budgetID:   "nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "cross-tenant access returns 404",
			tenantID:   "tenant-b",
			budgetID:   "budget-1",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateBudget("tenant-a", "", "test budget", 1000.0, "monthly", 0.8)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/costs/budgets/"+tc.budgetID, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleBudgetByID_PUT(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		budgetID   string
		body       string
		wantStatus int
	}{
		{
			name:       "update existing budget",
			tenantID:   "tenant-a",
			budgetID:   "budget-1",
			body:       `{"name":"updated","limit_usd":2000,"alert_threshold":0.9,"enabled":true}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "update nonexistent budget returns 404",
			tenantID:   "tenant-a",
			budgetID:   "nonexistent",
			body:       `{"name":"updated","enabled":true}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid JSON returns 400",
			tenantID:   "tenant-a",
			budgetID:   "budget-1",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateBudget("tenant-a", "", "original", 1000.0, "monthly", 0.8)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodPut, "/api/v1/costs/budgets/"+tc.budgetID, tc.tenantID, tc.body)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleBudgetByID_DELETE(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		budgetID   string
		wantStatus int
	}{
		{
			name:       "delete existing budget",
			tenantID:   "tenant-a",
			budgetID:   "budget-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete nonexistent budget returns 404",
			tenantID:   "tenant-a",
			budgetID:   "nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "cross-tenant delete returns 404",
			tenantID:   "tenant-b",
			budgetID:   "budget-1",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateBudget("tenant-a", "", "to delete", 1000.0, "monthly", 0.8)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodDelete, "/api/v1/costs/budgets/"+tc.budgetID, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleBudgetByID_Status(t *testing.T) {
	h, repo := newTestHandler()
	repo.CreateBudget("tenant-a", "", "monthly", 100.0, "monthly", 0.8)
	repo.RecordCost("tenant-a", "agent-1", "task-1", 50.0, 10000, "gpt-4", "inference")
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/costs/budgets/budget-1/status", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := responseBody(t, w)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data is not an object: %T", body["data"])
	}

	currentSpend, ok := data["current_spend"].(float64)
	if !ok {
		t.Fatalf("current_spend is not a float64: %T", data["current_spend"])
	}
	if currentSpend != 50.0 {
		t.Errorf("current_spend = %f, want %f", currentSpend, 50.0)
	}

	isOverBudget, ok := data["is_over_budget"].(bool)
	if !ok {
		t.Fatalf("is_over_budget is not a bool: %T", data["is_over_budget"])
	}
	if isOverBudget {
		t.Error("expected is_over_budget to be false")
	}
}

func TestHandleBudgetByID_Status_NotFound(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/costs/budgets/nonexistent/status", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleBudgetByID_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/costs/budgets/budget-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- Anomalies ----------

func TestHandleAnomalies_GET(t *testing.T) {
	tests := []struct {
		name             string
		tenantID         string
		seed             func(*Repository)
		wantStatus       int
		wantAnomalyCount int
	}{
		{
			name:             "empty when no costs exist",
			tenantID:         "tenant-a",
			seed:             func(r *Repository) {},
			wantStatus:       http.StatusOK,
			wantAnomalyCount: 0,
		},
		{
			name:     "detects spike anomaly",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				// Record baseline costs
				r.RecordCost("tenant-a", "agent-1", "t1", 0.10, 100, "gpt-4", "inference")
				r.RecordCost("tenant-a", "agent-1", "t2", 0.12, 120, "gpt-4", "inference")
				r.RecordCost("tenant-a", "agent-1", "t3", 0.11, 110, "gpt-4", "inference")
				// Record a spike
				r.RecordCost("tenant-a", "agent-1", "t4", 5.00, 50000, "gpt-4", "inference")
			},
			wantStatus:       http.StatusOK,
			wantAnomalyCount: 2, // spike_absolute + spike_percentage
		},
		{
			name:     "tenant isolation in anomaly detection",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.RecordCost("tenant-a", "agent-1", "t1", 0.10, 100, "gpt-4", "inference")
				r.RecordCost("tenant-a", "agent-1", "t2", 0.12, 120, "gpt-4", "inference")
				r.RecordCost("tenant-a", "agent-1", "t3", 0.11, 110, "gpt-4", "inference")
				// Spike only in tenant-b — should not affect tenant-a
				r.RecordCost("tenant-b", "agent-1", "t4", 50.00, 500000, "gpt-4", "inference")
			},
			wantStatus:       http.StatusOK,
			wantAnomalyCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/costs/anomalies", tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := responseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantAnomalyCount != 0 {
					t.Errorf("data is nil, want %d anomalies", tc.wantAnomalyCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantAnomalyCount {
				t.Errorf("anomaly count = %d, want %d", len(arr), tc.wantAnomalyCount)
			}
		})
	}
}

func TestHandleAnomalies_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/costs/anomalies")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleAnomalies_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodPost, "/api/v1/costs/anomalies", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ---------- Content-Type and Meta ----------

func TestHandleBreakdown_ContentType(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/costs/breakdown", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestHandleBreakdown_MetaTenantID(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/costs/breakdown", "tenant-meta", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := responseBody(t, w)
	meta, ok := body["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("meta is not an object: %T", body["meta"])
	}
	if meta["tenant_id"] != "tenant-meta" {
		t.Errorf("meta.tenant_id = %q, want %q", meta["tenant_id"], "tenant-meta")
	}
}

// ---------- Anomaly detector unit tests ----------

func TestAnomalyDetector_CalculateMean(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{name: "empty", values: []float64{}, want: 0},
		{name: "single", values: []float64{5.0}, want: 5.0},
		{name: "multiple", values: []float64{2.0, 4.0, 6.0}, want: 4.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := calculateMean(tc.values)
			if got != tc.want {
				t.Errorf("mean = %f, want %f", got, tc.want)
			}
		})
	}
}

func TestAnomalyDetector_CalculateStdDev(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		mean   float64
	}{
		{name: "single value returns 0", values: []float64{5.0}, mean: 5.0},
		{name: "identical values returns 0", values: []float64{5.0, 5.0, 5.0}, mean: 5.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := calculateStdDev(tc.values, tc.mean)
			if tc.name == "single value returns 0" && got != 0 {
				t.Errorf("stddev = %f, want 0", got)
			}
			if tc.name == "identical values returns 0" && got != 0 {
				t.Errorf("stddev = %f, want 0", got)
			}
		})
	}
}

// ---------- RegisterRoutes ----------

func TestRegisterRoutes(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "breakdown", method: http.MethodGet, path: "/api/v1/costs/breakdown"},
		{name: "trends", method: http.MethodGet, path: "/api/v1/costs/trends"},
		{name: "agent costs", method: http.MethodGet, path: "/api/v1/costs/agents/agent-1"},
		{name: "budgets list", method: http.MethodGet, path: "/api/v1/costs/budgets"},
		{name: "anomalies", method: http.MethodGet, path: "/api/v1/costs/anomalies"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := requestWithTenant(tc.method, tc.path, "test-tenant", "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound {
				t.Errorf("route %s %s returned 404 — not registered", tc.method, tc.path)
			}
		})
	}
}
