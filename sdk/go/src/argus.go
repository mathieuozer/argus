package argus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Client is the Argus SDK client for rich agent instrumentation.
type Client struct {
	endpoint   string
	tenantID   string
	agentID    string
	httpClient *http.Client
	batchSize  int
	flushEvery time.Duration

	mu       sync.Mutex
	spans    []spanData
	events   []eventData
	stopCh   chan struct{}
	stopped  bool
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

// WithAgentID sets the agent ID.
func WithAgentID(agentID string) Option {
	return func(c *Client) {
		c.agentID = agentID
	}
}

// WithBatchSize sets the maximum number of items to batch before flushing.
func WithBatchSize(size int) Option {
	return func(c *Client) {
		c.batchSize = size
	}
}

// WithFlushInterval sets how often the client flushes pending data.
func WithFlushInterval(d time.Duration) Option {
	return func(c *Client) {
		c.flushEvery = d
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

type spanData struct {
	SpanID        string            `json:"span_id"`
	TraceID       string            `json:"trace_id"`
	OperationName string            `json:"operation_name"`
	StartedAt     time.Time         `json:"started_at"`
	DurationMs    int64             `json:"duration_ms"`
	Attributes    map[string]string `json:"attributes"`
	ErrorCode     string            `json:"error_code,omitempty"`
}

type eventData struct {
	EventType string            `json:"event_type"`
	Payload   map[string]string `json:"payload"`
	Timestamp time.Time         `json:"timestamp"`
}

// NewClient creates a new Argus SDK client.
func NewClient(opts ...Option) *Client {
	c := &Client{
		endpoint:   "http://localhost:8080",
		batchSize:  100,
		flushEvery: 5 * time.Second,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		stopCh:     make(chan struct{}),
	}
	for _, opt := range opts {
		opt(c)
	}

	go c.flushLoop()
	return c
}

type spanContextKey struct{}

// Span represents a traced operation.
type Span struct {
	spanID    string
	traceID   string
	name      string
	startedAt time.Time
	attrs     map[string]string
	client    *Client
	errCode   string
}

// StartSpan creates and starts a new span. It propagates trace context
// through the returned context.
func (c *Client) StartSpan(ctx context.Context, name string) (*Span, context.Context) {
	traceID := ""
	if parent, ok := SpanFromContext(ctx); ok {
		traceID = parent.traceID
	}
	if traceID == "" {
		traceID = generateID()
	}

	span := &Span{
		spanID:    generateID(),
		traceID:   traceID,
		name:      name,
		startedAt: time.Now(),
		attrs:     make(map[string]string),
		client:    c,
	}
	ctx = context.WithValue(ctx, spanContextKey{}, span)
	return span, ctx
}

// SpanFromContext retrieves the current span from context.
func SpanFromContext(ctx context.Context) (*Span, bool) {
	span, ok := ctx.Value(spanContextKey{}).(*Span)
	return span, ok
}

// SetAttribute sets an attribute on the span.
func (s *Span) SetAttribute(key, value string) {
	s.attrs[key] = value
}

// SetError records an error on the span.
func (s *Span) SetError(err error) {
	if err != nil {
		s.errCode = err.Error()
		s.attrs["error"] = err.Error()
	}
}

// End finishes the span and queues it for sending.
func (s *Span) End() {
	duration := time.Since(s.startedAt).Milliseconds()
	s.client.addSpan(spanData{
		SpanID:        s.spanID,
		TraceID:       s.traceID,
		OperationName: s.name,
		StartedAt:     s.startedAt,
		DurationMs:    duration,
		Attributes:    s.attrs,
		ErrorCode:     s.errCode,
	})
}

// SpanID returns the span's unique identifier.
func (s *Span) SpanID() string { return s.spanID }

// TraceID returns the span's trace identifier.
func (s *Span) TraceID() string { return s.traceID }

func (c *Client) addSpan(sd spanData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.spans = append(c.spans, sd)
	if len(c.spans) >= c.batchSize {
		go func() { _ = c.Flush() }()
	}
}

// EmitEvent sends a business event to the platform.
func (c *Client) EmitEvent(ctx context.Context, eventType string, payload map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, eventData{
		EventType: eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	})
	return nil
}

// Flush sends all pending spans and events to the platform.
func (c *Client) Flush() error {
	c.mu.Lock()
	spans := c.spans
	events := c.events
	c.spans = nil
	c.events = nil
	c.mu.Unlock()

	if len(spans) == 0 && len(events) == 0 {
		return nil
	}

	var errs []error
	if len(spans) > 0 {
		if err := c.sendSpans(spans); err != nil {
			errs = append(errs, err)
		}
	}
	if len(events) > 0 {
		if err := c.sendEvents(events); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (c *Client) sendSpans(spans []spanData) error {
	body, err := json.Marshal(map[string]interface{}{
		"tenant_id": c.tenantID,
		"agent_id":  c.agentID,
		"spans":     spans,
	})
	if err != nil {
		return fmt.Errorf("marshal spans: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint+"/api/v1/telemetry/spans", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", c.tenantID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send spans: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("send spans: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) sendEvents(events []eventData) error {
	body, err := json.Marshal(map[string]interface{}{
		"tenant_id": c.tenantID,
		"agent_id":  c.agentID,
		"events":    events,
	})
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint+"/api/v1/telemetry/events", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", c.tenantID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("send events: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) flushLoop() {
	ticker := time.NewTicker(c.flushEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = c.Flush()
		case <-c.stopCh:
			return
		}
	}
}

// PendingSpans returns the number of spans waiting to be flushed.
func (c *Client) PendingSpans() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.spans)
}

// PendingEvents returns the number of events waiting to be flushed.
func (c *Client) PendingEvents() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.events)
}

// Close closes the client and flushes any pending data.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return nil
	}
	c.stopped = true
	c.mu.Unlock()

	close(c.stopCh)
	return c.Flush()
}

// generateID creates a simple unique ID using timestamp + counter.
var (
	idMu      sync.Mutex
	idCounter int64
)

func generateID() string {
	idMu.Lock()
	idCounter++
	id := idCounter
	idMu.Unlock()
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), id)
}
