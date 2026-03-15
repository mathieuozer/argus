package storage

import (
	"github.com/argus-platform/argus/services/telemetry/internal/classifier"
)

// Backend is the interface for telemetry storage.
type Backend interface {
	Store(tenantID string, tier classifier.DataTier, data map[string]string) error
}

// InMemoryBackend is a dev-mode in-memory storage backend.
type InMemoryBackend struct {
	data map[string]map[classifier.DataTier][]map[string]string
}

// NewInMemoryBackend creates a new in-memory storage backend.
func NewInMemoryBackend() *InMemoryBackend {
	return &InMemoryBackend{
		data: make(map[string]map[classifier.DataTier][]map[string]string),
	}
}

// Store stores classified telemetry data.
func (b *InMemoryBackend) Store(tenantID string, tier classifier.DataTier, data map[string]string) error {
	if b.data[tenantID] == nil {
		b.data[tenantID] = make(map[classifier.DataTier][]map[string]string)
	}
	b.data[tenantID][tier] = append(b.data[tenantID][tier], data)
	return nil
}
