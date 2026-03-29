package audit

import "context"

// Store defines the interface for reading audit log entries.
type Store interface {
	List(ctx context.Context, tenantID string) ([]*Entry, error)
	Search(ctx context.Context, tenantID, actor, action, resource string) ([]*Entry, error)
}

// AuditWriter defines the interface for writing audit log entries.
// Both Writer (in-memory) and PGWriter satisfy this interface.
type AuditWriter interface {
	Write(ctx context.Context, tenantID, actor, action, resource, details string) (*Entry, error)
}

// memStore wraps the in-memory Writer to satisfy the Store and AuditWriter interfaces.
type memStore struct{ w *Writer }

// NewMemStore wraps an in-memory Writer as a Store.
func NewMemStore(w *Writer) Store { return &memStore{w: w} }

// NewMemWriter wraps an in-memory Writer as an AuditWriter.
func NewMemWriter(w *Writer) AuditWriter { return &memStore{w: w} }

func (m *memStore) List(_ context.Context, tenantID string) ([]*Entry, error) {
	return m.w.List(tenantID), nil
}

func (m *memStore) Search(_ context.Context, tenantID, actor, action, resource string) ([]*Entry, error) {
	entries := m.w.List(tenantID)
	var filtered []*Entry
	for _, e := range entries {
		if actor != "" && e.Actor != actor {
			continue
		}
		if action != "" && e.Action != action {
			continue
		}
		if resource != "" && e.Resource != resource {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered, nil
}

func (m *memStore) Write(_ context.Context, tenantID, actor, action, resource, details string) (*Entry, error) {
	return m.w.Write(tenantID, actor, action, resource, details), nil
}
