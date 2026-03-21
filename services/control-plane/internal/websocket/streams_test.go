package websocket

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ---------- Hub construction ----------

func TestNewHub(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates hub with empty client maps"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hub := NewHub(zap.NewNop())
			if hub == nil {
				t.Fatal("NewHub returned nil")
			}
			if hub.ClientCount(StreamAgents) != 0 {
				t.Errorf("agents client count = %d, want 0", hub.ClientCount(StreamAgents))
			}
			if hub.ClientCount(StreamTelemetry) != 0 {
				t.Errorf("telemetry client count = %d, want 0", hub.ClientCount(StreamTelemetry))
			}
		})
	}
}

// ---------- RegisterRoutes ----------

func TestRegisterRoutes(t *testing.T) {
	hub := NewHub(zap.NewNop())
	mux := http.NewServeMux()
	hub.RegisterRoutes(mux)

	tests := []struct {
		name string
		path string
	}{
		{name: "agents stream path registered", path: "/ws/v1/agents/stream"},
		{name: "telemetry stream path registered", path: "/ws/v1/telemetry/stream"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Should not return 404 (route registered). It will return 400
			// because we do not supply the required tenant header.
			if w.Code == http.StatusNotFound {
				t.Errorf("route %s returned 404 — not registered", tc.path)
			}
		})
	}
}

// ---------- Tenant required ----------

func TestHandleStream_MissingTenant(t *testing.T) {
	hub := NewHub(zap.NewNop())
	mux := http.NewServeMux()
	hub.RegisterRoutes(mux)

	tests := []struct {
		name string
		path string
	}{
		{name: "agents stream without tenant", path: "/ws/v1/agents/stream"},
		{name: "telemetry stream without tenant", path: "/ws/v1/telemetry/stream"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}

			var body map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			errObj, ok := body["error"].(map[string]interface{})
			if !ok {
				t.Fatalf("error is not an object: %T", body["error"])
			}
			if errObj["code"] != "TENANT_REQUIRED" {
				t.Errorf("error.code = %q, want %q", errObj["code"], "TENANT_REQUIRED")
			}
		})
	}
}

// ---------- Tenant via query param ----------

func TestHandleStream_TenantViaQueryParam(t *testing.T) {
	hub := NewHub(zap.NewNop())
	mux := http.NewServeMux()
	hub.RegisterRoutes(mux)

	tests := []struct {
		name string
		path string
	}{
		{name: "agents stream with query tenant", path: "/ws/v1/agents/stream?tenant_id=test-tenant"},
		{name: "telemetry stream with query tenant", path: "/ws/v1/telemetry/stream?tenant_id=test-tenant"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Should not return 400 (TENANT_REQUIRED), because the query
			// param provides the tenant. Instead it will fail on the
			// WebSocket upgrade (missing headers), which is expected.
			if w.Code == http.StatusBadRequest {
				var body map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&body); err == nil {
					if errObj, ok := body["error"].(map[string]interface{}); ok {
						if errObj["code"] == "TENANT_REQUIRED" {
							t.Error("should accept tenant_id from query param")
						}
					}
				}
			}
		})
	}
}

// ---------- computeAcceptKey ----------

