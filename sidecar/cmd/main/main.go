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
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/sidecar/internal/discovery"
	"github.com/argus-platform/argus/sidecar/internal/identity"
	"github.com/argus-platform/argus/sidecar/internal/proxy"
	"github.com/argus-platform/argus/sidecar/internal/telemetry"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.Default()
	defer log.Sync()

	// Initialize components
	disc := discovery.New(log)
	identMgr := identity.NewManager(log)
	telEmitter := telemetry.NewEmitter(log, cfg.NATSUrl)
	transparentProxy := proxy.New(log)

	_ = disc
	_ = identMgr
	_ = telEmitter

	// HTTP server for health + proxy
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/", transparentProxy)

	srv := &http.Server{
		Addr:         ":8085",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Info("sidecar starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("sidecar failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down sidecar")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
