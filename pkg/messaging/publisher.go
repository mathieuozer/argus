package messaging

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Publisher publishes messages to tenant-scoped NATS subjects.
type Publisher struct {
	conn *Conn
}

// NewPublisher creates a new Publisher.
func NewPublisher(conn *Conn) *Publisher {
	return &Publisher{conn: conn}
}

// PublishTelemetry publishes a telemetry message for a tenant.
func (p *Publisher) PublishTelemetry(tenantID string, dataType string, data any) error {
	subject := fmt.Sprintf("tenant.%s.telemetry.%s", tenantID, dataType)
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	_, err = p.conn.js.Publish(subject, payload, nats.MsgId(fmt.Sprintf("%s-%d", tenantID, time.Now().UnixNano())))
	if err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}

	return nil
}

// PublishAsync publishes a message asynchronously.
func (p *Publisher) PublishAsync(subject string, data []byte) (nats.PubAckFuture, error) {
	return p.conn.js.PublishAsync(subject, data)
}

// Publish publishes a message synchronously.
func (p *Publisher) Publish(subject string, data []byte) (*nats.PubAck, error) {
	return p.conn.js.Publish(subject, data)
}
