package slo

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	calc := NewCalculator(repo)
	h := NewHandler(repo, calc)
	return h, repo
}

func serveMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

// ---------- SLOs CRUD ----------

func TestHandleSLOs_GET(t *testing.T) {
	tests := []struct {
		name         string
		tenantID     string
		queryParams  string
		seed         func(*Repository)
		wantStatus   int
		wantSLOCount int
	}{
		{
			name:         "empty list when no SLOs exist",
			tenantID:     "tenant-a",
			seed:         func(r *Repository) {},
			wantStatus:   http.StatusOK,
			wantSLOCount: 0,
		},
		{
			name:     "returns SLOs for tenant",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateSLO("tenant-a", "Availability", "99.9% uptime", "", SLOTypeAvailability, 99.9, "30d")
				r.CreateSLO("tenant-a", "Latency", "p99 < 500ms", "", SLOTypeLatency, 95.0, "7d")
			},
			wantStatus:   http.StatusOK,
			wantSLOCount: 2,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateSLO("tenant-a", "mine", "desc", "", SLOTypeAvailability, 99.9, "30d")
				r.CreateSLO("tenant-b", "theirs", "desc", "", SLOTypeLatency, 95.0, "7d")
			},
			wantStatus:   http.StatusOK,
			wantSLOCount: 1,
		},
		{
			name:        "filters by agent_id",
			tenantID:    "tenant-a",
			queryParams: "agent_id=agent-1",
			seed: func(r *Repository) {
				r.CreateSLO("tenant-a", "agent-1 SLO", "desc", "agent-1", SLOTypeAvailability, 99.9, "30d")
				r.CreateSLO("tenant-a", "agent-2 SLO", "desc", "agent-2", SLOTypeLatency, 95.0, "7d")
				r.CreateSLO("tenant-a", "global SLO", "desc", "", SLOTypeErrorRate, 99.0, "30d")
			},
			wantStatus:   http.StatusOK,
			wantSLOCount: 2, // agent-1 specific + global (empty agentID)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			path := "/api/v1/slos"
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
				if tc.wantSLOCount != 0 {
					t.Errorf("data is nil, want %d SLOs", tc.wantSLOCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantSLOCount {
				t.Errorf("SLO count = %d, want %d", len(arr), tc.wantSLOCount)
			}
		})
	}
}

