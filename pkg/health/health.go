package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckFunc is a function that checks the health of a dependency.
type CheckFunc func(ctx context.Context) error

// Component holds the status of a single dependency.
type Component struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
}

// Response is the health check response.
type Response struct {
	Status     Status      `json:"status"`
	Components []Component `json:"components,omitempty"`
	Uptime     string      `json:"uptime"`
}

// Checker manages health checks for a service.
type Checker struct {
	mu        sync.RWMutex
	checks    map[string]CheckFunc
	startedAt time.Time
}

// NewChecker creates a new health checker.
func NewChecker() *Checker {
	return &Checker{
		checks:    make(map[string]CheckFunc),
		startedAt: time.Now(),
	}
}

// AddCheck registers a named health check function.
func (c *Checker) AddCheck(name string, fn CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = fn
}

// LiveHandler returns an HTTP handler for liveness probes.
// Liveness checks only that the process is running.
func (c *Checker) LiveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Response{
			Status: StatusHealthy,
			Uptime: time.Since(c.startedAt).Truncate(time.Second).String(),
		})
	}
}

// ReadyHandler returns an HTTP handler for readiness probes.
// Readiness checks all registered dependencies.
func (c *Checker) ReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		c.mu.RLock()
		checks := make(map[string]CheckFunc, len(c.checks))
		for k, v := range c.checks {
			checks[k] = v
		}
		c.mu.RUnlock()

		overall := StatusHealthy
		components := make([]Component, 0, len(checks))

		for name, fn := range checks {
			comp := Component{Name: name, Status: StatusHealthy}
			if err := fn(ctx); err != nil {
				comp.Status = StatusUnhealthy
				comp.Message = err.Error()
				overall = StatusUnhealthy
			}
			components = append(components, comp)
		}

		resp := Response{
			Status:     overall,
			Components: components,
			Uptime:     time.Since(c.startedAt).Truncate(time.Second).String(),
		}

		w.Header().Set("Content-Type", "application/json")
		if overall == StatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// Handler returns an HTTP handler for the combined health endpoint.
// It runs readiness checks and returns the full status.
func (c *Checker) Handler() http.HandlerFunc {
	return c.ReadyHandler()
}
