package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/argus-platform/argus/pkg/tenancy"
	"go.uber.org/zap"
)

func TestTenantHTTP(t *testing.T) {
	// A downstream handler that records the tenant ID from context.
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid, err := tenancy.FromContext(r.Context())
		if err != nil {
			http.Error(w, "tenant not in context", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(tid))
	})

	handler := TenantHTTP(downstream)

	tests := []struct {
		name           string
		tenantHeader   string
		setHeader      bool
		wantStatus     int
		wantBodySubstr string
	}{
		{
			name:           "missing tenant header returns 400",
			setHeader:      false,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "TENANT_REQUIRED",
		},
		{
			name:           "empty tenant header returns 400",
			tenantHeader:   "",
			setHeader:      true,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "TENANT_REQUIRED",
		},
		{
			name:           "valid tenant header sets context and returns 200",
			tenantHeader:   "ministry-finance-tr",
			setHeader:      true,
			wantStatus:     http.StatusOK,
			wantBodySubstr: "ministry-finance-tr",
		},
		{
			name:           "different tenant ID propagated",
			tenantHeader:   "acme-corp",
			setHeader:      true,
			wantStatus:     http.StatusOK,
			wantBodySubstr: "acme-corp",
		},
		{
			name:           "UUID-style tenant ID",
			tenantHeader:   "550e8400-e29b-41d4-a716-446655440000",
			setHeader:      true,
			wantStatus:     http.StatusOK,
			wantBodySubstr: "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
			if tc.setHeader {
				req.Header.Set(TenantHeader, tc.tenantHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tc.wantStatus {
				t.Errorf("status = %d; want %d", res.StatusCode, tc.wantStatus)
			}

			body, _ := io.ReadAll(res.Body)
			if !strings.Contains(string(body), tc.wantBodySubstr) {
				t.Errorf("body = %q; want to contain %q", string(body), tc.wantBodySubstr)
			}
		})
	}
}

