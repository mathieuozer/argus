package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/argus-platform/argus/pkg/httputil"
	"github.com/argus-platform/argus/pkg/middleware"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
)

// buildAgentMux creates the /api/v1/agents/ handler with quarantine support,
// mirroring the handler from main.go but without database persistence so it
// can be unit-tested without external dependencies.
func buildAgentMux(agentRegistry *registry.Registry) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/api/v1/agents/", middleware.TenantHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := tenancy.FromContext(r.Context())
		agentID := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
		if agentID == "" {
			httputil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "agent ID required")
			return
		}

		// Handle quarantine sub-path
		if strings.HasSuffix(agentID, "/quarantine") {
			agentID = strings.TrimSuffix(agentID, "/quarantine")
			if r.Method == http.MethodPost {
				if err := agentRegistry.QuarantineAgent(tenantID, agentID); err != nil {
					httputil.WriteError(w, http.StatusNotFound, "AGENT_NOT_FOUND", err.Error())
					return
				}
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

		// Handle GET for single agent
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

	return mux
}

func TestQuarantineEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		tenantID       string
		agentID        string
		setup          func(*registry.Registry)
		wantStatus     int
		wantErrCode    string
		wantAgentState registry.AgentStatus
	}{
		{
			name:     "successfully quarantines an agent",
			method:   http.MethodPost,
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(r *registry.Registry) {
				r.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-1",
					Version:      "1.0.0",
					Framework:    "langchain",
					Capabilities: []string{"read:db"},
					NodeID:       "node-1",
				})
				_ = r.Heartbeat("tenant-1", "agent-1", registry.StatusHealthy)
			},
			wantStatus:     http.StatusOK,
			wantAgentState: registry.StatusQuarantined,
		},
		{
			name:     "returns 404 for nonexistent agent",
			method:   http.MethodPost,
			tenantID: "tenant-1",
			agentID:  "nonexistent",
			setup: func(r *registry.Registry) {
				r.Register("tenant-1", &registry.RegisterRequest{
					AgentID: "agent-1",
					Version: "1.0.0",
				})
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "AGENT_NOT_FOUND",
		},
		{
			name:     "returns 404 for nonexistent tenant",
			method:   http.MethodPost,
			tenantID: "nonexistent-tenant",
			agentID:  "agent-1",
			setup: func(r *registry.Registry) {
				r.Register("tenant-1", &registry.RegisterRequest{
					AgentID: "agent-1",
					Version: "1.0.0",
				})
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "AGENT_NOT_FOUND",
		},
		{
			name:     "returns 405 for GET method",
			method:   http.MethodGet,
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(r *registry.Registry) {
				r.Register("tenant-1", &registry.RegisterRequest{
					AgentID: "agent-1",
					Version: "1.0.0",
				})
			},
			wantStatus:  http.StatusMethodNotAllowed,
			wantErrCode: "METHOD_NOT_ALLOWED",
		},
		{
			name:     "returns 405 for PUT method",
			method:   http.MethodPut,
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(r *registry.Registry) {
				r.Register("tenant-1", &registry.RegisterRequest{
					AgentID: "agent-1",
					Version: "1.0.0",
				})
			},
			wantStatus:  http.StatusMethodNotAllowed,
			wantErrCode: "METHOD_NOT_ALLOWED",
		},
		{
			name:     "returns 405 for DELETE method",
			method:   http.MethodDelete,
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(r *registry.Registry) {
				r.Register("tenant-1", &registry.RegisterRequest{
					AgentID: "agent-1",
					Version: "1.0.0",
				})
			},
			wantStatus:  http.StatusMethodNotAllowed,
			wantErrCode: "METHOD_NOT_ALLOWED",
		},
		{
			name:     "cross-tenant quarantine is denied",
			method:   http.MethodPost,
			tenantID: "tenant-2",
			agentID:  "agent-1",
			setup: func(r *registry.Registry) {
				r.Register("tenant-1", &registry.RegisterRequest{
					AgentID: "agent-1",
					Version: "1.0.0",
				})
			},
			wantStatus:  http.StatusNotFound,
			wantErrCode: "AGENT_NOT_FOUND",
		},
		{
			name:     "quarantines already degraded agent",
			method:   http.MethodPost,
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(r *registry.Registry) {
				r.Register("tenant-1", &registry.RegisterRequest{
					AgentID:      "agent-1",
					Version:      "1.0.0",
					Framework:    "autogen",
					Capabilities: []string{"write:report"},
					NodeID:       "node-2",
				})
				_ = r.Heartbeat("tenant-1", "agent-1", registry.StatusDegraded)
			},
			wantStatus:     http.StatusOK,
			wantAgentState: registry.StatusQuarantined,
		},
		{
			name:     "quarantining already quarantined agent is idempotent",
			method:   http.MethodPost,
			tenantID: "tenant-1",
			agentID:  "agent-1",
			setup: func(r *registry.Registry) {
				r.Register("tenant-1", &registry.RegisterRequest{
					AgentID: "agent-1",
					Version: "1.0.0",
				})
				_ = r.QuarantineAgent("tenant-1", "agent-1")
			},
			wantStatus:     http.StatusOK,
			wantAgentState: registry.StatusQuarantined,
		},
		{
			name:     "missing tenant header returns 400",
			method:   http.MethodPost,
			tenantID: "", // empty = no header
			agentID:  "agent-1",
			setup: func(r *registry.Registry) {
				r.Register("tenant-1", &registry.RegisterRequest{
					AgentID: "agent-1",
					Version: "1.0.0",
				})
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := registry.New()
			tc.setup(reg)

			handler := buildAgentMux(reg)

			url := "/api/v1/agents/" + tc.agentID + "/quarantine"
			req := httptest.NewRequest(tc.method, url, nil)
			if tc.tenantID != "" {
				req.Header.Set("X-Tenant-ID", tc.tenantID)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tc.wantStatus, rr.Code, rr.Body.String())
			}

			// Skip JSON validation for non-JSON responses (e.g. middleware text errors)
			if rr.Code == http.StatusBadRequest && tc.tenantID == "" {
				return
			}

			var body map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}

			if tc.wantErrCode != "" {
				errObj, ok := body["error"].(map[string]interface{})
				if !ok {
					t.Fatalf("expected error object in response, got: %v", body)
				}
				code, _ := errObj["code"].(string)
				if code != tc.wantErrCode {
					t.Errorf("expected error code %q, got %q", tc.wantErrCode, code)
				}
			}

			if tc.wantAgentState != "" {
				// Verify the response data contains the agent with quarantined status
				dataObj, ok := body["data"].(map[string]interface{})
				if !ok {
					t.Fatalf("expected data object in response, got: %v", body)
				}
				status, _ := dataObj["status"].(string)
				if status != string(tc.wantAgentState) {
					t.Errorf("expected agent status %q in response, got %q", tc.wantAgentState, status)
				}

				// Verify meta contains tenant_id
				metaObj, ok := body["meta"].(map[string]interface{})
				if !ok {
					t.Fatalf("expected meta object in response, got: %v", body)
				}
				metaTenant, _ := metaObj["tenant_id"].(string)
				if metaTenant != tc.tenantID {
					t.Errorf("expected meta.tenant_id %q, got %q", tc.tenantID, metaTenant)
				}

				// Also verify the in-memory registry was actually updated
				agent, err := reg.Get(tc.tenantID, tc.agentID)
				if err != nil {
					t.Fatalf("failed to get agent from registry after quarantine: %v", err)
				}
				if agent.Status != tc.wantAgentState {
					t.Errorf("expected registry agent status %q, got %q", tc.wantAgentState, agent.Status)
				}
			}
		})
	}
}

