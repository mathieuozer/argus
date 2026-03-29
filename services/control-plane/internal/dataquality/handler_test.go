package dataquality

import (
	"encoding/json"
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
	store, repo := NewMemStore()
	h := NewHandler(store)
	return h, repo
}

func serveMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

// ---------- Rules CRUD ----------

func TestHandleRules_GET(t *testing.T) {
	tests := []struct {
		name          string
		tenantID      string
		queryParams   string
		seed          func(*Repository)
		wantStatus    int
		wantRuleCount int
	}{
		{
			name:          "empty list when no rules exist",
			tenantID:      "tenant-a",
			seed:          func(r *Repository) {},
			wantStatus:    http.StatusOK,
			wantRuleCount: 0,
		},
		{
			name:     "returns rules for tenant",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateRule("tenant-a", "rule-1", "desc", RuleTypeCompleteness, "", "latency_ms", "gt", "1000", SeverityWarning)
				r.CreateRule("tenant-a", "rule-2", "desc", RuleTypeAccuracy, "", "error_rate", "gt", "0.05", SeverityCritical)
			},
			wantStatus:    http.StatusOK,
			wantRuleCount: 2,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateRule("tenant-a", "mine", "desc", RuleTypeCompleteness, "", "field", "gt", "1", SeverityWarning)
				r.CreateRule("tenant-b", "theirs", "desc", RuleTypeAccuracy, "", "field", "gt", "1", SeverityCritical)
			},
			wantStatus:    http.StatusOK,
			wantRuleCount: 1,
		},
		{
			name:        "filters by agent_id",
			tenantID:    "tenant-a",
			queryParams: "agent_id=agent-1",
			seed: func(r *Repository) {
				r.CreateRule("tenant-a", "for-agent-1", "desc", RuleTypeCompleteness, "agent-1", "field", "gt", "1", SeverityWarning)
				r.CreateRule("tenant-a", "for-agent-2", "desc", RuleTypeAccuracy, "agent-2", "field", "gt", "1", SeverityCritical)
				r.CreateRule("tenant-a", "for-all", "desc", RuleTypeConsistency, "", "field", "gt", "1", SeverityInfo)
			},
			wantStatus:    http.StatusOK,
			wantRuleCount: 2, // agent-1 specific + global (empty agentID)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			path := "/api/v1/dataquality/rules"
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
				if tc.wantRuleCount != 0 {
					t.Errorf("data is nil, want %d rules", tc.wantRuleCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantRuleCount {
				t.Errorf("rule count = %d, want %d", len(arr), tc.wantRuleCount)
			}
		})
	}
}

