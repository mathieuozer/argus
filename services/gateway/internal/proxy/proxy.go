package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"

	"github.com/argus-platform/argus/pkg/config"
	"go.uber.org/zap"
)

// ServiceRoutes maps path prefixes to backend service addresses.
var ServiceRoutes = map[string]string{
	"/api/v1/agents":    "http://localhost:8082",
	"/api/v1/tasks":     "http://localhost:8082",
	"/api/v1/telemetry": "http://localhost:8083",
	"/api/v1/identity":  "http://localhost:8081",
	"/api/v1/":          "http://localhost:8084",
}

// sortedPrefixes returns route prefixes sorted by length descending
// so that longer (more specific) prefixes match first.
func sortedPrefixes() []string {
	prefixes := make([]string, 0, len(ServiceRoutes))
	for p := range ServiceRoutes {
		prefixes = append(prefixes, p)
	}
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i]) > len(prefixes[j])
	})
	return prefixes
}

// Proxy is a reverse proxy that routes requests to backend services.
type Proxy struct {
	cfg    *config.Base
	logger *zap.Logger
}

// New creates a new reverse proxy.
func New(cfg *config.Base, logger *zap.Logger) *Proxy {
	return &Proxy{cfg: cfg, logger: logger}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Match longest prefix first to avoid catch-all matching before specific routes
	for _, prefix := range sortedPrefixes() {
		if len(r.URL.Path) >= len(prefix) && r.URL.Path[:len(prefix)] == prefix {
			target := ServiceRoutes[prefix]
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
