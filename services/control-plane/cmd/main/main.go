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
	"github.com/argus-platform/argus/pkg/health"
	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/pkg/metrics"
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
		seedControlPlaneDemo(alertRouter, auditLog, traceSvc, sloRepo, costRepo,
			evalRepo, guardrailsRepo, ragRepo, promptsRepo, feedbackRepo,
			complianceRepo, dqRepo, catalogRepo, log)
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

	// Metrics registry
	metricsReg := metrics.NewRegistry()

	// Health checks with dependency verification
	healthChecker := health.NewChecker()
	if dbPool != nil {
		healthChecker.AddCheck("postgres", func(ctx context.Context) error {
			return dbPool.Ping(ctx)
		})
	}

	// HTTP REST API
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthChecker.Handler())
	mux.HandleFunc("/health/live", healthChecker.LiveHandler())
	mux.HandleFunc("/health/ready", healthChecker.ReadyHandler())
	mux.HandleFunc("/metrics", metricsReg.Handler())

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
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			Subject  string `json:"subject"`
			TenantID string `json:"tenant_id"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
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
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
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
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			TenantID string `json:"tenantId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		if req.Username == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "username is required")
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
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
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
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate refresh token")
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
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		oldClaims, err := jwtAuth.ValidateToken(req.RefreshToken)
		if err != nil {
			httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
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
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
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
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate refresh token")
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
			httputil.WriteJSON(w, http.StatusOK, rules, tenantID)
		case http.MethodPost:
			var rule policy.Rule
			if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
				httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
				return
			}
			policyEngine.AddRule(tenantID, &rule)
			auditLog.Write(tenantID, "system", "create_policy", "policy/"+rule.ID, "")
			httputil.WriteJSON(w, http.StatusCreated, rule, tenantID)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
	})))

	// Policy evaluation endpoint
	mux.Handle("/api/v1/policies/evaluate", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		if r.Method != http.MethodPost {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			Subject  string `json:"subject"`
			Action   string `json:"action"`
			Resource string `json:"resource"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		allowed, err := policyEngine.Evaluate(tenantID, req.Subject, policy.Action(req.Action), req.Resource)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		httputil.WriteJSON(w, http.StatusOK, map[string]bool{"allowed": allowed}, tenantID)
	})))

	// Metrics endpoint — aggregates from local modules
	mux.Handle("/api/v1/metrics", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())

		bd := costRepo.GetBreakdown(tenantID, time.Time{})
		sloStatuses := sloCalc.CalculateAllStatuses(tenantID, "")

		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"total_agents": len(bd.ByAgent),
			"active_tasks": bd.EntryCount,
			"total_cost":   bd.TotalCostUSD,
			"alert_count":  alertRouter.Count(tenantID),
			"slo_count":    len(sloStatuses),
		}, tenantID)
	})))

	// Metrics time series endpoint — platform-wide time series for dashboard charts
	mux.Handle("/api/v1/metrics/timeseries", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		now := time.Now()
		totalTasks := make([]map[string]interface{}, 0, 24)
		activeAgents := make([]map[string]interface{}, 0, 24)
		totalCostSeries := make([]map[string]interface{}, 0, 24)
		alertCountSeries := make([]map[string]interface{}, 0, 24)
		errorRateSeries := make([]map[string]interface{}, 0, 24)
		avgLatencySeries := make([]map[string]interface{}, 0, 24)

		// Use cost trends as a basis for time series data
		trends := costRepo.GetTrends(tenantID, 7)

		if len(trends) > 0 {
			for _, t := range trends {
				totalTasks = append(totalTasks, map[string]interface{}{"timestamp": t.Period, "value": t.TokensUsed / 1500})
				activeAgents = append(activeAgents, map[string]interface{}{"timestamp": t.Period, "value": 5})
				totalCostSeries = append(totalCostSeries, map[string]interface{}{"timestamp": t.Period, "value": t.CostUSD})
				alertCountSeries = append(alertCountSeries, map[string]interface{}{"timestamp": t.Period, "value": alertRouter.Count(tenantID)})
				errorRateSeries = append(errorRateSeries, map[string]interface{}{"timestamp": t.Period, "value": 0.02})
				avgLatencySeries = append(avgLatencySeries, map[string]interface{}{"timestamp": t.Period, "value": 185})
			}
		} else {
			// Generate synthetic time series when no real data
			for i := 23; i >= 0; i-- {
				ts := now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339)
				totalTasks = append(totalTasks, map[string]interface{}{"timestamp": ts, "value": 12 + (i%5)*3})
				activeAgents = append(activeAgents, map[string]interface{}{"timestamp": ts, "value": 5 + (i % 3)})
				totalCostSeries = append(totalCostSeries, map[string]interface{}{"timestamp": ts, "value": 0.45 + float64(i%4)*0.12})
				alertCountSeries = append(alertCountSeries, map[string]interface{}{"timestamp": ts, "value": i % 3})
				errorRateSeries = append(errorRateSeries, map[string]interface{}{"timestamp": ts, "value": 0.01 + float64(i%7)*0.003})
				avgLatencySeries = append(avgLatencySeries, map[string]interface{}{"timestamp": ts, "value": 150 + (i%5)*20})
			}
		}

		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"total_tasks":   totalTasks,
			"active_agents": activeAgents,
			"total_cost":    totalCostSeries,
			"alert_count":   alertCountSeries,
			"error_rate":    errorRateSeries,
			"avg_latency":   avgLatencySeries,
		}, tenantID)
	})))

	// Wrap with auth + tenant + logging + security middleware.
	tenantMw := tenantMiddlewareWithExclusions(
		middleware.TenantHTTP,
		"/health",
		"/api/v1/auth/",
	)
	handler := middleware.Recovery(log)(
		middleware.SecurityHeaders(
			middleware.CORSWithOrigin(
				middleware.MaxBodySize(1 << 20)( // 1MB max request body
					middleware.RequestID(
						metrics.HTTPMiddleware(metricsReg, "control-plane")(
							jwtAuth.Middleware(
								tenantMw(
									middleware.RequestLogger(log)(mux),
								),
							),
						),
					),
				),
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
	evalRepo *eval.Repository,
	guardrailsRepo *guardrails.Repository,
	ragRepo *rag.Repository,
	promptsRepo *prompts.Repository,
	feedbackRepo *feedback.Repository,
	complianceRepo *compliance.ReportRepository,
	dqRepo *dataquality.Repository,
	catalogRepo *catalog.Repository,
	log *zap.Logger,
) {
	tid := "default"
	now := time.Now()

	// Seed alerts
	alertRouter.Fire(tid, "budget-reconciler", alerts.SeverityWarning, "Latency spike detected", "p99 latency exceeded 3x p50 for 45s")
	alertRouter.Fire(tid, "ticket-classifier", alerts.SeverityCritical, "Token escalation pattern", "Agent stuck in retry loop, context window 92% full")
	a3 := alertRouter.Fire(tid, "doc-summarizer", alerts.SeverityInfo, "New agent version available", "v2.1.0 released with performance improvements")
	_, _ = alertRouter.UpdateStatus(tid, a3.ID, "resolved")
	alertRouter.Fire(tid, "data-pipeline", alerts.SeverityWarning, "Retry storm detected", "5 consecutive retries on upstream API in 30s")
	alertRouter.Fire(tid, "code-reviewer", alerts.SeverityInfo, "Cost acceleration", "Token spend rate increased 3x over baseline")

	// Seed audit log entries
	auditLog.Write(tid, "admin", "login", "auth/login", "role: admin")
	auditLog.Write(tid, "admin", "create_policy", "policy/deny-external", "block external API calls")
	auditLog.Write(tid, "system", "quarantine_agent", "agent/ticket-classifier", "auto-quarantine: failure probability 0.94")
	auditLog.Write(tid, "admin", "create_slo", "slo/api-availability", "99.9% availability target")
	auditLog.Write(tid, "admin", "create_guardrail", "guardrail/pii-detect", "PII detection rule enabled")
	auditLog.Write(tid, "system", "eval_completed", "eval/suite-001", "Score: 87% (13/15 passed)")

	// Seed traces
	type spanDef struct {
		op      string
		dur     int64
		parent  string
		errCode *string
	}
	traces := []struct {
		traceID string
		agentID string
		spans   []spanDef
	}{
		{"trace-001", "budget-reconciler", []spanDef{
			{"reconcile_budget", 1250, "", nil},
			{"fetch_transactions", 450, "span-001-0", nil},
			{"compute_totals", 320, "span-001-0", nil},
			{"write_report", 180, "span-001-0", nil},
		}},
		{"trace-002", "doc-summarizer", []spanDef{
			{"summarize_document", 2100, "", nil},
			{"extract_text", 800, "span-002-0", nil},
			{"generate_summary", 1100, "span-002-0", nil},
		}},
		{"trace-003", "ticket-classifier", []spanDef{
			{"classify_ticket", 3500, "", strPtr("TIMEOUT")},
			{"fetch_ticket", 200, "span-003-0", nil},
			{"run_classifier", 3000, "span-003-0", strPtr("CONTEXT_OVERFLOW")},
		}},
		{"trace-004", "data-pipeline", []spanDef{
			{"ingest_batch", 1800, "", nil},
			{"validate_schema", 300, "span-004-0", nil},
			{"transform_records", 900, "span-004-0", nil},
			{"write_to_store", 400, "span-004-0", nil},
		}},
	}

	for _, t := range traces {
		for i, sp := range t.spans {
			spanID := fmt.Sprintf("span-%s-%d", t.traceID[len(t.traceID)-3:], i)
			traceSvc.IngestSpan(&trace.Span{
				SpanID:        spanID,
				TraceID:       t.traceID,
				ParentSpanID:  sp.parent,
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

	// Seed eval test suites and runs
	completedAt := now.Add(-1 * time.Hour)
	evalRepo.AddSuite(&eval.TestSuite{
		ID: "suite-001", TenantID: tid, Name: "Budget Agent Accuracy",
		Description: "Validates budget reconciliation accuracy", AgentID: "budget-reconciler",
		TestCases: []eval.TestCase{
			{ID: "tc-1", Name: "Simple reconciliation", Input: "Reconcile Q1 budget", ExpectedOutput: "Budget balanced", Criteria: map[string]string{"accuracy": "exact"}, MaxLatencyMs: 5000},
			{ID: "tc-2", Name: "Multi-currency", Input: "Reconcile USD/EUR accounts", ExpectedOutput: "Converted and balanced", Criteria: map[string]string{"accuracy": "fuzzy"}, MaxLatencyMs: 8000},
			{ID: "tc-3", Name: "Error handling", Input: "Reconcile missing data", ExpectedOutput: "Error: missing records", Criteria: map[string]string{"error_handling": "graceful"}, MaxLatencyMs: 3000},
		},
		CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour),
	})
	evalRepo.AddSuite(&eval.TestSuite{
		ID: "suite-002", TenantID: tid, Name: "Document Summarizer Quality",
		Description: "Tests summarization quality and latency", AgentID: "doc-summarizer",
		TestCases: []eval.TestCase{
			{ID: "tc-4", Name: "Short document", Input: "Summarize 1-page report", ExpectedOutput: "Concise summary", Criteria: map[string]string{"brevity": "high"}, MaxLatencyMs: 3000},
			{ID: "tc-5", Name: "Long document", Input: "Summarize 50-page whitepaper", ExpectedOutput: "Executive summary", Criteria: map[string]string{"coverage": "comprehensive"}, MaxLatencyMs: 15000},
		},
		CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-12 * time.Hour),
	})
	evalRepo.AddRun(&eval.EvalRun{
		ID: "run-001", TenantID: tid, SuiteID: "suite-001", SuiteName: "Budget Agent Accuracy",
		AgentID: "budget-reconciler", Status: "completed", Score: 0.87,
		TotalCases: 3, PassedCases: 2, FailedCases: 1,
		Results: []eval.CaseResult{
			{TestCaseID: "tc-1", TestCaseName: "Simple reconciliation", Status: "passed", ActualOutput: "Budget balanced", LatencyMs: 1200, Score: 1.0, Reason: "Exact match"},
			{TestCaseID: "tc-2", TestCaseName: "Multi-currency", Status: "passed", ActualOutput: "Converted and balanced", LatencyMs: 3400, Score: 0.95, Reason: "Fuzzy match 95%"},
			{TestCaseID: "tc-3", TestCaseName: "Error handling", Status: "failed", ActualOutput: "Timeout", LatencyMs: 5100, Score: 0.0, Reason: "Exceeded max latency"},
		},
		StartedAt: now.Add(-2 * time.Hour), CompletedAt: &completedAt,
	})
	evalRepo.AddRun(&eval.EvalRun{
		ID: "run-002", TenantID: tid, SuiteID: "suite-002", SuiteName: "Document Summarizer Quality",
		AgentID: "doc-summarizer", Status: "completed", Score: 1.0,
		TotalCases: 2, PassedCases: 2, FailedCases: 0,
		Results: []eval.CaseResult{
			{TestCaseID: "tc-4", TestCaseName: "Short document", Status: "passed", ActualOutput: "Concise summary generated", LatencyMs: 1800, Score: 1.0, Reason: "Quality threshold met"},
			{TestCaseID: "tc-5", TestCaseName: "Long document", Status: "passed", ActualOutput: "Executive summary with key findings", LatencyMs: 12000, Score: 0.92, Reason: "Coverage 92%"},
		},
		StartedAt: now.Add(-90 * time.Minute), CompletedAt: &completedAt,
	})

	// Seed guardrail rules and violations
	guardrailsRepo.AddRule(&guardrails.Rule{
		ID: "gr-001", TenantID: tid, Name: "PII Detection", Description: "Detects personal identifiable information in agent outputs",
		Type: "pii_detection", Pattern: `\b\d{3}-\d{2}-\d{4}\b`, Action: "block", Enabled: true,
		AgentIDs: []string{"budget-reconciler", "doc-summarizer"}, CreatedAt: now.Add(-7 * 24 * time.Hour),
	})
	guardrailsRepo.AddRule(&guardrails.Rule{
		ID: "gr-002", TenantID: tid, Name: "Prompt Injection Guard", Description: "Blocks prompt injection attempts",
		Type: "prompt_injection", Pattern: "ignore previous|system prompt|you are now", Action: "block", Enabled: true,
		AgentIDs: []string{}, CreatedAt: now.Add(-5 * 24 * time.Hour),
	})
	guardrailsRepo.AddRule(&guardrails.Rule{
		ID: "gr-003", TenantID: tid, Name: "Toxicity Filter", Description: "Warns on toxic content in outputs",
		Type: "toxicity", Pattern: "", Action: "warn", Enabled: true,
		AgentIDs: []string{"ticket-classifier"}, CreatedAt: now.Add(-3 * 24 * time.Hour),
	})
	guardrailsRepo.AddViolation(&guardrails.Violation{
		ID: "gv-001", TenantID: tid, RuleID: "gr-001", RuleName: "PII Detection", RuleType: "pii_detection",
		AgentID: "budget-reconciler", SpanID: "span-001-0", Action: "block",
		Content: "Output contained SSN pattern: ***-**-****", CreatedAt: now.Add(-2 * time.Hour),
	})
	guardrailsRepo.AddViolation(&guardrails.Violation{
		ID: "gv-002", TenantID: tid, RuleID: "gr-002", RuleName: "Prompt Injection Guard", RuleType: "prompt_injection",
		AgentID: "ticket-classifier", SpanID: "span-003-1", Action: "block",
		Content: "Input contained: ignore previous instructions", CreatedAt: now.Add(-45 * time.Minute),
	})
	guardrailsRepo.AddViolation(&guardrails.Violation{
		ID: "gv-003", TenantID: tid, RuleID: "gr-003", RuleName: "Toxicity Filter", RuleType: "toxicity",
		AgentID: "ticket-classifier", SpanID: "span-003-2", Action: "warn",
		Content: "Output flagged for hostile tone", CreatedAt: now.Add(-30 * time.Minute),
	})

	// Seed RAG retrievals and sources
	ragRepo.AddSource(&rag.Source{
		ID: "src-001", TenantID: tid, Name: "Financial Regulations KB", Type: "document",
		TotalChunks: 2450, AvgRelevance: 0.87, UsageCount: 342,
	})
	ragRepo.AddSource(&rag.Source{
		ID: "src-002", TenantID: tid, Name: "Internal Policies DB", Type: "database",
		TotalChunks: 890, AvgRelevance: 0.92, UsageCount: 156,
	})
	ragRepo.AddSource(&rag.Source{
		ID: "src-003", TenantID: tid, Name: "Support Tickets Archive", Type: "document",
		TotalChunks: 5200, AvgRelevance: 0.78, UsageCount: 89,
	})
	for i := 0; i < 8; i++ {
		ragRepo.AddRetrieval(&rag.Retrieval{
			ID: fmt.Sprintf("ret-%03d", i+1), TenantID: tid, AgentID: agents[i%len(agents)],
			SpanID: fmt.Sprintf("span-ret-%d", i), Query: []string{
				"What are the Q4 budget limits?", "Summarize data residency requirements",
				"Classify this support ticket", "Review code for security issues",
				"Extract key financial metrics", "Find relevant compliance policies",
				"Get latest API documentation", "Search for error patterns",
			}[i],
			NumChunks: 3 + i%4, AvgRelevance: 0.75 + float64(i%5)*0.05,
			LatencyMs: int64(45 + i*12), SourceIDs: []string{"src-001", "src-002"},
			CreatedAt: now.Add(-time.Duration(i) * 30 * time.Minute),
		})
	}

	// Seed prompts with versions
	promptsRepo.AddPrompt(&prompts.Prompt{
		ID: "prompt-001", TenantID: tid, Name: "Budget Analysis Prompt",
		Description: "System prompt for budget reconciliation agent", AgentID: "budget-reconciler",
		ActiveVersion: 2, CreatedAt: now.Add(-14 * 24 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour),
	})
	promptsRepo.AddVersion(&prompts.PromptVersion{
		ID: "pv-001", PromptID: "prompt-001", TenantID: tid, Version: 1,
		Content: "You are a financial analysis agent. Reconcile budgets accurately.", ChangeLog: "Initial version",
		Metrics:   &prompts.VersionMetrics{AvgLatencyMs: 1450, SuccessRate: 0.89, TokensUsed: 45000, Invocations: 120},
		CreatedAt: now.Add(-14 * 24 * time.Hour),
	})
	promptsRepo.AddVersion(&prompts.PromptVersion{
		ID: "pv-002", PromptID: "prompt-001", TenantID: tid, Version: 2,
		Content: "You are a financial analysis agent specializing in budget reconciliation. Always verify totals before reporting.", ChangeLog: "Added verification step",
		Metrics:   &prompts.VersionMetrics{AvgLatencyMs: 1200, SuccessRate: 0.95, TokensUsed: 52000, Invocations: 85},
		CreatedAt: now.Add(-24 * time.Hour),
	})
	promptsRepo.AddPrompt(&prompts.Prompt{
		ID: "prompt-002", TenantID: tid, Name: "Document Summary Prompt",
		Description: "System prompt for document summarization", AgentID: "doc-summarizer",
		ActiveVersion: 1, CreatedAt: now.Add(-7 * 24 * time.Hour), UpdatedAt: now.Add(-7 * 24 * time.Hour),
	})
	promptsRepo.AddVersion(&prompts.PromptVersion{
		ID: "pv-003", PromptID: "prompt-002", TenantID: tid, Version: 1,
		Content: "Summarize the given document concisely, preserving key facts and figures.", ChangeLog: "Initial version",
		Metrics:   &prompts.VersionMetrics{AvgLatencyMs: 2100, SuccessRate: 0.97, TokensUsed: 38000, Invocations: 200},
		CreatedAt: now.Add(-7 * 24 * time.Hour),
	})

	// Seed feedback
	feedbackItems := []struct {
		agentID string
		rating  int
		comment string
	}{
		{"budget-reconciler", 5, "Accurate reconciliation, saved hours of manual work"},
		{"budget-reconciler", 4, "Good results but took longer than expected"},
		{"doc-summarizer", 5, "Excellent summary quality"},
		{"doc-summarizer", 5, "Very concise and accurate"},
		{"doc-summarizer", 3, "Missed some key details in long document"},
		{"ticket-classifier", 2, "Misclassified priority - should have been urgent"},
		{"ticket-classifier", 4, "Good classification for standard tickets"},
		{"code-reviewer", 5, "Found a critical security vulnerability I missed"},
		{"code-reviewer", 4, "Helpful suggestions, some were too verbose"},
		{"data-pipeline", 3, "Processing was slow for large batches"},
	}
	for i, fb := range feedbackItems {
		feedbackRepo.AddFeedback(&feedback.Feedback{
			ID: fmt.Sprintf("fb-%03d", i+1), TenantID: tid, AgentID: fb.agentID,
			SpanID: fmt.Sprintf("span-fb-%d", i), TaskID: fmt.Sprintf("task-fb-%d", i),
			Rating: fb.rating, Comment: fb.comment, UserID: "admin",
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
		})
	}

	// Seed compliance report
	complianceRepo.AddReport(&compliance.Report{
		ID: "cr-001", TenantID: tid, ProfileID: "gcc-sa", ProfileName: "GCC Saudi Arabia",
		Title: "Q1 2026 Compliance Assessment", Status: "completed", Format: "json",
		Sections: []compliance.Section{
			{Title: "Data Residency", Status: "compliant", Description: "All data stored within approved regions",
				Findings: []string{"Data stored in sa-east-1 region", "No cross-border transfers detected"},
				Evidence: []string{"Storage audit log", "Network flow analysis"}},
			{Title: "Access Control", Status: "compliant", Description: "RBAC policies enforced with tenant isolation",
				Findings: []string{"All API calls validated with JWT", "Tenant isolation verified via RLS"},
				Evidence: []string{"Auth middleware audit", "Database RLS policy review"}},
			{Title: "Encryption", Status: "partial", Description: "Encryption at rest and in transit",
				Findings: []string{"TLS 1.3 for all connections", "AES-256 at rest", "Key rotation pending for Q2"},
				Evidence: []string{"TLS configuration scan", "Encryption audit report"}},
			{Title: "Audit Logging", Status: "compliant", Description: "Immutable audit trail for all operations",
				Findings: []string{"All CRUD operations logged", "Logs retained for 5 years"},
				Evidence: []string{"Audit log analysis", "Retention policy document"}},
		},
		GeneratedAt: now.Add(-48 * time.Hour), PeriodStart: now.Add(-90 * 24 * time.Hour), PeriodEnd: now,
	})

	// Seed data quality scores and rules
	dqRepo.CreateRule(tid, "Latency Threshold", "Agent response must be under 5s", dataquality.RuleTypeTimeliness,
		"budget-reconciler", "latency_ms", "lt", "5000", dataquality.SeverityWarning)
	dqRepo.CreateRule(tid, "Output Completeness", "Outputs must have all required fields", dataquality.RuleTypeCompleteness,
		"", "output_fields", "not_null", "", dataquality.SeverityCritical)
	dqRepo.CreateRule(tid, "Token Consistency", "Token usage should not vary >50%", dataquality.RuleTypeConsistency,
		"doc-summarizer", "tokens_used", "lt", "50", dataquality.SeverityInfo)

	for _, agentID := range agents[:3] {
		dqRepo.RecordScore(tid, agentID, 85+float64(len(agentID)%10), 92, 88, 90, 87, 100, 87, 13)
	}

	// Seed catalog sources with lineage
	src1 := catalogRepo.CreateSource(tid, "Transaction Database", "Primary financial transaction store",
		catalog.SourceTypeDatabase, "finance-team", "budget-reconciler",
		[]string{"financial", "transactions", "production"}, map[string]string{"amount": "decimal", "currency": "string", "date": "timestamp"})
	src2 := catalogRepo.CreateSource(tid, "Report Storage", "Generated reports and summaries",
		catalog.SourceTypeFile, "analytics-team", "doc-summarizer",
		[]string{"reports", "summaries", "output"}, map[string]string{"title": "string", "content": "text", "format": "string"})
	src3 := catalogRepo.CreateSource(tid, "Ticket Queue API", "Support ticket management system",
		catalog.SourceTypeAPI, "support-team", "ticket-classifier",
		[]string{"support", "tickets", "api"}, map[string]string{"ticket_id": "string", "priority": "string", "category": "string"})
	_, _ = catalogRepo.AddLineageEdge(tid, src1.ID, src2.ID, "transform", "budget-reconciler", "Transactions → Budget Reports")
	_, _ = catalogRepo.AddLineageEdge(tid, src3.ID, src2.ID, "aggregate", "doc-summarizer", "Tickets → Summary Reports")

	log.Info("seeded control-plane demo data",
		zap.Int("alerts", 5),
		zap.Int("traces", len(traces)),
		zap.Int("slos", 3),
		zap.Int("eval_suites", 2),
		zap.Int("guardrail_rules", 3),
		zap.Int("rag_sources", 3),
		zap.Int("prompts", 2),
		zap.Int("feedback", len(feedbackItems)),
		zap.Int("catalog_sources", 3),
	)
}

func strPtr(s string) *string {
	return &s
}
