package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	agentv1 "github.com/argus-platform/argus/gen/go/agent"
	orchestrationv1 "github.com/argus-platform/argus/gen/go/orchestration"
	"github.com/argus-platform/argus/pkg/config"
	"github.com/argus-platform/argus/pkg/database"
	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/logger"
	"github.com/argus-platform/argus/pkg/middleware"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/orchestrator/internal/costtracker"
	"github.com/argus-platform/argus/services/orchestrator/internal/grpchandler"
	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
	"github.com/argus-platform/argus/services/orchestrator/internal/repository"
	"github.com/argus-platform/argus/services/orchestrator/internal/router"
	"github.com/argus-platform/argus/services/orchestrator/internal/statemachine"
	"github.com/argus-platform/argus/services/orchestrator/internal/versioning"
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

	agentRegistry := registry.New()
	taskRouter := router.New(agentRegistry)
	sm := statemachine.New()
	costs := costtracker.New()
	versions := versioning.New()

	// Seed demo agents so the dashboard shows data on first launch
	if os.Getenv("ARGUS_ENV") != "production" {
		seedDemoData(agentRegistry, sm, costs, versions, log)
	}

	// Initialize PostgreSQL persistence if DSN is configured
	var agentRepo *repository.AgentRepository
	var taskRepo *repository.TaskRepository
	if dsn := os.Getenv("ARGUS_DB_DSN"); dsn != "" {
		ctx := context.Background()
		pool, err := database.NewPool(ctx, dsn)
		if err != nil {
			log.Warn("failed to connect to database, using in-memory stores", zap.Error(err))
		} else {
			agentRepo = repository.NewAgentRepository(pool)
			taskRepo = repository.NewTaskRepository(pool)
			log.Info("PostgreSQL persistence enabled")
			defer pool.Close()
		}
	} else {
		log.Info("no ARGUS_DB_DSN set, using in-memory stores")
	}

	// gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.TenantUnaryInterceptor()),
	)

	// Register gRPC service handlers
	agentGRPC := grpchandler.NewAgentHandler(agentRegistry)
	agentv1.RegisterAgentServiceServer(grpcServer, agentGRPC)

	orchGRPC := grpchandler.NewOrchestrationHandler(sm, taskRouter)
	orchestrationv1.RegisterOrchestrationServiceServer(grpcServer, orchGRPC)

	grpcLis, err := net.Listen("tcp", ":9082")
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		log.Info("orchestrator gRPC server starting", zap.String("addr", ":9082"))
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// HTTP server with REST API
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Agent endpoints
	mux.Handle("/api/v1/agents", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		switch r.Method {
		case http.MethodGet:
			agents := agentRegistry.List(tenantID)
			httputil.WriteJSON(w, http.StatusOK, agents, tenantID)
		case http.MethodPost:
			var req registry.RegisterRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
				return
			}
			agent := agentRegistry.Register(tenantID, &req)
			versions.Set(tenantID, req.AgentID, req.Version, false)
			if agentRepo != nil {
				if _, err := agentRepo.Register(r.Context(), tenantID, &req); err != nil {
					log.Error("failed to persist agent to DB", zap.Error(err))
				}
			}
			httputil.WriteJSON(w, http.StatusCreated, agent, tenantID)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
	})))

	// Single agent endpoint
	mux.Handle("/api/v1/agents/", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		agentID := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
		if agentID == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "agent ID required")
			return
		}

		// Handle heartbeat sub-path
		if strings.HasSuffix(agentID, "/heartbeat") {
			agentID = strings.TrimSuffix(agentID, "/heartbeat")
			if r.Method == http.MethodPost {
				var req struct {
					Status string `json:"status"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
					return
				}
				if err := agentRegistry.Heartbeat(tenantID, agentID, registry.AgentStatus(req.Status)); err != nil {
					httputil.WriteError(w, http.StatusNotFound, "AGENT_NOT_FOUND", err.Error())
					return
				}
				if agentRepo != nil {
					if err := agentRepo.Heartbeat(r.Context(), tenantID, agentID, registry.AgentStatus(req.Status)); err != nil {
						log.Error("failed to persist heartbeat to DB", zap.Error(err))
					}
				}
				httputil.WriteJSON(w, http.StatusOK, map[string]bool{"acknowledged": true}, tenantID)
				return
			}
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		// Handle quarantine sub-path — used by the auto-quarantine pipeline
		// in the telemetry service when predictive failure probability > 0.9
		if strings.HasSuffix(agentID, "/quarantine") {
			agentID = strings.TrimSuffix(agentID, "/quarantine")
			if r.Method == http.MethodPost {
				if err := agentRegistry.QuarantineAgent(tenantID, agentID); err != nil {
					httputil.WriteError(w, http.StatusNotFound, "AGENT_NOT_FOUND", err.Error())
					return
				}
				if agentRepo != nil {
					if err := agentRepo.Quarantine(r.Context(), tenantID, agentID); err != nil {
						log.Error("failed to persist quarantine to DB", zap.Error(err))
					}
				}
				log.Info("agent quarantined",
					zap.String("tenant_id", tenantID),
					zap.String("agent_id", agentID),
				)
				agent, err := agentRegistry.Get(tenantID, agentID)
				if err != nil {
					httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "agent quarantined but failed to retrieve updated state")
					return
				}
				httputil.WriteJSON(w, http.StatusOK, agent, tenantID)
				return
			}
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		switch r.Method {
		case http.MethodGet:
			agent, err := agentRegistry.Get(tenantID, agentID)
			if err != nil {
				httputil.WriteError(w, http.StatusNotFound, "AGENT_NOT_FOUND", err.Error())
				return
			}
			httputil.WriteJSON(w, http.StatusOK, agent, tenantID)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
	})))

	// Task endpoints
	mux.Handle("/api/v1/tasks", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		switch r.Method {
		case http.MethodGet:
			tasks := sm.ListByTenant(tenantID)
			httputil.WriteJSON(w, http.StatusOK, tasks, tenantID)
		case http.MethodPost:
			var req struct {
				InputHash            string   `json:"input_hash"`
				RequiredCapabilities []string `json:"required_capabilities"`
				PreferredAgentID     string   `json:"preferred_agent_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
				return
			}

			// Hash the input if not provided
			inputHash := req.InputHash
			if inputHash == "" {
				h := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", tenantID, time.Now().UnixNano())))
				inputHash = hex.EncodeToString(h[:])
			}

			// Route to an agent
			agent, err := taskRouter.Route(tenantID, req.RequiredCapabilities, req.PreferredAgentID)
			if err != nil {
				httputil.WriteError(w, http.StatusUnprocessableEntity, "NO_AGENT_AVAILABLE", err.Error())
				return
			}

			taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
			task := sm.CreateTask(taskID, tenantID, agent.ID, inputHash)
			if taskRepo != nil {
				if err := taskRepo.Create(r.Context(), task); err != nil {
					log.Error("failed to persist task to DB", zap.Error(err))
				}
			}
			httputil.WriteJSON(w, http.StatusCreated, task, tenantID)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
	})))

	// Single task endpoint
	mux.Handle("/api/v1/tasks/", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		taskID := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
		if taskID == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "task ID required")
			return
		}

		switch r.Method {
		case http.MethodGet:
			task, err := sm.Get(taskID)
			if err != nil {
				httputil.WriteError(w, http.StatusNotFound, "TASK_NOT_FOUND", err.Error())
				return
			}
			if task.TenantID != tenantID {
				httputil.WriteError(w, http.StatusForbidden, "FORBIDDEN", "cross-tenant access denied")
				return
			}
			httputil.WriteJSON(w, http.StatusOK, task, tenantID)
		case http.MethodPut:
			var req struct {
				Status     string  `json:"status"`
				CostUSD    float64 `json:"cost_usd"`
				TokensUsed int64   `json:"tokens_used"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
				return
			}

			task, err := sm.Get(taskID)
			if err != nil {
				httputil.WriteError(w, http.StatusNotFound, "TASK_NOT_FOUND", err.Error())
				return
			}
			if task.TenantID != tenantID {
				httputil.WriteError(w, http.StatusForbidden, "FORBIDDEN", "cross-tenant access denied")
				return
			}

			if err := sm.Transition(taskID, statemachine.TaskStatus(req.Status)); err != nil {
				httputil.WriteError(w, http.StatusBadRequest, "INVALID_TRANSITION", err.Error())
				return
			}
			if taskRepo != nil {
				if err := taskRepo.UpdateStatus(r.Context(), tenantID, taskID, statemachine.TaskStatus(req.Status)); err != nil {
					log.Error("failed to persist task status to DB", zap.Error(err))
				}
			}

			if req.CostUSD > 0 {
				costs.Record(tenantID, task.AgentID, req.CostUSD)
				if taskRepo != nil {
					if err := taskRepo.UpdateCost(r.Context(), tenantID, taskID, req.CostUSD, req.TokensUsed); err != nil {
						log.Error("failed to persist task cost to DB", zap.Error(err))
					}
				}
			}

			updated, _ := sm.Get(taskID)
			httputil.WriteJSON(w, http.StatusOK, updated, tenantID)
		default:
			httputil.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
	})))

	// Metrics endpoint
	mux.Handle("/api/v1/metrics", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		agents := agentRegistry.List(tenantID)
		tasks := sm.ListByTenant(tenantID)

		activeTasks := 0
		for _, t := range tasks {
			if t.Status == statemachine.StatusRunning {
				activeTasks++
			}
		}

		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"total_agents": len(agents),
			"active_tasks": activeTasks,
			"total_tasks":  len(tasks),
			"total_cost":   costs.GetTenantCost(tenantID),
		}, tenantID)
	})))

	handler := middleware.CORS(middleware.RequestLogger(log)(mux))

	httpSrv := &http.Server{
		Addr:         ":8082",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("orchestrator HTTP server starting", zap.String("addr", ":8082"))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down orchestrator")
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}
	_ = cfg
}

func seedDemoData(reg *registry.Registry, sm *statemachine.StateMachine, costs *costtracker.Tracker, vers *versioning.Tracker, log *zap.Logger) {
	tenantID := "default"

	demoAgents := []registry.RegisterRequest{
		{AgentID: "budget-reconciler", Version: "1.2.0", Framework: "langchain", Capabilities: []string{"read:budget_db", "write:report_store"}, NodeID: "node-1"},
		{AgentID: "doc-summarizer", Version: "2.0.1", Framework: "autogen", Capabilities: []string{"read:documents", "write:summaries"}, NodeID: "node-1"},
		{AgentID: "ticket-classifier", Version: "1.0.0", Framework: "crewai", Capabilities: []string{"read:tickets", "write:classifications"}, NodeID: "node-2"},
		{AgentID: "code-reviewer", Version: "3.1.0", Framework: "custom", Capabilities: []string{"read:repos", "write:reviews"}, NodeID: "node-2"},
		{AgentID: "data-pipeline", Version: "1.5.2", Framework: "langchain", Capabilities: []string{"read:raw_data", "write:processed_data", "execute:transforms"}, NodeID: "node-3"},
	}

	statuses := []registry.AgentStatus{
		registry.StatusHealthy,
		registry.StatusHealthy,
		registry.StatusDegraded,
		registry.StatusHealthy,
		registry.StatusHealthy,
	}

	for i, req := range demoAgents {
		agent := reg.Register(tenantID, &req)
		agent.Status = statuses[i]
		vers.Set(tenantID, req.AgentID, req.Version, false)
	}

	// Create some demo tasks
	taskDefs := []struct {
		agentID string
		status  statemachine.TaskStatus
		cost    float64
	}{
		{"budget-reconciler", statemachine.StatusCompleted, 0.45},
		{"budget-reconciler", statemachine.StatusCompleted, 0.32},
		{"doc-summarizer", statemachine.StatusRunning, 0.0},
		{"ticket-classifier", statemachine.StatusFailed, 0.12},
		{"code-reviewer", statemachine.StatusCompleted, 1.20},
		{"data-pipeline", statemachine.StatusCompleted, 0.78},
		{"data-pipeline", statemachine.StatusRunning, 0.0},
	}

	for i, td := range taskDefs {
		taskID := fmt.Sprintf("demo-task-%d", i+1)
		inputHash := fmt.Sprintf("%x", sha256.Sum256([]byte(taskID)))
		task := sm.CreateTask(taskID, tenantID, td.agentID, inputHash)
		// Transition through states
		_ = sm.Transition(taskID, statemachine.StatusRunning)
		if td.status == statemachine.StatusCompleted || td.status == statemachine.StatusFailed {
			_ = sm.Transition(taskID, td.status)
		}
		if td.cost > 0 {
			costs.Record(tenantID, td.agentID, td.cost)
		}
		_ = task
	}

	log.Info("seeded demo data", zap.Int("agents", len(demoAgents)), zap.Int("tasks", len(taskDefs)))
}

