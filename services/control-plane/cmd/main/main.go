package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/argus-platform/argus/pkg/config"
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/pkg/middleware"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/control-plane/internal/audit"
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

	auditLog := audit.NewWriter()

	// gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.TenantUnaryInterceptor()),
	)

	grpcLis, err := net.Listen("tcp", ":9084")
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		log.Info("control-plane gRPC server starting", zap.String("addr", ":9084"))
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// HTTP REST API
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Dashboard API - agents list (proxied from orchestrator in prod, stub here)
	mux.Handle("/api/v1/agents", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
			"meta": map[string]string{"tenant_id": tenantID},
		})
	})))

	// Dashboard API - metrics
	mux.Handle("/api/v1/metrics", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"total_agents": 0,
				"active_tasks": 0,
				"total_cost":   0.0,
				"alert_count":  0,
			},
			"meta": map[string]string{"tenant_id": tenantID},
		})
	})))

	// Dashboard API - alerts
	mux.Handle("/api/v1/alerts", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
			"meta": map[string]string{"tenant_id": tenantID},
		})
	})))

	// Audit log endpoint
	mux.Handle("/api/v1/audit", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		entries := auditLog.List(tenantID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": entries,
			"meta": map[string]string{"tenant_id": tenantID},
		})
	})))

	handler := middleware.CORS(middleware.RequestLogger(log)(mux))

	httpSrv := &http.Server{
		Addr:         ":8084",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("control-plane HTTP server starting", zap.String("addr", ":8084"))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down control-plane")
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpSrv.Shutdown(ctx)
	_ = cfg
}