func TestHandleSLOs_POST(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		body       string
		wantStatus int
	}{
		{
			name:       "create SLO succeeds",
			tenantID:   "tenant-1",
			body:       `{"name":"Availability","description":"99.9% uptime","type":"availability","target":99.9,"window":"30d"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "create SLO with agent",
			tenantID:   "tenant-1",
			body:       `{"name":"Agent Latency","type":"latency","target":95.0,"agent_id":"agent-1"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing required fields returns 400",
			tenantID:   "tenant-1",
			body:       `{"description":"no name or type"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero target returns 400",
			tenantID:   "tenant-1",
			body:       `{"name":"Bad SLO","type":"availability","target":0}`,
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

			req := requestWithTenant(http.MethodPost, "/api/v1/slos", tc.tenantID, tc.body)
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

func TestHandleSLOs_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/slos")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleSLOs_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodDelete, "/api/v1/slos", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ---------- SLO by ID ----------

func TestHandleSLOByID_GET(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		sloID      string
		wantStatus int
	}{
		{
			name:       "returns existing SLO",
			tenantID:   "tenant-a",
			sloID:      "slo-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found for missing SLO",
			tenantID:   "tenant-a",
			sloID:      "nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "cross-tenant access returns 404",
			tenantID:   "tenant-b",
			sloID:      "slo-1",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateSLO("tenant-a", "test SLO", "desc", "", SLOTypeAvailability, 99.9, "30d")
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/slos/"+tc.sloID, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleSLOByID_PUT(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		sloID      string
		body       string
		wantStatus int
	}{
		{
			name:       "update existing SLO",
			tenantID:   "tenant-a",
			sloID:      "slo-1",
			body:       `{"name":"updated","description":"new desc","target":99.95,"enabled":true}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "update nonexistent SLO returns 404",
			tenantID:   "tenant-a",
			sloID:      "nonexistent",
			body:       `{"name":"updated","enabled":true}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid JSON returns 400",
			tenantID:   "tenant-a",
			sloID:      "slo-1",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateSLO("tenant-a", "original", "desc", "", SLOTypeAvailability, 99.9, "30d")
			mux := serveMux(h)

			req := requestWithTenant(http.MethodPut, "/api/v1/slos/"+tc.sloID, tc.tenantID, tc.body)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleSLOByID_DELETE(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		sloID      string
		wantStatus int
	}{
		{
			name:       "delete existing SLO",
			tenantID:   "tenant-a",
			sloID:      "slo-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete nonexistent SLO returns 404",
			tenantID:   "tenant-a",
			sloID:      "nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "cross-tenant delete returns 404",
			tenantID:   "tenant-b",
			sloID:      "slo-1",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateSLO("tenant-a", "to delete", "desc", "", SLOTypeAvailability, 99.9, "30d")
			mux := serveMux(h)

			req := requestWithTenant(http.MethodDelete, "/api/v1/slos/"+tc.sloID, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleSLOByID_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/slos/slo-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- SLO Status ----------

func TestHandleSLOByID_Status(t *testing.T) {
	h, repo := newTestHandler()
	repo.CreateSLO("tenant-a", "Availability", "desc", "", SLOTypeAvailability, 99.9, "30d")
	repo.RecordMeasurement("tenant-a", "slo-1", "agent-1", 99.95, 9995, 10000)
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/slos/slo-1/status", "tenant-a", "")
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

	compliance, ok := data["compliance"].(string)
	if !ok {
		t.Fatalf("compliance is not a string: %T", data["compliance"])
	}
	if compliance != "met" {
		t.Errorf("compliance = %q, want %q", compliance, "met")
	}
}

func TestHandleSLOByID_Status_Breached(t *testing.T) {
	h, repo := newTestHandler()
	repo.CreateSLO("tenant-a", "Availability", "desc", "", SLOTypeAvailability, 99.9, "30d")
	// Record measurements that breach the SLO
	repo.RecordMeasurement("tenant-a", "slo-1", "agent-1", 98.0, 980, 1000)
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/slos/slo-1/status", "tenant-a", "")
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

	compliance, ok := data["compliance"].(string)
	if !ok {
		t.Fatalf("compliance is not a string: %T", data["compliance"])
	}
	if compliance != "breached" {
		t.Errorf("compliance = %q, want %q", compliance, "breached")
	}
}

func TestHandleSLOByID_Status_NotFound(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/slos/nonexistent/status", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ---------- All statuses ----------

func TestHandleAllStatuses_GET(t *testing.T) {
	tests := []struct {
		name            string
		tenantID        string
		seed            func(*Repository)
		wantStatus      int
		wantStatusCount int
	}{
		{
			name:            "empty when no SLOs exist",
			tenantID:        "tenant-a",
			seed:            func(r *Repository) {},
			wantStatus:      http.StatusOK,
			wantStatusCount: 0,
		},
		{
			name:     "returns statuses for enabled SLOs",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateSLO("tenant-a", "SLO-1", "desc", "", SLOTypeAvailability, 99.9, "30d")
				r.CreateSLO("tenant-a", "SLO-2", "desc", "", SLOTypeLatency, 95.0, "7d")
				r.RecordMeasurement("tenant-a", "slo-1", "agent-1", 99.95, 9995, 10000)
				r.RecordMeasurement("tenant-a", "slo-2", "agent-1", 96.0, 960, 1000)
			},
			wantStatus:      http.StatusOK,
			wantStatusCount: 2,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateSLO("tenant-a", "mine", "desc", "", SLOTypeAvailability, 99.9, "30d")
				r.CreateSLO("tenant-b", "theirs", "desc", "", SLOTypeLatency, 95.0, "7d")
			},
			wantStatus:      http.StatusOK,
			wantStatusCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/slos/status", tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := responseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantStatusCount != 0 {
					t.Errorf("data is nil, want %d statuses", tc.wantStatusCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantStatusCount {
				t.Errorf("status count = %d, want %d", len(arr), tc.wantStatusCount)
			}
		})
	}
}

func TestHandleAllStatuses_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/slos/status")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- Calculator unit tests ----------

func TestCalculator_ErrorBudget(t *testing.T) {
	repo := NewRepository()
	calc := NewCalculator(repo)

	repo.CreateSLO("tenant-a", "Availability", "desc", "", SLOTypeAvailability, 99.9, "30d")
	// 10000 total events, 9990 good = 99.9% exactly at target
	repo.RecordMeasurement("tenant-a", "slo-1", "agent-1", 99.9, 9990, 10000)

	status := calc.CalculateStatus("tenant-a", "slo-1")
	if status == nil {
		t.Fatal("expected status, got nil")
	}

	// Error budget total = (1 - 99.9/100) * 10000 ≈ 10
	if math.Abs(status.ErrorBudgetTotal-10.0) > 0.001 {
		t.Errorf("ErrorBudgetTotal = %f, want %f", status.ErrorBudgetTotal, 10.0)
	}

	// Error budget used = 10000 - 9990 = 10
	if math.Abs(status.ErrorBudgetUsed-10.0) > 0.001 {
		t.Errorf("ErrorBudgetUsed = %f, want %f", status.ErrorBudgetUsed, 10.0)
	}

	// Error budget left = 10 - 10 ≈ 0
	if math.Abs(status.ErrorBudgetLeft-0.0) > 0.001 {
		t.Errorf("ErrorBudgetLeft = %f, want %f", status.ErrorBudgetLeft, 0.0)
	}
}

func TestCalculator_ComplianceStatuses(t *testing.T) {
	tests := []struct {
		name           string
		target         float64
		good           int64
		total          int64
		wantCompliance ComplianceStatus
	}{
		{
			name:           "met - well above target",
			target:         99.0,
			good:           999,
			total:          1000,
			wantCompliance: ComplianceStatusMet,
		},
		{
			name:           "breached - below target",
			target:         99.9,
			good:           980,
			total:          1000,
			wantCompliance: ComplianceStatusBreached,
		},
		{
			name:           "at risk - above target but error budget mostly consumed",
			target:         99.0,
			good:           991,
			total:          1000,
			wantCompliance: ComplianceStatusAtRisk,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewRepository()
			calc := NewCalculator(repo)

			repo.CreateSLO("tenant-a", "test", "desc", "", SLOTypeAvailability, tc.target, "30d")
			repo.RecordMeasurement("tenant-a", "slo-1", "agent-1", 0, tc.good, tc.total)

			status := calc.CalculateStatus("tenant-a", "slo-1")
			if status == nil {
				t.Fatal("expected status, got nil")
			}
			if status.Compliance != tc.wantCompliance {
				t.Errorf("compliance = %q, want %q", status.Compliance, tc.wantCompliance)
			}
		})
	}
}

func TestCalculator_NonexistentSLO(t *testing.T) {
	repo := NewRepository()
	calc := NewCalculator(repo)

	status := calc.CalculateStatus("tenant-a", "nonexistent")
	if status != nil {
		t.Errorf("expected nil status for nonexistent SLO")
	}
}

// ---------- Content-Type and Meta ----------

func TestHandleSLOs_ContentType(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/slos", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestHandleSLOs_MetaTenantID(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/slos", "tenant-meta", "")
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

// ---------- RegisterRoutes ----------

func TestRegisterRoutes(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "SLOs list", method: http.MethodGet, path: "/api/v1/slos"},
		{name: "all statuses", method: http.MethodGet, path: "/api/v1/slos/status"},
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
