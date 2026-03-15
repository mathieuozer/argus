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
	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
	"github.com/argus-platform/argus/services/orchestrator/internal/router"
	"github.com/argus-platform/argus/services/orchestrator/internal/statemachine"
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

	agentRegistry := registry.New()
	taskRouter := router.New(agentRegistry)
	_ = statemachine.New()
	_ = taskRouter

	// gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.TenantUnaryInterceptor()),
	)

	grpcLis, err := net.Listen("tcp", ":9082")
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		log.Info("orchestrator gRPC server starting", zap.String("addr", ":9082"))
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

	mux.Handle("/api/v1/agents", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		switch r.Method {
		case http.MethodGet:
			agents := agentRegistry.List(tenantID)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": agents,
				"meta": map[string]string{"tenant_id": tenantID},
			})
		case http.MethodPost:
			var req registry.RegisterRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":{"code":"VALIDATION_ERROR","message":"invalid request body"}}`, http.StatusBadRequest)
				return
			}
			agent := agentRegistry.Register(tenantID, &req)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": agent,
				"meta": map[string]string{"tenant_id": tenantID},
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	handler := middleware.CORS(middleware.RequestLogger(log)(mux))

	httpSrv := &http.Server{
		Addr:         ":8082",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("orchestrator HTTP server starting", zap.String("addr", ":8082"))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down orchestrator")
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpSrv.Shutdown(ctx)
	_ = cfg
}
