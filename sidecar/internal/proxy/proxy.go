package proxy

import (
	"io"
	"net/http"

	"go.uber.org/zap"
)

// Proxy intercepts agent I/O transparently.
type Proxy struct {
	logger *zap.Logger
}

// New creates a new transparent proxy.
func New(logger *zap.Logger) *Proxy {
	return &Proxy{logger: logger}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Stub: in production, this would intercept and forward agent I/O
	p.logger.Debug("proxy request",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	io.WriteString(w, `{"error":{"code":"NO_UPSTREAM","message":"no upstream agent configured"}}`)
}
