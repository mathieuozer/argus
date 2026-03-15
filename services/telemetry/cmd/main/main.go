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
	"github.com/argus-platform/argus/services/telemetry/internal/classifier"
	"github.com/argus-platform/argus/services/telemetry/internal/collector"
	"github.com/argus-platform/argus/services/telemetry/internal/pii"
	"github.com/argus-platform/argus/services/telemetry/internal/predictor"
	"github.com/argus-platform/argus/services/telemetry/internal/storage"
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

	spanCollector := collector.New(log)
	scrubber := pii.New()
	storageBackend := storage.NewInMemoryBackend()
	predictorClient := predictor.NewClient("localhost:8090")

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

	// Span ingestion endpoint
	mux.Handle("/api/v1/telemetry/spans", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		switch r.Method {
		case http.MethodPost:
			var req struct {
				Spans []*collector.Span `json:"spans"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
				return
			}

			// Set tenant ID, classify and scrub each span
			for _, span := range req.Spans {
				span.TenantID = tenantID

				// Classify attributes
				classified := classifier.ClassifyAttributes(span.Attributes)

				// Scrub PII from sensitive tier attributes
				for tier, attrs := range classified {
					if tier >= classifier.TierSensitive {
						classified[tier] = scrubber.ScrubMap(attrs)
					}
					storageBackend.Store(tenantID, tier, attrs)
				}
			}

			accepted := spanCollector.Ingest(req.Spans)
			writeJSON(w, http.StatusOK, map[string]int{"accepted": accepted}, tenantID)

		case http.MethodGet:
			agentID := r.URL.Query().Get("agent_id")
			traceID := r.URL.Query().Get("trace_id")
			spans := spanCollector.Query(tenantID, agentID, traceID, 100)
			writeJSON(w, http.StatusOK, spans, tenantID)

		default:
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
	})))

	// Metrics ingestion endpoint
	mux.Handle("/api/v1/telemetry/metrics", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			Metrics []struct {
				Name   string            `json:"name"`
				Value  float64           `json:"value"`
				Labels map[string]string `json:"labels"`
			} `json:"metrics"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		writeJSON(w, http.StatusOK, map[string]int{"accepted": len(req.Metrics)}, tenantID)
	})))

	// Prediction endpoint
	mux.Handle("/api/v1/telemetry/predict", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var features predictor.Features
		if err := json.NewDecoder(r.Body).Decode(&features); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		prediction, err := predictorClient.Predict(&features)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "PREDICTION_ERROR", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, prediction, tenantID)
	})))

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
