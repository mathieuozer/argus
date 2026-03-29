package trace

import (
	"sort"
	"sync"
	"time"
)

// Span represents a single traced operation within a distributed trace.
type Span struct {
	SpanID        string            `json:"span_id"`
	TraceID       string            `json:"trace_id"`
	ParentSpanID  string            `json:"parent_span_id,omitempty"`
	TenantID      string            `json:"tenant_id"`
	AgentID       string            `json:"agent_id"`
	TaskID        string            `json:"task_id"`
	OperationName string            `json:"operation_name"`
	StartedAt     time.Time         `json:"started_at"`
	DurationMs    int64             `json:"duration_ms"`
	Attributes    map[string]string `json:"attributes"`
	Tier          int               `json:"tier"`
	ErrorCode     *string           `json:"error_code"`
}

// SpanNode represents a span in a tree structure with children.
type SpanNode struct {
	SpanID        string            `json:"span_id"`
	OperationName string            `json:"operation_name"`
	DurationMs    int64             `json:"duration_ms"`
	StartedAt     time.Time         `json:"started_at"`
	AgentID       string            `json:"agent_id,omitempty"`
	Attributes    map[string]string `json:"attributes"`
	ErrorCode     *string           `json:"error_code,omitempty"`
	Children      []*SpanNode       `json:"children"`
}

// TraceSummary provides a high-level overview of a trace.
type TraceSummary struct {
	TraceID         string    `json:"trace_id"`
	RootOperation   string    `json:"root_operation"`
	AgentID         string    `json:"agent_id"`
	TotalSpans      int       `json:"total_spans"`
	TotalDurationMs int64     `json:"total_duration_ms"`
	HasErrors       bool      `json:"has_errors"`
	StartedAt       time.Time `json:"started_at"`
}

// TraceDetail provides the full span tree for a trace.
type TraceDetail struct {
	TraceID         string    `json:"trace_id"`
	RootSpan        *SpanNode `json:"root_span"`
	TotalSpans      int       `json:"total_spans"`
	TotalDurationMs int64     `json:"total_duration_ms"`
	HasErrors       bool      `json:"has_errors"`
}

// FlameGraphNode represents a node in a flame graph visualization.
type FlameGraphNode struct {
	Name     string            `json:"name"`
	Value    int64             `json:"value"`
	Children []*FlameGraphNode `json:"children"`
}

// Service manages trace data ingestion and querying.
type Service struct {
	mu    sync.RWMutex
	spans map[string][]*Span // traceID -> spans
}

// NewService creates a new trace service.
func NewService() *Service {
	return &Service{
		spans: make(map[string][]*Span),
	}
}

// IngestSpan adds a span to the trace store.
func (s *Service) IngestSpan(span *Span) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spans[span.TraceID] = append(s.spans[span.TraceID], span)
}

// ListTraces returns trace summaries for a tenant, optionally filtered by agent ID.
func (s *Service) ListTraces(tenantID, agentID string, limit int) []TraceSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var summaries []TraceSummary
	for traceID, spans := range s.spans {
		if len(spans) == 0 || spans[0].TenantID != tenantID {
			continue
		}
		if agentID != "" {
			found := false
			for _, sp := range spans {
				if sp.AgentID == agentID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		hasErrors := false
		var rootOp, rootAgent string
		var earliest time.Time
		var maxDuration int64

		for i, sp := range spans {
			if sp.ErrorCode != nil {
				hasErrors = true
			}
			if i == 0 || sp.StartedAt.Before(earliest) {
				earliest = sp.StartedAt
				rootOp = sp.OperationName
				rootAgent = sp.AgentID
			}
			if sp.DurationMs > maxDuration {
				maxDuration = sp.DurationMs
			}
		}
		// Find root span (no parent) — its values take precedence
		for _, sp := range spans {
			if sp.ParentSpanID == "" {
				rootOp = sp.OperationName
				rootAgent = sp.AgentID
				maxDuration = sp.DurationMs
				break
			}
		}

		summaries = append(summaries, TraceSummary{
			TraceID:         traceID,
			RootOperation:   rootOp,
			AgentID:         rootAgent,
			TotalSpans:      len(spans),
			TotalDurationMs: maxDuration,
			HasErrors:       hasErrors,
			StartedAt:       earliest,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].StartedAt.After(summaries[j].StartedAt)
	})

	if limit > 0 && len(summaries) > limit {
		summaries = summaries[:limit]
	}

	return summaries
}

// GetTrace returns the full trace tree for a given trace ID.
func (s *Service) GetTrace(tenantID, traceID string) *TraceDetail {
	s.mu.RLock()
	defer s.mu.RUnlock()

	spans, ok := s.spans[traceID]
	if !ok || len(spans) == 0 {
		return nil
	}
	if spans[0].TenantID != tenantID {
		return nil
	}

	return s.buildTree(traceID, spans)
}

func (s *Service) buildTree(traceID string, spans []*Span) *TraceDetail {
	nodeMap := make(map[string]*SpanNode)
	for _, sp := range spans {
		nodeMap[sp.SpanID] = &SpanNode{
			SpanID:        sp.SpanID,
			OperationName: sp.OperationName,
			DurationMs:    sp.DurationMs,
			StartedAt:     sp.StartedAt,
			AgentID:       sp.AgentID,
			Attributes:    sp.Attributes,
			ErrorCode:     sp.ErrorCode,
			Children:      make([]*SpanNode, 0),
		}
	}

	var root *SpanNode
	hasErrors := false

	for _, sp := range spans {
		if sp.ErrorCode != nil {
			hasErrors = true
		}
		node := nodeMap[sp.SpanID]
		if sp.ParentSpanID == "" {
			root = node
		} else if parent, ok := nodeMap[sp.ParentSpanID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	if root == nil && len(spans) > 0 {
		root = nodeMap[spans[0].SpanID]
	}

	return &TraceDetail{
		TraceID:         traceID,
		RootSpan:        root,
		TotalSpans:      len(spans),
		TotalDurationMs: root.DurationMs,
		HasErrors:       hasErrors,
	}
}

// ListSpansByAgent returns all spans for a given agent across all traces.
func (s *Service) ListSpansByAgent(agentID string) []*Span {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Span
	for _, spans := range s.spans {
		for _, sp := range spans {
			if sp.AgentID == agentID {
				result = append(result, sp)
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.After(result[j].StartedAt)
	})

	return result
}

// GetFlameGraph returns a flame graph representation for a trace.
func (s *Service) GetFlameGraph(tenantID, traceID string) *FlameGraphNode {
	detail := s.GetTrace(tenantID, traceID)
	if detail == nil || detail.RootSpan == nil {
		return nil
	}
	return spanToFlame(detail.RootSpan)
}

func spanToFlame(node *SpanNode) *FlameGraphNode {
	fg := &FlameGraphNode{
		Name:     node.OperationName,
		Value:    node.DurationMs,
		Children: make([]*FlameGraphNode, 0, len(node.Children)),
	}
	for _, child := range node.Children {
		fg.Children = append(fg.Children, spanToFlame(child))
	}
	return fg
}
