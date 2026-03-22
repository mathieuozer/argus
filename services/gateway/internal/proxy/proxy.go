package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"

	"github.com/argus-platform/argus/pkg/config"
	"go.uber.org/zap"
)

// getBackendAddr reads a backend service address from environment variables,
// falling back to the provided default (localhost for local dev).
func getBackendAddr(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

// buildServiceRoutes constructs the route map from environment variables.
// Environment variables:
//
//	ARGUS_BACKEND_ORCHESTRATOR (default: http://localhost:8082)
//	ARGUS_BACKEND_TELEMETRY   (default: http://localhost:8083)
//	ARGUS_BACKEND_IDENTITY    (default: http://localhost:8081)
//	ARGUS_BACKEND_CONTROL_PLANE (default: http://localhost:8084)
func buildServiceRoutes() map[string]string {
	orchestrator := getBackendAddr("ARGUS_BACKEND_ORCHESTRATOR", "http://localhost:8082")
	telemetryAddr := getBackendAddr("ARGUS_BACKEND_TELEMETRY", "http://localhost:8083")
	identityAddr := getBackendAddr("ARGUS_BACKEND_IDENTITY", "http://localhost:8081")
	controlPlane := getBackendAddr("ARGUS_BACKEND_CONTROL_PLANE", "http://localhost:8084")

	return map[string]string{
		"/api/v1/agents":    orchestrator,
		"/api/v1/tasks":     orchestrator,
		"/api/v1/telemetry": telemetryAddr,
		"/api/v1/identity":  identityAddr,
		"/api/v1/":          controlPlane,
	}
}

// Proxy is a reverse proxy that routes requests to backend services.
type Proxy struct {
	cfg    *config.Base
	logger *zap.Logger
	routes map[string]string
}

// New creates a new reverse proxy.
func New(cfg *config.Base, logger *zap.Logger) *Proxy {
	routes := buildServiceRoutes()
	logger.Info("gateway proxy routes configured",
		zap.Any("routes", routes),
	)
	return &Proxy{cfg: cfg, logger: logger, routes: routes}
}

// sortedPrefixes returns route prefixes sorted by length descending
// so that longer (more specific) prefixes match first.
func sortedPrefixes(routes map[string]string) []string {
	prefixes := make([]string, 0, len(routes))
	for p := range routes {
		prefixes = append(prefixes, p)
	}
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i]) > len(prefixes[j])
	})
	return prefixes
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Match longest prefix first to avoid catch-all matching before specific routes
	for _, prefix := range sortedPrefixes(p.routes) {
		if len(r.URL.Path) >= len(prefix) && r.URL.Path[:len(prefix)] == prefix {
			target := p.routes[prefix]
			targetURL, err := url.Parse(target)
			if err != nil {
				p.logger.Error("invalid target URL", zap.String("target", target), zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			proxy := httputil.NewSingleHostReverseProxy(targetURL)
			proxy.ServeHTTP(w, r)
			return
		}
	}

	http.NotFound(w, r)
}
