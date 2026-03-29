package catalog

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
	repo := NewRepository()
	h := NewHandler(NewMemStore(repo))
	return h, repo
}

func serveMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

// ---------- Sources CRUD ----------

func TestHandleSources_GET(t *testing.T) {
	tests := []struct {
		name            string
		tenantID        string
		queryParams     string
		seed            func(*Repository)
		wantStatus      int
		wantSourceCount int
	}{
		{
			name:            "empty list when no sources exist",
			tenantID:        "tenant-a",
			seed:            func(r *Repository) {},
			wantStatus:      http.StatusOK,
			wantSourceCount: 0,
		},
		{
			name:     "returns sources for tenant",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateSource("tenant-a", "budget-db", "Budget database", SourceTypeDatabase, "team-fin", "", []string{"finance"}, nil)
				r.CreateSource("tenant-a", "report-api", "Reporting API", SourceTypeAPI, "team-eng", "", []string{"reporting"}, nil)
			},
			wantStatus:      http.StatusOK,
			wantSourceCount: 2,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateSource("tenant-a", "mine", "desc", SourceTypeDatabase, "me", "", nil, nil)
				r.CreateSource("tenant-b", "theirs", "desc", SourceTypeAPI, "them", "", nil, nil)
			},
			wantStatus:      http.StatusOK,
			wantSourceCount: 1,
		},
		{
			name:        "filters by type",
			tenantID:    "tenant-a",
			queryParams: "type=database",
			seed: func(r *Repository) {
				r.CreateSource("tenant-a", "db-src", "desc", SourceTypeDatabase, "owner", "", nil, nil)
				r.CreateSource("tenant-a", "api-src", "desc", SourceTypeAPI, "owner", "", nil, nil)
			},
			wantStatus:      http.StatusOK,
			wantSourceCount: 1,
		},
		{
			name:        "filters by tag",
			tenantID:    "tenant-a",
			queryParams: "tag=finance",
			seed: func(r *Repository) {
				r.CreateSource("tenant-a", "budget", "desc", SourceTypeDatabase, "owner", "", []string{"finance", "critical"}, nil)
				r.CreateSource("tenant-a", "logs", "desc", SourceTypeStream, "owner", "", []string{"ops"}, nil)
			},
			wantStatus:      http.StatusOK,
			wantSourceCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			path := "/api/v1/catalog/sources"
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
				if tc.wantSourceCount != 0 {
					t.Errorf("data is nil, want %d sources", tc.wantSourceCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantSourceCount {
				t.Errorf("source count = %d, want %d", len(arr), tc.wantSourceCount)
			}
		})
	}
}

