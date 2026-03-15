package ratelimit

import (
	"net/http"
	"sync"
	"time"
)

// Limiter implements a simple per-IP rate limiter.
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

// Middleware returns an HTTP middleware that enforces rate limiting.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr
		if !l.Allow(key) {
			http.Error(w, `{"error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
