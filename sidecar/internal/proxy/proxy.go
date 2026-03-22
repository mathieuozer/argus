package proxy

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"go.uber.org/zap"
)

// Proxy intercepts agent I/O transparently.
type Proxy struct {
	logger       *zap.Logger
	upstreamAddr string
	reverseProxy *httputil.ReverseProxy
	requestCount int64
}

// New creates a new transparent proxy.
func New(logger *zap.Logger) *Proxy {
	upstreamAddr := os.Getenv("ARGUS_UPSTREAM_ADDR")
	if upstreamAddr == "" {
		upstreamAddr = "http://localhost:8000" // default agent port
	}

	p := &Proxy{
		logger:       logger,
		upstreamAddr: upstreamAddr,
	}

	target, err := url.Parse(upstreamAddr)
	if err != nil {
		logger.Error("invalid upstream address", zap.String("addr", upstreamAddr), zap.Error(err))
		return p
	}

	p.reverseProxy = &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
		},
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("proxy error",
				zap.String("path", r.URL.Path),
				zap.Error(err),
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			io.WriteString(w, `{"error":{"code":"UPSTREAM_ERROR","message":"upstream agent unavailable"}}`)
		},
	}

	return p
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	p.logger.Debug("proxy request",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("upstream", p.upstreamAddr),
	)

	if p.reverseProxy == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, `{"error":{"code":"NO_UPSTREAM","message":"no upstream agent configured"}}`)
		return
	}

	// Wrap response writer to capture status and size
	wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	p.reverseProxy.ServeHTTP(wrapped, r)

	p.requestCount++
	p.logger.Debug("proxy response",
		zap.Int("status", wrapped.statusCode),
		zap.Int64("bytes", wrapped.bytesWritten),
		zap.Duration("duration", time.Since(start)),
	)
}

// Stats returns proxy statistics.
func (p *Proxy) Stats() map[string]interface{} {
	return map[string]interface{}{
		"upstream_addr": p.upstreamAddr,
		"request_count": p.requestCount,
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}
