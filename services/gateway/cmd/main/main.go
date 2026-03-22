package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/argus-platform/argus/pkg/config"
	"github.com/argus-platform/argus/pkg/health"
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/pkg/middleware"
	"github.com/argus-platform/argus/services/gateway/internal/mtls"
	"github.com/argus-platform/argus/services/gateway/internal/proxy"
	"github.com/argus-platform/argus/services/gateway/internal/ratelimit"
	"github.com/argus-platform/argus/services/gateway/internal/websocket"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.Default()
	defer func() { _ = log.Sync() }()

	limiter := ratelimit.New(100, time.Minute)
	reverseProxy := proxy.New(cfg, log)

	// WebSocket proxy for real-time streams
	wsHandler := websocket.New(websocket.DefaultRoutes(), log)

	healthChecker := health.NewChecker()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthChecker.Handler())
	mux.HandleFunc("/health/live", healthChecker.LiveHandler())
	mux.HandleFunc("/health/ready", healthChecker.ReadyHandler())

	// Register WebSocket routes before the catch-all proxy.
	wsHandler.RegisterRoutes(mux)

	mux.Handle("/", reverseProxy)

	handler := middleware.Recovery(log)(
		middleware.SecurityHeaders(
			middleware.CORSWithOrigin(
				middleware.MaxBodySize(1<<20)(
					middleware.RequestID(
						middleware.RequestLogger(log)(
							limiter.Middleware(mux),
						),
					),
				),
			),
		),
	)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Configure mTLS if cert paths are provided
	caCertPath := os.Getenv("ARGUS_CA_CERT_PATH")
	serverCertPath := os.Getenv("ARGUS_SERVER_CERT_PATH")
	serverKeyPath := os.Getenv("ARGUS_SERVER_KEY_PATH")

	go func() {
		if caCertPath != "" && serverCertPath != "" && serverKeyPath != "" {
			tlsConfig, err := mtls.NewTLSConfig(&mtls.Config{
				CACertPath:     caCertPath,
				ServerCertPath: serverCertPath,
				ServerKeyPath:  serverKeyPath,
			})
			if err != nil {
				log.Fatal("failed to configure mTLS", zap.Error(err))
			}
			srv.TLSConfig = tlsConfig
			log.Info("gateway starting with mTLS", zap.String("addr", srv.Addr))
			if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Fatal("server failed", zap.Error(err))
			}
		} else {
			log.Info("gateway starting (no TLS)", zap.String("addr", srv.Addr))
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal("server failed", zap.Error(err))
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gateway")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}
}
