package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/control-plane/internal/alerts"
	"github.com/argus-platform/argus/services/control-plane/internal/audit"
)

// newTestHandler creates a Handler wired up with fresh in-memory alerts and audit stores.
func newTestHandler() (*Handler, *alerts.Router, *audit.Writer) {
	ar := alerts.NewRouter()
	aw := audit.NewWriter()
	h := New(ar, aw)
	return h, ar, aw
}

// serveMux returns an http.ServeMux with the dashboard routes registered.
func serveMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

// requestWithTenant builds an *http.Request whose context carries the given
// tenant ID, so the handler can extract it via tenancy.FromContext.
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

// requestWithoutTenant builds a request that has no tenant in the context.
func requestWithoutTenant(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

// responseBody is a helper that decodes the JSON response body into a generic map.
func responseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return body
}

// ---------- GET /api/v1/alerts ----------

func TestHandleAlerts_GET(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		seedAlerts     func(*alerts.Router)
		wantStatus     int
		wantAlertCount int
	}{
		{
			name:       "empty list when no alerts exist",
			tenantID:   "tenant-a",
			seedAlerts: func(r *alerts.Router) {},
			wantStatus: http.StatusOK,
			// nil slice marshals to JSON null, counted as 0
			wantAlertCount: 0,
		},
		{
			name:     "returns alerts belonging to tenant",
			tenantID: "tenant-a",
			seedAlerts: func(r *alerts.Router) {
				r.Fire("tenant-a", "agent-1", alerts.SeverityCritical, "alert-1", "msg-1")
				r.Fire("tenant-a", "agent-2", alerts.SeverityWarning, "alert-2", "msg-2")
			},
			wantStatus:     http.StatusOK,
			wantAlertCount: 2,
		},
		{
			name:     "tenant isolation - does not return other tenant alerts",
			tenantID: "tenant-a",
			seedAlerts: func(r *alerts.Router) {
				r.Fire("tenant-a", "agent-1", alerts.SeverityInfo, "mine", "m")
				r.Fire("tenant-b", "agent-2", alerts.SeverityCritical, "theirs", "t")
				r.Fire("tenant-b", "agent-3", alerts.SeverityWarning, "theirs-2", "t")
			},
			wantStatus:     http.StatusOK,
			wantAlertCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, ar, _ := newTestHandler()
			tc.seedAlerts(ar)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/alerts", tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := responseBody(t, w)
			data := body["data"]
			if data == nil {
				// nil JSON array
				if tc.wantAlertCount != 0 {
					t.Errorf("data is nil, want %d alerts", tc.wantAlertCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantAlertCount {
				t.Errorf("alert count = %d, want %d", len(arr), tc.wantAlertCount)
			}
		})
	}
}

func TestHandleAlerts_GET_ContentType(t *testing.T) {
	h, _, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/alerts", "tenant-ct", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestHandleAlerts_GET_MetaTenantID(t *testing.T) {
	h, _, _ := newTestHandler()
	mux := serveMux(h)

	tenantID := "tenant-meta"
	req := requestWithTenant(http.MethodGet, "/api/v1/alerts", tenantID, "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := responseBody(t, w)
	meta, ok := body["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("meta is not an object: %T", body["meta"])
	}
	if meta["tenant_id"] != tenantID {
		t.Errorf("meta.tenant_id = %q, want %q", meta["tenant_id"], tenantID)
	}
}

func TestHandleAlerts_MissingTenant(t *testing.T) {
	h, _, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/alerts")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	body := responseBody(t, w)
	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("error is not an object: %T", body["error"])
	}
	if errObj["code"] != "TENANT_REQUIRED" {
		t.Errorf("error.code = %q, want %q", errObj["code"], "TENANT_REQUIRED")
	}
}

// ---------- /api/v1/alerts/{id} ----------

func TestHandleAlertByID_PUT(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		alertID    string
		body       string
		wantStatus int
	}{
		{
			name:       "acknowledge existing alert",
			tenantID:   "tenant-1",
			alertID:    "alert-1",
			body:       `{"status":"acknowledged"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "resolve existing alert",
			tenantID:   "tenant-1",
			alertID:    "alert-1",
			body:       `{"status":"resolved"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "alert not found returns 404",
			tenantID:   "tenant-1",
			alertID:    "nonexistent",
			body:       `{"status":"acknowledged"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "cross-tenant access returns 404",
			tenantID:   "tenant-other",
			alertID:    "alert-1",
			body:       `{"status":"acknowledged"}`,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, ar, _ := newTestHandler()
			// Seed an alert owned by tenant-1
			ar.Fire("tenant-1", "agent-1", alerts.SeverityCritical, "test alert", "msg")
			mux := serveMux(h)

			path := "/api/v1/alerts/" + tc.alertID
			req := requestWithTenant(http.MethodPut, path, tc.tenantID, tc.body)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleAlertByID_GET_MethodNotAllowed(t *testing.T) {
	// The current handleAlertByID only supports PUT; GET should return 405.
	h, ar, _ := newTestHandler()
	ar.Fire("tenant-1", "agent-1", alerts.SeverityCritical, "test", "msg")
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/alerts/alert-1", "tenant-1", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}

	body := responseBody(t, w)
	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("error is not an object: %T", body["error"])
	}
	if errObj["code"] != "METHOD_NOT_ALLOWED" {
		t.Errorf("error.code = %q, want %q", errObj["code"], "METHOD_NOT_ALLOWED")
	}
}

func TestHandleAlertByID_ContentType(t *testing.T) {
	h, ar, _ := newTestHandler()
	ar.Fire("tenant-1", "agent-1", alerts.SeverityCritical, "test", "msg")
	mux := serveMux(h)

	req := requestWithTenant(http.MethodPut, "/api/v1/alerts/alert-1", "tenant-1", `{"status":"acknowledged"}`)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestHandleAlertByID_MissingTenant(t *testing.T) {
	h, ar, _ := newTestHandler()
	ar.Fire("tenant-1", "agent-1", alerts.SeverityCritical, "test", "msg")
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodPut, "/api/v1/alerts/alert-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- GET /api/v1/audit ----------

func TestHandleAuditLogs_GET(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		seedAudit      func(*audit.Writer)
		wantStatus     int
		wantEntryCount int
	}{
		{
			name:       "empty list when no entries exist",
			tenantID:   "tenant-a",
			seedAudit:  func(w *audit.Writer) {},
			wantStatus: http.StatusOK,
			// nil slice marshals to JSON null, counted as 0
			wantEntryCount: 0,
		},
		{
			name:     "returns audit entries for tenant",
			tenantID: "tenant-a",
			seedAudit: func(w *audit.Writer) {
				w.Write("tenant-a", "admin", "create_alert", "alert/1", "created alert")
				w.Write("tenant-a", "system", "update_alert", "alert/1", "acknowledged")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 2,
		},
		{
			name:     "tenant isolation - does not return other tenant entries",
			tenantID: "tenant-a",
			seedAudit: func(w *audit.Writer) {
				w.Write("tenant-a", "admin", "create_alert", "alert/1", "ours")
				w.Write("tenant-b", "admin", "create_alert", "alert/2", "theirs")
			},
			wantStatus:     http.StatusOK,
			wantEntryCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _, aw := newTestHandler()
			tc.seedAudit(aw)
			mux := serveMux(h)

			req := requestWithTenant(http.MethodGet, "/api/v1/audit", tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			body := responseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantEntryCount != 0 {
					t.Errorf("data is nil, want %d entries", tc.wantEntryCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantEntryCount {
				t.Errorf("entry count = %d, want %d", len(arr), tc.wantEntryCount)
			}
		})
	}
}

func TestHandleAuditLogs_GET_ContentType(t *testing.T) {
	h, _, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodGet, "/api/v1/audit", "tenant-ct", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestHandleAuditLogs_GET_MetaTenantID(t *testing.T) {
	h, _, _ := newTestHandler()
	mux := serveMux(h)

	tenantID := "tenant-meta-audit"
	req := requestWithTenant(http.MethodGet, "/api/v1/audit", tenantID, "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := responseBody(t, w)
	meta, ok := body["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("meta is not an object: %T", body["meta"])
	}
	if meta["tenant_id"] != tenantID {
		t.Errorf("meta.tenant_id = %q, want %q", meta["tenant_id"], tenantID)
	}
}

func TestHandleAuditLogs_POST_MethodNotAllowed(t *testing.T) {
	h, _, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithTenant(http.MethodPost, "/api/v1/audit", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleAuditLogs_MissingTenant(t *testing.T) {
	h, _, _ := newTestHandler()
	mux := serveMux(h)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/audit")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- POST /api/v1/alerts (creating an alert via the API) ----------

func TestHandleAlerts_POST(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		body       string
		wantStatus int
	}{
		{
			name:       "create alert succeeds",
			tenantID:   "tenant-1",
			body:       `{"agent_id":"agent-1","severity":"critical","title":"Failure imminent","message":"Token escalation"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid JSON body returns 400",
			tenantID:   "tenant-1",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _, _ := newTestHandler()
			mux := serveMux(h)

			req := requestWithTenant(http.MethodPost, "/api/v1/alerts", tc.tenantID, tc.body)
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

func TestHandleAlerts_POST_CreatesAuditEntry(t *testing.T) {
	h, _, aw := newTestHandler()
	mux := serveMux(h)

	tenantID := "tenant-audit"
	body := `{"agent_id":"agent-1","severity":"warning","title":"Latency spike","message":"p99 elevated"}`
	req := requestWithTenant(http.MethodPost, "/api/v1/alerts", tenantID, body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	entries := aw.List(tenantID)
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].Action != "create_alert" {
		t.Errorf("audit action = %q, want %q", entries[0].Action, "create_alert")
	}
}

// ---------- RegisterRoutes verifies all paths are served ----------

func TestRegisterRoutes(t *testing.T) {
	h, _, _ := newTestHandler()
	mux := serveMux(h)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "alerts list", method: http.MethodGet, path: "/api/v1/alerts"},
		{name: "alert by ID", method: http.MethodPut, path: "/api/v1/alerts/alert-1"},
		{name: "audit logs", method: http.MethodGet, path: "/api/v1/audit"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tenancy.WithTenant(context.Background(), "test-tenant")
			req := httptest.NewRequest(tc.method, tc.path, nil).WithContext(ctx)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// A 404 would mean the route is not registered at all
			if w.Code == http.StatusNotFound {
				t.Errorf("route %s %s returned 404 — not registered", tc.method, tc.path)
			}
		})
	}
}
