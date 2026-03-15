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

	// gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.TenantUnaryInterceptor()),
	)

	grpcLis, err := net.Listen("tcp", ":9083")
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		log.Info("telemetry gRPC server starting", zap.String("addr", ":9083"))
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

	handler := middleware.CORS(middleware.RequestLogger(log)(mux))

	httpSrv := &http.Server{
		Addr:         ":8083",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("telemetry HTTP server starting", zap.String("addr", ":8083"))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down telemetry service")
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpSrv.Shutdown(ctx)
	_ = cfg
}
