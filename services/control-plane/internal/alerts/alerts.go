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

// Status represents alert lifecycle states.
type Status string

const (
	StatusOpen           Status = "open"
	StatusAcknowledged   Status = "acknowledged"
	StatusResolved       Status = "resolved"
	StatusFalsePositive  Status = "false_positive"
)

// Alert represents an alert.
type Alert struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	AgentID        string    `json:"agent_id"`
	Severity       Severity  `json:"severity"`
	Title          string    `json:"title"`
	Message        string    `json:"message"`
	Status         Status    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}

// EscalationRule defines when and how to escalate an alert.
type EscalationRule struct {
	Severity   Severity      `json:"severity"`
	Timeout    time.Duration `json:"timeout"`
	NotifyType string        `json:"notify_type"` // "webhook", "email"
	Target     string        `json:"target"`
}

// Router routes alerts through escalation chains.
type Router struct {
	mu          sync.RWMutex
	alerts      map[string]*Alert // id -> alert
	counter     int
	escalations []EscalationRule
}

// NewRouter creates a new alert router.
func NewRouter() *Router {
	return &Router{
		alerts: make(map[string]*Alert),
		escalations: []EscalationRule{
			{Severity: SeverityCritical, Timeout: 5 * time.Minute, NotifyType: "webhook", Target: "default"},
			{Severity: SeverityWarning, Timeout: 30 * time.Minute, NotifyType: "webhook", Target: "default"},
		},
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
		Status:    StatusOpen,
		CreatedAt: time.Now(),
	}

	r.alerts[alert.ID] = alert
	return alert
}

// UpdateStatus updates the status of an alert.
func (r *Router) UpdateStatus(tenantID, alertID string, newStatus string) (*Alert, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	alert, ok := r.alerts[alertID]
	if !ok || alert.TenantID != tenantID {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}

	now := time.Now()
	switch Status(newStatus) {
	case StatusAcknowledged:
		alert.Status = StatusAcknowledged
		alert.AcknowledgedAt = &now
	case StatusResolved:
		alert.Status = StatusResolved
		alert.ResolvedAt = &now
	case StatusFalsePositive:
		alert.Status = StatusFalsePositive
		alert.ResolvedAt = &now
	default:
		return nil, fmt.Errorf("invalid status: %s", newStatus)
	}

	return alert, nil
}

// Get retrieves an alert by ID.
func (r *Router) Get(tenantID, alertID string) (*Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	alert, ok := r.alerts[alertID]
	if !ok || alert.TenantID != tenantID {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}
	return alert, nil
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

// ListOpen returns all open alerts for a tenant.
func (r *Router) ListOpen(tenantID string) []*Alert {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Alert
	for _, a := range r.alerts {
		if a.TenantID == tenantID && a.Status == StatusOpen {
			result = append(result, a)
		}
	}
	return result
}

// Count returns the total alert count for a tenant.
func (r *Router) Count(tenantID string) int {
	return len(r.List(tenantID))
}

// CountOpen returns the open alert count for a tenant.
func (r *Router) CountOpen(tenantID string) int {
	return len(r.ListOpen(tenantID))
}