func TestComputeAcceptKey(t *testing.T) {
	// RFC 6455 section 4.2.2 example:
	// Key: "dGhlIHNhbXBsZSBub25jZQ=="
	// Expected accept: "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "RFC 6455 example key",
			key:  "dGhlIHNhbXBsZSBub25jZQ==",
			want: "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeAcceptKey(tc.key)
			if got != tc.want {
				t.Errorf("computeAcceptKey(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
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

// ---------- buildWSFrame ----------

func TestBuildWSFrame(t *testing.T) {
	tests := []struct {
		name        string
		opcode      byte
		payload     []byte
		wantMinLen  int
		wantOpcode  byte
		wantPayload []byte
	}{
		{
			name:        "small text frame",
			opcode:      0x1,
			payload:     []byte("hello"),
			wantMinLen:  7, // 2 header bytes + 5 payload bytes
			wantOpcode:  0x1,
			wantPayload: []byte("hello"),
		},
		{
			name:       "empty payload",
			opcode:     0x1,
			payload:    []byte{},
			wantMinLen: 2, // 2 header bytes only
			wantOpcode: 0x1,
		},
		{
			name:       "close frame",
			opcode:     0x8,
			payload:    []byte{},
			wantMinLen: 2,
			wantOpcode: 0x8,
		},
		{
			name:       "ping frame",
			opcode:     0x9,
			payload:    []byte("ping"),
			wantMinLen: 6,
			wantOpcode: 0x9,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			frame := buildWSFrame(tc.opcode, tc.payload)

			if len(frame) < tc.wantMinLen {
				t.Errorf("frame length = %d, want >= %d", len(frame), tc.wantMinLen)
			}

			// Check FIN bit is set and opcode matches.
			if frame[0] != (0x80 | tc.wantOpcode) {
				t.Errorf("first byte = 0x%02X, want 0x%02X", frame[0], 0x80|tc.wantOpcode)
			}

			// Verify payload length encoding.
			payloadLen := frame[1] & 0x7F
			var headerLen int
			switch {
			case payloadLen <= 125:
				headerLen = 2
			case payloadLen == 126:
				headerLen = 4
			default:
				headerLen = 10
			}

			actualPayload := frame[headerLen:]
			if tc.wantPayload != nil {
				if string(actualPayload) != string(tc.wantPayload) {
					t.Errorf("payload = %q, want %q", string(actualPayload), string(tc.wantPayload))
				}
			}
		})
	}
}

// ---------- buildWSFrame medium payload (126-65535 bytes) ----------

func TestBuildWSFrame_MediumPayload(t *testing.T) {
	payload := make([]byte, 200)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	frame := buildWSFrame(0x1, payload)

	// For 200 bytes: first byte + 126 marker + 2-byte length + payload
	expectedLen := 1 + 1 + 2 + 200
	if len(frame) != expectedLen {
		t.Errorf("frame length = %d, want %d", len(frame), expectedLen)
	}

	// Check 126 marker.
	if frame[1] != 126 {
		t.Errorf("length byte = %d, want 126", frame[1])
	}

	// Check encoded length.
	encodedLen := binary.BigEndian.Uint16(frame[2:4])
	if encodedLen != 200 {
		t.Errorf("encoded length = %d, want 200", encodedLen)
	}
}

// ---------- readWSFrame ----------

func TestReadWSFrame(t *testing.T) {
	tests := []struct {
		name        string
		frame       []byte
		wantOpcode  byte
		wantPayload string
		wantErr     bool
	}{
		{
			name: "unmasked text frame",
			// FIN+text, length=5, "hello"
			frame:       append([]byte{0x81, 0x05}, []byte("hello")...),
			wantOpcode:  0x1,
			wantPayload: "hello",
		},
		{
			name: "masked text frame",
			// FIN+text, masked+length=5, mask=[0,0,0,0], "hello"
			frame: append(
				[]byte{0x81, 0x85, 0x00, 0x00, 0x00, 0x00},
				[]byte("hello")...),
			wantOpcode:  0x1,
			wantPayload: "hello",
		},
		{
			name:       "close frame",
			frame:      []byte{0x88, 0x00},
			wantOpcode: 0x8,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(string(tc.frame)))
			opcode, payload, err := readWSFrame(r)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if opcode != tc.wantOpcode {
				t.Errorf("opcode = 0x%02X, want 0x%02X", opcode, tc.wantOpcode)
			}

			if tc.wantPayload != "" && string(payload) != tc.wantPayload {
				t.Errorf("payload = %q, want %q", string(payload), tc.wantPayload)
			}
		})
	}
}

// ---------- Broadcast with tenant isolation ----------

