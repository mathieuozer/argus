package alerts

import (
	"fmt"
	"sync"
	"time"
)

// Severity represents alert severity levels.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Alert represents an alert.
type Alert struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	AgentID   string    `json:"agent_id"`
	Severity  Severity  `json:"severity"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Status    string    `json:"status"` // open, acknowledged, resolved
	CreatedAt time.Time `json:"created_at"`
}

// Router routes alerts through escalation chains.
type Router struct {
	mu      sync.RWMutex
	alerts  []*Alert
	counter int
}

// NewRouter creates a new alert router.
func NewRouter() *Router {
	return &Router{
		alerts: make([]*Alert, 0),
	}
}

// Fire creates and routes a new alert.
func (r *Router) Fire(tenantID, agentID string, severity Severity, title, message string) *Alert {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counter++
	alert := &Alert{
		ID:        fmt.Sprintf("alert-%d", r.counter),
		TenantID:  tenantID,
		AgentID:   agentID,
		Severity:  severity,
		Title:     title,
		Message:   message,
		Status:    "open",
		CreatedAt: time.Now(),
	}

	r.alerts = append(r.alerts, alert)
	return alert
}

// List returns all alerts for a tenant.
func (r *Router) List(tenantID string) []*Alert {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Alert
	for _, a := range r.alerts {
		if a.TenantID == tenantID {
			result = append(result, a)
		}
	}
	return result
}
