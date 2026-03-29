package trace

import "context"

// Store defines the interface for trace data access.
// Both in-memory Service and PGService satisfy this interface.
type Store interface {
	IngestSpan(ctx context.Context, span *Span) error
	ListTraces(ctx context.Context, tenantID, agentID string, limit int) ([]TraceSummary, error)
	GetTrace(ctx context.Context, tenantID, traceID string) (*TraceDetail, error)
	GetFlameGraph(ctx context.Context, tenantID, traceID string) (*FlameGraphNode, error)
}

// memStore wraps the in-memory Service to satisfy the Store interface.
type memStore struct{ svc *Service }

// NewMemStore wraps an in-memory Service as a Store.
func NewMemStore(svc *Service) Store { return &memStore{svc: svc} }

func (m *memStore) IngestSpan(_ context.Context, span *Span) error {
	m.svc.IngestSpan(span)
	return nil
}

func (m *memStore) ListTraces(_ context.Context, tenantID, agentID string, limit int) ([]TraceSummary, error) {
	return m.svc.ListTraces(tenantID, agentID, limit), nil
}

func (m *memStore) GetTrace(_ context.Context, tenantID, traceID string) (*TraceDetail, error) {
	return m.svc.GetTrace(tenantID, traceID), nil
}

func (m *memStore) GetFlameGraph(_ context.Context, tenantID, traceID string) (*FlameGraphNode, error) {
	return m.svc.GetFlameGraph(tenantID, traceID), nil
}
