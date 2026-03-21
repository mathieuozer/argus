package slo

import (
	"fmt"
	"sync"
	"time"
)

// SLOType defines the type of service level objective.
type SLOType string

const (
	SLOTypeAvailability SLOType = "availability"
	SLOTypeLatency      SLOType = "latency"
	SLOTypeErrorRate    SLOType = "error_rate"
	SLOTypeThroughput   SLOType = "throughput"
)

// SLO represents a Service Level Objective definition.
type SLO struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	AgentID     string    `json:"agent_id,omitempty"` // empty means all agents
	Type        SLOType   `json:"type"`
	Target      float64   `json:"target"`   // target value (e.g. 99.9 for availability)
	Window      string    `json:"window"`   // "7d", "30d", "90d"
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Measurement records a single SLO measurement data point.
type Measurement struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	SLOID     string    `json:"slo_id"`
	AgentID   string    `json:"agent_id"`
	Value     float64   `json:"value"`      // measured value
	Good      int64     `json:"good"`       // number of good events
	Total     int64     `json:"total"`      // total events
	Timestamp time.Time `json:"timestamp"`
}

// Repository provides in-memory storage for SLOs and measurements.
type Repository struct {
	mu           sync.RWMutex
	slos         []*SLO
	measurements []*Measurement
	sloSeq       int
	measSeq      int
}

// NewRepository creates a new SLO repository.
func NewRepository() *Repository {
	return &Repository{
		slos:         make([]*SLO, 0),
		measurements: make([]*Measurement, 0),
	}
}

// CreateSLO creates a new service level objective.
func (r *Repository) CreateSLO(tenantID, name, description, agentID string, sloType SLOType, target float64, window string) *SLO {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sloSeq++
	now := time.Now()
	if window == "" {
		window = "30d"
	}
	s := &SLO{
		ID:          fmt.Sprintf("slo-%d", r.sloSeq),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		AgentID:     agentID,
		Type:        sloType,
		Target:      target,
		Window:      window,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.slos = append(r.slos, s)
	return s
}

// ListSLOs returns all SLOs for a tenant, optionally filtered by agent.
func (r *Repository) ListSLOs(tenantID, agentID string) []*SLO {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*SLO
	for _, s := range r.slos {
		if s.TenantID != tenantID {
			continue
		}
		if agentID != "" && s.AgentID != "" && s.AgentID != agentID {
			continue
		}
		result = append(result, s)
	}
	return result
}

// GetSLO returns a specific SLO by ID within a tenant.
func (r *Repository) GetSLO(tenantID, sloID string) *SLO {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.slos {
		if s.TenantID == tenantID && s.ID == sloID {
			return s
		}
	}
	return nil
}

// UpdateSLO updates an SLO's fields.
func (r *Repository) UpdateSLO(tenantID, sloID, name, description string, target float64, enabled bool) (*SLO, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range r.slos {
		if s.TenantID == tenantID && s.ID == sloID {
			if name != "" {
				s.Name = name
			}
			if description != "" {
				s.Description = description
			}
			if target > 0 {
				s.Target = target
			}
			s.Enabled = enabled
			s.UpdatedAt = time.Now()
			return s, nil
		}
	}
	return nil, fmt.Errorf("SLO %s not found", sloID)
}

// DeleteSLO removes an SLO and its associated measurements.
func (r *Repository) DeleteSLO(tenantID, sloID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, s := range r.slos {
		if s.TenantID == tenantID && s.ID == sloID {
			r.slos = append(r.slos[:i], r.slos[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("SLO %s not found", sloID)
	}

	// Remove associated measurements
	filtered := make([]*Measurement, 0, len(r.measurements))
	for _, m := range r.measurements {
		if m.SLOID != sloID {
			filtered = append(filtered, m)
		}
	}
	r.measurements = filtered

	return nil
}

// RecordMeasurement records a measurement for an SLO.
func (r *Repository) RecordMeasurement(tenantID, sloID, agentID string, value float64, good, total int64) *Measurement {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.measSeq++
	m := &Measurement{
		ID:        fmt.Sprintf("meas-%d", r.measSeq),
		TenantID:  tenantID,
		SLOID:     sloID,
		AgentID:   agentID,
		Value:     value,
		Good:      good,
		Total:     total,
		Timestamp: time.Now(),
	}
	r.measurements = append(r.measurements, m)
	return m
}

// RecordMeasurementAt records a measurement at a specific time (for testing).
func (r *Repository) RecordMeasurementAt(tenantID, sloID, agentID string, value float64, good, total int64, ts time.Time) *Measurement {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.measSeq++
	m := &Measurement{
		ID:        fmt.Sprintf("meas-%d", r.measSeq),
		TenantID:  tenantID,
		SLOID:     sloID,
		AgentID:   agentID,
		Value:     value,
		Good:      good,
		Total:     total,
		Timestamp: ts,
	}
	r.measurements = append(r.measurements, m)
	return m
}

// GetMeasurements returns measurements for an SLO within a time window.
func (r *Repository) GetMeasurements(tenantID, sloID string, since time.Time) []*Measurement {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Measurement
	for _, m := range r.measurements {
		if m.TenantID != tenantID || m.SLOID != sloID {
			continue
		}
		if !since.IsZero() && m.Timestamp.Before(since) {
			continue
		}
		result = append(result, m)
	}
	return result
}
