package main

import (
	"context"
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
	"github.com/argus-platform/argus/services/identity/internal/ca"
	"github.com/argus-platform/argus/services/identity/internal/spiffe"
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
	_ = spiffeGen
	_ = authority

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

	// HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	httpSrv := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
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
	httpSrv.Shutdown(ctx)
	_ = cfg
}
