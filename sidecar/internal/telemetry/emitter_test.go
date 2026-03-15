package telemetry

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewEmitter(t *testing.T) {
	logger := zap.NewNop()
	natsURL := "nats://localhost:4222"

	e := NewEmitter(logger, natsURL)
	if e == nil {
		t.Fatal("expected non-nil emitter")
	}
	if e.logger != logger {
		t.Error("expected logger to be set")
	}
	if e.natsURL != natsURL {
		t.Errorf("expected natsURL %s, got %s", natsURL, e.natsURL)
	}
}

func TestEmitSpan(t *testing.T) {
	tests := []struct {
		name string
		span *Span
	}{
		{
			name: "basic span",
			span: &Span{
				SpanID:        "span-001",
				TraceID:       "trace-001",
				OperationName: "tool_call",
				StartedAt:     time.Now(),
				DurationMs:    150,
				Attributes:    map[string]string{"tool": "search"},
			},
		},
		{
			name: "span with empty attributes",
			span: &Span{
				SpanID:        "span-002",
				TraceID:       "trace-002",
				OperationName: "llm_inference",
				StartedAt:     time.Now(),
				DurationMs:    3200,
				Attributes:    map[string]string{},
			},
		},
		{
			name: "span with nil attributes",
			span: &Span{
				SpanID:        "span-003",
				TraceID:       "trace-003",
				OperationName: "agent_step",
				StartedAt:     time.Now(),
				DurationMs:    50,
				Attributes:    nil,
			},
		},
		{
			name: "span with zero duration",
			span: &Span{
				SpanID:        "span-004",
				TraceID:       "trace-004",
				OperationName: "cache_hit",
				StartedAt:     time.Now(),
				DurationMs:    0,
				Attributes:    map[string]string{"cache": "hit"},
			},
		},
	}

	e := NewEmitter(zap.NewNop(), "nats://localhost:4222")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := e.EmitSpan(tc.span)
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
		})
	}
}

func TestEmitMetric(t *testing.T) {
	tests := []struct {
		name   string
		metric string
		value  float64
		labels map[string]string
	}{
		{
			name:   "latency metric",
			metric: "agent.latency_ms",
			value:  245.5,
			labels: map[string]string{"agent_id": "budget-reconciler", "tenant_id": "test-tenant"},
		},
		{
			name:   "token count metric",
			metric: "agent.tokens_used",
			value:  1024,
			labels: map[string]string{"model": "gpt-4"},
		},
		{
			name:   "metric with no labels",
			metric: "system.uptime_seconds",
			value:  86400,
			labels: nil,
		},
		{
			name:   "metric with empty labels",
			metric: "agent.request_count",
			value:  0,
			labels: map[string]string{},
		},
		{
			name:   "negative metric value",
			metric: "agent.cost_delta",
			value:  -0.05,
			labels: map[string]string{"tenant_id": "test"},
		},
	}

	e := NewEmitter(zap.NewNop(), "nats://localhost:4222")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := e.EmitMetric(tc.metric, tc.value, tc.labels)
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
		})
	}
}
