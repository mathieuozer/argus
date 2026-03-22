package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkLiveHandler(b *testing.B) {
	checker := NewChecker()
	handler := checker.LiveHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health/live", nil)
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkReadyHandler_NoDeps(b *testing.B) {
	checker := NewChecker()
	handler := checker.ReadyHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health/ready", nil)
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkReadyHandler_WithDeps(b *testing.B) {
	checker := NewChecker()
	checker.AddCheck("db", func(_ context.Context) error { return nil })
	checker.AddCheck("nats", func(_ context.Context) error { return nil })
	checker.AddCheck("vault", func(_ context.Context) error { return nil })
	handler := checker.ReadyHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health/ready", nil)
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkHandler(b *testing.B) {
	checker := NewChecker()
	checker.AddCheck("db", func(_ context.Context) error { return nil })
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health/live":
			checker.LiveHandler().ServeHTTP(w, r)
		case "/health/ready":
			checker.ReadyHandler().ServeHTTP(w, r)
		default:
			checker.Handler().ServeHTTP(w, r)
		}
	})

	paths := []string{"/health", "/health/live", "/health/ready"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", paths[i%3], nil)
		handler.ServeHTTP(w, req)
	}
}
