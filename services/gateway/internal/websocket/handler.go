package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// wsGUID is the magic GUID defined by RFC 6455 for the WebSocket handshake.
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Route maps a WebSocket path prefix to its backend target address.
type Route struct {
	PathPrefix string
	Backend    string // e.g., "http://localhost:8084"
}

// DefaultRoutes returns the WebSocket routes proxying to the control-plane.
func DefaultRoutes() []Route {
	return []Route{
		{PathPrefix: "/ws/v1/agents/stream", Backend: "http://localhost:8084"},
		{PathPrefix: "/ws/v1/telemetry/stream", Backend: "http://localhost:8084"},
	}
}

// Handler proxies WebSocket connections from external clients to the
// appropriate backend service. It performs the initial HTTP-to-WebSocket
// upgrade with the client, then separately upgrades the connection to
// the backend, and finally relays frames bidirectionally.
type Handler struct {
	routes []Route
	logger *zap.Logger
}

// New creates a new WebSocket proxy handler.
func New(routes []Route, logger *zap.Logger) *Handler {
	return &Handler{
		routes: routes,
		logger: logger,
	}
}

// RegisterRoutes registers the WebSocket proxy routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	for _, route := range h.routes {
		r := route // capture for closure
		mux.HandleFunc(r.PathPrefix, h.proxyHandler(r))
	}
}

// proxyHandler returns an http.HandlerFunc that proxies a WebSocket
// connection to the backend.
func (h *Handler) proxyHandler(route Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate WebSocket upgrade request.
		if !headerContains(r.Header, "Connection", "upgrade") {
			http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"missing Connection: Upgrade header"}}`, http.StatusBadRequest)
			return
		}
		if !headerContains(r.Header, "Upgrade", "websocket") {
			http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"missing Upgrade: websocket header"}}`, http.StatusBadRequest)
			return
		}

		wsKey := r.Header.Get("Sec-WebSocket-Key")
		if wsKey == "" {
			http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"missing Sec-WebSocket-Key"}}`, http.StatusBadRequest)
			return
		}

		// Parse backend URL.
		backendURL, err := url.Parse(route.Backend)
		if err != nil {
			h.logger.Error("invalid backend URL", zap.String("backend", route.Backend), zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Connect to backend via raw TCP.
		backendAddr := backendURL.Host
		if !strings.Contains(backendAddr, ":") {
			if backendURL.Scheme == "https" || backendURL.Scheme == "wss" {
				backendAddr += ":443"
			} else {
				backendAddr += ":80"
			}
		}

		backendConn, err := net.Dial("tcp", backendAddr)
		if err != nil {
			h.logger.Error("failed to connect to backend", zap.String("addr", backendAddr), zap.Error(err))
			http.Error(w, `{"error":{"code":"SERVICE_UNAVAILABLE","message":"backend unavailable"}}`, http.StatusServiceUnavailable)
			return
		}

		// Build the WebSocket upgrade request for the backend.
		// Forward relevant headers (tenant, auth) so the backend can
		// authenticate and extract the tenant.
		backendPath := r.URL.Path
		if r.URL.RawQuery != "" {
			backendPath += "?" + r.URL.RawQuery
		}

		upgradeReq := "GET " + backendPath + " HTTP/1.1\r\n" +
			"Host: " + backendURL.Host + "\r\n" +
			"Connection: Upgrade\r\n" +
			"Upgrade: websocket\r\n" +
			"Sec-WebSocket-Version: 13\r\n" +
			"Sec-WebSocket-Key: " + wsKey + "\r\n"

		// Forward tenant header.
		if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
			upgradeReq += "X-Tenant-ID: " + tenantID + "\r\n"
		}

		// Forward authorization header.
		if auth := r.Header.Get("Authorization"); auth != "" {
			upgradeReq += "Authorization: " + auth + "\r\n"
		}

		upgradeReq += "\r\n"

		if _, err := backendConn.Write([]byte(upgradeReq)); err != nil {
			h.logger.Error("failed to write upgrade to backend", zap.Error(err))
			_ = backendConn.Close()
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Read the backend's upgrade response.
		// We read byte by byte looking for the end of HTTP headers (\r\n\r\n).
		backendResponse, err := readHTTPResponse(backendConn)
		if err != nil {
			h.logger.Error("failed to read backend upgrade response", zap.Error(err))
			_ = backendConn.Close()
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Check if backend accepted the upgrade.
		if !strings.Contains(backendResponse, "101") {
			h.logger.Error("backend did not accept WebSocket upgrade",
				zap.String("response", backendResponse),
			)
			_ = backendConn.Close()
			http.Error(w, `{"error":{"code":"WS_UPGRADE_FAILED","message":"backend rejected WebSocket upgrade"}}`, http.StatusBadGateway)
			return
		}

		// Hijack the client connection.
		hj, ok := w.(http.Hijacker)
		if !ok {
			h.logger.Error("server does not support hijacking")
			_ = backendConn.Close()
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		clientConn, _, err := hj.Hijack()
		if err != nil {
			h.logger.Error("hijack failed", zap.Error(err))
			_ = backendConn.Close()
			return
		}

		// Complete the client-side WebSocket handshake.
		acceptKey := computeAcceptKey(wsKey)
		clientResponse := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + acceptKey + "\r\n" +
			"\r\n"

		if _, err := clientConn.Write([]byte(clientResponse)); err != nil {
			h.logger.Error("failed to write upgrade to client", zap.Error(err))
			_ = clientConn.Close()
			_ = backendConn.Close()
			return
		}

		h.logger.Info("websocket connection proxied",
			zap.String("path", r.URL.Path),
			zap.String("backend", route.Backend),
			zap.String("remote_addr", r.RemoteAddr),
		)

		// Relay frames bidirectionally.
		relay(clientConn, backendConn)
	}
}

// relay copies data bidirectionally between two connections.
// When either direction finishes (EOF or error), both connections are closed.
func relay(client, backend net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(backend, client)
		_ = backend.Close()
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(client, backend)
		_ = client.Close()
	}()

	wg.Wait()
}

// readHTTPResponse reads the HTTP response headers from a connection.
// It reads until it finds the \r\n\r\n terminator.
func readHTTPResponse(conn net.Conn) (string, error) {
	var buf []byte
	oneByte := make([]byte, 1)
	for {
		_, err := conn.Read(oneByte)
		if err != nil {
			return string(buf), fmt.Errorf("read response: %w", err)
		}
		buf = append(buf, oneByte[0])
		if len(buf) >= 4 &&
			buf[len(buf)-4] == '\r' &&
			buf[len(buf)-3] == '\n' &&
			buf[len(buf)-2] == '\r' &&
			buf[len(buf)-1] == '\n' {
			return string(buf), nil
		}
		// Safety: don't read more than 8KB of headers.
		if len(buf) > 8192 {
			return string(buf), fmt.Errorf("response headers too large")
		}
	}
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
