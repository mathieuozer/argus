package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/argus-platform/argus/pkg/config"
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/pkg/middleware"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/identity/internal/ca"
	"github.com/argus-platform/argus/services/identity/internal/revocation"
	"github.com/argus-platform/argus/services/identity/internal/spiffe"
	"github.com/argus-platform/argus/services/identity/internal/vault"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.Default()
	defer log.Sync()

	authority, err := ca.NewDevCA()
	if err != nil {
		log.Fatal("failed to create CA", zap.Error(err))
	}

	spiffeGen := spiffe.NewGenerator("argus.local")
	revocationStore := revocation.NewStore()
	vaultClient := vault.NewInMemoryClient()

	// gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.TenantUnaryInterceptor()),
	)

	grpcLis, err := net.Listen("tcp", ":9081")
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		log.Info("identity gRPC server starting", zap.String("addr", ":9081"))
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// HTTP server with REST API
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// SVID creation endpoint
	mux.Handle("/api/v1/identity/svid", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			AgentID string `json:"agent_id"`
			Version string `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		spiffeID := spiffeGen.AgentID(tenantID, req.AgentID, req.Version)
		certPEM, keyPEM, err := authority.IssueCert(spiffeID, time.Hour)
		if err != nil {
			log.Error("failed to issue cert", zap.Error(err))
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue certificate")
			return
		}

		// Store key material in vault
		vaultPath := fmt.Sprintf("identity/%s/%s", tenantID, req.AgentID)
		if err := vaultClient.Store(vaultPath, map[string][]byte{
			"cert": certPEM,
			"key":  keyPEM,
		}); err != nil {
			log.Error("failed to store key material in vault", zap.String("path", vaultPath), zap.Error(err))
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"spiffe_id":  spiffeID,
			"cert_pem":   string(certPEM),
			"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
		}, tenantID)

		log.Info("issued SVID",
			zap.String("tenant_id", tenantID),
			zap.String("agent_id", req.AgentID),
			zap.String("spiffe_id", spiffeID),
		)
	})))

	// SVID validation endpoint
	mux.Handle("/api/v1/identity/validate", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			SpiffeID string `json:"spiffe_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		if revocationStore.IsRevoked(req.SpiffeID) {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"valid":   false,
				"reason":  "revoked",
			}, tenantID)
			return
		}

		// Parse the SPIFFE ID to validate tenant ownership
		parsedTenant, parsedAgent, parsedVersion, err := spiffeGen.Parse(req.SpiffeID)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"valid":  false,
				"reason": "invalid SPIFFE ID format",
			}, tenantID)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":     true,
			"tenant_id": parsedTenant,
			"agent_id":  parsedAgent,
			"version":   parsedVersion,
		}, tenantID)
	})))

	// Revocation endpoint
	mux.Handle("/api/v1/identity/revoke", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			SpiffeID string `json:"spiffe_id"`
			Reason   string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		// Verify tenant ownership of the SPIFFE ID
		if !strings.Contains(req.SpiffeID, "/tenant/"+tenantID+"/") {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "cannot revoke SVID from another tenant")
			return
		}

		revocationStore.Revoke(req.SpiffeID, req.Reason)

		writeJSON(w, http.StatusOK, map[string]bool{"revoked": true}, tenantID)

		log.Info("revoked SVID",
			zap.String("tenant_id", tenantID),
			zap.String("spiffe_id", req.SpiffeID),
			zap.String("reason", req.Reason),
		)
	})))

	// CRL endpoint
	mux.Handle("/api/v1/identity/crl", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		entries := revocationStore.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": entries,
		})
	}))

	// CA certificate endpoint (public)
	mux.HandleFunc("/api/v1/identity/ca", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Write(authority.CACertPEM())
	})

	handler := middleware.CORS(middleware.RequestLogger(log)(mux))

	httpSrv := &http.Server{
		Addr:         ":8081",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("identity HTTP server starting", zap.String("addr", ":8081"))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down identity service")
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}
	_ = cfg
}

func writeJSON(w http.ResponseWriter, status int, data interface{}, tenantID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": data,
		"meta": map[string]string{"tenant_id": tenantID},
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
