package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	telemetryv1 "github.com/argus-platform/argus/gen/go/telemetry"
	"github.com/argus-platform/argus/pkg/config"
	"github.com/argus-platform/argus/pkg/database"
	"github.com/argus-platform/argus/pkg/health"
	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/pkg/messaging"
	"github.com/argus-platform/argus/pkg/metrics"
	"github.com/argus-platform/argus/pkg/middleware"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/telemetry/internal/catalog"
	"github.com/argus-platform/argus/services/telemetry/internal/classifier"
	"github.com/argus-platform/argus/services/telemetry/internal/collector"
	"github.com/argus-platform/argus/services/telemetry/internal/dataquality"
	"github.com/argus-platform/argus/services/telemetry/internal/grpchandler"
	telguardrails "github.com/argus-platform/argus/services/telemetry/internal/guardrails"
	"github.com/argus-platform/argus/services/telemetry/internal/pii"
	"github.com/argus-platform/argus/services/telemetry/internal/predictor"
	telrepo "github.com/argus-platform/argus/services/telemetry/internal/repository"
	"github.com/argus-platform/argus/services/telemetry/internal/residency"
	"github.com/argus-platform/argus/services/telemetry/internal/storage"
	"github.com/nats-io/nats.go"
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

	spanCollector := collector.New(log)
	scrubber := pii.New()
	storageBackend := storage.NewInMemoryBackend()
	predictorClient := predictor.NewClient("localhost:8090")

	// Initialize PostgreSQL persistence if DSN is configured
	var dbPool *database.Pool
	var spanRepo *telrepo.SpanRepository
	if dsn := os.Getenv("ARGUS_DB_DSN"); dsn != "" {
		ctx := context.Background()
		pool, err := database.NewPool(ctx, dsn)
		if err != nil {
			log.Warn("failed to connect to database, using in-memory stores", zap.Error(err))
		} else {
			dbPool = pool
			spanRepo = telrepo.NewSpanRepository(pool)
			log.Info("PostgreSQL persistence enabled for telemetry")
			defer pool.Close()
		}
	} else {
		log.Info("no ARGUS_DB_DSN set, using in-memory stores")
	}

	// Initialize telemetry pipeline modules
	catalogDiscoverer := catalog.NewDiscoverer()
	residencyProver := residency.NewProver(os.Getenv("ARGUS_RESIDENCY_SIGNING_KEY"))
	dqValidator := dataquality.NewValidator()
	dqScorer := dataquality.NewScorer(dqValidator, 5*time.Minute)
	guardrailEngine := telguardrails.NewEngine(nil)

	// spanDataTracker accumulates recent span data per agent for quality scoring.
	spanDataTracker := newSpanDataTracker(1000)

	// Wire auto-quarantine callback
	predictorClient.SetQuarantineCallback(func(agentID, tenantID string) {
		log.Warn("auto-quarantine triggered",
			zap.String("agent_id", agentID),
			zap.String("tenant_id", tenantID),
		)
		// Call orchestrator to quarantine agent
		quarantineURL := fmt.Sprintf("http://localhost:8082/api/v1/agents/%s/quarantine", agentID)
		req, err := http.NewRequest(http.MethodPost, quarantineURL, nil)
		if err != nil {
			log.Error("failed to create quarantine request", zap.Error(err))
			return
		}
		req.Header.Set("X-Tenant-ID", tenantID)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Error("failed to send quarantine request", zap.Error(err))
			return
		}
		resp.Body.Close()
	})

	// Connect to NATS JetStream for span ingestion from sidecars
	var natsConn *messaging.Conn
	if natsURL := os.Getenv("ARGUS_NATS_URL"); natsURL != "" {
		conn, err := messaging.Connect(natsURL)
		if err != nil {
			log.Warn("failed to connect to NATS, spans will only be accepted via HTTP/gRPC", zap.Error(err))
		} else {
			natsConn = conn
			defer conn.Close()

			// Ensure telemetry stream exists
			if _, err := conn.EnsureStream(messaging.DefaultTelemetryStream()); err != nil {
				log.Error("failed to ensure NATS telemetry stream", zap.Error(err))
			}

			// Subscribe to all tenant telemetry spans
			sub := messaging.NewSubscriber(conn, log)
			natsCtx, natsCancel := context.WithCancel(context.Background())
			defer natsCancel()

			err = sub.SubscribeAll(natsCtx, "telemetry-collector", func(msg *nats.Msg) error {
				var span collector.Span
				if err := json.Unmarshal(msg.Data, &span); err != nil {
					log.Error("failed to unmarshal NATS span", zap.Error(err))
					return err
				}

				// Classify, scrub PII, and store
				classified := classifier.ClassifyAttributes(span.Attributes)
				for tier, attrs := range classified {
					if tier >= classifier.TierSensitive {
						classified[tier] = scrubber.ScrubMap(attrs)
					}
					if err := storageBackend.Store(span.TenantID, tier, attrs); err != nil {
						log.Error("failed to store telemetry from NATS", zap.String("tenant", span.TenantID), zap.Error(err))
					}
				}

				spanCollector.Ingest([]*collector.Span{&span})

				// Persist to DB if available
				if spanRepo != nil {
					ctx := context.Background()
					if err := spanRepo.Store(ctx, span.TenantID, []*collector.Span{&span}); err != nil {
						log.Error("failed to persist NATS span to DB", zap.Error(err))
					}
				}

				return nil
			})
			if err != nil {
				log.Error("failed to subscribe to NATS telemetry", zap.Error(err))
			} else {
				log.Info("subscribed to NATS telemetry stream for span ingestion")
			}
		}
	}

	// gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.TenantUnaryInterceptor()),
	)

	// Register gRPC service handlers
	telemetryGRPC := grpchandler.NewTelemetryHandler(spanCollector)
	telemetryv1.RegisterTelemetryServiceServer(grpcServer, telemetryGRPC)

	alertStore := grpchandler.NewInMemoryAlertStore()
	predictorGRPC := grpchandler.NewPredictorHandler(predictorClient, alertStore)
	telemetryv1.RegisterPredictorServiceServer(grpcServer, predictorGRPC)

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

	// Health checks with dependency verification
	healthChecker := health.NewChecker()
	if dbPool != nil {
		healthChecker.AddCheck("postgres", func(ctx context.Context) error {
			return dbPool.Ping(ctx)
		})
	}
	if natsConn != nil {
		healthChecker.AddCheck("nats", func(ctx context.Context) error {
			if !natsConn.NatsConn().IsConnected() {
				return fmt.Errorf("NATS disconnected")
			}
			return nil
		})
	}

	// Metrics registry
	metricsReg := metrics.NewRegistry()

	// HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthChecker.Handler())
	mux.HandleFunc("/health/live", healthChecker.LiveHandler())
	mux.HandleFunc("/health/ready", healthChecker.ReadyHandler())
	mux.HandleFunc("/metrics", metricsReg.Handler())

	// Span ingestion endpoint
	mux.Handle("/api/v1/telemetry/spans", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		switch r.Method {
		case http.MethodPost:
			var req struct {
				Spans []*collector.Span `json:"spans"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
				return
			}

			// Set tenant ID, classify, scrub, discover, validate quality, and check guardrails for each span
			for _, span := range req.Spans {
				span.TenantID = tenantID

				// Auto-discover data sources from span
				catalogDiscoverer.DiscoverFromSpan(tenantID, span.AgentID, span.OperationName, span.Attributes)

				// Run data quality validation on span attributes
				attrMap := toInterfaceMap(span.Attributes)
				attrMap["operation_name"] = span.OperationName
				attrMap["duration_ms"] = span.DurationMs
				dqResults := dqValidator.Validate(span.AgentID, attrMap)
				for _, result := range dqResults {
					if !result.Passed {
						log.Warn("data quality violation",
							zap.String("agent_id", span.AgentID),
							zap.String("span_id", span.SpanID),
							zap.String("rule", result.RuleName),
							zap.String("message", result.Message),
							zap.String("severity", result.Severity),
						)
					}
				}

				// Track span data for quality scoring
				spanDataTracker.Track(span.AgentID, attrMap, span.StartedAt)

				// Run guardrail checks on span content
				if content, ok := span.Attributes["input"]; ok {
					result := guardrailEngine.Check(tenantID, span.AgentID, span.SpanID, content)
					if !result.Passed {
						log.Warn("guardrail violation detected",
							zap.String("agent_id", span.AgentID),
							zap.String("span_id", span.SpanID),
							zap.Int("violations", len(result.Violations)),
						)
					}
				}

				// Classify attributes
				classified := classifier.ClassifyAttributes(span.Attributes)

				// Scrub PII from sensitive tier attributes
				for tier, attrs := range classified {
					if tier >= classifier.TierSensitive {
						classified[tier] = scrubber.ScrubMap(attrs)
					}
					if err := storageBackend.Store(tenantID, tier, attrs); err != nil {
						log.Error("failed to store telemetry", zap.String("tenant", tenantID), zap.Error(err))
					}

					// Create residency attestation for stored data
					nodeID := os.Getenv("ARGUS_NODE_ID")
					region := os.Getenv("ARGUS_REGION")
					if nodeID != "" && region != "" {
						residencyProver.Attest(tenantID, nodeID, region, []byte(fmt.Sprintf("%v", attrs)))
					}
				}
			}

			accepted := spanCollector.Ingest(req.Spans)

			// Persist to PostgreSQL if available
			if spanRepo != nil {
				if err := spanRepo.Store(r.Context(), tenantID, req.Spans); err != nil {
					log.Error("failed to persist spans to DB", zap.Error(err))
				}
			}

			httputil.WriteJSON(w, http.StatusOK, map[string]int{"accepted": accepted}, tenantID)

		case http.MethodGet:
			agentID := r.URL.Query().Get("agent_id")
			traceID := r.URL.Query().Get("trace_id")
			spans := spanCollector.Query(tenantID, agentID, traceID, 100)
			httputil.WriteJSON(w, http.StatusOK, spans, tenantID)

		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
	})))

	// Metrics ingestion endpoint
	mux.Handle("/api/v1/telemetry/metrics", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		if r.Method != http.MethodPost {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
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
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		httputil.WriteJSON(w, http.StatusOK, map[string]int{"accepted": len(req.Metrics)}, tenantID)
	})))

	// Prediction endpoint
	mux.Handle("/api/v1/telemetry/predict", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		if r.Method != http.MethodPost {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var features predictor.Features
		if err := json.NewDecoder(r.Body).Decode(&features); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		prediction, err := predictorClient.Predict(&features)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "PREDICTION_ERROR", err.Error())
			return
		}

		httputil.WriteJSON(w, http.StatusOK, prediction, tenantID)
	})))

	// Data catalog endpoint
	mux.Handle("/api/v1/telemetry/catalog", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		entries := catalogDiscoverer.ListSources(tenantID)
		httputil.WriteJSON(w, http.StatusOK, entries, tenantID)
	})))

	// Data residency proof endpoint
	mux.Handle("/api/v1/telemetry/residency/proof", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		allowedRegions := r.URL.Query()["region"]
		if len(allowedRegions) == 0 {
			allowedRegions = []string{os.Getenv("ARGUS_REGION")}
		}
		proof := residencyProver.GenerateProof(tenantID, allowedRegions)
		httputil.WriteJSON(w, http.StatusOK, proof, tenantID)
	})))

	// Data quality score endpoint
	mux.Handle("/api/v1/telemetry/quality", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		agentID := r.URL.Query().Get("agent_id")
		if agentID == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "agent_id is required")
			return
		}
		records, timestamps := spanDataTracker.Get(agentID)
		score := dqScorer.Score(agentID, records, nil, timestamps)
		httputil.WriteJSON(w, http.StatusOK, score, tenantID)
	})))

	handler := middleware.Recovery(log)(
		middleware.SecurityHeaders(
			middleware.CORSWithOrigin(
				middleware.MaxBodySize(1 << 20)(
					middleware.RequestID(
						metrics.HTTPMiddleware(metricsReg, "telemetry")(
							middleware.RequestLogger(log)(mux),
						),
					),
				),
			),
		),
	)

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
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}
	_ = cfg
}

// toInterfaceMap converts map[string]string to map[string]interface{} for the
// data quality validator which operates on generic maps.
func toInterfaceMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// spanDataTracker accumulates recent span data per agent for quality scoring.
type spanDataTracker struct {
	mu          sync.RWMutex
	maxPerAgent int
	records     map[string][]map[string]interface{}
	timestamps  map[string][]time.Time
}

func newSpanDataTracker(maxPerAgent int) *spanDataTracker {
	return &spanDataTracker{
		maxPerAgent: maxPerAgent,
		records:     make(map[string][]map[string]interface{}),
		timestamps:  make(map[string][]time.Time),
	}
}

func (t *spanDataTracker) Track(agentID string, data map[string]interface{}, ts time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records[agentID] = append(t.records[agentID], data)
	t.timestamps[agentID] = append(t.timestamps[agentID], ts)

	// Evict oldest entries if over limit.
	if len(t.records[agentID]) > t.maxPerAgent {
		t.records[agentID] = t.records[agentID][len(t.records[agentID])-t.maxPerAgent:]
		t.timestamps[agentID] = t.timestamps[agentID][len(t.timestamps[agentID])-t.maxPerAgent:]
	}
}

func (t *spanDataTracker) Get(agentID string) ([]map[string]interface{}, []time.Time) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.records[agentID], t.timestamps[agentID]
}
