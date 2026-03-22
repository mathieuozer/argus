package trace

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

func strPtr(s string) *string { return &s }

func seedTrace(svc *Service, tenantID, traceID, agentID string) {
	now := time.Now()
	svc.IngestSpan(&Span{
		SpanID:        traceID + "-root",
		TraceID:       traceID,
		TenantID:      tenantID,
		AgentID:       agentID,
		TaskID:        "task-1",
		OperationName: "root-op",
		StartedAt:     now,
		DurationMs:    500,
		Attributes:    map[string]string{"key": "val"},
	})
	svc.IngestSpan(&Span{
		SpanID:        traceID + "-child",
		TraceID:       traceID,
		ParentSpanID:  traceID + "-root",
		TenantID:      tenantID,
		AgentID:       agentID,
		TaskID:        "task-1",
		OperationName: "child-op",
		StartedAt:     now.Add(10 * time.Millisecond),
		DurationMs:    200,
		Attributes:    map[string]string{"step": "1"},
	})
}

func seedTraceWithError(svc *Service, tenantID, traceID, agentID string) {
	now := time.Now()
	svc.IngestSpan(&Span{
		SpanID:        traceID + "-root",
		TraceID:       traceID,
		TenantID:      tenantID,
		AgentID:       agentID,
		TaskID:        "task-err",
		OperationName: "error-op",
		StartedAt:     now,
		DurationMs:    300,
		Attributes:    map[string]string{},
		ErrorCode:     strPtr("TIMEOUT"),
	})
}

// ---------- Service unit tests ----------

func TestListTraces(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		agentID   string
		limit     int
		seed      func(*Service)
		wantCount int
	}{
		{
			name:      "empty service returns empty list",
			tenantID:  "tenant-a",
			limit:     50,
			seed:      func(s *Service) {},
			wantCount: 0,
		},
		{
			name:     "returns traces for matching tenant",
			tenantID: "tenant-a",
			limit:    50,
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
				seedTrace(s, "tenant-a", "trace-2", "agent-2")
			},
			wantCount: 2,
		},
		{
			name:     "tenant isolation - does not return other tenant traces",
			tenantID: "tenant-a",
			limit:    50,
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
				seedTrace(s, "tenant-b", "trace-2", "agent-2")
			},
			wantCount: 1,
		},
		{
			name:     "filters by agent ID",
			tenantID: "tenant-a",
			agentID:  "agent-1",
			limit:    50,
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
				seedTrace(s, "tenant-a", "trace-2", "agent-2")
			},
			wantCount: 1,
		},
		{
			name:     "respects limit",
			tenantID: "tenant-a",
			limit:    1,
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
				seedTrace(s, "tenant-a", "trace-2", "agent-2")
			},
			wantCount: 1,
		},
		{
			name:     "limit of zero means no limit",
			tenantID: "tenant-a",
			limit:    0,
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
				seedTrace(s, "tenant-a", "trace-2", "agent-2")
			},
			wantCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService()
			tc.seed(svc)

			traces := svc.ListTraces(tc.tenantID, tc.agentID, tc.limit)
			if len(traces) != tc.wantCount {
				t.Errorf("got %d traces, want %d", len(traces), tc.wantCount)
			}
		})
	}
}

func TestListTraces_HasErrors(t *testing.T) {
	svc := NewService()
	seedTraceWithError(svc, "tenant-a", "trace-err", "agent-1")
	seedTrace(svc, "tenant-a", "trace-ok", "agent-2")

	traces := svc.ListTraces("tenant-a", "", 50)
	errCount := 0
	for _, tr := range traces {
		if tr.HasErrors {
			errCount++
		}
	}
	if errCount != 1 {
		t.Errorf("expected 1 trace with errors, got %d", errCount)
	}
}

