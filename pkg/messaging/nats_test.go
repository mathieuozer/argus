package messaging

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestDefaultTelemetryStream(t *testing.T) {
	cfg := DefaultTelemetryStream()
	if cfg.Name != "ARGUS_TELEMETRY" {
		t.Errorf("expected stream name ARGUS_TELEMETRY, got %s", cfg.Name)
	}
	if len(cfg.Subjects) != 1 || cfg.Subjects[0] != "tenant.*.telemetry.>" {
		t.Errorf("unexpected subjects: %v", cfg.Subjects)
	}
	if cfg.MaxAge != 72*time.Hour {
		t.Errorf("expected 72h max age, got %v", cfg.MaxAge)
	}
	if cfg.Storage != nats.FileStorage {
		t.Error("expected file storage")
	}
}

func TestConnect_InvalidURL(t *testing.T) {
	_, err := Connect("nats://localhost:59998", nats.MaxReconnects(0))
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestNewPublisher(t *testing.T) {
	// Test publisher creation (no connection needed for construction)
	conn := &Conn{}
	pub := NewPublisher(conn)
	if pub == nil {
		t.Error("expected non-nil publisher")
	}
}

func TestNewSubscriber(t *testing.T) {
	conn := &Conn{}
	sub := NewSubscriber(conn)
	if sub == nil {
		t.Error("expected non-nil subscriber")
	}
}