func TestHandleRules_POST(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		body       string
		wantStatus int
	}{
		{
			name:       "create rule succeeds",
			tenantID:   "tenant-1",
			body:       `{"name":"latency check","description":"check latency","type":"completeness","field":"latency_ms","operator":"gt","threshold":"1000","severity":"warning"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "create rule with defaults",
			tenantID:   "tenant-1",
			body:       `{"name":"basic check","field":"error_rate","operator":"gt","threshold":"0.1"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing required fields returns 400",
			tenantID:   "tenant-1",
			body:       `{"description":"no name"}`,
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

			req := requestWithTenant(http.MethodPost, "/api/v1/dataquality/rules", tc.tenantID, tc.body)
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

func TestHandleRules_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/dataquality/rules")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRules_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodDelete, "/api/v1/dataquality/rules", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ---------- Rule by ID ----------

func TestHandleRuleByID_GET(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		ruleID     string
		wantStatus int
	}{
		{
			name:       "returns existing rule",
			tenantID:   "tenant-a",
			ruleID:     "dqr-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found for missing rule",
			tenantID:   "tenant-a",
			ruleID:     "nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "cross-tenant access returns 404",
			tenantID:   "tenant-b",
			ruleID:     "dqr-1",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateRule("tenant-a", "test rule", "desc", RuleTypeCompleteness, "", "field", "gt", "1", SeverityWarning)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/dataquality/rules/"+tc.ruleID, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleRuleByID_PUT(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		ruleID     string
		body       string
		wantStatus int
	}{
		{
			name:       "update existing rule",
			tenantID:   "tenant-a",
			ruleID:     "dqr-1",
			body:       `{"name":"updated","description":"new desc","enabled":false}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "update nonexistent rule returns 404",
			tenantID:   "tenant-a",
			ruleID:     "nonexistent",
			body:       `{"name":"updated","enabled":true}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid JSON returns 400",
			tenantID:   "tenant-a",
			ruleID:     "dqr-1",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateRule("tenant-a", "original", "desc", RuleTypeCompleteness, "", "field", "gt", "1", SeverityWarning)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodPut, "/api/v1/dataquality/rules/"+tc.ruleID, tc.tenantID, tc.body)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleRuleByID_DELETE(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		ruleID     string
		wantStatus int
	}{
		{
			name:       "delete existing rule",
			tenantID:   "tenant-a",
			ruleID:     "dqr-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete nonexistent rule returns 404",
			tenantID:   "tenant-a",
			ruleID:     "nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "cross-tenant delete returns 404",
			tenantID:   "tenant-b",
			ruleID:     "dqr-1",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateRule("tenant-a", "to delete", "desc", RuleTypeCompleteness, "", "field", "gt", "1", SeverityWarning)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodDelete, "/api/v1/dataquality/rules/"+tc.ruleID, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleRuleByID_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/dataquality/rules/dqr-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- Scores ----------

func TestHandleScores_GET(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		queryParams    string
		seed           func(*Repository)
		wantStatus     int
		wantScoreCount int
		singleResult   bool
	}{
		{
			name:           "empty list when no scores exist",
			tenantID:       "tenant-a",
			seed:           func(r *Repository) {},
			wantStatus:     http.StatusOK,
			wantScoreCount: 0,
		},
		{
			name:     "returns all scores for tenant",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.RecordScore("tenant-a", "agent-1", 95.0, 98.0, 92.0, 94.0, 96.0, 100, 95, 5)
				r.RecordScore("tenant-a", "agent-2", 88.0, 90.0, 85.0, 89.0, 88.0, 80, 70, 10)
			},
			wantStatus:     http.StatusOK,
			wantScoreCount: 2,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.RecordScore("tenant-a", "agent-1", 95.0, 98.0, 92.0, 94.0, 96.0, 100, 95, 5)
				r.RecordScore("tenant-b", "agent-2", 88.0, 90.0, 85.0, 89.0, 88.0, 80, 70, 10)
			},
			wantStatus:     http.StatusOK,
			wantScoreCount: 1,
		},
		{
			name:        "filters by agent_id returns single score",
			tenantID:    "tenant-a",
			queryParams: "agent_id=agent-1",
			seed: func(r *Repository) {
				r.RecordScore("tenant-a", "agent-1", 95.0, 98.0, 92.0, 94.0, 96.0, 100, 95, 5)
			},
			wantStatus:   http.StatusOK,
			singleResult: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			path := "/api/v1/dataquality/scores"
			if tc.queryParams != "" {
				path += "?" + tc.queryParams
			}
			req := requestWithTenant(http.MethodGet, path, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			if tc.singleResult {
				body := responseBody(t, w)
				data, ok := body["data"].(map[string]interface{})
				if !ok {
					t.Fatalf("expected single score object, got %T", body["data"])
				}
				if data["agent_id"] != "agent-1" {
					t.Errorf("agent_id = %q, want %q", data["agent_id"], "agent-1")
				}
				return
			}

			if !tc.singleResult {
				body := responseBody(t, w)
				data := body["data"]
				if data == nil {
					if tc.wantScoreCount != 0 {
						t.Errorf("data is nil, want %d scores", tc.wantScoreCount)
					}
					return
				}
				arr, ok := data.([]interface{})
				if !ok {
					t.Fatalf("data is not an array: %T", data)
				}
				if len(arr) != tc.wantScoreCount {
					t.Errorf("score count = %d, want %d", len(arr), tc.wantScoreCount)
				}
			}
		})
	}
}

func TestHandleScores_GET_AgentNotFound(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/dataquality/scores?agent_id=nonexistent", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleScores_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/dataquality/scores")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleScores_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodDelete, "/api/v1/dataquality/scores", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ---------- Violations ----------

func TestHandleViolations_GET(t *testing.T) {
	tests := []struct {
		name               string
		tenantID           string
		queryParams        string
		seed               func(*Repository)
		wantStatus         int
		wantViolationCount int
	}{
		{
			name:               "empty list when no violations exist",
			tenantID:           "tenant-a",
			seed:               func(r *Repository) {},
			wantStatus:         http.StatusOK,
			wantViolationCount: 0,
		},
		{
			name:     "returns violations for tenant",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.RecordViolation("tenant-a", "dqr-1", "latency check", "agent-1", "latency_ms", "5000", "<1000", SeverityCritical, "latency exceeded threshold")
				r.RecordViolation("tenant-a", "dqr-2", "error check", "agent-2", "error_rate", "0.15", "<0.05", SeverityWarning, "error rate too high")
			},
			wantStatus:         http.StatusOK,
			wantViolationCount: 2,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.RecordViolation("tenant-a", "dqr-1", "mine", "agent-1", "field", "val", "exp", SeverityWarning, "msg")
				r.RecordViolation("tenant-b", "dqr-2", "theirs", "agent-2", "field", "val", "exp", SeverityCritical, "msg")
			},
			wantStatus:         http.StatusOK,
			wantViolationCount: 1,
		},
		{
			name:        "filters by agent_id",
			tenantID:    "tenant-a",
			queryParams: "agent_id=agent-1",
			seed: func(r *Repository) {
				r.RecordViolation("tenant-a", "dqr-1", "rule-1", "agent-1", "field", "val", "exp", SeverityWarning, "msg")
				r.RecordViolation("tenant-a", "dqr-2", "rule-2", "agent-2", "field", "val", "exp", SeverityCritical, "msg")
			},
			wantStatus:         http.StatusOK,
			wantViolationCount: 1,
		},
		{
			name:        "filters by rule_id",
			tenantID:    "tenant-a",
			queryParams: "rule_id=dqr-1",
			seed: func(r *Repository) {
				r.RecordViolation("tenant-a", "dqr-1", "rule-1", "agent-1", "field", "val", "exp", SeverityWarning, "msg")
				r.RecordViolation("tenant-a", "dqr-2", "rule-2", "agent-1", "field", "val", "exp", SeverityCritical, "msg")
			},
			wantStatus:         http.StatusOK,
			wantViolationCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			path := "/api/v1/dataquality/violations"
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
				if tc.wantViolationCount != 0 {
					t.Errorf("data is nil, want %d violations", tc.wantViolationCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantViolationCount {
				t.Errorf("violation count = %d, want %d", len(arr), tc.wantViolationCount)
			}
		})
	}
}

func TestHandleViolations_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/dataquality/violations")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleViolations_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodPost, "/api/v1/dataquality/violations", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ---------- Content-Type and Meta ----------

func TestHandleRules_ContentType(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/dataquality/rules", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestHandleRules_MetaTenantID(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/dataquality/rules", "tenant-meta", "")
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
		{name: "rules list", method: http.MethodGet, path: "/api/v1/dataquality/rules"},
		{name: "scores", method: http.MethodGet, path: "/api/v1/dataquality/scores"},
		{name: "violations", method: http.MethodGet, path: "/api/v1/dataquality/violations"},
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
