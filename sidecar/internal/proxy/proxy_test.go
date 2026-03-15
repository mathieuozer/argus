package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestNew_DefaultUpstream(t *testing.T) {
	t.Setenv("ARGUS_UPSTREAM_ADDR", "")

	p := New(zap.NewNop())
	if p == nil {
		t.Fatal("expected non-nil proxy")
	}
	if p.upstreamAddr != "http://localhost:8000" {
		t.Errorf("expected default upstream http://localhost:8000, got %s", p.upstreamAddr)
	}
	if p.reverseProxy == nil {
		t.Error("expected reverseProxy to be initialized for valid default address")
	}
}

func TestNew_CustomUpstream(t *testing.T) {
	t.Setenv("ARGUS_UPSTREAM_ADDR", "http://myagent:9090")

	p := New(zap.NewNop())
	if p.upstreamAddr != "http://myagent:9090" {
		t.Errorf("expected custom upstream http://myagent:9090, got %s", p.upstreamAddr)
	}
}

func TestNew_InvalidUpstream(t *testing.T) {
	t.Setenv("ARGUS_UPSTREAM_ADDR", "://invalid-url")

	p := New(zap.NewNop())
	if p == nil {
		t.Fatal("expected non-nil proxy even with invalid upstream")
	}
	if p.reverseProxy != nil {
		t.Error("expected reverseProxy to be nil for invalid upstream address")
	}
}

func TestServeHTTP_NilReverseProxy(t *testing.T) {
	p := &Proxy{
		logger:       zap.NewNop(),
		upstreamAddr: "http://localhost:8000",
		reverseProxy: nil,
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	p.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "NO_UPSTREAM" {
		t.Errorf("expected error code NO_UPSTREAM, got %s", errResp.Error.Code)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

func TestStats(t *testing.T) {
	p := &Proxy{
		logger:       zap.NewNop(),
		upstreamAddr: "http://localhost:9000",
		requestCount: 42,
	}

	stats := p.Stats()
	if stats["upstream_addr"] != "http://localhost:9000" {
		t.Errorf("expected upstream_addr http://localhost:9000, got %v", stats["upstream_addr"])
	}
	if stats["request_count"] != int64(42) {
		t.Errorf("expected request_count 42, got %v", stats["request_count"])
	}
}

func TestResponseWriter_CapturesStatusAndBytes(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		body           string
		expectedStatus int
		expectedBytes  int64
	}{
		{
			name:           "200 with body",
			statusCode:     http.StatusOK,
			body:           "hello world",
			expectedStatus: http.StatusOK,
			expectedBytes:  11,
		},
		{
			name:           "404 with JSON error",
			statusCode:     http.StatusNotFound,
			body:           `{"error":"not found"}`,
			expectedStatus: http.StatusNotFound,
			expectedBytes:  21,
		},
		{
			name:           "500 with empty body",
			statusCode:     http.StatusInternalServerError,
			body:           "",
			expectedStatus: http.StatusInternalServerError,
			expectedBytes:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

			rw.WriteHeader(tc.statusCode)
			if tc.body != "" {
				_, _ = rw.Write([]byte(tc.body))
			}

			if rw.statusCode != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, rw.statusCode)
			}
			if rw.bytesWritten != tc.expectedBytes {
				t.Errorf("expected %d bytes written, got %d", tc.expectedBytes, rw.bytesWritten)
			}
		})
	}
}

func TestServeHTTP_FullRoundTrip(t *testing.T) {
	// Start a mock upstream server that echoes a response.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"data":"ok","path":"`+r.URL.Path+`"}`)
	}))
	defer upstream.Close()

	t.Setenv("ARGUS_UPSTREAM_ADDR", upstream.URL)

	p := New(zap.NewNop())
	if p.reverseProxy == nil {
		t.Fatal("expected reverseProxy to be non-nil")
	}

	// Make a request through the proxy.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)

	p.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if data["data"] != "ok" {
		t.Errorf("expected data=ok, got %s", data["data"])
	}
	if data["path"] != "/api/v1/tasks" {
		t.Errorf("expected path=/api/v1/tasks, got %s", data["path"])
	}

	if p.requestCount != 1 {
		t.Errorf("expected requestCount 1, got %d", p.requestCount)
	}
}

func TestServeHTTP_UpstreamDown(t *testing.T) {
	// Point to a server that immediately closes.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	upstreamURL := upstream.URL
	upstream.Close() // close it so connections fail

	t.Setenv("ARGUS_UPSTREAM_ADDR", upstreamURL)

	p := New(zap.NewNop())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	p.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	// The ErrorHandler in proxy.go returns 502 with UPSTREAM_ERROR.
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502 when upstream is down, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var errResp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "UPSTREAM_ERROR" {
		t.Errorf("expected error code UPSTREAM_ERROR, got %s", errResp.Error.Code)
	}
}
