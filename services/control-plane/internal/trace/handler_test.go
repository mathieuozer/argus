package trace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---------- handler test helpers ----------

func newHandlerWithMux(seed func(*Service)) (*Handler, *http.ServeMux) {
	svc := NewService()
	if seed != nil {
		seed(svc)
	}
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return h, mux
}

// ---------- GET /api/v1/traces ----------

func TestHandler_ListTraces_Success(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		queryParams    string
		seed           func(*Service)
		wantStatus     int
		wantTraceCount int
	}{
		{
			name:           "empty result with no data",
			tenantID:       "tenant-x",
			seed:           func(s *Service) {},
			wantStatus:     http.StatusOK,
			wantTraceCount: 0,
		},
		{
			name:     "returns all traces for tenant",
			tenantID: "tenant-a",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "t1", "agent-1")
				seedTrace(s, "tenant-a", "t2", "agent-2")
				seedTrace(s, "tenant-a", "t3", "agent-3")
			},
			wantStatus:     http.StatusOK,
			wantTraceCount: 3,
		},
		{
			name:        "agent_id filter returns matching traces only",
			tenantID:    "tenant-a",
			queryParams: "agent_id=agent-2",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "t1", "agent-1")
				seedTrace(s, "tenant-a", "t2", "agent-2")
				seedTrace(s, "tenant-a", "t3", "agent-3")
			},
			wantStatus:     http.StatusOK,
			wantTraceCount: 1,
		},
		{
			name:        "limit=2 caps results",
			tenantID:    "tenant-a",
			queryParams: "limit=2",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "t1", "agent-1")
				seedTrace(s, "tenant-a", "t2", "agent-2")
				seedTrace(s, "tenant-a", "t3", "agent-3")
			},
			wantStatus:     http.StatusOK,
			wantTraceCount: 2,
		},
		{
			name:        "invalid limit ignored, uses default",
			tenantID:    "tenant-a",
			queryParams: "limit=abc",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "t1", "agent-1")
			},
			wantStatus:     http.StatusOK,
			wantTraceCount: 1,
		},
		{
			name:        "negative limit ignored, uses default",
			tenantID:    "tenant-a",
			queryParams: "limit=-5",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "t1", "agent-1")
			},
			wantStatus:     http.StatusOK,
			wantTraceCount: 1,
		},
		{
			name:        "combined agent_id and limit filters",
			tenantID:    "tenant-a",
			queryParams: "agent_id=agent-1&limit=1",
			seed: func(s *Service) {
				seedTrace(s, "tenant-a", "t1", "agent-1")
				seedTrace(s, "tenant-a", "t2", "agent-1")
				seedTrace(s, "tenant-a", "t3", "agent-2")
			},
			wantStatus:     http.StatusOK,
			wantTraceCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, mux := newHandlerWithMux(tc.seed)

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

func TestHandler_ListTraces_TenantIsolation(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-a", "t1", "agent-1")
		seedTrace(s, "tenant-a", "t2", "agent-2")
		seedTrace(s, "tenant-b", "t3", "agent-3")
		seedTrace(s, "tenant-c", "t4", "agent-4")
	})

	tests := []struct {
		tenantID  string
		wantCount int
	}{
		{"tenant-a", 2},
		{"tenant-b", 1},
		{"tenant-c", 1},
		{"tenant-nonexistent", 0},
	}

	for _, tc := range tests {
		t.Run("tenant="+tc.tenantID, func(t *testing.T) {
			req := requestWithTenant(http.MethodGet, "/api/v1/traces", tc.tenantID, "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}

			body := responseBody(t, w)
			data := body["data"]
			if data == nil {
				if tc.wantCount != 0 {
					t.Errorf("data is nil, want %d traces", tc.wantCount)
				}
				return
			}
			arr := data.([]interface{})
			if len(arr) != tc.wantCount {
				t.Errorf("trace count = %d, want %d", len(arr), tc.wantCount)
			}
		})
	}
}

func TestHandler_ListTraces_MissingTenant(t *testing.T) {
	_, mux := newHandlerWithMux(nil)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/traces")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	body := responseBody(t, w)
	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "TENANT_REQUIRED" {
		t.Errorf("error code = %q, want %q", errObj["code"], "TENANT_REQUIRED")
	}
}

