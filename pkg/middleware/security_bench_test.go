package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

var nopHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func BenchmarkRecovery(b *testing.B) {
	logger := zap.NewNop()
	handler := Recovery(logger)(nopHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkSecurityHeaders(b *testing.B) {
	handler := SecurityHeaders(nopHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkRequestID(b *testing.B) {
	handler := RequestID(nopHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkCORSWithOrigin(b *testing.B) {
	b.Setenv("ARGUS_ENV", "development")
	handler := CORSWithOrigin(nopHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkFullMiddlewareChain(b *testing.B) {
	logger := zap.NewNop()
	b.Setenv("ARGUS_ENV", "development")

	handler := Recovery(logger)(
		SecurityHeaders(
			CORSWithOrigin(
				RequestID(nopHandler),
			),
		),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/agents", nil)
		handler.ServeHTTP(w, req)
	}
}
