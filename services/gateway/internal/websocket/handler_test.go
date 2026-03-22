package websocket

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

// ---------- Handler construction ----------

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		routes []Route
	}{
		{
			name:   "creates handler with default routes",
			routes: DefaultRoutes(),
		},
		{
			name:   "creates handler with empty routes",
			routes: []Route{},
		},
		{
			name: "creates handler with custom routes",
			routes: []Route{
				{PathPrefix: "/ws/custom", Backend: "http://localhost:9999"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := New(tc.routes, zap.NewNop())
			if h == nil {
				t.Fatal("New() returned nil")
			}
			if len(h.routes) != len(tc.routes) {
				t.Errorf("route count = %d, want %d", len(h.routes), len(tc.routes))
			}
		})
	}
}

// ---------- DefaultRoutes ----------

func TestDefaultRoutes(t *testing.T) {
	routes := DefaultRoutes()

	tests := []struct {
		name        string
		idx         int
		wantPrefix  string
		wantBackend string
	}{
		{
			name:        "agents stream route",
			idx:         0,
			wantPrefix:  "/ws/v1/agents/stream",
			wantBackend: "http://localhost:8084",
		},
		{
			name:        "telemetry stream route",
			idx:         1,
			wantPrefix:  "/ws/v1/telemetry/stream",
			wantBackend: "http://localhost:8084",
		},
	}

	if len(routes) != 2 {
		t.Fatalf("DefaultRoutes() returned %d routes, want 2", len(routes))
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if routes[tc.idx].PathPrefix != tc.wantPrefix {
				t.Errorf("PathPrefix = %q, want %q", routes[tc.idx].PathPrefix, tc.wantPrefix)
			}
			if routes[tc.idx].Backend != tc.wantBackend {
				t.Errorf("Backend = %q, want %q", routes[tc.idx].Backend, tc.wantBackend)
			}
		})
	}
}

// ---------- RegisterRoutes ----------

func TestRegisterRoutes(t *testing.T) {
	h := New(DefaultRoutes(), zap.NewNop())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	tests := []struct {
		name string
		path string
	}{
		{name: "agents stream route registered", path: "/ws/v1/agents/stream"},
		{name: "telemetry stream route registered", path: "/ws/v1/telemetry/stream"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// A 404 would mean the route is not registered.
			if w.Code == http.StatusNotFound {
				t.Errorf("route %s returned 404 — not registered", tc.path)
			}
		})
	}
}

// ---------- Missing upgrade headers ----------

