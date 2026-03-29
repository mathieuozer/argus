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
	"github.com/argus-platform/argus/services/control-plane/internal/governance"
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

	// Partially-implemented PG repos — these don't yet satisfy the full Store interface
	// and will be wired once complete.
	_ = catalogPG
	_ = dqPG

	// Initialize components
	auditLog := audit.NewWriter()
	alertRouter := alerts.NewRouter()
	policyEngine := policy.New()
	jwtAuth := auth.New(os.Getenv("ARGUS_JWT_SECRET"))
	dashHandler := dashboard.New(alertRouter, auditLog)

	// Initialize observability module handlers with PG persistence when available
	traceSvc := trace.NewService()
	var traceStore trace.Store = trace.NewMemStore(traceSvc)
	if tracePG != nil {
		traceStore = tracePG
	}
	traceHandler := trace.NewHandler(traceStore)

	dqStore, dqRepo := dataquality.NewMemStore()
	dqHandler := dataquality.NewHandler(dqStore)

	catalogRepo := catalog.NewRepository()
	catalogHandler := catalog.NewHandler(catalog.NewMemStore(catalogRepo))

	costRepo := costgov.NewRepository()
	costDetector := costgov.NewAnomalyDetector()
	costHandler := costgov.NewHandler(costRepo, costDetector)

	sloRepo := slo.NewRepository()
	var sloStore slo.Store = slo.NewMemStore(sloRepo)
	if sloPG != nil {
		sloStore = sloPG
	}
	sloCalc := slo.NewCalculator(sloStore)
	sloHandler := slo.NewHandler(sloStore, sloCalc)

	var auditStore audit.Store = audit.NewMemStore(auditLog)
	if auditPG != nil {
		auditStore = auditPG
	}
	auditHandler := audit.NewHandler(auditStore)

	// Initialize feature module handlers with PG dual-write when available
	evalRepo := eval.NewRepository()
	evalHandler := eval.NewHandler(evalRepo)
	if evalPG != nil {
		evalHandler.SetPG(evalPG)
	}

	feedbackRepo := feedback.NewRepository()
	feedbackHandler := feedback.NewHandler(feedbackRepo)
	if feedbackPG != nil {
		feedbackHandler.SetPG(feedbackPG)
	}

	guardrailsRepo := guardrails.NewRepository()
	guardrailsHandler := guardrails.NewHandler(guardrailsRepo)
	if guardrailsPG != nil {
		guardrailsHandler.SetPG(guardrailsPG)
	}

	promptsRepo := prompts.NewRepository()
	promptsHandler := prompts.NewHandler(promptsRepo)
	if promptsPG != nil {
		promptsHandler.SetPG(promptsPG)
	}

	ragRepo := rag.NewRepository()
	ragHandler := rag.NewHandler(ragRepo)
	if ragPG != nil {
		ragHandler.SetPG(ragPG)
	}

	complianceRepo := compliance.NewReportRepository()
	complianceHandler := compliance.NewReportHandler(complianceRepo)
	if compliancePG != nil {
		complianceHandler.SetPG(compliancePG)
	}

	if costPG != nil {
		costHandler.SetPG(costPG)
	}

	govRepo := governance.NewRepository()
	govHandler := governance.NewHandler(govRepo)

	// Seed demo data for dashboard (non-production only)
	if os.Getenv("ARGUS_ENV") != "production" {
		seedControlPlaneDemo(alertRouter, auditLog, traceSvc, sloRepo, costRepo,
			evalRepo, guardrailsRepo, ragRepo, promptsRepo, feedbackRepo,
			complianceRepo, dqRepo, catalogRepo, govRepo, log)
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
	govHandler.RegisterRoutes(mux)

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
		sloStatuses, _ := sloCalc.CalculateAllStatuses(r.Context(), tenantID, "")

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

	// Agent and task stub endpoints (served from control-plane for dev convenience
	// when the orchestrator is not running separately).
	mux.Handle("/api/v1/agents", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, seedAgents, tenantID)
	})))
	mux.Handle("/api/v1/agents/", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		agentID := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
		agentID = strings.SplitN(agentID, "/", 2)[0]
		for _, a := range seedAgents {
			if a["id"] == agentID {
				httputil.WriteJSON(w, http.StatusOK, a, tenantID)
				return
			}
		}
		httputil.WriteError(w, http.StatusNotFound, "AGENT_NOT_FOUND", "agent not found")
	})))
	mux.Handle("/api/v1/tasks", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		if r.Method != http.MethodGet {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, seedTasks, tenantID)
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

// seedAgents and seedTasks are dev-mode stub data for the agents/tasks pages
// when running the control-plane standalone (without the orchestrator).
var seedAgents = []map[string]interface{}{
	{
		"id": "budget-reconciler", "tenant_id": "default", "version": "2.1.0",
		"framework": "langchain", "capabilities": []string{"read:budget_db", "write:report_store"},
		"status": "healthy", "svid_uri": "spiffe://argus.local/tenant/default/agent/budget-reconciler/v2.1.0",
		"last_seen": time.Now().Add(-30 * time.Second).Format(time.RFC3339), "node_id": "node-eu-west-1a",
		"tasks_completed": 1247, "tasks_failed": 3, "total_cost_usd": 45.82,
		"total_tokens": 2850000, "avg_latency_ms": 185, "uptime_pct": 99.8,
	},
	{
		"id": "code-reviewer", "tenant_id": "default", "version": "1.4.2",
		"framework": "autogen", "capabilities": []string{"read:git_repo", "write:review_comments"},
		"status": "healthy", "svid_uri": "spiffe://argus.local/tenant/default/agent/code-reviewer/v1.4.2",
		"last_seen": time.Now().Add(-15 * time.Second).Format(time.RFC3339), "node_id": "node-eu-west-1b",
		"tasks_completed": 892, "tasks_failed": 12, "total_cost_usd": 28.45,
		"total_tokens": 1920000, "avg_latency_ms": 320, "uptime_pct": 98.7,
	},
	{
		"id": "data-pipeline", "tenant_id": "default", "version": "3.0.1",
		"framework": "custom", "capabilities": []string{"read:data_lake", "write:warehouse", "execute:etl"},
		"status": "degraded", "svid_uri": "spiffe://argus.local/tenant/default/agent/data-pipeline/v3.0.1",
		"last_seen": time.Now().Add(-5 * time.Minute).Format(time.RFC3339), "node_id": "node-eu-west-1a",
		"tasks_completed": 5623, "tasks_failed": 47, "total_cost_usd": 312.90,
		"total_tokens": 18500000, "avg_latency_ms": 890, "uptime_pct": 97.2,
	},
	{
		"id": "customer-support", "tenant_id": "default", "version": "1.0.0",
		"framework": "langchain", "capabilities": []string{"read:ticket_db", "write:responses", "read:knowledge_base"},
		"status": "healthy", "svid_uri": "spiffe://argus.local/tenant/default/agent/customer-support/v1.0.0",
		"last_seen": time.Now().Add(-10 * time.Second).Format(time.RFC3339), "node_id": "node-eu-west-1c",
		"tasks_completed": 3215, "tasks_failed": 8, "total_cost_usd": 156.30,
		"total_tokens": 9400000, "avg_latency_ms": 245, "uptime_pct": 99.5,
	},
	{
		"id": "security-scanner", "tenant_id": "default", "version": "2.3.0",
		"framework": "crewai", "capabilities": []string{"read:network_logs", "write:alerts", "execute:scan"},
		"status": "healthy", "svid_uri": "spiffe://argus.local/tenant/default/agent/security-scanner/v2.3.0",
		"last_seen": time.Now().Add(-45 * time.Second).Format(time.RFC3339), "node_id": "node-eu-west-1b",
		"tasks_completed": 782, "tasks_failed": 2, "total_cost_usd": 67.15,
		"total_tokens": 4200000, "avg_latency_ms": 540, "uptime_pct": 99.9,
	},
}

var seedTasks = func() []map[string]interface{} {
	now := time.Now()
	tasks := []map[string]interface{}{
		{"id": "task-001", "tenant_id": "default", "agent_id": "budget-reconciler", "status": "completed",
			"input_hash": "a1b2c3d4e5", "started_at": now.Add(-2 * time.Hour).Format(time.RFC3339),
			"completed_at": now.Add(-1 * time.Hour).Format(time.RFC3339), "cost_usd": 0.032, "tokens_used": 15200},
		{"id": "task-002", "tenant_id": "default", "agent_id": "code-reviewer", "status": "running",
			"input_hash": "f6g7h8i9j0", "started_at": now.Add(-10 * time.Minute).Format(time.RFC3339),
			"completed_at": nil, "cost_usd": 0.018, "tokens_used": 8400},
		{"id": "task-003", "tenant_id": "default", "agent_id": "data-pipeline", "status": "completed",
			"input_hash": "k1l2m3n4o5", "started_at": now.Add(-3 * time.Hour).Format(time.RFC3339),
			"completed_at": now.Add(-2*time.Hour - 30*time.Minute).Format(time.RFC3339), "cost_usd": 0.087, "tokens_used": 42300},
		{"id": "task-004", "tenant_id": "default", "agent_id": "customer-support", "status": "completed",
			"input_hash": "p6q7r8s9t0", "started_at": now.Add(-45 * time.Minute).Format(time.RFC3339),
			"completed_at": now.Add(-30 * time.Minute).Format(time.RFC3339), "cost_usd": 0.024, "tokens_used": 11800},
		{"id": "task-005", "tenant_id": "default", "agent_id": "security-scanner", "status": "pending",
			"input_hash": "u1v2w3x4y5", "started_at": now.Format(time.RFC3339),
			"completed_at": nil, "cost_usd": 0.0, "tokens_used": 0},
		{"id": "task-006", "tenant_id": "default", "agent_id": "data-pipeline", "status": "failed",
			"input_hash": "z6a7b8c9d0", "started_at": now.Add(-4 * time.Hour).Format(time.RFC3339),
			"completed_at": now.Add(-3*time.Hour - 45*time.Minute).Format(time.RFC3339), "cost_usd": 0.054, "tokens_used": 26100},
		{"id": "task-007", "tenant_id": "default", "agent_id": "budget-reconciler", "status": "completed",
			"input_hash": "e1f2g3h4i5", "started_at": now.Add(-6 * time.Hour).Format(time.RFC3339),
			"completed_at": now.Add(-5 * time.Hour).Format(time.RFC3339), "cost_usd": 0.041, "tokens_used": 19700},
		{"id": "task-008", "tenant_id": "default", "agent_id": "customer-support", "status": "running",
			"input_hash": "j6k7l8m9n0", "started_at": now.Add(-5 * time.Minute).Format(time.RFC3339),
			"completed_at": nil, "cost_usd": 0.012, "tokens_used": 5600},
	}
	return tasks
}()

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
	govRepo *governance.Repository,
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

	// Seed enhanced catalog sources with full metadata
	src1 := catalogRepo.CreateSourceFull(tid, "Transaction Database", "Primary financial transaction store with all payment records",
		catalog.SourceTypeDatabase, "finance-team", "budget-reconciler",
		[]string{"financial", "transactions", "production", "critical"},
		map[string]string{"amount": "decimal", "currency": "string", "date": "timestamp", "account_id": "string", "status": "string"},
		"confidential", "finance", "active", "Sarah Chen",
		92.5,
		&catalog.FreshnessInfo{LastRefreshed: now.Add(-30 * time.Minute), RefreshFrequency: "30m", SLASeconds: 3600, IsStale: false},
		&catalog.ProfileInfo{RowCount: 2500000, ColumnCount: 12, SizeBytes: 4800000000, NullRate: 0.02, DuplicateRate: 0.001, Completeness: 98.5, LastProfiled: now.Add(-2 * time.Hour)},
		[]*catalog.Column{
			{Name: "transaction_id", Type: "uuid", Description: "Unique transaction identifier", Classification: "internal", UniqueCount: 2500000, NullRate: 0},
			{Name: "amount", Type: "decimal(18,2)", Description: "Transaction amount", Classification: "confidential", NullRate: 0, MinValue: "0.01", MaxValue: "9999999.99"},
			{Name: "currency", Type: "varchar(3)", Description: "ISO 4217 currency code", Classification: "internal", UniqueCount: 12, NullRate: 0},
			{Name: "account_id", Type: "uuid", Description: "Customer account reference", IsPII: true, Classification: "restricted", UniqueCount: 45000, NullRate: 0},
			{Name: "customer_name", Type: "varchar(255)", Description: "Customer full name", IsPII: true, Classification: "restricted", UniqueCount: 42000, NullRate: 0.01},
			{Name: "email", Type: "varchar(255)", Description: "Customer email address", IsPII: true, Classification: "restricted", UniqueCount: 41000, NullRate: 0.05},
			{Name: "status", Type: "varchar(20)", Description: "Transaction status", Classification: "internal", UniqueCount: 5, NullRate: 0},
			{Name: "created_at", Type: "timestamp", Description: "Transaction creation time", Classification: "internal", NullRate: 0},
		},
	)

	src2 := catalogRepo.CreateSourceFull(tid, "Report Storage", "Generated reports, summaries, and analytics outputs",
		catalog.SourceTypeFile, "analytics-team", "doc-summarizer",
		[]string{"reports", "summaries", "output", "analytics"},
		map[string]string{"title": "string", "content": "text", "format": "string", "generated_by": "string"},
		"internal", "analytics", "active", "Mike Johnson",
		88.0,
		&catalog.FreshnessInfo{LastRefreshed: now.Add(-1 * time.Hour), RefreshFrequency: "1h", SLASeconds: 7200, IsStale: false},
		&catalog.ProfileInfo{RowCount: 15000, ColumnCount: 8, SizeBytes: 250000000, NullRate: 0.05, DuplicateRate: 0.02, Completeness: 95.0, LastProfiled: now.Add(-3 * time.Hour)},
		[]*catalog.Column{
			{Name: "report_id", Type: "uuid", Description: "Unique report identifier", Classification: "internal", UniqueCount: 15000, NullRate: 0},
			{Name: "title", Type: "varchar(500)", Description: "Report title", Classification: "internal", NullRate: 0},
			{Name: "content", Type: "text", Description: "Full report content", Classification: "confidential", NullRate: 0.02},
			{Name: "format", Type: "varchar(10)", Description: "Output format (pdf, html, json)", Classification: "internal", UniqueCount: 3, NullRate: 0},
		},
	)

	src3 := catalogRepo.CreateSourceFull(tid, "Ticket Queue API", "Support ticket management system via REST API",
		catalog.SourceTypeAPI, "support-team", "ticket-classifier",
		[]string{"support", "tickets", "api", "customer-facing"},
		map[string]string{"ticket_id": "string", "priority": "string", "category": "string", "description": "text"},
		"confidential", "support", "active", "Alex Rivera",
		85.0,
		&catalog.FreshnessInfo{LastRefreshed: now.Add(-5 * time.Minute), RefreshFrequency: "5m", SLASeconds: 600, IsStale: false},
		&catalog.ProfileInfo{RowCount: 89000, ColumnCount: 15, SizeBytes: 180000000, NullRate: 0.08, DuplicateRate: 0.005, Completeness: 92.0, LastProfiled: now.Add(-1 * time.Hour)},
		[]*catalog.Column{
			{Name: "ticket_id", Type: "varchar(20)", Description: "Support ticket identifier", Classification: "internal", UniqueCount: 89000, NullRate: 0},
			{Name: "customer_email", Type: "varchar(255)", Description: "Customer email", IsPII: true, Classification: "restricted", UniqueCount: 55000, NullRate: 0.01},
			{Name: "priority", Type: "enum", Description: "Ticket priority level", Classification: "internal", UniqueCount: 4, NullRate: 0},
			{Name: "category", Type: "varchar(50)", Description: "Ticket category", Classification: "internal", UniqueCount: 12, NullRate: 0.02},
			{Name: "description", Type: "text", Description: "Ticket description text", Classification: "confidential", NullRate: 0},
		},
	)

	src4 := catalogRepo.CreateSourceFull(tid, "ML Feature Store", "Feature store for ML model training and inference",
		catalog.SourceTypeDatabase, "ml-team", "data-pipeline",
		[]string{"ml", "features", "training", "inference"},
		map[string]string{"feature_name": "string", "feature_value": "float64", "entity_id": "string"},
		"internal", "ml-engineering", "active", "Dr. Priya Patel",
		95.0,
		&catalog.FreshnessInfo{LastRefreshed: now.Add(-15 * time.Minute), RefreshFrequency: "15m", SLASeconds: 1800, IsStale: false},
		&catalog.ProfileInfo{RowCount: 12000000, ColumnCount: 45, SizeBytes: 24000000000, NullRate: 0.01, DuplicateRate: 0.0001, Completeness: 99.2, LastProfiled: now.Add(-30 * time.Minute)},
		nil,
	)

	src5 := catalogRepo.CreateSourceFull(tid, "Compliance Audit Log", "Immutable audit trail for regulatory compliance",
		catalog.SourceTypeStream, "security-team", "code-reviewer",
		[]string{"audit", "compliance", "immutable", "security"},
		map[string]string{"event_id": "string", "actor": "string", "action": "string", "resource": "string"},
		"restricted", "security", "active", "James Wilson",
		97.0,
		&catalog.FreshnessInfo{LastRefreshed: now.Add(-1 * time.Minute), RefreshFrequency: "1m", SLASeconds: 120, IsStale: false},
		&catalog.ProfileInfo{RowCount: 5800000, ColumnCount: 18, SizeBytes: 8900000000, NullRate: 0.0, DuplicateRate: 0.0, Completeness: 100.0, LastProfiled: now.Add(-10 * time.Minute)},
		nil,
	)

	// Lineage edges
	_, _ = catalogRepo.AddLineageEdge(tid, src1.ID, src2.ID, "transform", "budget-reconciler", "Transactions processed into budget reports")
	_, _ = catalogRepo.AddLineageEdge(tid, src3.ID, src2.ID, "aggregate", "doc-summarizer", "Tickets summarized into reports")
	_, _ = catalogRepo.AddLineageEdge(tid, src1.ID, src4.ID, "extract", "data-pipeline", "Financial features extracted for ML")
	_, _ = catalogRepo.AddLineageEdge(tid, src3.ID, src4.ID, "extract", "data-pipeline", "Support ticket features for classification model")
	_, _ = catalogRepo.AddLineageEdge(tid, src4.ID, src2.ID, "predict", "data-pipeline", "ML predictions included in reports")
	_, _ = catalogRepo.AddLineageEdge(tid, src1.ID, src5.ID, "audit", "code-reviewer", "Financial transactions audited")
	_, _ = catalogRepo.AddLineageEdge(tid, src3.ID, src5.ID, "audit", "code-reviewer", "Ticket access events logged")

	// Glossary terms
	catalogRepo.CreateGlossaryTerm(tid, "PII", "Personally Identifiable Information — any data that can identify a specific individual", "security", "James Wilson", []string{"GDPR", "Data Classification"}, []string{src1.ID, src3.ID})
	catalogRepo.CreateGlossaryTerm(tid, "SLA", "Service Level Agreement — contractual uptime and freshness guarantees", "operations", "Sarah Chen", []string{"SLO", "Uptime"}, []string{})
	catalogRepo.CreateGlossaryTerm(tid, "Feature Store", "Centralized repository of ML features for training and serving", "ml-engineering", "Dr. Priya Patel", []string{"ML Pipeline", "Feature Engineering"}, []string{src4.ID})
	catalogRepo.CreateGlossaryTerm(tid, "Data Contract", "Formal agreement between data producer and consumer on schema, quality, and freshness", "data-governance", "Mike Johnson", []string{"Schema", "Data Quality"}, []string{})
	catalogRepo.CreateGlossaryTerm(tid, "Data Lineage", "Record of how data flows and transforms across the platform", "data-governance", "Alex Rivera", []string{"ETL", "Provenance"}, []string{})

	// Seed enhanced data quality: contracts, profiles, incidents, anomalies
	dqRepo.RecordProfile(tid, "budget-reconciler", src1.ID, 2500000, 12, 0.02, 0.001, 98.5, []*dataquality.ColumnProfile{
		{Name: "amount", Type: "decimal", NullRate: 0, UniqueRate: 0.85, MinValue: "0.01", MaxValue: "9999999.99", MeanValue: "1245.67"},
		{Name: "account_id", Type: "uuid", NullRate: 0, UniqueRate: 0.018},
	})
	dqRepo.RecordProfile(tid, "doc-summarizer", src2.ID, 15000, 8, 0.05, 0.02, 95.0, nil)
	dqRepo.RecordProfile(tid, "ticket-classifier", src3.ID, 89000, 15, 0.08, 0.005, 92.0, nil)

	dqRepo.CreateContract(tid, "Transaction Data SLA", "Guarantees freshness and completeness of financial transaction data",
		"budget-reconciler", []string{"doc-summarizer", "data-pipeline"}, src1.ID,
		map[string]string{"amount": "decimal", "currency": "string", "date": "timestamp"},
		&dataquality.FreshnessSpec{MaxStalenessSeconds: 3600, RefreshSchedule: "*/30 * * * *"},
		&dataquality.QualitySpec{MinCompleteness: 98.0, MinAccuracy: 99.5, MaxNullRate: 0.05},
	)
	dqRepo.CreateContract(tid, "Ticket Data Contract", "Schema and quality agreement for support ticket data",
		"ticket-classifier", []string{"doc-summarizer"}, src3.ID,
		map[string]string{"ticket_id": "string", "priority": "string", "category": "string"},
		&dataquality.FreshnessSpec{MaxStalenessSeconds: 600, RefreshSchedule: "*/5 * * * *"},
		&dataquality.QualitySpec{MinCompleteness: 92.0, MinAccuracy: 95.0, MaxNullRate: 0.10},
	)
	dqRepo.CreateContract(tid, "ML Feature Freshness", "Ensures ML features are up-to-date for inference",
		"data-pipeline", []string{"budget-reconciler", "ticket-classifier"}, src4.ID,
		map[string]string{"feature_name": "string", "feature_value": "float64"},
		&dataquality.FreshnessSpec{MaxStalenessSeconds: 1800, RefreshSchedule: "*/15 * * * *"},
		&dataquality.QualitySpec{MinCompleteness: 99.0, MinAccuracy: 99.0, MaxNullRate: 0.01},
	)

	dqRepo.RecordIncident(tid, "ticket-classifier", "", "High null rate in ticket categories",
		"Null rate for category field spiked to 15%, above 10% threshold", dataquality.SeverityWarning, nil)
	dqRepo.RecordIncident(tid, "budget-reconciler", "", "Duplicate transactions detected",
		"0.5% duplicate rate detected in last batch, normally <0.01%", dataquality.SeverityCritical, nil)

	dqRepo.RecordAnomaly(tid, "budget-reconciler", "completeness", 98.5, 94.2, 4.3, dataquality.SeverityWarning)
	dqRepo.RecordAnomaly(tid, "ticket-classifier", "null_rate", 0.08, 0.15, 87.5, dataquality.SeverityCritical)
	dqRepo.RecordAnomaly(tid, "doc-summarizer", "freshness_lag_seconds", 3600, 7800, 116.7, dataquality.SeverityInfo)

	// Seed governance: classification policies, retention policies, PII scans, compliance mappings, stewards
	govRepo.CreateClassificationPolicy(tid, "PII Auto-Detect", "Automatically classify fields containing PII patterns",
		`(email|phone|ssn|national_id|name|address)`, "field_name", "restricted", true)
	govRepo.CreateClassificationPolicy(tid, "Financial Data", "Classify financial data as confidential",
		`(amount|balance|salary|revenue|cost|price)`, "field_name", "confidential", true)
	govRepo.CreateClassificationPolicy(tid, "Internal Metadata", "Classify system metadata as internal",
		`(created_at|updated_at|id|status|type)`, "field_name", "internal", true)

	govRepo.CreateRetentionPolicy(tid, "Restricted Data Retention", "Restricted data must be deleted after 90 days",
		"restricted", 90, "delete")
	govRepo.CreateRetentionPolicy(tid, "Confidential Data Retention", "Confidential data archived after 1 year",
		"confidential", 365, "archive")
	govRepo.CreateRetentionPolicy(tid, "Audit Log Retention", "Audit logs retained for 5 years per compliance",
		"audit", 1825, "archive")

	govRepo.RecordAccessLog(tid, src1.ID, "Transaction Database", "budget-reconciler", "read", "system")
	govRepo.RecordAccessLog(tid, src1.ID, "Transaction Database", "data-pipeline", "read", "system")
	govRepo.RecordAccessLog(tid, src3.ID, "Ticket Queue API", "ticket-classifier", "read_write", "system")
	govRepo.RecordAccessLog(tid, src2.ID, "Report Storage", "doc-summarizer", "write", "system")
	govRepo.RecordAccessLog(tid, src1.ID, "Transaction Database", "", "read", "admin")

	govRepo.RecordPIIScan(tid, src1.ID, "Transaction Database", []*governance.PIIField{
		{FieldName: "customer_name", PIIType: "person_name", Confidence: 0.98, SampleCount: 42000, Recommendation: "Apply encryption at rest and mask in non-prod"},
		{FieldName: "email", PIIType: "email_address", Confidence: 0.99, SampleCount: 41000, Recommendation: "Hash for analytics, encrypt for storage"},
		{FieldName: "account_id", PIIType: "account_number", Confidence: 0.85, SampleCount: 45000, Recommendation: "Tokenize for cross-system references"},
	}, 12)
	govRepo.RecordPIIScan(tid, src3.ID, "Ticket Queue API", []*governance.PIIField{
		{FieldName: "customer_email", PIIType: "email_address", Confidence: 0.99, SampleCount: 55000, Recommendation: "Hash for analytics, encrypt for storage"},
	}, 15)

	govRepo.CreateComplianceMapping(tid, src1.ID, "Transaction Database", "GDPR", "Art. 5(1)(f)",
		"Integrity and confidentiality of personal data", "compliant",
		[]string{"Encryption at rest enabled", "Access logging active", "PII fields identified"})
	govRepo.CreateComplianceMapping(tid, src1.ID, "Transaction Database", "GDPR", "Art. 17",
		"Right to erasure (right to be forgotten)", "partial",
		[]string{"Deletion workflow exists", "Pending: automated erasure on request"})
	govRepo.CreateComplianceMapping(tid, src3.ID, "Ticket Queue API", "KVKK", "Art. 12",
		"Data security obligations", "compliant",
		[]string{"mTLS enforced", "Data classified", "Access logs maintained"})
	govRepo.CreateComplianceMapping(tid, src4.ID, "ML Feature Store", "GDPR", "Art. 22",
		"Automated individual decision-making", "not_assessed",
		nil)

	govRepo.CreateSteward(tid, "sarah.chen", "Sarah Chen", "sarah.chen@company.com",
		[]string{"finance"}, []string{src1.ID}, "lead_steward")
	govRepo.CreateSteward(tid, "mike.johnson", "Mike Johnson", "mike.johnson@company.com",
		[]string{"analytics"}, []string{src2.ID}, "steward")
	govRepo.CreateSteward(tid, "alex.rivera", "Alex Rivera", "alex.rivera@company.com",
		[]string{"support"}, []string{src3.ID}, "steward")
	govRepo.CreateSteward(tid, "james.wilson", "James Wilson", "james.wilson@company.com",
		[]string{"security", "compliance"}, []string{src5.ID}, "lead_steward")

	log.Info("seeded control-plane demo data",
		zap.Int("alerts", 5),
		zap.Int("traces", len(traces)),
		zap.Int("slos", 3),
		zap.Int("eval_suites", 2),
		zap.Int("guardrail_rules", 3),
		zap.Int("rag_sources", 3),
		zap.Int("prompts", 2),
		zap.Int("feedback", len(feedbackItems)),
		zap.Int("catalog_sources", 5),
		zap.Int("glossary_terms", 5),
		zap.Int("data_contracts", 3),
		zap.Int("governance_policies", 6),
		zap.Int("pii_scans", 2),
		zap.Int("compliance_mappings", 4),
		zap.Int("data_stewards", 4),
	)
}

func strPtr(s string) *string {
	return &s
}
