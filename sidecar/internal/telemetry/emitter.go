package telemetry

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/argus-platform/argus/pkg/messaging"
	"go.uber.org/zap"
)

// Span represents a telemetry span to emit.
type Span struct {
	SpanID        string            `json:"span_id"`
	TraceID       string            `json:"trace_id"`
	TenantID      string            `json:"tenant_id"`
	AgentID       string            `json:"agent_id"`
	OperationName string            `json:"operation_name"`
	StartedAt     time.Time         `json:"started_at"`
	DurationMs    int64             `json:"duration_ms"`
	Attributes    map[string]string `json:"attributes"`
}

// Metric represents a telemetry metric to emit.
type Metric struct {
	Name     string            `json:"name"`
	Value    float64           `json:"value"`
	Labels   map[string]string `json:"labels"`
	TenantID string            `json:"tenant_id"`
	AgentID  string            `json:"agent_id"`
	Time     time.Time         `json:"time"`
}

// Emitter emits telemetry data to NATS JetStream.
type Emitter struct {
	logger    *zap.Logger
	natsURL   string
	conn      *messaging.Conn
	publisher *messaging.Publisher
	tenantID  string
	agentID   string
}

// NewEmitter creates a new telemetry emitter.
func NewEmitter(logger *zap.Logger, natsURL string) *Emitter {
	return &Emitter{
		logger:  logger,
		natsURL: natsURL,
	}
}

// SetIdentity sets the tenant and agent ID for telemetry attribution.
func (e *Emitter) SetIdentity(tenantID, agentID string) {
	e.tenantID = tenantID
	e.agentID = agentID
}

// Connect establishes a connection to NATS JetStream and ensures the telemetry stream exists.
func (e *Emitter) Connect() error {
	if e.natsURL == "" {
		e.logger.Warn("NATS URL not configured, telemetry will be logged locally only")
		return nil
	}

	conn, err := messaging.Connect(e.natsURL)
	if err != nil {
		e.logger.Warn("failed to connect to NATS, telemetry will be logged locally only",
			zap.String("url", e.natsURL),
			zap.Error(err),
		)
		return nil // graceful degradation: log locally
	}

	// Ensure the telemetry stream exists
	if _, err := conn.EnsureStream(messaging.DefaultTelemetryStream()); err != nil {
		e.logger.Warn("failed to ensure NATS telemetry stream", zap.Error(err))
		conn.Close()
		return nil
	}

	e.conn = conn
	e.publisher = messaging.NewPublisher(conn)
	e.logger.Info("connected to NATS JetStream for telemetry", zap.String("url", e.natsURL))
	return nil
}

// EmitSpan publishes a span to the NATS telemetry stream.
func (e *Emitter) EmitSpan(span *Span) error {
	e.logger.Debug("emitting span",
		zap.String("span_id", span.SpanID),
		zap.String("operation", span.OperationName),
	)

	if e.publisher == nil {
		return nil // NATS not connected; span was logged above
	}

	tenantID := span.TenantID
	if tenantID == "" {
		tenantID = e.tenantID
	}
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required for telemetry emission")
	}

	subject := fmt.Sprintf("tenant.%s.telemetry.spans", tenantID)
	payload, err := json.Marshal(span)
	if err != nil {
		return fmt.Errorf("marshal span: %w", err)
	}

	if _, err := e.publisher.Publish(subject, payload); err != nil {
		e.logger.Error("failed to publish span to NATS",
			zap.String("span_id", span.SpanID),
			zap.Error(err),
		)
		return fmt.Errorf("publish span: %w", err)
	}

	return nil
}

// EmitMetric publishes a metric to the NATS telemetry stream.
func (e *Emitter) EmitMetric(name string, value float64, labels map[string]string) error {
	e.logger.Debug("emitting metric",
		zap.String("name", name),
		zap.Float64("value", value),
	)

	if e.publisher == nil {
		return nil // NATS not connected; metric was logged above
	}

	tenantID := e.tenantID
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required for telemetry emission")
	}

	metric := &Metric{
		Name:     name,
		Value:    value,
		Labels:   labels,
		TenantID: tenantID,
		AgentID:  e.agentID,
		Time:     time.Now(),
	}

	subject := fmt.Sprintf("tenant.%s.telemetry.metrics", tenantID)
	payload, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("marshal metric: %w", err)
	}

	if _, err := e.publisher.Publish(subject, payload); err != nil {
		e.logger.Error("failed to publish metric to NATS",
			zap.String("name", name),
			zap.Error(err),
		)
		return fmt.Errorf("publish metric: %w", err)
	}

	return nil
}

// Close shuts down the NATS connection.
func (e *Emitter) Close() {
	if e.conn != nil {
		e.conn.Close()
	}
}