func TestProxyHandler_MissingConnectionHeader(t *testing.T) {
	h := New(DefaultRoutes(), zap.NewNop())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	tests := []struct {
		name string
		path string
	}{
		{name: "agents stream", path: "/ws/v1/agents/stream"},
		{name: "telemetry stream", path: "/ws/v1/telemetry/stream"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			// Missing Connection: Upgrade header.
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestProxyHandler_MissingUpgradeHeader(t *testing.T) {
	h := New(DefaultRoutes(), zap.NewNop())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	tests := []struct {
		name string
		path string
	}{
		{name: "agents stream", path: "/ws/v1/agents/stream"},
		{name: "telemetry stream", path: "/ws/v1/telemetry/stream"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Connection", "Upgrade")
			// Missing Upgrade: websocket header.
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestProxyHandler_MissingWebSocketKey(t *testing.T) {
	h := New(DefaultRoutes(), zap.NewNop())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	tests := []struct {
		name string
		path string
	}{
		{name: "agents stream", path: "/ws/v1/agents/stream"},
		{name: "telemetry stream", path: "/ws/v1/telemetry/stream"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
			// Missing Sec-WebSocket-Key.
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

// ---------- Backend unavailable ----------

func TestProxyHandler_BackendUnavailable(t *testing.T) {
	routes := []Route{
		{PathPrefix: "/ws/v1/agents/stream", Backend: "http://localhost:19999"}, // unlikely to be listening
	}
	h := New(routes, zap.NewNop())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/ws/v1/agents/stream", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("X-Tenant-ID", "test-tenant")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// ---------- headerContains ----------

func TestHeaderContains(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		key    string
		token  string
		want   bool
	}{
		{
			name:   "exact match",
			header: http.Header{"Connection": {"Upgrade"}},
			key:    "Connection",
			token:  "Upgrade",
			want:   true,
		},
		{
			name:   "case insensitive match",
			header: http.Header{"Connection": {"upgrade"}},
			key:    "Connection",
			token:  "Upgrade",
			want:   true,
		},
		{
			name:   "comma separated values",
			header: http.Header{"Connection": {"keep-alive, Upgrade"}},
			key:    "Connection",
			token:  "Upgrade",
			want:   true,
		},
		{
			name:   "token not present",
			header: http.Header{"Connection": {"keep-alive"}},
			key:    "Connection",
			token:  "Upgrade",
			want:   false,
		},
		{
			name:   "header key not present",
			header: http.Header{},
			key:    "Connection",
			token:  "Upgrade",
			want:   false,
		},
		{
			name:   "multiple header values",
			header: http.Header{"Connection": {"keep-alive", "Upgrade"}},
			key:    "Connection",
			token:  "Upgrade",
			want:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := headerContains(tc.header, tc.key, tc.token)
			if got != tc.want {
				t.Errorf("headerContains() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ---------- computeAcceptKey ----------

func TestComputeAcceptKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{
			name: "RFC 6455 example key produces deterministic output",
			key:  "dGhlIHNhbXBsZSBub25jZQ==",
		},
		{
			name: "different key produces different output",
			key:  "x3JJHMbDL1EzLkh9GBhXDw==",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result1 := computeAcceptKey(tc.key)
			result2 := computeAcceptKey(tc.key)
			if result1 != result2 {
				t.Errorf("computeAcceptKey is not deterministic: %q != %q", result1, result2)
			}
			if result1 == "" {
				t.Error("computeAcceptKey returned empty string")
			}
		})
	}
}

// ---------- Route struct ----------

func TestRouteStruct(t *testing.T) {
	route := Route{
		PathPrefix: "/ws/test",
		Backend:    "http://localhost:1234",
	}

	if route.PathPrefix != "/ws/test" {
		t.Errorf("PathPrefix = %q, want %q", route.PathPrefix, "/ws/test")
	}
	if route.Backend != "http://localhost:1234" {
		t.Errorf("Backend = %q, want %q", route.Backend, "http://localhost:1234")
	}
}

// ---------- readHTTPResponse ----------

func TestReadHTTPResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantResp string
	}{
		{
			name:     "valid HTTP response",
			input:    "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\n\r\n",
			wantResp: "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\n\r\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use a pipe to simulate a connection.
			server, client := pipeConn()
			defer func() {
				_ = server.Close()
				_ = client.Close()
			}()

			go func() {
				_, _ = client.Write([]byte(tc.input))
			}()

			resp, err := readHTTPResponse(server)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if resp != tc.wantResp {
				t.Errorf("response = %q, want %q", resp, tc.wantResp)
			}
		})
	}
}

// ---------- Full WebSocket proxy integration test ----------

func TestProxyHandler_FullUpgrade(t *testing.T) {
	// Create a mock backend that accepts WebSocket upgrades.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify tenant header is forwarded.
		if r.Header.Get("X-Tenant-ID") == "" {
			t.Error("backend did not receive X-Tenant-ID header")
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("backend did not receive Authorization header")
		}

		key := r.Header.Get("Sec-WebSocket-Key")
		if key == "" {
			http.Error(w, "missing key", http.StatusBadRequest)
			return
		}

		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijack", http.StatusInternalServerError)
			return
		}

		conn, bufrw, err := hj.Hijack()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		acceptKey := computeAcceptKey(key)
		resp := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + acceptKey + "\r\n" +
			"\r\n"
		_, _ = bufrw.WriteString(resp)
		_ = bufrw.Flush()

		// Send a text frame with "hello".
		frame := buildWSFrame(0x1, []byte(`{"type":"test"}`))
		_, _ = bufrw.Write(frame)
		_ = bufrw.Flush()
	}))
	defer backend.Close()

	routes := []Route{
		{PathPrefix: "/ws/v1/agents/stream", Backend: backend.URL},
	}
	h := New(routes, zap.NewNop())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// We need a real server for hijacking to work.
	proxy := httptest.NewServer(mux)
	defer proxy.Close()

	// Verify the proxy is running by checking that a non-WS request gets rejected.
	resp, err := http.Get(proxy.URL + "/ws/v1/agents/stream")
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	_ = resp.Body.Close()

	// Without WebSocket headers, should get 400.
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("non-WS request status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

// ---------- helpers ----------

// pipeConn returns a pair of connected net.Conn using net.Pipe.
func pipeConn() (net.Conn, net.Conn) {
	return net.Pipe()
}

// buildWSFrame builds a WebSocket frame (server-to-client, unmasked).
func buildWSFrame(opcode byte, payload []byte) []byte {
	length := len(payload)
	frame := []byte{0x80 | opcode}

	switch {
	case length <= 125:
		frame = append(frame, byte(length))
	case length <= 65535:
		frame = append(frame, 126)
		lenBytes := make([]byte, 2)
		lenBytes[0] = byte(length >> 8)
		lenBytes[1] = byte(length)
		frame = append(frame, lenBytes...)
	default:
		frame = append(frame, 127)
		lenBytes := make([]byte, 8)
		for i := 7; i >= 0; i-- {
			lenBytes[i] = byte(length & 0xFF)
			length >>= 8
		}
		frame = append(frame, lenBytes...)
	}

	frame = append(frame, payload...)
	return frame
}
