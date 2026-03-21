package registry

import (
	"sync"
	"time"

	"github.com/argus-platform/argus/pkg/errors"
)

// AgentStatus represents the status of an agent.
type AgentStatus string

const (
	StatusDiscovered  AgentStatus = "discovered"
	StatusHealthy     AgentStatus = "healthy"
	StatusDegraded    AgentStatus = "degraded"
	StatusFailed      AgentStatus = "failed"
	StatusQuarantined AgentStatus = "quarantined"
)

// Agent represents a registered agent instance.
type Agent struct {
	ID           string      `json:"id"`
	TenantID     string      `json:"tenant_id"`
	Version      string      `json:"version"`
	Framework    string      `json:"framework"`
	Capabilities []string    `json:"capabilities"`
	Status       AgentStatus `json:"status"`
	SVIDURI      string      `json:"svid_uri"`
	LastSeen     time.Time   `json:"last_seen"`
	NodeID       string      `json:"node_id"`
}

// RegisterRequest is the request to register an agent.
type RegisterRequest struct {
	AgentID      string   `json:"agent_id"`
	Version      string   `json:"version"`
	Framework    string   `json:"framework"`
	Capabilities []string `json:"capabilities"`
	NodeID       string   `json:"node_id"`
}

// Registry manages agent registration and discovery.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]map[string]*Agent // tenant_id -> agent_id -> agent
}

// New creates a new in-memory agent registry.
func New() *Registry {
	return &Registry{
		agents: make(map[string]map[string]*Agent),
	}
}

// Register registers a new agent.
func (r *Registry) Register(tenantID string, req *RegisterRequest) *Agent {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.agents[tenantID] == nil {
		r.agents[tenantID] = make(map[string]*Agent)
	}

	agent := &Agent{
		ID:           req.AgentID,
		TenantID:     tenantID,
		Version:      req.Version,
		Framework:    req.Framework,
		Capabilities: req.Capabilities,
		Status:       StatusDiscovered,
		LastSeen:     time.Now(),
		NodeID:       req.NodeID,
	}

	r.agents[tenantID][req.AgentID] = agent
	return agent
}

// Get retrieves an agent by tenant and agent ID.
func (r *Registry) Get(tenantID, agentID string) (*Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tenantAgents, ok := r.agents[tenantID]
	if !ok {
		return nil, errors.New(errors.CodeAgentNotFound, "agent not found")
	}

	agent, ok := tenantAgents[agentID]
	if !ok {
		return nil, errors.New(errors.CodeAgentNotFound, "agent not found")
	}

	return agent, nil
}

// List returns all agents for a tenant.
func (r *Registry) List(tenantID string) []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tenantAgents := r.agents[tenantID]
	agents := make([]*Agent, 0, len(tenantAgents))
	for _, a := range tenantAgents {
		agents = append(agents, a)
	}
	return agents
}

// Heartbeat updates the last seen time and status of an agent.
func (r *Registry) Heartbeat(tenantID, agentID string, status AgentStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tenantAgents, ok := r.agents[tenantID]
	if !ok {
		return errors.New(errors.CodeAgentNotFound, "agent not found")
	}

	agent, ok := tenantAgents[agentID]
	if !ok {
		return errors.New(errors.CodeAgentNotFound, "agent not found")
	}

	agent.Status = status
	agent.LastSeen = time.Now()
	return nil
}

// QuarantineAgent sets an agent's status to quarantined. This is used by the
// auto-quarantine pipeline when the predictive failure model determines that
// an agent has a very high probability of imminent failure (> 0.9). A
// quarantined agent is excluded from task routing but can still be inspected.
func (r *Registry) QuarantineAgent(tenantID, agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tenantAgents, ok := r.agents[tenantID]
	if !ok {
		return errors.New(errors.CodeAgentNotFound, "agent not found")
	}

	agent, ok := tenantAgents[agentID]
	if !ok {
		return errors.New(errors.CodeAgentNotFound, "agent not found")
	}

	agent.Status = StatusQuarantined
	agent.LastSeen = time.Now()
	return nil
}

// FindByCapabilities returns agents that have all the specified capabilities.
func (r *Registry) FindByCapabilities(tenantID string, capabilities []string) []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Agent
	for _, agent := range r.agents[tenantID] {
		if agent.Status == StatusQuarantined || agent.Status == StatusFailed {
			continue
		}
		if hasAllCapabilities(agent.Capabilities, capabilities) {
			result = append(result, agent)
		}
	}
	return result
}

func hasAllCapabilities(agentCaps, required []string) bool {
	capSet := make(map[string]bool, len(agentCaps))
	for _, c := range agentCaps {
		capSet[c] = true
	}
	for _, r := range required {
		if !capSet[r] {
			return false
		}
	}
	return true
}
