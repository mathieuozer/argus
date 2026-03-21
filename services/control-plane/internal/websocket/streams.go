package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// wsGUID is the magic GUID defined by RFC 6455 for the WebSocket handshake.
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// StreamType identifies which event stream a client subscribes to.
type StreamType string

const (
	StreamAgents    StreamType = "agents"
	StreamTelemetry StreamType = "telemetry"
)

// EventType classifies the kind of event being broadcast.
type EventType string

const (
	// Agent events
	EventAgentRegistered   EventType = "agent.registered"
	EventAgentHeartbeat    EventType = "agent.heartbeat"
	EventAgentStatusChange EventType = "agent.status_change"
	EventAgentQuarantined  EventType = "agent.quarantined"

	// Telemetry events
	EventSpanCreated       EventType = "telemetry.span_created"
	EventAlertFired        EventType = "telemetry.alert_fired"
	EventPredictiveWarning EventType = "telemetry.predictive_warning"
)

// Event is the JSON envelope sent to connected WebSocket clients.
type Event struct {
	Type      EventType   `json:"type"`
	TenantID  string      `json:"tenant_id"`
	Timestamp string      `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// client represents a single connected WebSocket client.
type client struct {
	tenantID string
	conn     net.Conn
	bufrw    *bufio.ReadWriter
	mu       sync.Mutex // serialises writes to this connection
}

// Hub manages WebSocket client connections and event broadcasting.
// It maintains separate subscriber lists for agent and telemetry streams
// and enforces tenant isolation so that events are only delivered to
// clients whose tenant matches the event's TenantID.
type Hub struct {
	mu      sync.RWMutex
	clients map[StreamType]map[*client]struct{}
	logger  *zap.Logger

	// pingInterval controls how often the hub sends WebSocket ping frames.
	pingInterval time.Duration
}

// NewHub creates a new WebSocket stream hub.
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients: map[StreamType]map[*client]struct{}{
			StreamAgents:    {},
			StreamTelemetry: {},
		},
		logger:       logger,
		pingInterval: 30 * time.Second,
	}
}

// RegisterRoutes registers the WebSocket endpoints on the provided mux.
func (h *Hub) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ws/v1/agents/stream", h.handleStream(StreamAgents))
	mux.HandleFunc("/ws/v1/telemetry/stream", h.handleStream(StreamTelemetry))
}

// handleStream returns an http.HandlerFunc that upgrades the connection
// to WebSocket and subscribes the client to the given stream.
func (h *Hub) handleStream(stream StreamType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")

		// Also check query param for WebSocket clients that cannot set headers.
		if tenantID == "" {
			tenantID = r.URL.Query().Get("tenant_id")
		}
		if tenantID == "" {
			writeWSError(w, http.StatusBadRequest, "TENANT_REQUIRED", "tenant identification required")
			return
		}

		conn, bufrw, err := upgradeWebSocket(w, r)
		if err != nil {
			h.logger.Error("websocket upgrade failed",
				zap.String("stream", string(stream)),
				zap.Error(err),
			)
			return
		}

		c := &client{
			tenantID: tenantID,
			conn:     conn,
			bufrw:    bufrw,
		}

		h.addClient(stream, c)
		h.logger.Info("websocket client connected",
			zap.String("stream", string(stream)),
			zap.String("tenant_id", tenantID),
			zap.String("remote_addr", conn.RemoteAddr().String()),
		)

		// Send a welcome message so the client knows the connection is live.
		welcome := Event{
			Type:      EventType("connected"),
			TenantID:  tenantID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Payload:   map[string]string{"stream": string(stream)},
		}
		if err := h.sendEvent(c, &welcome); err != nil {
			h.removeClient(stream, c)
			return
		}

		// Block on reading; when the client disconnects or sends a close
		// frame the read loop exits and we clean up.
		h.readLoop(stream, c)
	}
}

// Broadcast sends an event to all clients on the specified stream whose
// tenant matches the event's TenantID. This is the primary API used by
// other services to push real-time updates.
func (h *Hub) Broadcast(stream StreamType, event *Event) {
	h.mu.RLock()
	clients := h.clients[stream]
	// Snapshot current clients under read lock.
	targets := make([]*client, 0, len(clients))
	for c := range clients {
		if c.tenantID == event.TenantID {
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range targets {
		if err := h.sendEvent(c, event); err != nil {
			h.logger.Debug("removing disconnected client",
				zap.String("stream", string(stream)),
				zap.String("tenant_id", c.tenantID),
			)
			h.removeClient(stream, c)
		}
	}
}

// BroadcastAgentEvent is a convenience method for broadcasting agent events.
func (h *Hub) BroadcastAgentEvent(eventType EventType, tenantID string, payload interface{}) {
	h.Broadcast(StreamAgents, &Event{
		Type:      eventType,
		TenantID:  tenantID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	})
}

// BroadcastTelemetryEvent is a convenience method for broadcasting telemetry events.
func (h *Hub) BroadcastTelemetryEvent(eventType EventType, tenantID string, payload interface{}) {
	h.Broadcast(StreamTelemetry, &Event{
		Type:      eventType,
		TenantID:  tenantID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	})
}

// ClientCount returns the number of connected clients for a stream.
func (h *Hub) ClientCount(stream StreamType) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[stream])
}

// ClientCountForTenant returns the number of connected clients for a
// specific tenant on a given stream.
func (h *Hub) ClientCountForTenant(stream StreamType, tenantID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for c := range h.clients[stream] {
		if c.tenantID == tenantID {
			count++
		}
	}
	return count
}

// Shutdown closes all connected clients.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for stream, clients := range h.clients {
		for c := range clients {
			_ = c.conn.Close()
			delete(clients, c)
		}
		h.clients[stream] = make(map[*client]struct{})
	}
}

// addClient registers a client for the given stream.
func (h *Hub) addClient(stream StreamType, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[stream][c] = struct{}{}
}

// removeClient unregisters and closes a client.
func (h *Hub) removeClient(stream StreamType, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[stream][c]; ok {
		_ = c.conn.Close()
		delete(h.clients[stream], c)
	}
}

// sendEvent marshals an event to JSON and sends it as a WebSocket text frame.
func (h *Hub) sendEvent(c *client, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return writeWSTextFrame(c, data)
}

// readLoop reads from the WebSocket connection until it closes or errors.
// It handles ping/pong and close frames per RFC 6455.
func (h *Hub) readLoop(stream StreamType, c *client) {
	defer h.removeClient(stream, c)

	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		opcode, payload, err := readWSFrame(c.bufrw.Reader)
		if err != nil {
			return
		}

		switch opcode {
		case 0x8: // close
			// Send close frame back.
			closeFrame := buildWSFrame(0x8, payload)
			c.mu.Lock()
			_, _ = c.bufrw.Write(closeFrame)
			_ = c.bufrw.Flush()
			c.mu.Unlock()
			return
		case 0x9: // ping
			// Respond with pong.
			pongFrame := buildWSFrame(0xA, payload)
			c.mu.Lock()
			_, _ = c.bufrw.Write(pongFrame)
			_ = c.bufrw.Flush()
			c.mu.Unlock()
		case 0xA: // pong — ignore
		case 0x1: // text frame — we ignore client messages for now
		case 0x2: // binary frame — ignore
		}
	}
}

// ---------- WebSocket protocol helpers (RFC 6455) ----------

// upgradeWebSocket performs the HTTP-to-WebSocket upgrade handshake.
func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, *bufio.ReadWriter, error) {
	if !headerContains(r.Header, "Connection", "upgrade") {
		http.Error(w, "missing Connection: Upgrade header", http.StatusBadRequest)
		return nil, nil, fmt.Errorf("missing Connection: Upgrade")
	}
	if !headerContains(r.Header, "Upgrade", "websocket") {
		http.Error(w, "missing Upgrade: websocket header", http.StatusBadRequest)
		return nil, nil, fmt.Errorf("missing Upgrade: websocket")
	}

	wsKey := r.Header.Get("Sec-WebSocket-Key")
	if wsKey == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return nil, nil, fmt.Errorf("missing Sec-WebSocket-Key")
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "server does not support hijacking", http.StatusInternalServerError)
		return nil, nil, fmt.Errorf("http.Hijacker not supported")
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, nil, fmt.Errorf("hijack: %w", err)
	}

	// Compute accept key per RFC 6455 section 4.2.2.
	acceptKey := computeAcceptKey(wsKey)

	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n" +
		"\r\n"

	if _, err := bufrw.WriteString(response); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("write upgrade response: %w", err)
	}
	if err := bufrw.Flush(); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("flush upgrade response: %w", err)
	}

	return conn, bufrw, nil
}

// computeAcceptKey computes the Sec-WebSocket-Accept value per RFC 6455.
func computeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte(wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// headerContains checks whether a header value contains a token
// (case-insensitive, comma-separated).
func headerContains(h http.Header, key, token string) bool {
	for _, v := range h[http.CanonicalHeaderKey(key)] {
		for _, s := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(s), token) {
				return true
			}
		}
	}
	return false
}

// writeWSTextFrame sends a text frame (opcode 0x1) to the client.
func writeWSTextFrame(c *client, data []byte) error {
	frame := buildWSFrame(0x1, data)
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.bufrw.Write(frame)
	if err != nil {
		return err
	}
	return c.bufrw.Flush()
}

// buildWSFrame builds a WebSocket frame with the given opcode and payload.
// The FIN bit is always set (no fragmentation). Server-to-client frames
// are never masked per RFC 6455.
func buildWSFrame(opcode byte, payload []byte) []byte {
	length := len(payload)

	// First byte: FIN bit + opcode.
	frame := []byte{0x80 | opcode}

	// Second byte onward: payload length encoding.
	switch {
	case length <= 125:
		frame = append(frame, byte(length))
	case length <= 65535:
		frame = append(frame, 126)
		lenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lenBytes, uint16(length))
		frame = append(frame, lenBytes...)
	default:
		frame = append(frame, 127)
		lenBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lenBytes, uint64(length))
		frame = append(frame, lenBytes...)
	}

	frame = append(frame, payload...)
	return frame
}

// readWSFrame reads a single WebSocket frame from the reader.
// Client-to-server frames are always masked per RFC 6455.
func readWSFrame(r *bufio.Reader) (opcode byte, payload []byte, err error) {
	// Read first two bytes: FIN+opcode, mask+length.
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}

	opcode = header[0] & 0x0F
	masked := (header[1] & 0x80) != 0
	length := uint64(header[1] & 0x7F)

	switch length {
	case 126:
		lenBytes := make([]byte, 2)
		if _, err := io.ReadFull(r, lenBytes); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(lenBytes))
	case 127:
		lenBytes := make([]byte, 8)
		if _, err := io.ReadFull(r, lenBytes); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(lenBytes)
	}

	var maskKey []byte
	if masked {
		maskKey = make([]byte, 4)
		if _, err := io.ReadFull(r, maskKey); err != nil {
			return 0, nil, err
		}
	}

	payload = make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, nil, err
		}
	}

	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	return opcode, payload, nil
}

// writeWSError writes an HTTP JSON error response. This is used before
// the WebSocket upgrade, so standard HTTP writing is still valid.
func writeWSError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
