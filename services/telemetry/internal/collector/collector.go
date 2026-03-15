package collector

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// Span represents a telemetry span.
type Span struct {
	SpanID        string            `json:"span_id"`
	TraceID       string            `json:"trace_id"`
	TenantID      string            `json:"tenant_id"`
	AgentID       string            `json:"agent_id"`
	TaskID        string            `json:"task_id"`
	OperationName string            `json:"operation_name"`
	StartedAt     time.Time         `json:"started_at"`
	DurationMs    int64             `json:"duration_ms"`
	Tier          int               `json:"tier"`
	Attributes    map[string]string `json:"attributes"`
	ErrorCode     *string           `json:"error_code,omitempty"`
}

// Collector receives and stores telemetry spans.
type Collector struct {
	mu     sync.RWMutex
	spans  []*Span
	logger *zap.Logger
}

// New creates a new telemetry collector.
func New(logger *zap.Logger) *Collector {
	return &Collector{
		spans:  make([]*Span, 0),
		logger: logger,
	}
}

// Ingest accepts a batch of spans.
func (c *Collector) Ingest(spans []*Span) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.spans = append(c.spans, spans...)
	c.logger.Debug("ingested spans", zap.Int("count", len(spans)))
	return len(spans)
}

// Query returns spans matching the given criteria.
func (c *Collector) Query(tenantID, agentID, traceID string, limit int) []*Span {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*Span
	for _, s := range c.spans {
		if s.TenantID != tenantID {
			continue
		}
		if agentID != "" && s.AgentID != agentID {
			continue
		}
		if traceID != "" && s.TraceID != traceID {
			continue
		}
		result = append(result, s)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}
