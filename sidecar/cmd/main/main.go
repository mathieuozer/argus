package main

import (
	"context"
	"encoding/json"
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

	// Read agent config from env
	agentID := getEnv("ARGUS_AGENT_ID", "unknown-agent")
	agentVersion := getEnv("ARGUS_AGENT_VERSION", "0.0.1")
	agentFramework := getEnv("ARGUS_AGENT_FRAMEWORK", "custom")

	// Initialize components
	disc := discovery.New(log)
	identMgr := identity.NewManager(log)
	telEmitter := telemetry.NewEmitter(log, cfg.NATSUrl)
	transparentProxy := proxy.New(log)

	// Auto-register with orchestrator
	go func() {
		for i := 0; i < 5; i++ {
			err := disc.Register(agentID, agentVersion, agentFramework, []string{})
			if err == nil {
				break
			}
			log.Warn("registration attempt failed, retrying...",
				zap.Int("attempt", i+1),
				zap.Error(err),
			)
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
		}

		if disc.IsRegistered() {
			disc.StartHeartbeatLoop(30 * time.Second)

			// Request SVID
			if err := identMgr.RequestSVID(getEnv("ARGUS_TENANT_ID", "default"), agentID, agentVersion); err != nil {
				log.Warn("failed to request SVID", zap.Error(err))
			}
		}
	}()

	// HTTP server for health + proxy + status
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "ok",
			"registered": disc.IsRegistered(),
			"agent_id":   agentID,
			"spiffe_id":  identMgr.GetSpiffeID(),
		})
	})

	// Status endpoint with detailed info
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agent_id":   agentID,
			"version":    agentVersion,
			"framework":  agentFramework,
			"registered": disc.IsRegistered(),
			"spiffe_id":  identMgr.GetSpiffeID(),
			"proxy":      transparentProxy.Stats(),
		})
	})

	// All other traffic goes through the transparent proxy
	mux.Handle("/", transparentProxy)

	srv := &http.Server{
		Addr:         ":8085",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Info("sidecar starting",
			zap.String("addr", srv.Addr),
			zap.String("agent_id", agentID),
			zap.String("version", agentVersion),
		)
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
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	_ = telEmitter
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