func TestTenantHTTPErrorResponseFormat(t *testing.T) {
	handler := TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("downstream should not be called when tenant header is missing")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	bodyStr := string(body)

	// Verify the error response contains expected JSON fields
	if !strings.Contains(bodyStr, `"error"`) {
		t.Errorf("error response missing 'error' key: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"code"`) {
		t.Errorf("error response missing 'code' key: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"message"`) {
		t.Errorf("error response missing 'message' key: %s", bodyStr)
	}
}

func TestRequestLogger(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		tenantHeader string
		wantStatus   int
	}{
		{
			name:         "GET request is logged and forwarded",
			method:       http.MethodGet,
			path:         "/api/v1/agents",
			tenantHeader: "test-tenant",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "POST request is logged and forwarded",
			method:       http.MethodPost,
			path:         "/api/v1/tasks",
			tenantHeader: "another-tenant",
			wantStatus:   http.StatusOK,
		},
		{
			name:       "request without tenant header is still logged",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
		},
		{
			name:         "DELETE request is logged",
			method:       http.MethodDelete,
			path:         "/api/v1/agents/agent-123",
			tenantHeader: "delete-tenant",
			wantStatus:   http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l, err := zap.NewDevelopment()
			if err != nil {
				t.Fatalf("zap.NewDevelopment() error: %v", err)
			}

			called := false
			downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			handler := RequestLogger(l)(downstream)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.tenantHeader != "" {
				req.Header.Set(TenantHeader, tc.tenantHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if !called {
				t.Error("downstream handler was not called")
			}

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tc.wantStatus {
				t.Errorf("status = %d; want %d", res.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := CORS(downstream)

	tests := []struct {
		name              string
		method            string
		wantStatus        int
		wantBody          string
		checkCORSHeaders  bool
		expectNoBody      bool
	}{
		{
			name:             "OPTIONS preflight returns 204",
			method:           http.MethodOptions,
			wantStatus:       http.StatusNoContent,
			checkCORSHeaders: true,
			expectNoBody:     true,
		},
		{
			name:             "GET request has CORS headers and forwards to downstream",
			method:           http.MethodGet,
			wantStatus:       http.StatusOK,
			wantBody:         "ok",
			checkCORSHeaders: true,
		},
		{
			name:             "POST request has CORS headers and forwards to downstream",
			method:           http.MethodPost,
			wantStatus:       http.StatusOK,
			wantBody:         "ok",
			checkCORSHeaders: true,
		},
		{
			name:             "PUT request has CORS headers and forwards to downstream",
			method:           http.MethodPut,
			wantStatus:       http.StatusOK,
			wantBody:         "ok",
			checkCORSHeaders: true,
		},
		{
			name:             "DELETE request has CORS headers and forwards to downstream",
			method:           http.MethodDelete,
			wantStatus:       http.StatusOK,
			wantBody:         "ok",
			checkCORSHeaders: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/agents", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tc.wantStatus {
				t.Errorf("status = %d; want %d", res.StatusCode, tc.wantStatus)
			}

			if tc.checkCORSHeaders {
				corsHeaders := []struct {
					header string
					want   string
				}{
					{"Access-Control-Allow-Origin", "*"},
					{"Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS"},
					{"Access-Control-Allow-Headers", "Content-Type, Authorization, " + TenantHeader},
				}

				for _, h := range corsHeaders {
					got := res.Header.Get(h.header)
					if got != h.want {
						t.Errorf("header %q = %q; want %q", h.header, got, h.want)
					}
				}
			}

			body, _ := io.ReadAll(res.Body)
			if tc.expectNoBody {
				if len(body) != 0 {
					t.Errorf("expected empty body for preflight; got %q", string(body))
				}
			} else if tc.wantBody != "" {
				if string(body) != tc.wantBody {
					t.Errorf("body = %q; want %q", string(body), tc.wantBody)
				}
			}
		})
	}
}

func TestCORSPreflightDoesNotCallDownstream(t *testing.T) {
	called := false
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := CORS(downstream)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Error("CORS preflight should not call downstream handler")
	}
}

func TestCORSRegularRequestCallsDownstream(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			called := false
			downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			})

			handler := CORS(downstream)
			req := httptest.NewRequest(method, "/api/v1/agents", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if !called {
				t.Errorf("downstream not called for %s request", method)
			}
		})
	}
}

func TestMiddlewareChaining(t *testing.T) {
	// Verify that CORS, RequestLogger, and TenantHTTP can be chained together.
	l, _ := zap.NewDevelopment()

	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid, err := tenancy.FromContext(r.Context())
		if err != nil {
			http.Error(w, "no tenant", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(tid))
	})

	// Chain: CORS -> RequestLogger -> TenantHTTP -> downstream
	handler := CORS(RequestLogger(l)(TenantHTTP(downstream)))

	tests := []struct {
		name         string
		method       string
		tenantHeader string
		wantStatus   int
		wantBody     string
	}{
		{
			name:         "full chain with valid tenant",
			method:       http.MethodGet,
			tenantHeader: "chain-tenant",
			wantStatus:   http.StatusOK,
			wantBody:     "chain-tenant",
		},
		{
			name:       "preflight bypasses downstream",
			method:     http.MethodOptions,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/agents", nil)
			if tc.tenantHeader != "" {
				req.Header.Set(TenantHeader, tc.tenantHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tc.wantStatus {
				t.Errorf("status = %d; want %d", res.StatusCode, tc.wantStatus)
			}

			// Always check CORS headers are present
			if res.Header.Get("Access-Control-Allow-Origin") != "*" {
				t.Error("CORS header missing from chained response")
			}

			if tc.wantBody != "" {
				body, _ := io.ReadAll(res.Body)
				if string(body) != tc.wantBody {
					t.Errorf("body = %q; want %q", string(body), tc.wantBody)
				}
			}
		})
	}
}
