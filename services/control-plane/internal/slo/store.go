package slo

import (
	"context"
	"time"
)

// Store defines the persistence interface for SLOs and measurements.
// All methods accept a context.Context as the first argument and return errors
// so that both the in-memory Repository and the PostgreSQL PGRepository can
// satisfy the interface transparently.
type Store interface {
	// CreateSLO creates a new service level objective.
	CreateSLO(ctx context.Context, tenantID, name, description, agentID string, sloType SLOType, target float64, window string) (*SLO, error)

	// ListSLOs returns all SLOs for a tenant, optionally filtered by agent.
	ListSLOs(ctx context.Context, tenantID, agentID string) ([]*SLO, error)

	// GetSLO returns a specific SLO by ID within a tenant. Returns (nil, nil) when not found.
	GetSLO(ctx context.Context, tenantID, sloID string) (*SLO, error)

	// UpdateSLO updates mutable fields of an SLO.
	UpdateSLO(ctx context.Context, tenantID, sloID, name, description string, target float64, enabled bool) (*SLO, error)

	// DeleteSLO removes an SLO and its associated measurements.
	DeleteSLO(ctx context.Context, tenantID, sloID string) error

	// RecordMeasurement records a single measurement data point for an SLO.
	RecordMeasurement(ctx context.Context, tenantID, sloID, agentID string, value float64, good, total int64) (*Measurement, error)

	// GetMeasurements returns measurements for an SLO within a time window.
	GetMeasurements(ctx context.Context, tenantID, sloID string, since time.Time) ([]*Measurement, error)
}

// memStore wraps *Repository and satisfies the Store interface by ignoring ctx
// and wrapping the non-error return values in nil errors.
type memStore struct {
	repo *Repository
}

// NewMemStore creates a Store backed by the given in-memory Repository.
func NewMemStore(repo *Repository) Store {
	return &memStore{repo: repo}
}

// Unwrap returns the underlying *Repository so callers (e.g. tests) can use the
// original context-free seeding helpers directly.
func (m *memStore) Unwrap() *Repository {
	return m.repo
}

func (m *memStore) CreateSLO(_ context.Context, tenantID, name, description, agentID string, sloType SLOType, target float64, window string) (*SLO, error) {
	return m.repo.CreateSLO(tenantID, name, description, agentID, sloType, target, window), nil
}

func (m *memStore) ListSLOs(_ context.Context, tenantID, agentID string) ([]*SLO, error) {
	return m.repo.ListSLOs(tenantID, agentID), nil
}

func (m *memStore) GetSLO(_ context.Context, tenantID, sloID string) (*SLO, error) {
	return m.repo.GetSLO(tenantID, sloID), nil
}

func (m *memStore) UpdateSLO(_ context.Context, tenantID, sloID, name, description string, target float64, enabled bool) (*SLO, error) {
	return m.repo.UpdateSLO(tenantID, sloID, name, description, target, enabled)
}

func (m *memStore) DeleteSLO(_ context.Context, tenantID, sloID string) error {
	return m.repo.DeleteSLO(tenantID, sloID)
}

func (m *memStore) RecordMeasurement(_ context.Context, tenantID, sloID, agentID string, value float64, good, total int64) (*Measurement, error) {
	return m.repo.RecordMeasurement(tenantID, sloID, agentID, value, good, total), nil
}

func (m *memStore) GetMeasurements(_ context.Context, tenantID, sloID string, since time.Time) ([]*Measurement, error) {
	return m.repo.GetMeasurements(tenantID, sloID, since), nil
}

// Compile-time assertions: both memStore and PGRepository must satisfy Store.
var _ Store = (*memStore)(nil)
var _ Store = (*PGRepository)(nil)
