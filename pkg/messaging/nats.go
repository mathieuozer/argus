package messaging

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Conn wraps a NATS connection with JetStream support.
type Conn struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

// Connect establishes a NATS connection with JetStream.
func Connect(url string, opts ...nats.Option) (*Conn, error) {
	defaults := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * time.Second),
		nats.PingInterval(10 * time.Second),
		nats.MaxPingsOutstanding(3),
		nats.Name("argus-service"),
	}
	allOpts := append(defaults, opts...)

	nc, err := nats.Connect(url, allOpts...)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	return &Conn{nc: nc, js: js}, nil
}

// JetStream returns the JetStream context.
func (c *Conn) JetStream() nats.JetStreamContext {
	return c.js
}

// NatsConn returns the underlying NATS connection.
func (c *Conn) NatsConn() *nats.Conn {
	return c.nc
}

// EnsureStream creates a stream if it doesn't exist.
func (c *Conn) EnsureStream(cfg *nats.StreamConfig) (*nats.StreamInfo, error) {
	info, err := c.js.StreamInfo(cfg.Name)
	if err == nil {
		return info, nil
	}
	info, err = c.js.AddStream(cfg)
	if err != nil {
		return nil, fmt.Errorf("add stream %s: %w", cfg.Name, err)
	}
	return info, nil
}

// Close closes the NATS connection.
func (c *Conn) Close() {
	c.nc.Drain()
	c.nc.Close()
}

// DefaultTelemetryStream returns the default stream config for telemetry.
func DefaultTelemetryStream() *nats.StreamConfig {
	return &nats.StreamConfig{
		Name:      "ARGUS_TELEMETRY",
		Subjects:  []string{"tenant.*.telemetry.>"},
		Retention: nats.LimitsPolicy,
		MaxAge:    72 * time.Hour,
		Storage:   nats.FileStorage,
		Replicas:  1,
		Discard:   nats.DiscardOld,
		MaxBytes:  10 * 1024 * 1024 * 1024, // 10GB
	}
}
