package discovery

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func TestNew_Defaults(t *testing.T) {
	t.Setenv("ARGUS_ORCHESTRATOR_ADDR", "")
	t.Setenv("ARGUS_TENANT_ID", "")

	d := New(zap.NewNop())
	if d == nil {
		t.Fatal("expected non-nil discovery")
	}
	if d.orchestratorAddr != "http://localhost:8082" {
		t.Errorf("expected default orchestrator addr http://localhost:8082, got %s", d.orchestratorAddr)
	}
	if d.tenantID != "default" {
		t.Errorf("expected default tenant ID 'default', got %s", d.tenantID)
	}
	if d.registered {
		t.Error("expected registered to be false on creation")
	}
	if d.agentID != "" {
		t.Error("expected agentID to be empty on creation")
	}
}

func TestNew_CustomEnv(t *testing.T) {
	t.Setenv("ARGUS_ORCHESTRATOR_ADDR", "http://orchestrator:9090")
	t.Setenv("ARGUS_TENANT_ID", "ministry-finance-tr")

	d := New(zap.NewNop())
	if d.orchestratorAddr != "http://orchestrator:9090" {
		t.Errorf("expected orchestrator addr http://orchestrator:9090, got %s", d.orchestratorAddr)
	}
	if d.tenantID != "ministry-finance-tr" {
		t.Errorf("expected tenant ID ministry-finance-tr, got %s", d.tenantID)
	}
}

func TestRegister_Success(t *testing.T) {
	var mu sync.Mutex
	var receivedBody map[string]interface{}
	var receivedHeaders http.Header

	mockOrchestrator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/agents" {
			t.Errorf("expected path /api/v1/agents, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		mu.Lock()
		json.Unmarshal(body, &receivedBody)
		receivedHeaders = r.Header.Clone()
		mu.Unlock()

		w.WriteHeader(http.StatusCreated)
	}))
	defer mockOrchestrator.Close()

	t.Setenv("ARGUS_ORCHESTRATOR_ADDR", mockOrchestrator.URL)
	t.Setenv("ARGUS_TENANT_ID", "test-tenant")

	d := New(zap.NewNop())
	err := d.Register("budget-reconciler", "1.2.0", "langchain", []string{"read:budget_db", "write:report_store"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if receivedBody["agent_id"] != "budget-reconciler" {
		t.Errorf("expected agent_id budget-reconciler, got %v", receivedBody["agent_id"])
	}
	if receivedBody["version"] != "1.2.0" {
		t.Errorf("expected version 1.2.0, got %v", receivedBody["version"])
	}
	if receivedBody["framework"] != "langchain" {
		t.Errorf("expected framework langchain, got %v", receivedBody["framework"])
	}

	caps, ok := receivedBody["capabilities"].([]interface{})
	if !ok {
		t.Fatal("expected capabilities to be an array")
	}
	if len(caps) != 2 {
		t.Errorf("expected 2 capabilities, got %d", len(caps))
	}

	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", receivedHeaders.Get("Content-Type"))
	}
	if receivedHeaders.Get("X-Tenant-ID") != "test-tenant" {
		t.Errorf("expected X-Tenant-ID test-tenant, got %s", receivedHeaders.Get("X-Tenant-ID"))
	}

	if !d.IsRegistered() {
		t.Error("expected IsRegistered to be true after successful registration")
	}
	if d.AgentID() != "budget-reconciler" {
		t.Errorf("expected AgentID budget-reconciler, got %s", d.AgentID())
	}
}

func TestRegister_ServerError(t *testing.T) {
	mockOrchestrator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockOrchestrator.Close()

	t.Setenv("ARGUS_ORCHESTRATOR_ADDR", mockOrchestrator.URL)
	t.Setenv("ARGUS_TENANT_ID", "test-tenant")

	d := New(zap.NewNop())
	err := d.Register("test-agent", "1.0.0", "custom", nil)
	if err == nil {
		t.Fatal("expected error on 500 response")
	}

	if d.IsRegistered() {
		t.Error("expected IsRegistered to be false after failed registration")
	}
}