func TestHandleSources_POST(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		body       string
		wantStatus int
	}{
		{
			name:       "create source succeeds",
			tenantID:   "tenant-1",
			body:       `{"name":"budget-db","description":"Budget database","type":"database","owner":"team-fin","tags":["finance"]}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "create source with schema",
			tenantID:   "tenant-1",
			body:       `{"name":"users-api","type":"api","schema":{"user_id":"string","email":"string"}}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing required fields returns 400",
			tenantID:   "tenant-1",
			body:       `{"description":"no name or type"}`,
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

			req := requestWithTenant(http.MethodPost, "/api/v1/catalog/sources", tc.tenantID, tc.body)
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

func TestHandleSources_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/catalog/sources")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleSources_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodDelete, "/api/v1/catalog/sources", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ---------- Source by ID ----------

func TestHandleSourceByID_GET(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		sourceID   string
		wantStatus int
	}{
		{
			name:       "returns existing source",
			tenantID:   "tenant-a",
			sourceID:   "src-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found for missing source",
			tenantID:   "tenant-a",
			sourceID:   "nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "cross-tenant access returns 404",
			tenantID:   "tenant-b",
			sourceID:   "src-1",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateSource("tenant-a", "budget-db", "desc", SourceTypeDatabase, "owner", "", nil, nil)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/catalog/sources/"+tc.sourceID, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleSourceByID_PUT(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		sourceID   string
		body       string
		wantStatus int
	}{
		{
			name:       "update existing source",
			tenantID:   "tenant-a",
			sourceID:   "src-1",
			body:       `{"name":"updated-db","description":"new desc","owner":"new-owner","tags":["updated"]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "update nonexistent source returns 404",
			tenantID:   "tenant-a",
			sourceID:   "nonexistent",
			body:       `{"name":"updated"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid JSON returns 400",
			tenantID:   "tenant-a",
			sourceID:   "src-1",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateSource("tenant-a", "original", "desc", SourceTypeDatabase, "owner", "", nil, nil)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodPut, "/api/v1/catalog/sources/"+tc.sourceID, tc.tenantID, tc.body)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleSourceByID_DELETE(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		sourceID   string
		wantStatus int
	}{
		{
			name:       "delete existing source",
			tenantID:   "tenant-a",
			sourceID:   "src-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete nonexistent source returns 404",
			tenantID:   "tenant-a",
			sourceID:   "nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "cross-tenant delete returns 404",
			tenantID:   "tenant-b",
			sourceID:   "src-1",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			repo.CreateSource("tenant-a", "to delete", "desc", SourceTypeDatabase, "owner", "", nil, nil)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodDelete, "/api/v1/catalog/sources/"+tc.sourceID, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleSourceByID_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/catalog/sources/src-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- Lineage ----------

func TestHandleSourceLineage(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		sourceID   string
		seed       func(*Repository)
		wantStatus int
	}{
		{
			name:     "returns lineage for existing source",
			tenantID: "tenant-a",
			sourceID: "src-1",
			seed: func(r *Repository) {
				r.CreateSource("tenant-a", "source-a", "desc", SourceTypeDatabase, "owner", "", nil, nil) // src-1
				r.CreateSource("tenant-a", "source-b", "desc", SourceTypeAPI, "owner", "", nil, nil)      // src-2
				_, _ = r.AddLineageEdge("tenant-a", "src-1", "src-2", "transform", "agent-1", "ETL pipeline")
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found for missing source",
			tenantID:   "tenant-a",
			sourceID:   "nonexistent",
			seed:       func(r *Repository) {},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "cross-tenant access returns 404",
			tenantID: "tenant-b",
			sourceID: "src-1",
			seed: func(r *Repository) {
				r.CreateSource("tenant-a", "source-a", "desc", SourceTypeDatabase, "owner", "", nil, nil)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/catalog/sources/"+tc.sourceID+"/lineage", tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleSourceLineage_VerifyGraph(t *testing.T) {
	h, repo := newTestHandler()
	repo.CreateSource("tenant-a", "raw-data", "desc", SourceTypeDatabase, "owner", "", nil, nil) // src-1
	repo.CreateSource("tenant-a", "processed", "desc", SourceTypeAPI, "owner", "", nil, nil)     // src-2
	repo.CreateSource("tenant-a", "report", "desc", SourceTypeFile, "owner", "", nil, nil)       // src-3
	_, _ = repo.AddLineageEdge("tenant-a", "src-1", "src-2", "transform", "agent-1", "ETL")
	_, _ = repo.AddLineageEdge("tenant-a", "src-2", "src-3", "aggregate", "agent-2", "Report gen")
	mux := serveMux(h)

	// Check lineage from the middle node (src-2)
	req := requestWithTenant(http.MethodGet, "/api/v1/catalog/sources/src-2/lineage", "tenant-a", "")
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

	upstream, ok := data["upstream"].([]interface{})
	if !ok {
		t.Fatalf("upstream is not an array: %T", data["upstream"])
	}
	if len(upstream) != 1 {
		t.Errorf("upstream count = %d, want 1", len(upstream))
	}

	downstream, ok := data["downstream"].([]interface{})
	if !ok {
		t.Fatalf("downstream is not an array: %T", data["downstream"])
	}
	if len(downstream) != 1 {
		t.Errorf("downstream count = %d, want 1", len(downstream))
	}
}

// ---------- Lineage edges ----------

func TestHandleLineageEdges_GET(t *testing.T) {
	tests := []struct {
		name          string
		tenantID      string
		seed          func(*Repository)
		wantStatus    int
		wantEdgeCount int
	}{
		{
			name:          "empty list when no edges exist",
			tenantID:      "tenant-a",
			seed:          func(r *Repository) {},
			wantStatus:    http.StatusOK,
			wantEdgeCount: 0,
		},
		{
			name:     "returns edges for tenant",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateSource("tenant-a", "a", "desc", SourceTypeDatabase, "o", "", nil, nil)
				r.CreateSource("tenant-a", "b", "desc", SourceTypeAPI, "o", "", nil, nil)
				_, _ = r.AddLineageEdge("tenant-a", "src-1", "src-2", "transform", "agent-1", "desc")
			},
			wantStatus:    http.StatusOK,
			wantEdgeCount: 1,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			seed: func(r *Repository) {
				r.CreateSource("tenant-a", "a", "desc", SourceTypeDatabase, "o", "", nil, nil)
				r.CreateSource("tenant-a", "b", "desc", SourceTypeAPI, "o", "", nil, nil)
				_, _ = r.AddLineageEdge("tenant-a", "src-1", "src-2", "copy", "agent-1", "desc")
				r.CreateSource("tenant-b", "c", "desc", SourceTypeDatabase, "o", "", nil, nil)
				r.CreateSource("tenant-b", "d", "desc", SourceTypeAPI, "o", "", nil, nil)
				_, _ = r.AddLineageEdge("tenant-b", "src-3", "src-4", "copy", "agent-2", "desc")
			},
			wantStatus:    http.StatusOK,
			wantEdgeCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/catalog/lineage", tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := responseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantEdgeCount != 0 {
					t.Errorf("data is nil, want %d edges", tc.wantEdgeCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantEdgeCount {
				t.Errorf("edge count = %d, want %d", len(arr), tc.wantEdgeCount)
			}
		})
	}
}

func TestHandleLineageEdges_POST(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		body       string
		seed       func(*Repository)
		wantStatus int
	}{
		{
			name:     "create edge succeeds",
			tenantID: "tenant-a",
			body:     `{"source_id":"src-1","target_id":"src-2","transform_type":"transform","agent_id":"agent-1","description":"ETL pipeline"}`,
			seed: func(r *Repository) {
				r.CreateSource("tenant-a", "a", "desc", SourceTypeDatabase, "o", "", nil, nil)
				r.CreateSource("tenant-a", "b", "desc", SourceTypeAPI, "o", "", nil, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing required fields returns 400",
			tenantID:   "tenant-a",
			body:       `{"description":"no source or target"}`,
			seed:       func(r *Repository) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON returns 400",
			tenantID:   "tenant-a",
			body:       `{invalid`,
			seed:       func(r *Repository) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "nonexistent source returns 400",
			tenantID: "tenant-a",
			body:     `{"source_id":"nonexistent","target_id":"src-1"}`,
			seed: func(r *Repository) {
				r.CreateSource("tenant-a", "a", "desc", SourceTypeDatabase, "o", "", nil, nil)
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, repo := newTestHandler()
			tc.seed(repo)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodPost, "/api/v1/catalog/lineage", tc.tenantID, tc.body)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleLineageEdges_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/catalog/lineage")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- Content-Type and Meta ----------

func TestHandleSources_ContentType(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/catalog/sources", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestHandleSources_MetaTenantID(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/catalog/sources", "tenant-meta", "")
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
		{name: "sources list", method: http.MethodGet, path: "/api/v1/catalog/sources"},
		{name: "lineage edges", method: http.MethodGet, path: "/api/v1/catalog/lineage"},
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