func TestGetTrace(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		traceID   string
		seed      func(*Service)
		wantNil   bool
		wantSpans int
	}{
		{
			name:     "returns trace detail",
			tenantID: "tenant-a",
			traceID:  "trace-1",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
			},
			wantNil:   false,
			wantSpans: 2,
		},
		{
			name:     "not found for missing trace",
			tenantID: "tenant-a",
			traceID:  "nonexistent",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
			},
			wantNil: true,
		},
		{
			name:     "tenant isolation - cannot access other tenant trace",
			tenantID: "tenant-a",
			traceID:  "trace-1",
			seed: func(s *Service) {
				seedTrace(s, "tenant-b", "trace-1", "agent-1")
			},
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService()
			tc.seed(svc)

			detail := svc.GetTrace(tc.tenantID, tc.traceID)
			if tc.wantNil {
				if detail != nil {
					t.Errorf("expected nil, got trace detail")
				}
				return
			}
			if detail == nil {
				t.Fatal("expected trace detail, got nil")
			}
			if detail.TotalSpans != tc.wantSpans {
				t.Errorf("TotalSpans = %d, want %d", detail.TotalSpans, tc.wantSpans)
			}
		})
	}
}

func TestGetTrace_TreeStructure(t *testing.T) {
	svc := NewService()
	seedTrace(svc, "tenant-a", "trace-1", "agent-1")

	detail := svc.GetTrace("tenant-a", "trace-1")
	if detail == nil {
		t.Fatal("expected trace detail, got nil")
	}
	if detail.RootSpan == nil {
		t.Fatal("expected root span, got nil")
	}
	if detail.RootSpan.OperationName != "root-op" {
		t.Errorf("root operation = %q, want %q", detail.RootSpan.OperationName, "root-op")
	}
	if len(detail.RootSpan.Children) != 1 {
		t.Fatalf("expected 1 child span, got %d", len(detail.RootSpan.Children))
	}
	if detail.RootSpan.Children[0].OperationName != "child-op" {
		t.Errorf("child operation = %q, want %q", detail.RootSpan.Children[0].OperationName, "child-op")
	}
}

func TestGetTrace_HasErrors(t *testing.T) {
	svc := NewService()
	seedTraceWithError(svc, "tenant-a", "trace-err", "agent-1")

	detail := svc.GetTrace("tenant-a", "trace-err")
	if detail == nil {
		t.Fatal("expected trace detail, got nil")
	}
	if !detail.HasErrors {
		t.Error("expected HasErrors to be true")
	}
}

func TestGetFlameGraph(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		traceID  string
		seed     func(*Service)
		wantNil  bool
		wantName string
	}{
		{
			name:     "returns flame graph for existing trace",
			tenantID: "tenant-a",
			traceID:  "trace-1",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
			},
			wantNil:  false,
			wantName: "root-op",
		},
		{
			name:     "returns nil for missing trace",
			tenantID: "tenant-a",
			traceID:  "nonexistent",
			seed:     func(s *Service) {},
			wantNil:  true,
		},
		{
			name:     "tenant isolation - cannot access other tenant flame graph",
			tenantID: "tenant-a",
			traceID:  "trace-1",
			seed: func(s *Service) {
				seedTrace(s, "tenant-b", "trace-1", "agent-1")
			},
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService()
			tc.seed(svc)

			fg := svc.GetFlameGraph(tc.tenantID, tc.traceID)
			if tc.wantNil {
				if fg != nil {
					t.Errorf("expected nil, got flame graph")
				}
				return
			}
			if fg == nil {
				t.Fatal("expected flame graph, got nil")
			}
			if fg.Name != tc.wantName {
				t.Errorf("root name = %q, want %q", fg.Name, tc.wantName)
			}
		})
	}
}

func TestGetFlameGraph_Children(t *testing.T) {
	svc := NewService()
	seedTrace(svc, "tenant-a", "trace-1", "agent-1")

	fg := svc.GetFlameGraph("tenant-a", "trace-1")
	if fg == nil {
		t.Fatal("expected flame graph, got nil")
	}
	if len(fg.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(fg.Children))
	}
	if fg.Children[0].Name != "child-op" {
		t.Errorf("child name = %q, want %q", fg.Children[0].Name, "child-op")
	}
	if fg.Children[0].Value != 200 {
		t.Errorf("child value = %d, want %d", fg.Children[0].Value, 200)
	}
}

