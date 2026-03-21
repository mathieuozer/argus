package integration

import (
	"context"
	"testing"

	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
	"github.com/argus-platform/argus/services/orchestrator/internal/repository"
)

func TestAgentRepository_RegisterAndGet(t *testing.T) {
	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	tenantID := CreateTestTenant(t, pool, "test-agent-tenant")
	repo := repository.NewAgentRepository(pool)
	ctx := context.Background()

	req := &registry.RegisterRequest{
		AgentID:      "test-agent-1",
		Version:      "1.0.0",
		Framework:    "langchain",
		Capabilities: []string{"read:db", "write:reports"},
		NodeID:       "node-1",
	}

	agent, err := repo.Register(ctx, tenantID, req)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if agent.ID != "test-agent-1" {
		t.Errorf("expected agent ID test-agent-1, got %s", agent.ID)
	}
	if agent.Status != registry.StatusDiscovered {
		t.Errorf("expected status discovered, got %s", agent.Status)
	}

	// Get the agent
	got, err := repo.Get(ctx, tenantID, "test-agent-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Framework != "langchain" {
		t.Errorf("expected framework langchain, got %s", got.Framework)
	}
	if len(got.Capabilities) != 2 {
		t.Errorf("expected 2 capabilities, got %d", len(got.Capabilities))
	}
}

func TestAgentRepository_List(t *testing.T) {
	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	tenantID := CreateTestTenant(t, pool, "test-list-tenant")
	repo := repository.NewAgentRepository(pool)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := repo.Register(ctx, tenantID, &registry.RegisterRequest{
			AgentID:      "agent-" + string(rune('a'+i)),
			Version:      "1.0.0",
			Framework:    "custom",
			Capabilities: []string{"read"},
			NodeID:       "node-1",
		})
		if err != nil {
			t.Fatalf("Register agent %d failed: %v", i, err)
		}
	}

	agents, err := repo.List(ctx, tenantID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(agents))
	}
}

func TestAgentRepository_Heartbeat(t *testing.T) {
	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	tenantID := CreateTestTenant(t, pool, "test-heartbeat-tenant")
	repo := repository.NewAgentRepository(pool)
	ctx := context.Background()

	_, err := repo.Register(ctx, tenantID, &registry.RegisterRequest{
		AgentID:   "agent-hb",
		Version:   "1.0.0",
		Framework: "autogen",
		NodeID:    "node-1",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = repo.Heartbeat(ctx, tenantID, "agent-hb", registry.StatusHealthy)
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	agent, err := repo.Get(ctx, tenantID, "agent-hb")
	if err != nil {
		t.Fatalf("Get after heartbeat failed: %v", err)
	}
	if agent.Status != registry.StatusHealthy {
		t.Errorf("expected status healthy, got %s", agent.Status)
	}
}

func TestAgentRepository_Quarantine(t *testing.T) {
	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	tenantID := CreateTestTenant(t, pool, "test-quarantine-tenant")
	repo := repository.NewAgentRepository(pool)
	ctx := context.Background()

	_, err := repo.Register(ctx, tenantID, &registry.RegisterRequest{
		AgentID:   "agent-q",
		Version:   "1.0.0",
		Framework: "custom",
		NodeID:    "node-1",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = repo.Quarantine(ctx, tenantID, "agent-q")
	if err != nil {
		t.Fatalf("Quarantine failed: %v", err)
	}

	agent, err := repo.Get(ctx, tenantID, "agent-q")
	if err != nil {
		t.Fatalf("Get after quarantine failed: %v", err)
	}
	if agent.Status != registry.StatusQuarantined {
		t.Errorf("expected status quarantined, got %s", agent.Status)
	}
}

func TestAgentRepository_CrossTenantIsolation(t *testing.T) {
	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	tenantA := CreateTestTenant(t, pool, "tenant-a")
	tenantB := CreateTestTenant(t, pool, "tenant-b")
	repo := repository.NewAgentRepository(pool)
	ctx := context.Background()

	_, err := repo.Register(ctx, tenantA, &registry.RegisterRequest{
		AgentID:   "shared-name",
		Version:   "1.0.0",
		Framework: "custom",
		NodeID:    "node-1",
	})
	if err != nil {
		t.Fatalf("Register in tenant A failed: %v", err)
	}

	// Tenant B should not see tenant A's agent
	agents, err := repo.List(ctx, tenantB)
	if err != nil {
		t.Fatalf("List tenant B failed: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 agents for tenant B, got %d", len(agents))
	}

	// Direct get from tenant B should fail
	_, err = repo.Get(ctx, tenantB, "shared-name")
	if err == nil {
		t.Error("expected error when accessing tenant A's agent from tenant B")
	}
}