func TestBroadcast_TenantIsolation(t *testing.T) {
	hub := NewHub(zap.NewNop())

	// Create mock clients via pipe connections.
	tenantA := createMockClient("tenant-a")
	tenantB := createMockClient("tenant-b")

	hub.addClient(StreamAgents, tenantA.client)
	hub.addClient(StreamAgents, tenantB.client)

	if hub.ClientCount(StreamAgents) != 2 {
		t.Fatalf("client count = %d, want 2", hub.ClientCount(StreamAgents))
	}

	// Broadcast an event for tenant-a only.
	event := &Event{
		Type:      EventAgentRegistered,
		TenantID:  "tenant-a",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   map[string]string{"agent_id": "agent-1"},
	}

	// Run broadcast in a goroutine because net.Pipe writes block
	// until the remote end reads.
	go hub.Broadcast(StreamAgents, event)

	// tenant-a should receive the event.
	received := readFromPipe(t, tenantA.remote)
	if received == nil {
		t.Fatal("tenant-a did not receive event")
	}
	if received.Type != EventAgentRegistered {
		t.Errorf("event type = %q, want %q", received.Type, EventAgentRegistered)
	}
	if received.TenantID != "tenant-a" {
		t.Errorf("tenant_id = %q, want %q", received.TenantID, "tenant-a")
	}

	// tenant-b should NOT receive the event.
	noData := tryReadFromPipe(tenantB.remote)
	if noData != nil {
		t.Error("tenant-b received an event that was not intended for them")
	}

	// Cleanup.
	tenantA.close()
	tenantB.close()
}

// ---------- ClientCountForTenant ----------

func TestClientCountForTenant(t *testing.T) {
	hub := NewHub(zap.NewNop())

	tests := []struct {
		name     string
		tenantID string
		stream   StreamType
		addCount int
		want     int
	}{
		{
			name:     "no clients",
			tenantID: "tenant-x",
			stream:   StreamAgents,
			addCount: 0,
			want:     0,
		},
		{
			name:     "two clients for same tenant",
			tenantID: "tenant-y",
			stream:   StreamAgents,
			addCount: 2,
			want:     2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHub(zap.NewNop())
			mocks := make([]mockConn, 0, tc.addCount)
			for i := 0; i < tc.addCount; i++ {
				m := createMockClient(tc.tenantID)
				mocks = append(mocks, m)
				h.addClient(tc.stream, m.client)
			}

			got := h.ClientCountForTenant(tc.stream, tc.tenantID)
			if got != tc.want {
				t.Errorf("ClientCountForTenant() = %d, want %d", got, tc.want)
			}

			for _, m := range mocks {
				m.close()
			}
		})
	}

	_ = hub // suppress unused
}

// ---------- BroadcastAgentEvent convenience ----------

func TestBroadcastAgentEvent(t *testing.T) {
	hub := NewHub(zap.NewNop())
	mock := createMockClient("tenant-1")
	hub.addClient(StreamAgents, mock.client)

	// Run in goroutine because net.Pipe writes block until read.
	go hub.BroadcastAgentEvent(EventAgentHeartbeat, "tenant-1", map[string]string{"agent_id": "a1"})

	received := readFromPipe(t, mock.remote)
	if received == nil {
		t.Fatal("did not receive event")
	}
	if received.Type != EventAgentHeartbeat {
		t.Errorf("type = %q, want %q", received.Type, EventAgentHeartbeat)
	}

	mock.close()
}

// ---------- BroadcastTelemetryEvent convenience ----------

func TestBroadcastTelemetryEvent(t *testing.T) {
	hub := NewHub(zap.NewNop())
	mock := createMockClient("tenant-2")
	hub.addClient(StreamTelemetry, mock.client)

	// Run in goroutine because net.Pipe writes block until read.
	go hub.BroadcastTelemetryEvent(EventAlertFired, "tenant-2", map[string]string{"alert_id": "alert-1"})

	received := readFromPipe(t, mock.remote)
	if received == nil {
		t.Fatal("did not receive event")
	}
	if received.Type != EventAlertFired {
		t.Errorf("type = %q, want %q", received.Type, EventAlertFired)
	}

	mock.close()
}

// ---------- Shutdown ----------

