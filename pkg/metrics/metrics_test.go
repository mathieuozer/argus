package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCounter_Inc(t *testing.T) {
	reg := NewRegistry()
	c := reg.NewCounter("test_requests_total", "Total requests")

	c.Inc(`method="GET"`)
	c.Inc(`method="GET"`)
	c.Inc(`method="POST"`)

	body := getMetrics(t, reg)
	assertContains(t, body, `test_requests_total{method="GET"} 2`)
	assertContains(t, body, `test_requests_total{method="POST"} 1`)
	assertContains(t, body, "# TYPE test_requests_total counter")
	assertContains(t, body, "# HELP test_requests_total Total requests")
}

func TestCounter_Add(t *testing.T) {
	reg := NewRegistry()
	c := reg.NewCounter("test_bytes_total", "Total bytes")

	c.Add("", 100)
	c.Add("", 250)

	body := getMetrics(t, reg)
	assertContains(t, body, "test_bytes_total 350")
}

func TestGauge_SetIncDec(t *testing.T) {
	reg := NewRegistry()
	g := reg.NewGauge("test_active_connections", "Active connections")

	g.Set("", 5)
	body := getMetrics(t, reg)
	assertContains(t, body, "test_active_connections 5")
	assertContains(t, body, "# TYPE test_active_connections gauge")

	g.Inc("")
	body = getMetrics(t, reg)
	assertContains(t, body, "test_active_connections 6")

	g.Dec("")
	g.Dec("")
	body = getMetrics(t, reg)
	assertContains(t, body, "test_active_connections 4")
}

func TestGauge_WithLabels(t *testing.T) {
	reg := NewRegistry()
	g := reg.NewGauge("test_gauge", "Test gauge")

	g.Set(`service="api"`, 10)
	g.Set(`service="web"`, 20)

	body := getMetrics(t, reg)
	assertContains(t, body, `test_gauge{service="api"} 10`)
	assertContains(t, body, `test_gauge{service="web"} 20`)
}

func TestHistogram_Observe(t *testing.T) {
	reg := NewRegistry()
	h := reg.NewHistogram("test_duration_seconds", "Request duration", []float64{0.1, 0.5, 1.0, 5.0})

	h.Observe("", 0.05) // bucket 0.1
	h.Observe("", 0.3)  // bucket 0.5
	h.Observe("", 0.8)  // bucket 1.0
	h.Observe("", 2.0)  // bucket 5.0
	h.Observe("", 10.0) // +Inf only

	body := getMetrics(t, reg)
	assertContains(t, body, "# TYPE test_duration_seconds histogram")
	assertContains(t, body, `test_duration_seconds_bucket{le="0.100"} 1`)
	assertContains(t, body, `test_duration_seconds_bucket{le="0.500"} 2`)
	assertContains(t, body, `test_duration_seconds_bucket{le="1.000"} 3`)
	assertContains(t, body, `test_duration_seconds_bucket{le="5.000"} 4`)
	assertContains(t, body, `test_duration_seconds_bucket{le="+Inf"} 5`)
	assertContains(t, body, "test_duration_seconds_count 5")
}

func TestHistogram_WithLabels(t *testing.T) {
	reg := NewRegistry()
	h := reg.NewHistogram("test_latency", "Latency", []float64{0.01, 0.1, 1.0})

	h.Observe(`method="GET"`, 0.005)
	h.Observe(`method="GET"`, 0.05)

	body := getMetrics(t, reg)
	assertContains(t, body, `test_latency_bucket{method="GET",le="0.010"} 1`)
	assertContains(t, body, `test_latency_bucket{method="GET",le="0.100"} 2`)
	assertContains(t, body, `test_latency_count{method="GET"} 2`)
}

func TestHistogram_BucketsAreSorted(t *testing.T) {
	reg := NewRegistry()
	h := reg.NewHistogram("test_sorted", "Test", []float64{5.0, 0.1, 1.0, 0.5})

	h.Observe("", 0.3)

	body := getMetrics(t, reg)
	// Should be cumulative buckets in sorted order
	assertContains(t, body, `test_sorted_bucket{le="0.100"} 0`)
	assertContains(t, body, `test_sorted_bucket{le="0.500"} 1`)
	assertContains(t, body, `test_sorted_bucket{le="1.000"} 1`)
	assertContains(t, body, `test_sorted_bucket{le="5.000"} 1`)
}

func TestMetricsHandler_ContentType(t *testing.T) {
	reg := NewRegistry()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	reg.Handler()(w, req)

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
}

func TestHTTPMiddleware_TracksRequests(t *testing.T) {
	reg := NewRegistry()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := HTTPMiddleware(reg, "test-service")(inner)

	// Make a few requests
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/agents", nil)
		handler.ServeHTTP(w, req)
	}

	body := getMetrics(t, reg)
	assertContains(t, body, "http_requests_total")
	assertContains(t, body, "http_request_duration_seconds")
	assertContains(t, body, `service="test-service"`)
}

func TestHTTPMiddleware_SkipsMetricsPath(t *testing.T) {
	reg := NewRegistry()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := HTTPMiddleware(reg, "test")(inner)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	handler.ServeHTTP(w, req)

	// The /metrics request itself should not be tracked
	body := getMetrics(t, reg)
	if strings.Contains(body, `/metrics"`) {
		t.Error("/metrics path should be excluded from instrumentation")
	}
}

func TestHTTPMiddleware_SkipsHealthPath(t *testing.T) {
	reg := NewRegistry()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := HTTPMiddleware(reg, "test")(inner)

	for _, path := range []string{"/health", "/health/live", "/health/ready"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", path, nil)
		handler.ServeHTTP(w, req)
	}

	body := getMetrics(t, reg)
	if strings.Contains(body, `/health"`) {
		t.Error("/health paths should be excluded from instrumentation")
	}
}

func TestHTTPMiddleware_TracksErrors(t *testing.T) {
	reg := NewRegistry()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	handler := HTTPMiddleware(reg, "test")(inner)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/error", nil)
	handler.ServeHTTP(w, req)

	body := getMetrics(t, reg)
	assertContains(t, body, "http_errors_total")
	assertContains(t, body, `status="500"`)
}

func TestEmptyRegistry(t *testing.T) {
	reg := NewRegistry()
	body := getMetrics(t, reg)
	if body != "" {
		t.Errorf("empty registry should produce empty output, got: %q", body)
	}
}

// helpers

func getMetrics(t *testing.T, reg *Registry) string {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	reg.Handler()(w, req)
	return w.Body.String()
}

func assertContains(t *testing.T, body, substr string) {
	t.Helper()
	if !strings.Contains(body, substr) {
		t.Errorf("expected output to contain %q\ngot:\n%s", substr, body)
	}
}
