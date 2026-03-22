package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLiveHandler_AlwaysReturns200(t *testing.T) {
	checker := NewChecker()

	// Even with a failing dependency check, liveness should pass
	checker.AddCheck("always-fail", func(ctx context.Context) error {
		return fmt.Errorf("database down")
	})

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	checker.LiveHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("LiveHandler returned %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Status != StatusHealthy {
		t.Errorf("status = %q, want %q", resp.Status, StatusHealthy)
	}
	if resp.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}

func TestReadyHandler_AllHealthy(t *testing.T) {
	checker := NewChecker()
	checker.AddCheck("postgres", func(ctx context.Context) error { return nil })
	checker.AddCheck("nats", func(ctx context.Context) error { return nil })

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	checker.ReadyHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ReadyHandler returned %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Status != StatusHealthy {
		t.Errorf("status = %q, want %q", resp.Status, StatusHealthy)
	}
	if len(resp.Components) != 2 {
		t.Errorf("components count = %d, want 2", len(resp.Components))
	}
	for _, comp := range resp.Components {
		if comp.Status != StatusHealthy {
			t.Errorf("component %q status = %q, want %q", comp.Name, comp.Status, StatusHealthy)
		}
	}
}

func TestReadyHandler_OneUnhealthy(t *testing.T) {
	checker := NewChecker()
	checker.AddCheck("postgres", func(ctx context.Context) error { return nil })
	checker.AddCheck("nats", func(ctx context.Context) error {
		return fmt.Errorf("connection refused")
	})

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	checker.ReadyHandler()(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("ReadyHandler returned %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Status != StatusUnhealthy {
		t.Errorf("status = %q, want %q", resp.Status, StatusUnhealthy)
	}

	found := false
	for _, comp := range resp.Components {
		if comp.Name == "nats" {
			found = true
			if comp.Status != StatusUnhealthy {
				t.Errorf("nats status = %q, want %q", comp.Status, StatusUnhealthy)
			}
			if comp.Message == "" {
				t.Error("unhealthy component should have a message")
			}
		}
	}
	if !found {
		t.Error("expected to find 'nats' component in response")
	}
}

func TestReadyHandler_NoChecks(t *testing.T) {
	checker := NewChecker()

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	checker.ReadyHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ReadyHandler with no checks returned %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Status != StatusHealthy {
		t.Errorf("status = %q, want %q", resp.Status, StatusHealthy)
	}
}

func TestHandler_SameAsReady(t *testing.T) {
	checker := NewChecker()
	checker.AddCheck("test", func(ctx context.Context) error { return nil })

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	checker.Handler()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Handler returned %d, want %d", w.Code, http.StatusOK)
	}
}

func TestResponse_ContentType(t *testing.T) {
	checker := NewChecker()

	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"live", checker.LiveHandler()},
		{"ready", checker.ReadyHandler()},
		{"handler", checker.Handler()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.handler(w, httptest.NewRequest("GET", "/", nil))
			ct := w.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

func TestReadyHandler_ContextCancellation(t *testing.T) {
	checker := NewChecker()
	checker.AddCheck("slow", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	// The internal 5s timeout should eventually cancel
	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	checker.ReadyHandler()(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("ReadyHandler returned %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
