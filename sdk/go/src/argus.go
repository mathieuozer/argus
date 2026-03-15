package argus

import (
	"context"
	"time"
)

// Client is the Argus SDK client.
type Client struct {
	endpoint string
	tenantID string
}

// Option configures the client.
type Option func(*Client)

// WithEndpoint sets the Argus platform endpoint.
func WithEndpoint(endpoint string) Option {
	return func(c *Client) {
		c.endpoint = endpoint
	}
}

// WithTenantID sets the tenant ID.
func WithTenantID(tenantID string) Option {
	return func(c *Client) {
		c.tenantID = tenantID
	}
}

// NewClient creates a new Argus SDK client.
func NewClient(opts ...Option) *Client {
	c := &Client{
		endpoint: "localhost:8080",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Span represents a traced operation.
type Span struct {
	name      string
	startedAt time.Time
	attrs     map[string]string
	client    *Client
}

// StartSpan creates and starts a new span.
func (c *Client) StartSpan(ctx context.Context, name string) (*Span, context.Context) {
	span := &Span{
		name:      name,
		startedAt: time.Now(),
		attrs:     make(map[string]string),
		client:    c,
	}
	return span, ctx
}

// SetAttribute sets an attribute on the span.
func (s *Span) SetAttribute(key, value string) {
	s.attrs[key] = value
}

// End finishes the span and sends it to the platform.
func (s *Span) End() {
	// Stub: in production, send span to Argus platform
}

// EmitEvent sends a business event to the platform.
func (c *Client) EmitEvent(ctx context.Context, eventType string, payload map[string]string) error {
	// Stub: in production, send event to Argus platform
	return nil
}

// Close closes the client and flushes any pending data.
func (c *Client) Close() error {
	return nil
}
