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

	"github.com/argus-platform/argus/pkg/config"
	"github.com/argus-platform/argus/pkg/database"
	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/pkg/middleware"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/control-plane/internal/alerts"
	"github.com/argus-platform/argus/services/control-plane/internal/audit"
	"github.com/argus-platform/argus/services/control-plane/internal/auth"
	"github.com/argus-platform/argus/services/control-plane/internal/catalog"
	"github.com/argus-platform/argus/services/control-plane/internal/compliance"
	"github.com/argus-platform/argus/services/control-plane/internal/costgov"
	"github.com/argus-platform/argus/services/control-plane/internal/dashboard"
	"github.com/argus-platform/argus/services/control-plane/internal/dataquality"
	"github.com/argus-platform/argus/services/control-plane/internal/eval"
	"github.com/argus-platform/argus/services/control-plane/internal/feedback"
	"github.com/argus-platform/argus/services/control-plane/internal/guardrails"
	"github.com/argus-platform/argus/services/control-plane/internal/policy"
	"github.com/argus-platform/argus/services/control-plane/internal/prompts"
	"github.com/argus-platform/argus/services/control-plane/internal/rag"
	"github.com/argus-platform/argus/services/control-plane/internal/slo"
	"github.com/argus-platform/argus/services/control-plane/internal/trace"
	ws "github.com/argus-platform/argus/services/control-plane/internal/websocket"
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

	// Initialize PostgreSQL persistence if ARGUS_DB_DSN is configured
	var dbPool *database.Pool
	var catalogPG *catalog.PGRepository
	var costPG *costgov.PGRepository
	var dqPG *dataquality.PGRepository
	var sloPG *slo.PGRepository
	var tracePG *trace.PGService
	var auditPG *audit.PGWriter
	var evalPG *eval.PGRepository
	var feedbackPG *feedback.PGRepository
	var guardrailsPG *guardrails.PGRepository
	var promptsPG *prompts.PGRepository
	var ragPG *rag.PGRepository
	var compliancePG *compliance.PGReportRepository

	if dsn := os.Getenv("ARGUS_DB_DSN"); dsn != "" {
		ctx := context.Background()
		pool, err := database.NewPool(ctx, dsn)
		if err != nil {
			log.Warn("failed to connect to database, using in-memory stores", zap.Error(err))
		} else {
			dbPool = pool
			catalogPG = catalog.NewPGRepository(pool)
			costPG = costgov.NewPGRepository(pool)
			dqPG = dataquality.NewPGRepository(pool)
			sloPG = slo.NewPGRepository(pool)
			tracePG = trace.NewPGService(pool)
			auditPG = audit.NewPGWriter(pool)
			evalPG = eval.NewPGRepository(pool)
			feedbackPG = feedback.NewPGRepository(pool)
			guardrailsPG = guardrails.NewPGRepository(pool)
			promptsPG = prompts.NewPGRepository(pool)
			ragPG = rag.NewPGRepository(pool)
			compliancePG = compliance.NewPGReportRepository(pool)
			log.Info("PostgreSQL persistence enabled for control-plane")
		}
	} else {
		log.Info("no ARGUS_DB_DSN set, using in-memory stores")
	}

	// Suppress unused variable warnings for PG repositories.
	// These are available for dual-write persistence in request handlers
	// and for future migration to DB-primary reads.
	_ = catalogPG
	_ = costPG
	_ = dqPG
	_ = sloPG
	_ = tracePG
	_ = auditPG
	_ = evalPG
	_ = feedbackPG
	_ = guardrailsPG
	_ = promptsPG
	_ = ragPG
	_ = compliancePG

	// Initialize components
	auditLog := audit.NewWriter()
	alertRouter := alerts.NewRouter()
	policyEngine := policy.New()
	jwtAuth := auth.New(os.Getenv("ARGUS_JWT_SECRET"))
	dashHandler := dashboard.New(alertRouter, auditLog)

	// Initialize observability module handlers
	traceSvc := trace.NewService()
	traceHandler := trace.NewHandler(traceSvc)

	dqRepo := dataquality.NewRepository()
	dqHandler := dataquality.NewHandler(dqRepo)

	catalogRepo := catalog.NewRepository()
	catalogHandler := catalog.NewHandler(catalogRepo)

	costRepo := costgov.NewRepository()
	costDetector := costgov.NewAnomalyDetector()
	costHandler := costgov.NewHandler(costRepo, costDetector)

	sloRepo := slo.NewRepository()
	sloCalc := slo.NewCalculator(sloRepo)
	sloHandler := slo.NewHandler(sloRepo, sloCalc)

	auditHandler := audit.NewHandler(auditLog)

	// Initialize feature module handlers
	evalRepo := eval.NewRepository()
	evalHandler := eval.NewHandler(evalRepo)

	feedbackRepo := feedback.NewRepository()
	feedbackHandler := feedback.NewHandler(feedbackRepo)

	guardrailsRepo := guardrails.NewRepository()
	guardrailsHandler := guardrails.NewHandler(guardrailsRepo)

	promptsRepo := prompts.NewRepository()
	promptsHandler := prompts.NewHandler(promptsRepo)

	ragRepo := rag.NewRepository()
	ragHandler := rag.NewHandler(ragRepo)

	complianceRepo := compliance.NewReportRepository()
	complianceHandler := compliance.NewReportHandler(complianceRepo)

	// Seed demo data for dashboard (non-production only)
	if os.Getenv("ARGUS_ENV") != "production" {
		seedControlPlaneDemo(alertRouter, auditLog, traceSvc, sloRepo, costRepo, log)
	}

	// WebSocket stream hub for real-time agent and telemetry events
	wsHub := ws.NewHub(log)

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
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Dashboard endpoints (alerts, audit)
	dashHandler.RegisterRoutes(mux)

	// Observability module routes
	traceHandler.RegisterRoutes(mux)
	dqHandler.RegisterRoutes(mux)
	catalogHandler.RegisterRoutes(mux)
	costHandler.RegisterRoutes(mux)
	sloHandler.RegisterRoutes(mux)
	auditHandler.RegisterRoutes(mux)

	// Feature module routes
	evalHandler.RegisterRoutes(mux)
	feedbackHandler.RegisterRoutes(mux)
	guardrailsHandler.RegisterRoutes(mux)
	promptsHandler.RegisterRoutes(mux)
	ragHandler.RegisterRoutes(mux)
	complianceHandler.RegisterRoutes(mux)

	// WebSocket stream routes (real-time agent and telemetry events)
	wsHub.RegisterRoutes(mux)

	// Auth endpoint - generate tokens (dev mode)
	mux.HandleFunc("/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httputil.WriteError(w,http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			Subject  string `json:"subject"`
			TenantID string `json:"tenant_id"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w,http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		claims := &auth.Claims{
			Sub:      req.Subject,
			TenantID: req.TenantID,
			Role:     auth.Role(req.Role),
			Iat:      time.Now().Unix(),
			Exp:      time.Now().Add(24 * time.Hour).Unix(),
		}

		token, err := jwtAuth.GenerateToken(claims)
		if err != nil {
			httputil.WriteError(w,http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
			return
		}

		auditLog.Write(req.TenantID, req.Subject, "generate_token", "auth/token", "role: "+req.Role)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"token":      token,
				"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
		})
	})

	// Login endpoint for the dashboard (dev mode: accepts any username/password)
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httputil.WriteError(w,http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			TenantID string `json:"tenantId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w,http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		if req.Username == "" {
			httputil.WriteError(w,http.StatusBadRequest, "VALIDATION_ERROR", "username is required")
			return
		}
		if req.TenantID == "" {
			req.TenantID = "default"
		}

		role := auth.RoleAdmin
		claims := &auth.Claims{
			Sub:      req.Username,
			TenantID: req.TenantID,
			Role:     role,
			Iat:      time.Now().Unix(),
			Exp:      time.Now().Add(24 * time.Hour).Unix(),
		}

		token, err := jwtAuth.GenerateToken(claims)
		if err != nil {
			httputil.WriteError(w,http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
			return
		}

		refreshClaims := &auth.Claims{
			Sub:      req.Username,
			TenantID: req.TenantID,
			Role:     role,
			Iat:      time.Now().Unix(),
			Exp:      time.Now().Add(7 * 24 * time.Hour).Unix(),
		}
		refreshToken, err := jwtAuth.GenerateToken(refreshClaims)
		if err != nil {
			httputil.WriteError(w,http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate refresh token")
			return
		}

		auditLog.Write(req.TenantID, req.Username, "login", "auth/login", "role: "+string(role))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"token":        token,
				"refreshToken": refreshToken,
				"user": map[string]interface{}{
					"id":         req.Username,
					"username":   req.Username,
					"email":      req.Username + "@argus.dev",
					"tenantId":   req.TenantID,
					"tenantName": req.TenantID,
					"role":       string(role),
				},
				"expiresAt": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
		})
	})

	// Token refresh endpoint for the dashboard
	mux.HandleFunc("/api/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httputil.WriteError(w,http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w,http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		oldClaims, err := jwtAuth.ValidateToken(req.RefreshToken)
		if err != nil {
			httputil.WriteError(w,http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
			return
		}

		claims := &auth.Claims{
			Sub:      oldClaims.Sub,
			TenantID: oldClaims.TenantID,
			Role:     oldClaims.Role,
			Iat:      time.Now().Unix(),
			Exp:      time.Now().Add(24 * time.Hour).Unix(),
		}

		token, err := jwtAuth.GenerateToken(claims)
		if err != nil {
			httputil.WriteError(w,http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
			return
		}

		refreshClaims := &auth.Claims{
			Sub:      oldClaims.Sub,
			TenantID: oldClaims.TenantID,
			Role:     oldClaims.Role,
			Iat:      time.Now().Unix(),
			Exp:      time.Now().Add(7 * 24 * time.Hour).Unix(),
		}
		newRefresh, err := jwtAuth.GenerateToken(refreshClaims)
		if err != nil {
			httputil.WriteError(w,http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate refresh token")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"token":        token,
				"refreshToken": newRefresh,
				"user": map[string]interface{}{
					"id":         oldClaims.Sub,
					"username":   oldClaims.Sub,
					"email":      oldClaims.Sub + "@argus.dev",
					"tenantId":   oldClaims.TenantID,
					"tenantName": oldClaims.TenantID,
					"role":       string(oldClaims.Role),
				},
				"expiresAt": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
		})
	})

	// Policy endpoints
	mux.Handle("/api/v1/policies", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		switch r.Method {
		case http.MethodGet:
			rules := policyEngine.ListRules(tenantID)
			httputil.WriteJSON(w,http.StatusOK, rules, tenantID)
		case http.MethodPost:
			var rule policy.Rule
			if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
				httputil.WriteError(w,http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
				return
			}
			policyEngine.AddRule(tenantID, &rule)
			auditLog.Write(tenantID, "system", "create_policy", "policy/"+rule.ID, "")
			httputil.WriteJSON(w,http.StatusCreated, rule, tenantID)
		default:
			httputil.WriteError(w,http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
	})))

	// Policy evaluation endpoint
	mux.Handle("/api/v1/policies/evaluate", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		if r.Method != http.MethodPost {
			httputil.WriteError(w,http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			Subject  string `json:"subject"`
			Action   string `json:"action"`
			Resource string `json:"resource"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w,http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		allowed, err := policyEngine.Evaluate(tenantID, req.Subject, policy.Action(req.Action), req.Resource)
		if err != nil {
			httputil.WriteError(w,http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		httputil.WriteJSON(w,http.StatusOK, map[string]bool{"allowed": allowed}, tenantID)
	})))

	// Metrics endpoint
	mux.Handle("/api/v1/metrics", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"total_agents": 0,
			"active_tasks": 0,
			"total_cost":   0.0,
			"alert_count":  alertRouter.Count(tenantID),
		}, tenantID)
	})))


	// Wrap with auth + tenant + logging middleware.
	// TenantHTTP extracts X-Tenant-ID header into tenancy context.
	// It is applied after auth so that all API routes have tenant context.
	// /health and /api/v1/auth/token are excluded (they don't need tenant).
	tenantMw := tenantMiddlewareWithExclusions(
		middleware.TenantHTTP,
		"/health",
		"/api/v1/auth/",
	)
	handler := middleware.CORS(
		jwtAuth.Middleware(
			tenantMw(
				middleware.RequestLogger(log)(mux),
			),
		),
	)

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
	wsHub.Shutdown()
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Close database pool on shutdown
	if dbPool != nil {
		dbPool.Close()
		log.Info("PostgreSQL connection pool closed")
	}

	_ = cfg
}

// tenantMiddlewareWithExclusions wraps a tenant middleware so that certain
// path prefixes bypass it (e.g., /health, /api/v1/auth/token).
func tenantMiddlewareWithExclusions(mw func(http.Handler) http.Handler, excludedPrefixes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		wrapped := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, prefix := range excludedPrefixes {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}
			wrapped.ServeHTTP(w, r)
		})
	}
}

func seedControlPlaneDemo(
	alertRouter *alerts.Router,
	auditLog *audit.Writer,
	traceSvc *trace.Service,
	sloRepo *slo.Repository,
	costRepo *costgov.Repository,
	log *zap.Logger,
) {
	tid := "default"

	// Seed alerts
	alertRouter.Fire(tid, "budget-reconciler", alerts.SeverityWarning, "Latency spike detected", "p99 latency exceeded 3x p50 for 45s")
	alertRouter.Fire(tid, "ticket-classifier", alerts.SeverityCritical, "Token escalation pattern", "Agent stuck in retry loop, context window 92% full")
	a3 := alertRouter.Fire(tid, "doc-summarizer", alerts.SeverityInfo, "New agent version available", "v2.1.0 released with performance improvements")
	_, _ = alertRouter.UpdateStatus(tid, a3.ID, "resolved")

	// Seed audit log entries
	auditLog.Write(tid, "admin", "login", "auth/login", "role: admin")
	auditLog.Write(tid, "admin", "create_policy", "policy/deny-external", "block external API calls")
	auditLog.Write(tid, "system", "quarantine_agent", "agent/ticket-classifier", "auto-quarantine: failure probability 0.94")
	auditLog.Write(tid, "admin", "create_slo", "slo/api-availability", "99.9% availability target")

	// Seed traces
	now := time.Now()
	traces := []struct {
		traceID string
		agentID string
		spans   []struct {
			op       string
			dur      int64
			parent   string
			errCode  *string
		}
	}{
		{
			traceID: "trace-001",
			agentID: "budget-reconciler",
			spans: []struct {
				op      string
				dur     int64
				parent  string
				errCode *string
			}{
				{"reconcile_budget", 1250, "", nil},
				{"fetch_transactions", 450, "span-001-0", nil},
				{"compute_totals", 320, "span-001-0", nil},
				{"write_report", 180, "span-001-0", nil},
			},
		},
		{
			traceID: "trace-002",
			agentID: "doc-summarizer",
			spans: []struct {
				op      string
				dur     int64
				parent  string
				errCode *string
			}{
				{"summarize_document", 2100, "", nil},
				{"extract_text", 800, "span-002-0", nil},
				{"generate_summary", 1100, "span-002-0", nil},
			},
		},
		{
			traceID: "trace-003",
			agentID: "ticket-classifier",
			spans: []struct {
				op      string
				dur     int64
				parent  string
				errCode *string
			}{
				{"classify_ticket", 3500, "", strPtr("TIMEOUT")},
				{"fetch_ticket", 200, "span-003-0", nil},
				{"run_classifier", 3000, "span-003-0", strPtr("CONTEXT_OVERFLOW")},
			},
		},
	}

	for _, t := range traces {
		for i, sp := range t.spans {
			spanID := fmt.Sprintf("span-%s-%d", t.traceID[len(t.traceID)-3:], i)
			parentID := sp.parent
			traceSvc.IngestSpan(&trace.Span{
				SpanID:        spanID,
				TraceID:       t.traceID,
				ParentSpanID:  parentID,
				TenantID:      tid,
				AgentID:       t.agentID,
				TaskID:        "demo-task",
				OperationName: sp.op,
				StartedAt:     now.Add(-time.Duration(i) * time.Minute),
				DurationMs:    sp.dur,
				Attributes:    map[string]string{"demo": "true"},
				ErrorCode:     sp.errCode,
			})
		}
	}

	// Seed SLOs with measurements
	slo1 := sloRepo.CreateSLO(tid, "API Availability", "Agent API uptime", "budget-reconciler", slo.SLOTypeAvailability, 99.9, "30d")
	slo2 := sloRepo.CreateSLO(tid, "Latency P99", "Response time under 2s", "doc-summarizer", slo.SLOTypeLatency, 95.0, "7d")
	slo3 := sloRepo.CreateSLO(tid, "Error Rate", "Error rate below 1%", "ticket-classifier", slo.SLOTypeErrorRate, 99.0, "30d")

	// Add measurements over the past week
	for i := 7; i >= 0; i-- {
		ts := now.Add(-time.Duration(i) * 24 * time.Hour)
		sloRepo.RecordMeasurementAt(tid, slo1.ID, "budget-reconciler", 99.95, 999, 1000, ts)
		sloRepo.RecordMeasurementAt(tid, slo2.ID, "doc-summarizer", 96.5, 965, 1000, ts)
		sloRepo.RecordMeasurementAt(tid, slo3.ID, "ticket-classifier", 97.0, 970, 1000, ts)
	}

	// Seed cost data
	agents := []string{"budget-reconciler", "doc-summarizer", "ticket-classifier", "code-reviewer", "data-pipeline"}
	models := []string{"gpt-4", "claude-3", "gpt-3.5-turbo"}
	for i := 7; i >= 0; i-- {
		ts := now.Add(-time.Duration(i) * 24 * time.Hour)
		for _, agentID := range agents {
			costRepo.RecordCostAt(tid, agentID, fmt.Sprintf("task-%s-%d", agentID, i),
				0.10+float64(i)*0.05, int64(500+i*100), models[i%len(models)], "inference", ts)
		}
	}

	log.Info("seeded control-plane demo data",
		zap.Int("alerts", 3),
		zap.Int("traces", len(traces)),
		zap.Int("slos", 3),
	)
}

func strPtr(s string) *string {
	return &s
}
