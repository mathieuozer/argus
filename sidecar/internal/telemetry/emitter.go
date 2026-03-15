package telemetry

import (
	"time"

	"go.uber.org/zap"
)

// Span represents a telemetry span to emit.
type Span struct {
	SpanID        string
	TraceID       string
	OperationName string
	StartedAt     time.Time
	DurationMs    int64
	Attributes    map[string]string
}

// Emitter emits telemetry data to NATS.
type Emitter struct {
	logger  *zap.Logger
	natsURL string
}

// NewEmitter creates a new telemetry emitter.
func NewEmitter(logger *zap.Logger, natsURL string) *Emitter {
	return &Emitter{
		logger:  logger,
		natsURL: natsURL,
	}
}

// EmitSpan publishes a span to the telemetry stream.
func (e *Emitter) EmitSpan(span *Span) error {
	e.logger.Debug("emitting span",
		zap.String("span_id", span.SpanID),
		zap.String("operation", span.OperationName),
	)
	// Stub: in production, publish to NATS JetStream
	return nil
}

// EmitMetric publishes a metric to the telemetry stream.
func (e *Emitter) EmitMetric(name string, value float64, labels map[string]string) error {
	e.logger.Debug("emitting metric",
		zap.String("name", name),
		zap.Float64("value", value),
	)
	// Stub: in production, publish to NATS JetStream
	return nil
}
