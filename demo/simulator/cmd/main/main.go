package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Service URLs
var (
	orchestratorURL = env("SIMULATOR_ORCHESTRATOR_URL", "http://localhost:8082")
	controlPlaneURL = env("SIMULATOR_CONTROL_PLANE_URL", "http://localhost:8084")
	telemetryURL    = env("SIMULATOR_TELEMETRY_URL", "http://localhost:8083")
	tenantID        = "default"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Println("[simulator] waiting for services...")
	waitForHealth(ctx)
	fmt.Println("[simulator] services healthy, bootstrapping...")

	bootstrap()
	fmt.Println("[simulator] bootstrap complete, starting continuous simulation")

	var wg sync.WaitGroup
	loop := func(name string, interval time.Duration, fn func(int)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tick := 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(interval):
					tick++
					fn(tick)
				}
			}
		}()
	}

	loop("heartbeat", 30*time.Second, heartbeatAll)
	loop("tasks", 8*time.Second, taskLifecycle)
	loop("traces", 12*time.Second, generateTrace)
	loop("alerts", 40*time.Second, generateAlert)
	loop("costs", 15*time.Second, recordCost)
	loop("slos", 45*time.Second, recordSLOMeasurement)
	loop("evals", 4*time.Minute, runEval)
	loop("guardrails", 60*time.Second, checkGuardrail)
	loop("feedback", 90*time.Second, submitFeedback)
	loop("prompts", 3*time.Minute, rotatePrompt)
	loop("rag", 2*time.Minute, recordRAGRetrieval)
	loop("compliance", 8*time.Minute, generateComplianceReport)
	loop("dataquality", 30*time.Second, updateDataQuality)

	// Late agent discovery after 2 minutes
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Minute):
			registerAgent("email-drafter", "autogen", []string{"write:emails", "read:templates"})
			fmt.Println("[simulator] late agent 'email-drafter' discovered")
		}
	}()

	<-ctx.Done()
	fmt.Println("[simulator] shutting down...")
	wg.Wait()
}

// ── Health Wait ─────────────────────────────────────────────────────────

func waitForHealth(ctx context.Context) {
	services := []string{orchestratorURL, controlPlaneURL, telemetryURL}
	for _, svc := range services {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			resp, err := httpClient.Get(svc + "/health")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(2 * time.Second)
		}
	}
}

// ── HTTP Helpers ────────────────────────────────────────────────────────

