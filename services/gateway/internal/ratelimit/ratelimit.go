package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Limiter implements a per-key rate limiter supporting per-tenant and per-IP modes.
type Limiter struct {
	maxRequests int
	window      time.Duration
	mu          sync.Mutex
	clients     map[string]*clientInfo
}

type clientInfo struct {
	count   int
	resetAt time.Time
}

// New creates a new rate limiter.
func New(maxRequests int, window time.Duration) *Limiter {
	return &Limiter{
		maxRequests: maxRequests,
		window:      window,
		clients:     make(map[string]*clientInfo),
	}
}

// Allow checks if a request from the given key is allowed.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	client, exists := l.clients[key]
	if !exists || now.After(client.resetAt) {
		l.clients[key] = &clientInfo{count: 1, resetAt: now.Add(l.window)}
		return true
	}

	if client.count >= l.maxRequests {
		return false
	}

	client.count++
	return true
}

// extractKey determines the rate limit key from the request.
// Priority: X-Argus-Tenant-ID header (per-tenant) > X-Real-IP > X-Forwarded-For > RemoteAddr (per-IP).
func extractKey(r *http.Request) string {
	// Prefer per-tenant rate limiting when tenant ID is available
	if tenantID := r.Header.Get("X-Argus-Tenant-ID"); tenantID != "" {
		return "tenant:" + tenantID
	}

	// Fall back to IP-based: prefer trusted proxy headers
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return "ip:" + realIP
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For can contain multiple IPs; use the first (client IP)
		for i, c := range forwarded {
			if c == ',' {
				return "ip:" + forwarded[:i]
			}
		}
		return "ip:" + forwarded
	}

	// Strip port from RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "ip:" + r.RemoteAddr
	}
	return "ip:" + host
}

// Middleware returns an HTTP middleware that enforces rate limiting.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractKey(r)
		if !l.Allow(key) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