func TestHandler_ListTraces_MethodNotAllowed(t *testing.T) {
	_, mux := newHandlerWithMux(nil)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := requestWithTenant(method, "/api/v1/traces", "tenant-a", "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// ---------- GET /api/v1/traces/{id} ----------

func TestHandler_GetTraceByID_Success(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-a", "trace-100", "agent-1")
	})

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/trace-100", "tenant-a", "")
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
	if data["trace_id"] != "trace-100" {
		t.Errorf("trace_id = %q, want %q", data["trace_id"], "trace-100")
	}
	if data["total_spans"].(float64) != 2 {
		t.Errorf("total_spans = %v, want 2", data["total_spans"])
	}
}

func TestHandler_GetTraceByID_NotFound(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-a", "trace-1", "agent-1")
	})

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/does-not-exist", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	body := responseBody(t, w)
	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "TRACE_NOT_FOUND" {
		t.Errorf("error code = %q, want %q", errObj["code"], "TRACE_NOT_FOUND")
	}
}

func TestHandler_GetTraceByID_CrossTenantReturns404(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-a", "trace-1", "agent-1")
	})

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/trace-1", "tenant-b", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d for cross-tenant access", w.Code, http.StatusNotFound)
	}
}

func TestHandler_GetTraceByID_MissingTenant(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-a", "trace-1", "agent-1")
	})

	req := requestWithoutTenant(http.MethodGet, "/api/v1/traces/trace-1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_GetTraceByID_MethodNotAllowed(t *testing.T) {
	_, mux := newHandlerWithMux(nil)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := requestWithTenant(method, "/api/v1/traces/trace-1", "tenant-a", "")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// ---------- GET /api/v1/traces/{id}/flamegraph ----------

func TestHandler_FlameGraph_Success(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-a", "trace-fg", "agent-1")
	})

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/trace-fg/flamegraph", "tenant-a", "")
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
	if data["name"] != "root-op" {
		t.Errorf("flame graph root name = %q, want %q", data["name"], "root-op")
	}
	children, ok := data["children"].([]interface{})
	if !ok {
		t.Fatalf("children is not an array: %T", data["children"])
	}
	if len(children) != 1 {
		t.Errorf("children count = %d, want 1", len(children))
	}
}

func TestHandler_FlameGraph_NotFound(t *testing.T) {
	_, mux := newHandlerWithMux(nil)

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/nonexistent/flamegraph", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandler_FlameGraph_CrossTenantReturns404(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-a", "trace-1", "agent-1")
	})

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/trace-1/flamegraph", "tenant-b", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d for cross-tenant flamegraph access", w.Code, http.StatusNotFound)
	}
}

// ---------- Response format tests ----------

func TestHandler_ResponseContentType(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-a", "trace-1", "agent-1")
	})

	paths := []string{
		"/api/v1/traces",
		"/api/v1/traces/trace-1",
		"/api/v1/traces/trace-1/flamegraph",
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

func TestHandler_ResponseMetaTenantID(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-verify", "trace-1", "agent-1")
	})

	req := requestWithTenant(http.MethodGet, "/api/v1/traces", "tenant-verify", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := responseBody(t, w)
	meta, ok := body["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("meta is not an object: %T", body["meta"])
	}
	if meta["tenant_id"] != "tenant-verify" {
		t.Errorf("meta.tenant_id = %q, want %q", meta["tenant_id"], "tenant-verify")
	}
}

func TestHandler_ErrorResponseFormat(t *testing.T) {
	_, mux := newHandlerWithMux(nil)

	req := requestWithoutTenant(http.MethodGet, "/api/v1/traces")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q for error response", ct, "application/json")
	}

	body := responseBody(t, w)
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

// ---------- GET /api/v1/traces with trace detail verification ----------

