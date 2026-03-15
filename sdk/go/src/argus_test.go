package argus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	defer c.Close()

	if c.endpoint != "http://localhost:8080" {
		t.Errorf("endpoint = %q, want %q", c.endpoint, "http://localhost:8080")
	}
	if c.batchSize != 100 {
		t.Errorf("batchSize = %d, want 100", c.batchSize)
	}
}

func TestNewClientWithOptions(t *testing.T) {
	c := NewClient(
		WithEndpoint("http://custom:9090"),
		WithTenantID("tenant-1"),
		WithAgentID("agent-1"),
		WithBatchSize(50),
		WithFlushInterval(10*time.Second),
	)
	defer c.Close()

	if c.endpoint != "http://custom:9090" {
		t.Errorf("endpoint = %q", c.endpoint)
	}
	if c.tenantID != "tenant-1" {
		t.Errorf("tenantID = %q", c.tenantID)
	}
	if c.agentID != "agent-1" {
		t.Errorf("agentID = %q", c.agentID)
	}
	if c.batchSize != 50 {
		t.Errorf("batchSize = %d", c.batchSize)
	}
}

func TestStartSpan(t *testing.T) {
	c := NewClient()
	defer c.Close()

	ctx := context.Background()
	span, newCtx := c.StartSpan(ctx, "test-operation")

	if span.name != "test-operation" {
		t.Errorf("span name = %q, want %q", span.name, "test-operation")
	}
	if span.spanID == "" {
		t.Error("span ID should not be empty")
	}
	if span.traceID == "" {
		t.Error("trace ID should not be empty")
	}
	if span.SpanID() != span.spanID {
		t.Error("SpanID() mismatch")
	}
	if span.TraceID() != span.traceID {
		t.Error("TraceID() mismatch")
	}

	// Context should contain the span
	recovered, ok := SpanFromContext(newCtx)
	if !ok {
		t.Fatal("expected span in context")
	}
	if recovered.spanID != span.spanID {
		t.Error("context span mismatch")
	}
}

func TestSpanContextPropagation(t *testing.T) {
	c := NewClient()
	defer c.Close()

	ctx := context.Background()
	parentSpan, ctx := c.StartSpan(ctx, "parent")
	childSpan, _ := c.StartSpan(ctx, "child")

	// Child should inherit parent's trace ID
	if childSpan.traceID != parentSpan.traceID {
		t.Errorf("child traceID = %q, parent traceID = %q", childSpan.traceID, parentSpan.traceID)
	}
	// But should have a different span ID
	if childSpan.spanID == parentSpan.spanID {
		t.Error("child and parent should have different span IDs")
	}
}

func TestSpanFromContextEmpty(t *testing.T) {
	_, ok := SpanFromContext(context.Background())
	if ok {
		t.Error("expected no span in empty context")
	}
}

func TestSpanSetAttribute(t *testing.T) {
	c := NewClient()
	defer c.Close()

	span, _ := c.StartSpan(context.Background(), "test")
	span.SetAttribute("key1", "value1")
	span.SetAttribute("key2", "value2")

	if span.attrs["key1"] != "value1" {
		t.Errorf("attr key1 = %q", span.attrs["key1"])
	}
	if span.attrs["key2"] != "value2" {
		t.Errorf("attr key2 = %q", span.attrs["key2"])
	}
}

func TestSpanSetError(t *testing.T) {
	c := NewClient()
	defer c.Close()

	span, _ := c.StartSpan(context.Background(), "test")

	// nil error should be no-op
	span.SetError(nil)
	if span.errCode != "" {
		t.Error("nil error should not set errCode")
	}

	span.SetError(context.DeadlineExceeded)
	if span.errCode == "" {
		t.Error("expected errCode to be set")
	}
	if span.attrs["error"] == "" {
		t.Error("expected error attribute to be set")
	}
}

func TestSpanEnd(t *testing.T) {
	c := NewClient(WithFlushInterval(1 * time.Hour)) // prevent auto flush
	defer c.Close()

	span, _ := c.StartSpan(context.Background(), "test")
	time.Sleep(1 * time.Millisecond)
	span.End()

	if c.PendingSpans() != 1 {
		t.Errorf("PendingSpans = %d, want 1", c.PendingSpans())
	}
}

func TestEmitEvent(t *testing.T) {
	c := NewClient(WithFlushInterval(1 * time.Hour))
	defer c.Close()

	err := c.EmitEvent(context.Background(), "task.completed", map[string]string{
		"task_id": "t1",
		"status":  "success",
	})
	if err != nil {
		t.Fatalf("EmitEvent failed: %v", err)
	}

	if c.PendingEvents() != 1 {
		t.Errorf("PendingEvents = %d, want 1", c.PendingEvents())
	}
}

func TestFlushSendsToServer(t *testing.T) {
	var spanCount int32
	var eventCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&body)

		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type: application/json")
		}
		if r.Header.Get("X-Tenant-ID") != "t1" {
			t.Errorf("X-Tenant-ID = %q", r.Header.Get("X-Tenant-ID"))
		}

		switch r.URL.Path {
		case "/api/v1/telemetry/spans":
			atomic.AddInt32(&spanCount, 1)
		case "/api/v1/telemetry/events":
			atomic.AddInt32(&eventCount, 1)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(
		WithEndpoint(srv.URL),
		WithTenantID("t1"),
		WithAgentID("a1"),
		WithFlushInterval(1*time.Hour),
	)
	defer c.Close()

	span, _ := c.StartSpan(context.Background(), "op1")
	span.End()

	c.EmitEvent(context.Background(), "test", map[string]string{"k": "v"})

	err := c.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if atomic.LoadInt32(&spanCount) != 1 {
		t.Errorf("span requests = %d, want 1", spanCount)
	}
	if atomic.LoadInt32(&eventCount) != 1 {
		t.Errorf("event requests = %d, want 1", eventCount)
	}

	// After flush, pending should be 0
	if c.PendingSpans() != 0 {
		t.Errorf("PendingSpans after flush = %d", c.PendingSpans())
	}
	if c.PendingEvents() != 0 {
		t.Errorf("PendingEvents after flush = %d", c.PendingEvents())
	}
}

func TestFlushEmptyIsNoop(t *testing.T) {
	c := NewClient(WithFlushInterval(1 * time.Hour))
	defer c.Close()

	err := c.Flush()
	if err != nil {
		t.Fatalf("empty Flush should not error: %v", err)
	}
}

func TestFlushServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(
		WithEndpoint(srv.URL),
		WithFlushInterval(1*time.Hour),
	)
	defer c.Close()

	span, _ := c.StartSpan(context.Background(), "test")
	span.End()

	err := c.Flush()
	if err == nil {
		t.Error("expected error on server 500")
	}
}

func TestCloseFlushesAndStops(t *testing.T) {
	var received int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&received, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(
		WithEndpoint(srv.URL),
		WithFlushInterval(1*time.Hour),
	)

	span, _ := c.StartSpan(context.Background(), "test")
	span.End()

	err := c.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if atomic.LoadInt32(&received) != 1 {
		t.Errorf("Close should have flushed, received = %d", received)
	}

	// Double close should be safe
	err = c.Close()
	if err != nil {
		t.Fatalf("double Close failed: %v", err)
	}
}

func TestGenerateID(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateID()
		if ids[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}
