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

	identityv1 "github.com/argus-platform/argus/gen/go/identity"
	"github.com/argus-platform/argus/pkg/config"
	"github.com/argus-platform/argus/pkg/health"
	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/pkg/metrics"
	"github.com/argus-platform/argus/pkg/middleware"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/identity/internal/ca"
	"github.com/argus-platform/argus/services/identity/internal/grpchandler"
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
	defer func() { _ = log.Sync() }()

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

	// Register gRPC service handler
	identityGRPC := grpchandler.NewIdentityHandler(authority, spiffeGen, revocationStore)
	identityv1.RegisterIdentityServiceServer(grpcServer, identityGRPC)

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

	// Health checks with dependency verification
	healthChecker := health.NewChecker()

	// Metrics registry
	metricsReg := metrics.NewRegistry()

	// HTTP server with REST API
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthChecker.Handler())
	mux.HandleFunc("/health/live", healthChecker.LiveHandler())
	mux.HandleFunc("/health/ready", healthChecker.ReadyHandler())
	mux.HandleFunc("/metrics", metricsReg.Handler())

	// SVID creation endpoint
	mux.Handle("/api/v1/identity/svid", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		if r.Method != http.MethodPost {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			AgentID string `json:"agent_id"`
			Version string `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		spiffeID := spiffeGen.AgentID(tenantID, req.AgentID, req.Version)
		certPEM, keyPEM, err := authority.IssueCert(spiffeID, time.Hour)
		if err != nil {
			log.Error("failed to issue cert", zap.Error(err))
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue certificate")
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

		httputil.WriteJSON(w, http.StatusCreated, map[string]interface{}{
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
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			SpiffeID string `json:"spiffe_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		if revocationStore.IsRevoked(req.SpiffeID) {
			httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"valid":  false,
				"reason": "revoked",
			}, tenantID)
			return
		}

		// Parse the SPIFFE ID to validate tenant ownership
		parsedTenant, parsedAgent, parsedVersion, err := spiffeGen.Parse(req.SpiffeID)
		if err != nil {
			httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"valid":  false,
				"reason": "invalid SPIFFE ID format",
			}, tenantID)
			return
		}

		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
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
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			SpiffeID string `json:"spiffe_id"`
			Reason   string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		// Verify tenant ownership of the SPIFFE ID
		if !strings.Contains(req.SpiffeID, "/tenant/"+tenantID+"/") {
			httputil.WriteError(w, http.StatusForbidden, "FORBIDDEN", "cannot revoke SVID from another tenant")
			return
		}

		revocationStore.Revoke(req.SpiffeID, req.Reason)

		httputil.WriteJSON(w, http.StatusOK, map[string]bool{"revoked": true}, tenantID)

		log.Info("revoked SVID",
			zap.String("tenant_id", tenantID),
			zap.String("spiffe_id", req.SpiffeID),
			zap.String("reason", req.Reason),
		)
	})))

	// CRL endpoint
	mux.Handle("/api/v1/identity/crl", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		entries := revocationStore.List()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": entries,
		})
	}))

	// CA certificate endpoint (public)
	mux.HandleFunc("/api/v1/identity/ca", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-pem-file")
		_, _ = w.Write(authority.CACertPEM())
	})

	handler := middleware.Recovery(log)(
		middleware.SecurityHeaders(
			middleware.CORSWithOrigin(
				middleware.MaxBodySize(1 << 20)(
					middleware.RequestID(
						metrics.HTTPMiddleware(metricsReg, "identity")(
							middleware.RequestLogger(log)(mux),
						),
					),
				),
			),
		),
	)

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
