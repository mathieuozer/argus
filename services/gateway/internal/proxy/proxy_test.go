package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/argus-platform/argus/pkg/config"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *config.Base
		logger *zap.Logger
	}{
		{
			name:   "creates proxy with default config",
			cfg:    &config.Base{},
			logger: zap.NewNop(),
		},
		{
			name: "creates proxy with populated config",
			cfg: &config.Base{
				Env:      "production",
				LogLevel: "info",
			},
			logger: zap.NewNop(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := New(tc.cfg, tc.logger)
			if p == nil {
				t.Fatal("New() returned nil")
			}
			if p.cfg != tc.cfg {
				t.Error("proxy cfg does not match input")
			}
			if p.logger != tc.logger {
				t.Error("proxy logger does not match input")
			}
		})
	}
}

func TestServiceRoutes(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantURL string
	}{
		{
			name:    "agents route to orchestrator",
			prefix:  "/api/v1/agents",
			wantURL: "http://localhost:8082",
		},
		{
			name:    "tasks route to orchestrator",
			prefix:  "/api/v1/tasks",
			wantURL: "http://localhost:8082",
		},
		{
			name:    "telemetry route to telemetry service",
			prefix:  "/api/v1/telemetry",
			wantURL: "http://localhost:8083",
		},
		{
			name:    "identity route to identity service",
			prefix:  "/api/v1/identity",
			wantURL: "http://localhost:8081",
		},
		{
			name:    "catch-all api route to control-plane",
			prefix:  "/api/v1/",
			wantURL: "http://localhost:8084",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ServiceRoutes[tc.prefix]
			if !ok {
				t.Fatalf("ServiceRoutes missing prefix %q", tc.prefix)
			}
			if got != tc.wantURL {
				t.Errorf("ServiceRoutes[%q] = %q, want %q", tc.prefix, got, tc.wantURL)
			}
		})
	}

	t.Run("expected number of routes", func(t *testing.T) {
		want := 5
		if len(ServiceRoutes) != want {
			t.Errorf("ServiceRoutes has %d entries, want %d", len(ServiceRoutes), want)
		}
	})
}

func TestServeHTTP_UnknownPath_Returns404(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "completely unknown path",
			path: "/unknown/path",
		},
		{
			name: "root path",
			path: "/",
		},
		{
			name: "partial api prefix without v1",
			path: "/api/agents",
		},
		{
			name: "health check path not in routes",
			path: "/healthz",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := New(&config.Base{}, zap.NewNop())
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()

			p.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d for path %q", w.Code, http.StatusNotFound, tc.path)
			}
		})
	}
}

func TestServeHTTP_KnownPath_ProxiesRequest(t *testing.T) {
	// Spin up a real backend that responds with 200 and a known body.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "test")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend-ok"))
	}))
	defer backend.Close()

	// Temporarily replace one of the ServiceRoutes entries so the proxy
	// forwards to our test backend. We restore the original after the test.
	original := ServiceRoutes["/api/v1/agents"]
	ServiceRoutes["/api/v1/agents"] = backend.URL
	defer func() { ServiceRoutes["/api/v1/agents"] = original }()

	p := New(&config.Base{}, zap.NewNop())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	w := httptest.NewRecorder()

	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("X-Backend"); got != "test" {
		t.Errorf("X-Backend header = %q, want %q", got, "test")
	}
	if body := w.Body.String(); body != "backend-ok" {
		t.Errorf("body = %q, want %q", body, "backend-ok")
	}
}

func TestServeHTTP_PreservesPath(t *testing.T) {
	// Verify that the full request path is forwarded to the backend.
	var receivedPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	original := ServiceRoutes["/api/v1/agents"]
	ServiceRoutes["/api/v1/agents"] = backend.URL
	defer func() { ServiceRoutes["/api/v1/agents"] = original }()

	p := New(&config.Base{}, zap.NewNop())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/agent-123", nil)
	w := httptest.NewRecorder()

	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if receivedPath != "/api/v1/agents/agent-123" {
		t.Errorf("backend received path = %q, want %q", receivedPath, "/api/v1/agents/agent-123")
	}
}

func TestServeHTTP_ForwardsMethod(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var receivedMethod string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			original := ServiceRoutes["/api/v1/agents"]
			ServiceRoutes["/api/v1/agents"] = backend.URL
			defer func() { ServiceRoutes["/api/v1/agents"] = original }()

			p := New(&config.Base{}, zap.NewNop())
			req := httptest.NewRequest(method, "/api/v1/agents", nil)
			w := httptest.NewRecorder()

			p.ServeHTTP(w, req)

			if receivedMethod != method {
				t.Errorf("backend received method = %q, want %q", receivedMethod, method)
			}
		})
	}
}