func TestShutdown(t *testing.T) {
	hub := NewHub(zap.NewNop())

	mock1 := createMockClient("tenant-1")
	mock2 := createMockClient("tenant-2")
	hub.addClient(StreamAgents, mock1.client)
	hub.addClient(StreamTelemetry, mock2.client)

	hub.Shutdown()

	if hub.ClientCount(StreamAgents) != 0 {
		t.Errorf("agents client count after shutdown = %d, want 0", hub.ClientCount(StreamAgents))
	}
	if hub.ClientCount(StreamTelemetry) != 0 {
		t.Errorf("telemetry client count after shutdown = %d, want 0", hub.ClientCount(StreamTelemetry))
	}
}

// ---------- Event types ----------

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		want      string
	}{
		{name: "agent registered", eventType: EventAgentRegistered, want: "agent.registered"},
		{name: "agent heartbeat", eventType: EventAgentHeartbeat, want: "agent.heartbeat"},
		{name: "agent status change", eventType: EventAgentStatusChange, want: "agent.status_change"},
		{name: "agent quarantined", eventType: EventAgentQuarantined, want: "agent.quarantined"},
		{name: "span created", eventType: EventSpanCreated, want: "telemetry.span_created"},
		{name: "alert fired", eventType: EventAlertFired, want: "telemetry.alert_fired"},
		{name: "predictive warning", eventType: EventPredictiveWarning, want: "telemetry.predictive_warning"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.eventType) != tc.want {
				t.Errorf("EventType = %q, want %q", tc.eventType, tc.want)
			}
		})
	}
}

// ---------- Event JSON serialization ----------

func TestEvent_JSON(t *testing.T) {
	event := Event{
		Type:      EventAgentRegistered,
		TenantID:  "tenant-1",
		Timestamp: "2026-03-22T10:00:00Z",
		Payload:   map[string]string{"agent_id": "agent-1"},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	tests := []struct {
		key  string
		want string
	}{
		{key: "type", want: "agent.registered"},
		{key: "tenant_id", want: "tenant-1"},
		{key: "timestamp", want: "2026-03-22T10:00:00Z"},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			got, ok := decoded[tc.key].(string)
			if !ok {
				t.Fatalf("%s is not a string: %T", tc.key, decoded[tc.key])
			}
			if got != tc.want {
				t.Errorf("%s = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

// ---------- Stream type constants ----------

func TestStreamTypes(t *testing.T) {
	tests := []struct {
		name   string
		stream StreamType
		want   string
	}{
		{name: "agents stream", stream: StreamAgents, want: "agents"},
		{name: "telemetry stream", stream: StreamTelemetry, want: "telemetry"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.stream) != tc.want {
				t.Errorf("StreamType = %q, want %q", tc.stream, tc.want)
			}
		})
	}
}

// ---------- helpers ----------

// mockConn wraps a net.Pipe pair and the client struct for testing.
type mockConn struct {
	client *client
	remote net.Conn // the "remote" end of the pipe for reading what was sent
}

func createMockClient(tenantID string) mockConn {
	serverConn, remoteConn := net.Pipe()
	bufrw := bufio.NewReadWriter(
		bufio.NewReader(serverConn),
		bufio.NewWriter(serverConn),
	)
	c := &client{
		tenantID: tenantID,
		conn:     serverConn,
		bufrw:    bufrw,
	}
	return mockConn{client: c, remote: remoteConn}
}

func (m mockConn) close() {
	_ = m.client.conn.Close()
	_ = m.remote.Close()
}

// readFromPipe reads a WebSocket text frame from the remote end of a pipe
// and decodes it as an Event.
func readFromPipe(t *testing.T, conn net.Conn) *Event {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	r := bufio.NewReader(conn)
	_, payload, err := readWSFrame(r)
	if err != nil {
		t.Fatalf("readWSFrame from pipe: %v", err)
	}

	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	return &event
}

// tryReadFromPipe attempts a non-blocking read from a pipe. Returns nil
// if nothing is available within a short timeout.
func tryReadFromPipe(conn net.Conn) *Event {
	_ = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	r := bufio.NewReader(conn)
	_, payload, err := readWSFrame(r)
	if err != nil {
		return nil
	}
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil
	}
	return &event
}
