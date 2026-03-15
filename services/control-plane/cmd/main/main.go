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
	"github.com/argus-platform/argus/services/control-plane/internal/alerts"
	"github.com/argus-platform/argus/services/control-plane/internal/audit"
	"github.com/argus-platform/argus/services/control-plane/internal/auth"
	"github.com/argus-platform/argus/services/control-plane/internal/dashboard"
	"github.com/argus-platform/argus/services/control-plane/internal/policy"
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

	// Initialize components
	auditLog := audit.NewWriter()
	alertRouter := alerts.NewRouter()
	policyEngine := policy.New()
	jwtAuth := auth.New(os.Getenv("ARGUS_JWT_SECRET"))
	dashHandler := dashboard.New(alertRouter, auditLog)

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
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Dashboard endpoints (alerts, audit)
	dashHandler.RegisterRoutes(mux)

	// Auth endpoint - generate tokens (dev mode)
	mux.HandleFunc("/api/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			Subject  string `json:"subject"`
			TenantID string `json:"tenant_id"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
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
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
			return
		}

		auditLog.Write(req.TenantID, req.Subject, "generate_token", "auth/token", "role: "+req.Role)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"token":      token,
				"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
		})
	})

	// Policy endpoints
	mux.Handle("/api/v1/policies", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		switch r.Method {
		case http.MethodGet:
			rules := policyEngine.ListRules(tenantID)
			writeJSON(w, http.StatusOK, rules, tenantID)
		case http.MethodPost:
			var rule policy.Rule
			if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
				return
			}
			policyEngine.AddRule(tenantID, &rule)
			auditLog.Write(tenantID, "system", "create_policy", "policy/"+rule.ID, "")
			writeJSON(w, http.StatusCreated, rule, tenantID)
		default:
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
	})))

	// Policy evaluation endpoint
	mux.Handle("/api/v1/policies/evaluate", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		var req struct {
			Subject  string `json:"subject"`
			Action   string `json:"action"`
			Resource string `json:"resource"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
			return
		}

		allowed, err := policyEngine.Evaluate(tenantID, req.Subject, policy.Action(req.Action), req.Resource)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]bool{"allowed": allowed}, tenantID)
	})))

	// Metrics endpoint
	mux.Handle("/api/v1/metrics", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"total_agents": 0,
			"active_tasks": 0,
			"total_cost":   0.0,
			"alert_count":  alertRouter.Count(tenantID),
		}, tenantID)
	})))

	// Wrap with auth + tenant + logging middleware
	handler := middleware.CORS(
		jwtAuth.Middleware(
			middleware.RequestLogger(log)(mux),
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
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}
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