func TestHandler_ListTraces_VerifyTraceFields(t *testing.T) {
	now := time.Now()
	_, mux := newHandlerWithMux(func(s *Service) {
		s.IngestSpan(&Span{
			SpanID:        "span-root",
			TraceID:       "trace-fields",
			TenantID:      "tenant-a",
			AgentID:       "agent-field-test",
			TaskID:        "task-1",
			OperationName: "main-operation",
			StartedAt:     now,
			DurationMs:    1000,
			Attributes:    map[string]string{"env": "test"},
		})
		s.IngestSpan(&Span{
			SpanID:        "span-child",
			TraceID:       "trace-fields",
			ParentSpanID:  "span-root",
			TenantID:      "tenant-a",
			AgentID:       "agent-field-test",
			TaskID:        "task-1",
			OperationName: "sub-operation",
			StartedAt:     now.Add(10 * time.Millisecond),
			DurationMs:    200,
			Attributes:    map[string]string{},
		})
	})

	req := requestWithTenant(http.MethodGet, "/api/v1/traces", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := responseBody(t, w)
	arr := body["data"].([]interface{})
	if len(arr) != 1 {
		t.Fatalf("expected 1 trace summary, got %d", len(arr))
	}

	summary := arr[0].(map[string]interface{})
	if summary["trace_id"] != "trace-fields" {
		t.Errorf("trace_id = %q, want %q", summary["trace_id"], "trace-fields")
	}
	if summary["root_operation"] != "main-operation" {
		t.Errorf("root_operation = %q, want %q", summary["root_operation"], "main-operation")
	}
	if summary["agent_id"] != "agent-field-test" {
		t.Errorf("agent_id = %q, want %q", summary["agent_id"], "agent-field-test")
	}
	if summary["total_spans"].(float64) != 2 {
		t.Errorf("total_spans = %v, want 2", summary["total_spans"])
	}
	if summary["has_errors"].(bool) != false {
		t.Error("expected has_errors to be false")
	}
}

func TestHandler_ListTraces_WithErrors(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTraceWithError(s, "tenant-a", "trace-err", "agent-1")
		seedTrace(s, "tenant-a", "trace-ok", "agent-2")
	})

	req := requestWithTenant(http.MethodGet, "/api/v1/traces", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := responseBody(t, w)
	arr := body["data"].([]interface{})
	if len(arr) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(arr))
	}

	errCount := 0
	for _, item := range arr {
		trace := item.(map[string]interface{})
		if trace["has_errors"].(bool) {
			errCount++
		}
	}
	if errCount != 1 {
		t.Errorf("expected 1 trace with errors, got %d", errCount)
	}
}

// ---------- GET /api/v1/traces/{id} detail verification ----------

func TestHandler_GetTraceByID_VerifyTreeStructure(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-a", "trace-tree", "agent-1")
	})

	req := requestWithTenant(http.MethodGet, "/api/v1/traces/trace-tree", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := responseBody(t, w)
	data := body["data"].(map[string]interface{})

	rootSpan, ok := data["root_span"].(map[string]interface{})
	if !ok {
		t.Fatal("root_span is not an object")
	}
	if rootSpan["operation_name"] != "root-op" {
		t.Errorf("root operation = %q, want %q", rootSpan["operation_name"], "root-op")
	}

	children, ok := rootSpan["children"].([]interface{})
	if !ok {
		t.Fatal("children is not an array")
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}

	child := children[0].(map[string]interface{})
	if child["operation_name"] != "child-op" {
		t.Errorf("child operation = %q, want %q", child["operation_name"], "child-op")
	}
}

// ---------- RegisterRoutes ----------

func TestHandler_RegisterRoutes(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "test-tenant", "trace-1", "agent-1")
	})

	routes := []struct {
		name   string
		method string
		path   string
	}{
		{name: "traces list", method: http.MethodGet, path: "/api/v1/traces"},
		{name: "trace by ID", method: http.MethodGet, path: "/api/v1/traces/trace-1"},
		{name: "flamegraph", method: http.MethodGet, path: "/api/v1/traces/trace-1/flamegraph"},
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

// ---------- writeJSON / writeError ----------

func TestHandler_WriteJSON_ValidJSON(t *testing.T) {
	_, mux := newHandlerWithMux(func(s *Service) {
		seedTrace(s, "tenant-a", "t1", "agent-1")
	})

	req := requestWithTenant(http.MethodGet, "/api/v1/traces", "tenant-a", "")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var raw json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}
}
