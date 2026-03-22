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

func TestBuildServiceRoutes_Defaults(t *testing.T) {
	routes := buildServiceRoutes()

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
			got, ok := routes[tc.prefix]
			if !ok {
				t.Fatalf("routes missing prefix %q", tc.prefix)
			}
			if got != tc.wantURL {
				t.Errorf("routes[%q] = %q, want %q", tc.prefix, got, tc.wantURL)
			}
		})
	}

	t.Run("expected number of routes", func(t *testing.T) {
		want := 5
		if len(routes) != want {
			t.Errorf("routes has %d entries, want %d", len(routes), want)
		}
	})
}

func TestBuildServiceRoutes_FromEnv(t *testing.T) {
	t.Setenv("ARGUS_BACKEND_ORCHESTRATOR", "http://orchestrator:8082")
	t.Setenv("ARGUS_BACKEND_TELEMETRY", "http://telemetry:8083")
	t.Setenv("ARGUS_BACKEND_IDENTITY", "http://identity:8081")
	t.Setenv("ARGUS_BACKEND_CONTROL_PLANE", "http://control-plane:8084")

	routes := buildServiceRoutes()

	if routes["/api/v1/agents"] != "http://orchestrator:8082" {
		t.Errorf("agents = %q, want http://orchestrator:8082", routes["/api/v1/agents"])
	}
	if routes["/api/v1/telemetry"] != "http://telemetry:8083" {
		t.Errorf("telemetry = %q, want http://telemetry:8083", routes["/api/v1/telemetry"])
	}
	if routes["/api/v1/identity"] != "http://identity:8081" {
		t.Errorf("identity = %q, want http://identity:8081", routes["/api/v1/identity"])
	}
	if routes["/api/v1/"] != "http://control-plane:8084" {
		t.Errorf("control-plane = %q, want http://control-plane:8084", routes["/api/v1/"])
	}
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

func newProxyWithBackend(t *testing.T, prefix, backendURL string) *Proxy {
	t.Helper()
	p := New(&config.Base{}, zap.NewNop())
	p.routes[prefix] = backendURL
	return p
}

func TestServeHTTP_KnownPath_ProxiesRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "test")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("backend-ok"))
	}))
	defer backend.Close()

	p := newProxyWithBackend(t, "/api/v1/agents", backend.URL)
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
	var receivedPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	p := newProxyWithBackend(t, "/api/v1/agents", backend.URL)
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

			p := newProxyWithBackend(t, "/api/v1/agents", backend.URL)
			req := httptest.NewRequest(method, "/api/v1/agents", nil)
			w := httptest.NewRecorder()

			p.ServeHTTP(w, req)

			if receivedMethod != method {
				t.Errorf("backend received method = %q, want %q", receivedMethod, method)
			}
		})
	}
}
