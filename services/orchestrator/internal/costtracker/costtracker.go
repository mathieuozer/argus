package costtracker

import (
	"sync"
)

// Tracker tracks costs per agent and tenant.
type Tracker struct {
	mu    sync.RWMutex
	costs map[string]map[string]float64 // tenant_id -> agent_id -> total_cost
}

// New creates a new cost tracker.
func New() *Tracker {
	return &Tracker{
		costs: make(map[string]map[string]float64),
	}
}

// Record records a cost for a task.
func (t *Tracker) Record(tenantID, agentID string, costUSD float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.costs[tenantID] == nil {
		t.costs[tenantID] = make(map[string]float64)
	}
	t.costs[tenantID][agentID] += costUSD
}

// GetAgentCost returns the total cost for an agent.
func (t *Tracker) GetAgentCost(tenantID, agentID string) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if agents, ok := t.costs[tenantID]; ok {
		return agents[agentID]
	}
	return 0
}

// GetTenantCost returns the total cost for a tenant.
func (t *Tracker) GetTenantCost(tenantID string) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := 0.0
	if agents, ok := t.costs[tenantID]; ok {
		for _, cost := range agents {
			total += cost
		}
	}
	return total
}