func TestQuarantineEndpoint_AgentExcludedFromRouting(t *testing.T) {
	// Verify that a quarantined agent is no longer returned by FindByCapabilities,
	// which is what the task router uses to find eligible agents.
	reg := registry.New()
	reg.Register("tenant-1", &registry.RegisterRequest{
		AgentID:      "agent-1",
		Version:      "1.0.0",
		Framework:    "langchain",
		Capabilities: []string{"read:db"},
		NodeID:       "node-1",
	})
	_ = reg.Heartbeat("tenant-1", "agent-1", registry.StatusHealthy)

	// Before quarantine: agent is findable
	before := reg.FindByCapabilities("tenant-1", []string{"read:db"})
	if len(before) != 1 {
		t.Fatalf("expected 1 agent before quarantine, got %d", len(before))
	}

	// Quarantine via HTTP
	handler := buildAgentMux(reg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/agent-1/quarantine", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("quarantine request failed with status %d: %s", rr.Code, rr.Body.String())
	}

	// After quarantine: agent is excluded from capability search
	after := reg.FindByCapabilities("tenant-1", []string{"read:db"})
	if len(after) != 0 {
		t.Errorf("expected 0 agents after quarantine, got %d", len(after))
	}

	// But agent is still retrievable via direct GET
	agent, err := reg.Get("tenant-1", "agent-1")
	if err != nil {
		t.Fatalf("expected agent to still be retrievable after quarantine: %v", err)
	}
	if agent.Status != registry.StatusQuarantined {
		t.Errorf("expected status %q, got %q", registry.StatusQuarantined, agent.Status)
	}
}