// ---------- Handler HTTP tests ----------

func TestHandleTraces_GET(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		queryParams    string
		seed           func(*Service)
		wantStatus     int
		wantTraceCount int
	}{
		{
			name:           "empty list when no traces exist",
			tenantID:       "tenant-a",
			seed:           func(s *Service) {},
			wantStatus:     http.StatusOK,
			wantTraceCount: 0,
		},
		{
			name:     "returns traces for tenant",
			tenantID: "tenant-a",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
				seedTrace(s, "tenant-a", "trace-2", "agent-2")
			},
			wantStatus:     http.StatusOK,
			wantTraceCount: 2,
		},
		{
			name:        "filters by agent_id query param",
			tenantID:    "tenant-a",
			queryParams: "agent_id=agent-1",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
				seedTrace(s, "tenant-a", "trace-2", "agent-2")
			},
			wantStatus:     http.StatusOK,
			wantTraceCount: 1,
		},
		{
			name:        "respects limit query param",
			tenantID:    "tenant-a",
			queryParams: "limit=1",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
				seedTrace(s, "tenant-a", "trace-2", "agent-2")
			},
			wantStatus:     http.StatusOK,
			wantTraceCount: 1,
		},
		{
			name:     "tenant isolation",
			tenantID: "tenant-a",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
				seedTrace(s, "tenant-b", "trace-2", "agent-2")
			},
			wantStatus:     http.StatusOK,
			wantTraceCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService()
			tc.seed(svc)
			h := NewHandler(svc)
			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			path := "/api/v1/traces"
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
				if tc.wantTraceCount != 0 {
					t.Errorf("data is nil, want %d traces", tc.wantTraceCount)
				}
				return
			}
			arr, ok := data.([]interface{})
			if !ok {
				t.Fatalf("data is not an array: %T", data)
			}
			if len(arr) != tc.wantTraceCount {
				t.Errorf("trace count = %d, want %d", len(arr), tc.wantTraceCount)
			}
		})
	}
}

func TestHandleTraces_MissingTenant(t *testing.T) {
	svc := NewService()
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/traces")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleTraces_MethodNotAllowed(t *testing.T) {
	svc := NewService()
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := requestWithTenant(http.MethodPost, "/api/v1/traces", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleTraceByID_GET(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		traceID    string
		seed       func(*Service)
		wantStatus int
	}{
		{
			name:     "returns trace detail",
			tenantID: "tenant-a",
			traceID:  "trace-1",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "not found for missing trace",
			tenantID: "tenant-a",
			traceID:  "nonexistent",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "trace-1", "agent-1")
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "cross-tenant access returns 404",
			tenantID: "tenant-a",
			traceID:  "trace-1",
			seed: func(s *Service) {
				seedTrace(s, "tenant-b", "trace-1", "agent-1")
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService()
			tc.seed(svc)
			h := NewHandler(svc)
			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			req := requestWithTenant(http.MethodGet, "/api/v1/traces/"+tc.traceID, tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleTraceByID_FlameGraph(t *testing.T) {
	svc := NewService()
	seedTrace(svc, "tenant-a", "trace-1", "agent-1")
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/trace-1/flamegraph", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := responseBody(t, w)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data is not an object: %T", body["data"])
	}
	if data["name"] != "root-op" {
		t.Errorf("flame graph root name = %q, want %q", data["name"], "root-op")
	}
}

func TestHandleTraceByID_FlameGraph_NotFound(t *testing.T) {
	svc := NewService()
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/nonexistent/flamegraph", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleTraceByID_MissingTenant(t *testing.T) {
	svc := NewService()
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/traces/trace-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleTraceByID_ContentType(t *testing.T) {
	svc := NewService()
	seedTrace(svc, "tenant-a", "trace-1", "agent-1")
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/trace-1", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestHandleTraceByID_MetaTenantID(t *testing.T) {
	svc := NewService()
	seedTrace(svc, "tenant-meta", "trace-1", "agent-1")
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/trace-1", "tenant-meta", "")
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