func TestRegister_ConnectionRefused(t *testing.T) {
	t.Setenv("ARGUS_ORCHESTRATOR_ADDR", "http://127.0.0.1:1") // port 1 should refuse connections
	t.Setenv("ARGUS_TENANT_ID", "test-tenant")

	d := New(zap.NewNop())
	err := d.Register("test-agent", "1.0.0", "custom", nil)
	if err == nil {
		t.Fatal("expected error when orchestrator is unreachable")
	}

	if d.IsRegistered() {
		t.Error("expected IsRegistered to be false after connection error")
	}
}

func TestSendHeartbeat_NotRegistered(t *testing.T) {
	t.Setenv("ARGUS_ORCHESTRATOR_ADDR", "")
	t.Setenv("ARGUS_TENANT_ID", "")

	d := New(zap.NewNop())
	err := d.SendHeartbeat("healthy")
	if err == nil {
		t.Fatal("expected error when sending heartbeat before registration")
	}
	if err.Error() != "agent not registered" {
		t.Errorf("expected 'agent not registered' error, got %v", err)
	}
}

func TestSendHeartbeat_AfterRegistration(t *testing.T) {
	var mu sync.Mutex
	var heartbeatPath string
	var heartbeatBody map[string]string
	var heartbeatTenantID string

	mockOrchestrator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.URL.Path == "/api/v1/agents" {
			w.WriteHeader(http.StatusCreated)
			return
		}

		// Heartbeat endpoint
		heartbeatPath = r.URL.Path
		heartbeatTenantID = r.Header.Get("X-Tenant-ID")

		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		json.Unmarshal(body, &heartbeatBody)

		w.WriteHeader(http.StatusOK)
	}))
	defer mockOrchestrator.Close()

	t.Setenv("ARGUS_ORCHESTRATOR_ADDR", mockOrchestrator.URL)
	t.Setenv("ARGUS_TENANT_ID", "heartbeat-tenant")

	d := New(zap.NewNop())

	// Register first
	err := d.Register("my-agent", "1.0.0", "autogen", []string{"read:data"})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Now send heartbeat
	err = d.SendHeartbeat("healthy")
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	expectedPath := "/api/v1/agents/my-agent/heartbeat"
	if heartbeatPath != expectedPath {
		t.Errorf("expected heartbeat path %s, got %s", expectedPath, heartbeatPath)
	}
	if heartbeatBody["status"] != "healthy" {
		t.Errorf("expected heartbeat status 'healthy', got %s", heartbeatBody["status"])
	}
	if heartbeatTenantID != "heartbeat-tenant" {
		t.Errorf("expected X-Tenant-ID heartbeat-tenant, got %s", heartbeatTenantID)
	}
}

func TestIsRegistered_And_AgentID(t *testing.T) {
	tests := []struct {
		name           string
		registered     bool
		agentID        string
		expectRegister bool
		expectAgentID  string
	}{
		{
			name:           "not registered",
			registered:     false,
			agentID:        "",
			expectRegister: false,
			expectAgentID:  "",
		},
		{
			name:           "registered with agent ID",
			registered:     true,
			agentID:        "budget-reconciler",
			expectRegister: true,
			expectAgentID:  "budget-reconciler",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := &Discovery{
				logger:     zap.NewNop(),
				registered: tc.registered,
				agentID:    tc.agentID,
			}

			if d.IsRegistered() != tc.expectRegister {
				t.Errorf("expected IsRegistered=%v, got %v", tc.expectRegister, d.IsRegistered())
			}
			if d.AgentID() != tc.expectAgentID {
				t.Errorf("expected AgentID=%s, got %s", tc.expectAgentID, d.AgentID())
			}
		})
	}
}