func post(baseURL, path string, body interface{}) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", baseURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func put(baseURL, path string, body interface{}) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", baseURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func get(baseURL, path string) map[string]interface{} {
	req, _ := http.NewRequest("GET", baseURL+path, nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

// ── Agent Definitions ──────────────────────────────────────────────────

type agentDef struct {
	ID           string
	Framework    string
	Capabilities []string
	Stable       bool // if false, this agent will degrade
}

var agents = []agentDef{
	{"budget-reconciler", "langchain", []string{"read:budget_db", "write:reports"}, true},
	{"doc-summarizer", "autogen", []string{"read:documents", "write:summaries"}, true},
	{"ticket-classifier", "crewai", []string{"read:tickets", "write:classifications"}, false}, // degrades
	{"code-reviewer", "custom", []string{"read:repos", "write:reviews"}, true},
	{"data-pipeline", "langchain", []string{"read:raw_data", "write:processed_data"}, false}, // retry storms
	{"report-generator", "crewai", []string{"read:data_warehouse", "write:reports"}, true},
	{"fraud-detector", "langchain", []string{"read:transactions", "write:alerts"}, true},
}

var activeTasks = make(map[string]string) // taskID -> agentID
var taskCounter int
var degradePhase int // cycles 0-9, 0-3=healthy, 4-6=degraded, 7-8=failed, 9=recovery

// ── Bootstrap ──────────────────────────────────────────────────────────

func bootstrap() {
	// Register agents
	for _, a := range agents {
		registerAgent(a.ID, a.Framework, a.Capabilities)
	}

	// Create SLOs
	slos := []map[string]interface{}{
		{"name": "API Availability", "description": "Overall API uptime", "agent_id": "budget-reconciler", "type": "availability", "target": 0.999, "window": "30d"},
		{"name": "Latency P99", "description": "P99 response time under 2s", "agent_id": "doc-summarizer", "type": "latency_p99", "target": 2000, "window": "7d"},
		{"name": "Error Rate", "description": "Error rate below 1%", "agent_id": "ticket-classifier", "type": "error_rate", "target": 0.01, "window": "7d"},
		{"name": "Task Success Rate", "description": "Task completion rate above 95%", "agent_id": "data-pipeline", "type": "availability", "target": 0.95, "window": "7d"},
	}
	for _, s := range slos {
		post(controlPlaneURL, "/api/v1/slos", s)
	}

	// Create cost budgets
	budgets := []map[string]interface{}{
		{"agent_id": "budget-reconciler", "name": "Budget Agent Monthly", "budget_usd": 500, "period": "monthly", "alert_threshold": 0.8},
		{"agent_id": "doc-summarizer", "name": "Summarizer Weekly", "budget_usd": 100, "period": "weekly", "alert_threshold": 0.9},
		{"agent_id": "fraud-detector", "name": "Fraud Detection Daily", "budget_usd": 50, "period": "daily", "alert_threshold": 0.75},
	}
	for _, b := range budgets {
		post(controlPlaneURL, "/api/v1/costs/budgets", b)
	}

	// Create guardrail rules
	rules := []map[string]interface{}{
		{"name": "PII Detection", "description": "Detect personal identifiable information in agent outputs", "type": "pii_detection", "pattern": `(?i)(ssn|social.security|credit.card|\b\d{3}-\d{2}-\d{4}\b)`, "action": "block", "enabled": true, "agent_ids": []string{"budget-reconciler", "doc-summarizer"}},
		{"name": "Prompt Injection Guard", "description": "Detect prompt injection attempts", "type": "prompt_injection", "pattern": `(?i)(ignore previous|system prompt|you are now)`, "action": "block", "enabled": true, "agent_ids": []string{}},
		{"name": "Toxicity Filter", "description": "Filter toxic or harmful content", "type": "toxicity", "pattern": `(?i)(harmful|illegal|exploit)`, "action": "warn", "enabled": true, "agent_ids": []string{}},
		{"name": "Output Length Guard", "description": "Warn on excessively long outputs", "type": "custom_regex", "pattern": `.{10000,}`, "action": "warn", "enabled": true, "agent_ids": []string{"doc-summarizer"}},
	}
	for _, r := range rules {
		post(controlPlaneURL, "/api/v1/guardrails/rules", r)
	}

	// Create eval suites
	suites := []map[string]interface{}{
		{
			"name": "Budget Accuracy Suite", "description": "Tests budget reconciliation accuracy", "agent_id": "budget-reconciler",
			"test_cases": []map[string]interface{}{
				{"id": "tc-budget-1", "name": "Valid Q1 budget", "input": "Reconcile Q1 2026 budget for department A", "expected_output": "Budget balanced", "criteria": map[string]string{"accuracy": ">=0.9"}, "max_latency_ms": 5000},
				{"id": "tc-budget-2", "name": "Missing entries", "input": "Reconcile budget with 3 missing line items", "expected_output": "3 discrepancies found", "criteria": map[string]string{"accuracy": ">=0.85"}, "max_latency_ms": 8000},
				{"id": "tc-budget-3", "name": "Currency conversion", "input": "Convert and reconcile multi-currency budget", "expected_output": "Converted at market rates", "criteria": map[string]string{"accuracy": ">=0.95"}, "max_latency_ms": 10000},
			},
		},
		{
			"name": "Document Summarizer Quality", "description": "Tests summarization quality", "agent_id": "doc-summarizer",
			"test_cases": []map[string]interface{}{
				{"id": "tc-doc-1", "name": "Short document", "input": "Summarize 1-page memo", "expected_output": "Concise summary under 100 words", "criteria": map[string]string{"quality": ">=0.8"}, "max_latency_ms": 3000},
				{"id": "tc-doc-2", "name": "Long report", "input": "Summarize 50-page annual report", "expected_output": "Executive summary with key metrics", "criteria": map[string]string{"quality": ">=0.85"}, "max_latency_ms": 15000},
			},
		},
	}
	for _, s := range suites {
		post(controlPlaneURL, "/api/v1/evals/suites", s)
	}

	// Create prompts
	prompts := []map[string]interface{}{
		{"name": "Budget Analysis Prompt", "description": "System prompt for budget reconciliation agent", "agent_id": "budget-reconciler"},
		{"name": "Document Summary Prompt", "description": "System prompt for document summarization", "agent_id": "doc-summarizer"},
		{"name": "Ticket Classification Prompt", "description": "System prompt for support ticket classification", "agent_id": "ticket-classifier"},
	}
	promptIDs := []string{"sim-prompt-budget", "sim-prompt-docs", "sim-prompt-tickets"}
	for i, p := range prompts {
		post(controlPlaneURL, "/api/v1/prompts", p)
		// Add initial version
		post(controlPlaneURL, fmt.Sprintf("/api/v1/prompts/%s/versions", promptIDs[i]), map[string]interface{}{
			"content":    fmt.Sprintf("You are a specialized AI agent for %s. Follow all safety guidelines.", p["description"]),
			"change_log": "Initial version",
		})
	}

	// Create data quality rules
	dqRules := []map[string]interface{}{
		{"name": "Output completeness", "description": "Ensure agent outputs contain required fields", "type": "schema", "agent_id": "budget-reconciler", "field": "output", "operator": "not_empty", "threshold": 1.0, "severity": "critical"},
		{"name": "Latency threshold", "description": "Flag responses taking over 5 seconds", "type": "range", "agent_id": "doc-summarizer", "field": "latency_ms", "operator": "lte", "threshold": 5000, "severity": "warning"},
		{"name": "Token budget", "description": "Monitor token usage per request", "type": "range", "agent_id": "ticket-classifier", "field": "tokens_used", "operator": "lte", "threshold": 2000, "severity": "info"},
	}
	for _, r := range dqRules {
		post(controlPlaneURL, "/api/v1/dataquality/rules", r)
	}

	// Create catalog sources and lineage
	sources := []map[string]interface{}{
		{"name": "Production PostgreSQL", "description": "Main application database", "type": "database", "owner": "platform-team", "agent_id": "budget-reconciler", "tags": []string{"production", "financial"}, "schema": map[string]string{"host": "prod-db-1", "port": "5432"}},
		{"name": "Document Store S3", "description": "S3 bucket for document storage", "type": "storage", "owner": "data-team", "agent_id": "doc-summarizer", "tags": []string{"documents", "s3"}, "schema": map[string]string{"bucket": "argus-docs", "region": "eu-west-1"}},
		{"name": "Ticket API", "description": "External ticketing system REST API", "type": "api", "owner": "support-team", "agent_id": "ticket-classifier", "tags": []string{"external", "tickets"}, "schema": map[string]string{"base_url": "https://tickets.internal/api"}},
		{"name": "ML Model Registry", "description": "Model versioning and serving", "type": "api", "owner": "ml-team", "agent_id": "fraud-detector", "tags": []string{"ml", "models"}, "schema": map[string]string{"base_url": "https://models.internal"}},
	}
	for _, s := range sources {
		post(controlPlaneURL, "/api/v1/catalog/sources", s)
	}

	// Create lineage edges
	lineageEdges := []map[string]interface{}{
		{"source_id": "sim-source-1", "target_id": "sim-source-2", "transform_type": "etl", "agent_id": "budget-reconciler", "description": "Budget data flows from DB to reports"},
		{"source_id": "sim-source-3", "target_id": "sim-source-1", "transform_type": "api_call", "agent_id": "ticket-classifier", "description": "Tickets classified and stored in DB"},
	}
	for _, e := range lineageEdges {
		post(controlPlaneURL, "/api/v1/catalog/lineage", e)
	}

	// Create RAG sources
	ragSources := []map[string]interface{}{
		{"name": "Financial Regulations KB", "type": "document", "total_chunks": 8500, "avg_relevance": 0.87, "usage_count": 450},
		{"name": "Company Policies", "type": "database", "total_chunks": 3200, "avg_relevance": 0.92, "usage_count": 280},
		{"name": "Support Knowledge Base", "type": "document", "total_chunks": 12000, "avg_relevance": 0.78, "usage_count": 890},
	}
	for _, s := range ragSources {
		post(controlPlaneURL, "/api/v1/rag/sources", s)
	}

	// Generate initial compliance report
	post(controlPlaneURL, "/api/v1/compliance/reports", map[string]interface{}{
		"profile_id":   "gov-tr",
		"period_start": time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		"period_end":   time.Now().Format(time.RFC3339),
	})

	fmt.Println("[simulator] registered", len(agents), "agents + SLOs + budgets + guardrails + evals + prompts + DQ + catalog + RAG + compliance")
}

func registerAgent(id, framework string, capabilities []string) {
	post(orchestratorURL, "/api/v1/agents", map[string]interface{}{
		"agent_id":     id,
		"version":      "2.1.0",
		"framework":    framework,
		"capabilities": capabilities,
		"node_id":      "node-" + id[:3],
	})
}

// ── Continuous Generators ──────────────────────────────────────────────

func heartbeatAll(tick int) {
	degradePhase = tick % 20 // cycle every ~10 minutes (20 * 30s)
	for _, a := range agents {
		status := "healthy"
		if !a.Stable {
			if a.ID == "ticket-classifier" {
				switch {
				case degradePhase >= 8 && degradePhase <= 12:
					status = "degraded"
				case degradePhase >= 13 && degradePhase <= 16:
					status = "failed"
				case degradePhase >= 17 && degradePhase <= 18:
					status = "quarantined"
				}
			} else if a.ID == "data-pipeline" {
				if degradePhase >= 10 && degradePhase <= 14 {
					status = "degraded"
				}
			}
		}
		post(orchestratorURL, fmt.Sprintf("/api/v1/agents/%s/heartbeat", a.ID), map[string]string{"status": status})
	}
}

func taskLifecycle(tick int) {
	// Create a new task
	agent := agents[rand.Intn(len(agents))]
	taskCounter++
	taskID := fmt.Sprintf("sim-task-%d-%d", time.Now().Unix(), taskCounter)

	post(orchestratorURL, "/api/v1/tasks", map[string]interface{}{
		"input_hash":            fmt.Sprintf("%x", taskID),
		"required_capabilities": agent.Capabilities[:1],
		"preferred_agent_id":    agent.ID,
	})

	activeTasks[taskID] = agent.ID

	// Complete some existing tasks
	for id, agentID := range activeTasks {
		if rand.Float64() < 0.4 { // 40% chance to complete each tick
			status := "completed"
			if rand.Float64() < 0.08 { // 8% failure rate
				status = "failed"
			}
			// Higher failure for degrading agents
			if agentID == "ticket-classifier" && degradePhase >= 13 {
				if rand.Float64() < 0.6 {
					status = "failed"
				}
			}

			cost := 0.01 + rand.Float64()*0.5
			tokens := 200 + rand.Intn(3000)
			put(orchestratorURL, fmt.Sprintf("/api/v1/tasks/%s", id), map[string]interface{}{
				"status":      status,
				"cost_usd":    cost,
				"tokens_used": tokens,
			})
			delete(activeTasks, id)
			break // complete one per tick
		}
	}
}

func generateTrace(tick int) {
	agent := agents[rand.Intn(len(agents))]
	traceID := fmt.Sprintf("sim-trace-%d-%d", time.Now().Unix(), tick)
	now := time.Now()

	operations := []struct {
		name     string
		duration int
	}{
		{"fetch_data", 100 + rand.Intn(200)},
		{"validate_input", 10 + rand.Intn(30)},
		{"run_inference", 500 + rand.Intn(1500)},
		{"generate_embedding", 100 + rand.Intn(300)},
		{"compose_response", 200 + rand.Intn(500)},
		{"write_result", 30 + rand.Intn(100)},
	}

	// Apply degradation multiplier
	latencyMult := 1.0
	hasError := false
	if agent.ID == "ticket-classifier" && degradePhase >= 8 {
		latencyMult = 2.0 + float64(degradePhase-8)*0.5
		if degradePhase >= 13 {
			hasError = rand.Float64() < 0.5
		}
	}

	numSpans := 3 + rand.Intn(4) // 3-6 spans
	if numSpans > len(operations) {
		numSpans = len(operations)
	}

	spans := make([]map[string]interface{}, numSpans)
	totalDuration := 0
	for i := 0; i < numSpans; i++ {
		op := operations[i]
		dur := int(float64(op.duration) * latencyMult)
		totalDuration += dur

		span := map[string]interface{}{
			"span_id":        fmt.Sprintf("%s-span-%d", traceID, i),
			"trace_id":       traceID,
			"agent_id":       agent.ID,
			"operation_name": op.name,
			"started_at":     now.Add(time.Duration(totalDuration-dur) * time.Millisecond).Format(time.RFC3339Nano),
			"duration_ms":    dur,
			"attributes": map[string]string{
				"model":    pickModel(),
				"tokens":   fmt.Sprintf("%d", 100+rand.Intn(500)),
				"agent_id": agent.ID,
			},
		}
		if hasError && i == numSpans-1 {
			span["error_code"] = pickError()
		}
		spans[i] = span
	}

	// Send to telemetry service (ingestion pipeline)
	post(telemetryURL, "/api/v1/telemetry/spans", map[string]interface{}{"spans": spans})

	// Also post trace to control-plane for dashboard viewing
	post(controlPlaneURL, "/api/v1/traces", map[string]interface{}{
		"trace_id":       traceID,
		"agent_id":       agent.ID,
		"root_operation": operations[0].name,
		"total_spans":    numSpans,
		"duration_ms":    totalDuration,
		"has_errors":     hasError,
		"started_at":     now.Format(time.RFC3339),
		"spans":          spans,
	})
}

func generateAlert(tick int) {
	// Normal alert from degrading agent
	if degradePhase >= 8 && degradePhase <= 16 {
		severity := "warning"
		if degradePhase >= 13 {
			severity = "critical"
		}
		precursor := "latency_spike"
		if degradePhase >= 10 {
			precursor = "token_escalation"
		}

		prob := 0.4 + float64(degradePhase-8)*0.08
		if prob > 1.0 {
			prob = 0.98
		}

		post(controlPlaneURL, "/api/v1/alerts", map[string]interface{}{
			"agent_id":       "ticket-classifier",
			"severity":       severity,
			"title":          fmt.Sprintf("Agent ticket-classifier %s detected", precursor),
			"message":        fmt.Sprintf("Failure probability %.0f%%, precursor: %s", prob*100, precursor),
			"probability":    prob,
			"estimated_ttf":  300 - (degradePhase-8)*30,
			"precursor_type": precursor,
		})
	}

	// Send predictive features to predictor
	p99Ratio := 1.5 + rand.Float64()*0.5
	retryRate := 0.02 + rand.Float64()*0.03
	contextFill := 0.3 + rand.Float64()*0.1
	if degradePhase >= 8 && degradePhase <= 16 {
		p99Ratio = 3.0 + float64(degradePhase-8)*0.5
		retryRate = 0.1 + float64(degradePhase-8)*0.06
		contextFill = 0.6 + float64(degradePhase-8)*0.04
	}

	post(telemetryURL, "/api/v1/telemetry/predict", map[string]interface{}{
		"latency_p99_ratio": p99Ratio,
		"token_velocity":    20 + rand.Float64()*30,
		"retry_rate":        retryRate,
		"error_rate_delta":  0.001 + rand.Float64()*0.01,
		"context_fill_pct":  contextFill,
		"tool_call_depth":   1.0 + rand.Float64()*3,
		"consecutive_slow":  float64(rand.Intn(3)),
		"cost_acceleration": 0.8 + rand.Float64()*0.5,
	})

	// Resolve old alerts occasionally
	if tick%5 == 0 {
		// Try to acknowledge or resolve some alerts
		result := get(controlPlaneURL, "/api/v1/alerts")
		if result != nil {
			if data, ok := result["data"].([]interface{}); ok {
				for _, item := range data {
					if alert, ok := item.(map[string]interface{}); ok {
						if alert["status"] == "open" && rand.Float64() < 0.2 {
							if id, ok := alert["id"].(string); ok {
								put(controlPlaneURL, fmt.Sprintf("/api/v1/alerts/%s", id), map[string]string{"status": "acknowledged"})
							}
						}
					}
				}
			}
		}
	}
}

func recordCost(tick int) {
	agent := agents[rand.Intn(len(agents))]
	model := pickModel()
	cost := 0.005 + rand.Float64()*0.3
	tokens := 100 + rand.Intn(2000)

	// Record via cost trends (POST to costs endpoint)
	post(controlPlaneURL, "/api/v1/costs/record", map[string]interface{}{
		"agent_id":    agent.ID,
		"model":       model,
		"cost_usd":    cost,
		"tokens_used": tokens,
		"category":    pickCategory(),
	})
}

func recordSLOMeasurement(tick int) {
	// SLO measurements are derived from task completion data
	// The SLO status endpoint computes from task data, so just
	// ensure tasks keep flowing (handled by taskLifecycle)
	// We can also query SLO status to trigger budget calculations
	get(controlPlaneURL, "/api/v1/slos/status")
}

func runEval(tick int) {
	// Run eval on first suite
	suites := get(controlPlaneURL, "/api/v1/evals/suites")
	if suites == nil {
		return
	}
	data, ok := suites["data"].([]interface{})
	if !ok || len(data) == 0 {
		return
	}
	suite, ok := data[rand.Intn(len(data))].(map[string]interface{})
	if !ok {
		return
	}
	suiteID, ok := suite["id"].(string)
	if !ok {
		return
	}
	post(controlPlaneURL, fmt.Sprintf("/api/v1/evals/suites/%s/run", suiteID), map[string]interface{}{})
	fmt.Printf("[simulator] eval run triggered for suite %s\n", suiteID)
}

func checkGuardrail(tick int) {
	// Guardrail violations are generated by the telemetry pipeline when spans
	// contain suspicious content. We inject some via traces.
	// Also directly create a violation for visibility
	if rand.Float64() < 0.3 {
		agent := agents[rand.Intn(len(agents))]
		violations := []string{
			"Detected PII: SSN pattern 123-45-6789 in output",
			"Prompt injection attempt: 'ignore previous instructions'",
			"Output exceeds 10000 character limit",
			"Potentially harmful content detected in response",
		}
		post(controlPlaneURL, "/api/v1/guardrails/violations", map[string]interface{}{
			"rule_id":  fmt.Sprintf("rule-%d", 1+rand.Intn(4)),
			"agent_id": agent.ID,
			"span_id":  fmt.Sprintf("sim-span-%d", time.Now().UnixNano()),
			"action":   []string{"block", "warn", "log"}[rand.Intn(3)],
			"content":  violations[rand.Intn(len(violations))],
		})
	}
}

func submitFeedback(tick int) {
	agent := agents[rand.Intn(len(agents))]
	rating := 1 // positive
	if rand.Float64() < 0.2 {
		rating = -1 // 20% negative
	}
	comments := []string{
		"Great results, very accurate",
		"Response was fast and helpful",
		"Could be more detailed",
		"Incorrect classification, needs improvement",
		"Perfect analysis of the budget data",
		"Missed some edge cases",
		"Excellent summarization quality",
		"Too slow for production use",
		"Good but formatting could be better",
		"Outstanding performance on complex task",
	}
	post(controlPlaneURL, "/api/v1/feedback", map[string]interface{}{
		"agent_id": agent.ID,
		"task_id":  fmt.Sprintf("sim-task-%d", time.Now().Unix()),
		"rating":   rating,
		"comment":  comments[rand.Intn(len(comments))],
		"user_id":  fmt.Sprintf("user-%d", 1+rand.Intn(5)),
	})
}

func rotatePrompt(tick int) {
	// Create a new version for a random prompt
	promptIDs := []string{"sim-prompt-budget", "sim-prompt-docs", "sim-prompt-tickets"}
	pid := promptIDs[rand.Intn(len(promptIDs))]
	version := tick + 2 // start from v2

	changes := []string{
		"Improved accuracy for edge cases",
		"Added output format instructions",
		"Reduced hallucination rate",
		"Optimized for lower token usage",
		"Added safety guidelines",
		"Enhanced multi-language support",
	}

	post(controlPlaneURL, fmt.Sprintf("/api/v1/prompts/%s/versions", pid), map[string]interface{}{
		"content":    fmt.Sprintf("You are a specialized AI agent. Version %d with improvements. Follow all guidelines strictly.", version),
		"change_log": changes[rand.Intn(len(changes))],
	})
}

func recordRAGRetrieval(tick int) {
	agent := agents[rand.Intn(len(agents))]
	queries := []string{
		"What are the budget allocation rules for Q1?",
		"Find relevant compliance regulations for data storage",
		"Retrieve customer complaint patterns from last month",
		"What are the fraud detection thresholds?",
		"Find document templates for annual reports",
		"Search for security incident response procedures",
	}

	post(controlPlaneURL, "/api/v1/rag/retrievals", map[string]interface{}{
		"agent_id":      agent.ID,
		"span_id":       fmt.Sprintf("sim-span-%d", time.Now().UnixNano()),
		"query":         queries[rand.Intn(len(queries))],
		"num_chunks":    5 + rand.Intn(20),
		"avg_relevance": 0.6 + rand.Float64()*0.35,
		"latency_ms":    50 + rand.Intn(400),
		"source_ids":    []string{fmt.Sprintf("src-%d", 1+rand.Intn(3))},
	})
}

func generateComplianceReport(tick int) {
	profiles := []string{"gov-tr", "eu-gdpr", "fedramp-moderate", "gcc-sa"}
	post(controlPlaneURL, "/api/v1/compliance/reports", map[string]interface{}{
		"profile_id":   profiles[rand.Intn(len(profiles))],
		"period_start": time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		"period_end":   time.Now().Format(time.RFC3339),
	})
}

func updateDataQuality(tick int) {
	// Data quality scores are computed by the telemetry pipeline from spans.
	// The spans we send via generateTrace feed into the DQ system.
	// We also send quality data points directly for richer dashboard display.
	agent := agents[rand.Intn(len(agents))]

	completeness := 0.85 + rand.Float64()*0.15
	consistency := 0.80 + rand.Float64()*0.18
	timeliness := 0.90 + rand.Float64()*0.10
	validity := 0.88 + rand.Float64()*0.12

	// Degrade quality for unstable agents
	if agent.ID == "ticket-classifier" && degradePhase >= 8 {
		completeness -= 0.15
		consistency -= 0.20
	}

	post(controlPlaneURL, "/api/v1/dataquality/scores", map[string]interface{}{
		"agent_id":     agent.ID,
		"completeness": math.Max(0, completeness),
		"consistency":  math.Max(0, consistency),
		"timeliness":   timeliness,
		"validity":     validity,
	})
}

// ── Helpers ─────────────────────────────────────────────────────────────

func pickModel() string {
	models := []string{"gpt-4o", "claude-3.5-sonnet", "gpt-4o-mini", "claude-3-haiku", "gemini-1.5-pro"}
	return models[rand.Intn(len(models))]
}

func pickCategory() string {
	cats := []string{"inference", "embedding", "fine-tuning", "retrieval"}
	return cats[rand.Intn(len(cats))]
}

func pickError() string {
	errors := []string{"TIMEOUT", "CONTEXT_OVERFLOW", "RATE_LIMITED", "MODEL_ERROR", "TOOL_FAILURE"}
	return errors[rand.Intn(len(errors))]
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// Support comma-separated tenant list but use first one as default
	if t := os.Getenv("SIMULATOR_TENANTS"); t != "" {
		parts := strings.SplitN(t, ",", 2)
		tenantID = parts[0]
	}
}
