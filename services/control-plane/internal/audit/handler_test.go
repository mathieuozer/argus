package audit

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

func decodeResponseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return body
}

func newTestHandler() (*Handler, *Writer) {
	writer := NewWriter()
	h := NewHandler(NewMemStore(writer))
	return h, writer
}

func serveMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

// ---------- GET /api/v1/audit/logs ----------

func TestHandleLogs_GET(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		queryParams    string
		seed           func(*Writer)
		wantStatus     int
		wantEntryCount int
	}{
		{
			name:           "empty list when no entries exist",
			tenantID:       "tenant-a",
			seed:           func(w *Writer) {},
			wantStatus:     http.StatusOK,
			wantEntryCount: 0,
		},
		{
			name:     "returns entries for tenant",
			tenantID: "tenant-a",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "agent.register", "agent/budget", "registered")
				w.Write("tenant-a", "system", "cert.rotate", "identity/ca", "rotated")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
		},
		{
			name:     "tenant isolation - only returns own entries",
			tenantID: "tenant-a",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
				w.Write("tenant-b", "admin", "action2", "resource2", "")
				w.Write("tenant-a", "admin", "action3", "resource3", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
		},
		{
			name:        "respects limit",
			tenantID:    "tenant-a",
			queryParams: "limit=1",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
				w.Write("tenant-a", "admin", "action2", "resource2", "")
				w.Write("tenant-a", "admin", "action3", "resource3", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
		{
			name:        "respects offset",
			tenantID:    "tenant-a",
			queryParams: "offset=1",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
				w.Write("tenant-a", "admin", "action2", "resource2", "")
				w.Write("tenant-a", "admin", "action3", "resource3", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
		},
		{
			name:        "offset and limit combined",
			tenantID:    "tenant-a",
			queryParams: "offset=1&limit=1",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
				w.Write("tenant-a", "admin", "action2", "resource2", "")
				w.Write("tenant-a", "admin", "action3", "resource3", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
		{
			name:        "offset beyond entries returns empty",
			tenantID:    "tenant-a",
			queryParams: "offset=100",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 0,
		},
		{
			name:        "invalid limit ignored, uses default",
			tenantID:    "tenant-a",
			queryParams: "limit=abc",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
		{
			name:        "negative limit ignored, uses default",
			tenantID:    "tenant-a",
			queryParams: "limit=-5",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
		{
			name:        "invalid offset ignored, uses default of 0",
			tenantID:    "tenant-a",
			queryParams: "offset=abc",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, writer := newTestHandler()
			tc.seed(writer)
			mux := serveMux(h)

			path := "/api/v1/audit/logs"
			if tc.queryParams != "" {
				path += "?" + tc.queryParams
			}
			req := requestWithTenant(http.MethodGet, path, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := decodeResponseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantEntryCount != 0 {
					t.Errorf("data is nil, want %d entries", tc.wantEntryCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				if tc.wantEntryCount == 0 {
					return
				}
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantEntryCount {
				t.Errorf("entry count = %d, want %d", len(arr), tc.wantEntryCount)
			}
		})
	}
}

func TestHandleLogs_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/audit/logs")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	body := decodeResponseBody(t, w)
	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "TENANT_REQUIRED" {
		t.Errorf("error code = %q, want %q", errObj["code"], "TENANT_REQUIRED")
	}
}

func TestHandleLogs_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := requestWithTenant(method, "/api/v1/audit/logs", "tenant-a", "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// ---------- GET /api/v1/audit/search ----------

func TestHandleSearch_GET(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		queryParams    string
		seed           func(*Writer)
		wantStatus     int
		wantEntryCount int
	}{
		{
			name:     "returns all entries when no filters",
			tenantID: "tenant-a",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "agent.register", "agent/budget", "")
				w.Write("tenant-a", "system", "cert.rotate", "identity/ca", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
		},
		{
			name:        "filters by actor",
			tenantID:    "tenant-a",
			queryParams: "actor=admin",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "agent.register", "agent/budget", "")
				w.Write("tenant-a", "system", "cert.rotate", "identity/ca", "")
				w.Write("tenant-a", "admin", "agent.deregister", "agent/report", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
		},
		{
			name:        "filters by action",
			tenantID:    "tenant-a",
			queryParams: "action=cert.rotate",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "agent.register", "agent/budget", "")
				w.Write("tenant-a", "system", "cert.rotate", "identity/ca", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
		{
			name:        "filters by resource",
			tenantID:    "tenant-a",
			queryParams: "resource=identity/ca",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "agent.register", "agent/budget", "")
				w.Write("tenant-a", "system", "cert.rotate", "identity/ca", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
		{
			name:        "combines actor and action filters",
			tenantID:    "tenant-a",
			queryParams: "actor=admin&action=agent.register",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "agent.register", "agent/budget", "")
				w.Write("tenant-a", "admin", "cert.rotate", "identity/ca", "")
				w.Write("tenant-a", "system", "agent.register", "agent/report", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
		{
			name:        "combines all three filters",
			tenantID:    "tenant-a",
			queryParams: "actor=admin&action=agent.register&resource=agent/budget",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "agent.register", "agent/budget", "")
				w.Write("tenant-a", "admin", "agent.register", "agent/report", "")
				w.Write("tenant-a", "system", "cert.rotate", "identity/ca", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
		{
			name:        "no matches returns empty",
			tenantID:    "tenant-a",
			queryParams: "actor=nonexistent",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 0,
		},
		{
			name:        "tenant isolation in search",
			tenantID:    "tenant-a",
			queryParams: "actor=admin",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
				w.Write("tenant-b", "admin", "action2", "resource2", "")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, writer := newTestHandler()
			tc.seed(writer)
			mux := serveMux(h)

			path := "/api/v1/audit/search"
			if tc.queryParams != "" {
				path += "?" + tc.queryParams
			}
			req := requestWithTenant(http.MethodGet, path, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := decodeResponseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantEntryCount != 0 {
					t.Errorf("data is nil, want %d entries", tc.wantEntryCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				if tc.wantEntryCount == 0 {
					return
				}
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantEntryCount {
				t.Errorf("entry count = %d, want %d", len(arr), tc.wantEntryCount)
			}
		})
	}
}

func TestHandleSearch_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/audit/search")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleSearch_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := requestWithTenant(method, "/api/v1/audit/search", "tenant-a", "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// ---------- GET /api/v1/audit/stats ----------

func TestHandleStats_GET(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		seed       func(*Writer)
		wantStatus int
		wantTotal  int
	}{
		{
			name:       "empty stats when no entries",
			tenantID:   "tenant-a",
			seed:       func(w *Writer) {},
			wantStatus: http.StatusOK,
			wantTotal:  0,
		},
		{
			name:     "returns correct total count",
			tenantID: "tenant-a",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "agent.register", "agent/budget", "")
				w.Write("tenant-a", "system", "cert.rotate", "identity/ca", "")
				w.Write("tenant-a", "admin", "agent.deregister", "agent/report", "")
			},
			wantStatus: http.StatusOK,
			wantTotal:  3,
		},
		{
			name:     "tenant isolation in stats",
			tenantID: "tenant-a",
			seed: func(w *Writer) {
				w.Write("tenant-a", "admin", "action1", "resource1", "")
				w.Write("tenant-b", "admin", "action2", "resource2", "")
				w.Write("tenant-a", "admin", "action3", "resource3", "")
			},
			wantStatus: http.StatusOK,
			wantTotal:  2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, writer := newTestHandler()
			tc.seed(writer)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/audit/stats", tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := decodeResponseBody(t, w)
			data, ok := body["data"].(map[string]interface{})
			if !ok {
				t.Fatalf("data is not an object: %T", body["data"])
			}
			total := int(data["total_entries"].(float64))
			if total != tc.wantTotal {
				t.Errorf("total_entries = %d, want %d", total, tc.wantTotal)
			}
		})
	}
}

func TestHandleStats_Aggregations(t *testing.T) {
	h, writer := newTestHandler()
	writer.Write("tenant-a", "admin", "agent.register", "agent/budget", "")
	writer.Write("tenant-a", "admin", "cert.rotate", "identity/ca", "")
	writer.Write("tenant-a", "system", "agent.register", "agent/report", "")
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/audit/stats", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := decodeResponseBody(t, w)
	data := body["data"].(map[string]interface{})

	// Verify total
	if int(data["total_entries"].(float64)) != 3 {
		t.Errorf("total_entries = %v, want 3", data["total_entries"])
	}

	// Verify unique_actions: agent.register and cert.rotate = 2
	if int(data["unique_actions"].(float64)) != 2 {
		t.Errorf("unique_actions = %v, want 2", data["unique_actions"])
	}

	// Verify unique_actors: admin and system = 2
	if int(data["unique_actors"].(float64)) != 2 {
		t.Errorf("unique_actors = %v, want 2", data["unique_actors"])
	}

	// Verify by_action breakdown
	byAction, ok := data["by_action"].(map[string]interface{})
	if !ok {
		t.Fatal("by_action is not an object")
	}
	if int(byAction["agent.register"].(float64)) != 2 {
		t.Errorf("by_action[agent.register] = %v, want 2", byAction["agent.register"])
	}
	if int(byAction["cert.rotate"].(float64)) != 1 {
		t.Errorf("by_action[cert.rotate] = %v, want 1", byAction["cert.rotate"])
	}

	// Verify by_actor breakdown
	byActor, ok := data["by_actor"].(map[string]interface{})
	if !ok {
		t.Fatal("by_actor is not an object")
	}
	if int(byActor["admin"].(float64)) != 2 {
		t.Errorf("by_actor[admin] = %v, want 2", byActor["admin"])
	}
	if int(byActor["system"].(float64)) != 1 {
		t.Errorf("by_actor[system] = %v, want 1", byActor["system"])
	}
}

func TestHandleStats_MissingTenant(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/audit/stats")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleStats_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := requestWithTenant(method, "/api/v1/audit/stats", "tenant-a", "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// ---------- Response format tests ----------

func TestHandler_ContentType(t *testing.T) {
	h, writer := newTestHandler()
	writer.Write("tenant-a", "admin", "action", "resource", "")
	mux := serveMux(h)

	paths := []string{
		"/api/v1/audit/logs",
		"/api/v1/audit/search",
		"/api/v1/audit/stats",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := requestWithTenant(http.MethodGet, path, "tenant-a", "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			ct := w.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}
		})
	}
}

func TestHandler_MetaTenantID(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/audit/logs", "tenant-meta-check", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := decodeResponseBody(t, w)
	meta, ok := body["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("meta is not an object: %T", body["meta"])
	}
	if meta["tenant_id"] != "tenant-meta-check" {
		t.Errorf("meta.tenant_id = %q, want %q", meta["tenant_id"], "tenant-meta-check")
	}
}

func TestHandler_ErrorResponseFormat(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/audit/logs")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q for error response", ct, "application/json")
	}

	body := decodeResponseBody(t, w)
	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if _, ok := errObj["code"]; !ok {
		t.Error("error object missing 'code' field")
	}
	if _, ok := errObj["message"]; !ok {
		t.Error("error object missing 'message' field")
	}
}

func TestHandler_ValidJSON(t *testing.T) {
	h, writer := newTestHandler()
	writer.Write("tenant-a", "admin", "action", "resource", "details")
	mux := serveMux(h)

	paths := []string{
		"/api/v1/audit/logs",
		"/api/v1/audit/search",
		"/api/v1/audit/stats",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := requestWithTenant(http.MethodGet, path, "tenant-a", "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			var raw json.RawMessage
			if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
				t.Errorf("response is not valid JSON: %v", err)
			}
		})
	}
}

// ---------- RegisterRoutes ----------

func TestHandler_RegisterRoutes(t *testing.T) {
	h, _ := newTestHandler()
	mux := serveMux(h)

	routes := []struct {
		name   string
		method string
		path   string
	}{
		{name: "audit logs", method: http.MethodGet, path: "/api/v1/audit/logs"},
		{name: "audit search", method: http.MethodGet, path: "/api/v1/audit/search"},
		{name: "audit stats", method: http.MethodGet, path: "/api/v1/audit/stats"},
	}

	for _, tc := range routes {
		t.Run(tc.name, func(t *testing.T) {
			req := requestWithTenant(tc.method, tc.path, "test-tenant", "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound {
				t.Errorf("route %s %s returned 404 -- not registered", tc.method, tc.path)
			}
		})
	}
}

// ---------- Cross-tenant isolation ----------

func TestHandler_CrossTenantIsolation_Logs(t *testing.T) {
	h, writer := newTestHandler()
	writer.Write("tenant-a", "admin", "secret.action", "secret/resource", "sensitive details")
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/audit/logs", "tenant-b", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := decodeResponseBody(t, w)
	data := body["data"]
	if data != nil {
		arr, ok := data.([]interface{})
		if ok && len(arr) != 0 {
			t.Errorf("tenant-b should see 0 entries from tenant-a, got %d", len(arr))
		}
	}
}

func TestHandler_CrossTenantIsolation_Search(t *testing.T) {
	h, writer := newTestHandler()
	writer.Write("tenant-a", "admin", "action1", "resource1", "")
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/audit/search?actor=admin", "tenant-b", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := decodeResponseBody(t, w)
	data := body["data"]
	if data != nil {
		arr, ok := data.([]interface{})
		if ok && len(arr) != 0 {
			t.Errorf("tenant-b should see 0 entries from tenant-a search, got %d", len(arr))
		}
	}
}

func TestHandler_CrossTenantIsolation_Stats(t *testing.T) {
	h, writer := newTestHandler()
	writer.Write("tenant-a", "admin", "action1", "resource1", "")
	writer.Write("tenant-a", "admin", "action2", "resource2", "")
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/audit/stats", "tenant-b", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := decodeResponseBody(t, w)
	data := body["data"].(map[string]interface{})
	if int(data["total_entries"].(float64)) != 0 {
		t.Errorf("tenant-b should see 0 entries in stats, got %v", data["total_entries"])
	}
}

// ---------- Entry field verification ----------

func TestHandleLogs_VerifyEntryFields(t *testing.T) {
	h, writer := newTestHandler()
	writer.Write("tenant-a", "admin@example.com", "agent.register", "agent/budget-reconciler", "Registered v1.0.0")
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/audit/logs", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := decodeResponseBody(t, w)
	arr := body["data"].([]interface{})
	if len(arr) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(arr))
	}

	entry := arr[0].(map[string]interface{})
	if entry["id"] == nil || entry["id"] == "" {
		t.Error("entry missing 'id' field")
	}
	if entry["tenant_id"] != "tenant-a" {
		t.Errorf("tenant_id = %q, want %q", entry["tenant_id"], "tenant-a")
	}
	if entry["actor"] != "admin@example.com" {
		t.Errorf("actor = %q, want %q", entry["actor"], "admin@example.com")
	}
	if entry["action"] != "agent.register" {
		t.Errorf("action = %q, want %q", entry["action"], "agent.register")
	}
	if entry["resource"] != "agent/budget-reconciler" {
		t.Errorf("resource = %q, want %q", entry["resource"], "agent/budget-reconciler")
	}
	if entry["details"] != "Registered v1.0.0" {
		t.Errorf("details = %q, want %q", entry["details"], "Registered v1.0.0")
	}
	if entry["timestamp"] == nil || entry["timestamp"] == "" {
		t.Error("entry missing 'timestamp' field")
	}
}
