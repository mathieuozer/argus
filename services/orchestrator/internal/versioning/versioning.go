package versioning

import (
	"fmt"
	"sync"
)

// AgentVersion tracks an agent's version history.
type AgentVersion struct {
	AgentID  string `json:"agent_id"`
	Version  string `json:"version"`
	IsCanary bool   `json:"is_canary"`
}

// Tracker tracks agent versions.
type Tracker struct {
	mu       sync.RWMutex
	versions map[string]map[string]*AgentVersion // tenant -> agent -> version
}

// New creates a new version tracker.
func New() *Tracker {
	return &Tracker{
		versions: make(map[string]map[string]*AgentVersion),
	}
}

// Set records an agent's current version.
func (t *Tracker) Set(tenantID, agentID, version string, isCanary bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.versions[tenantID] == nil {
		t.versions[tenantID] = make(map[string]*AgentVersion)
	}

	t.versions[tenantID][agentID] = &AgentVersion{
		AgentID:  agentID,
		Version:  version,
		IsCanary: isCanary,
	}
}

// Get returns the current version for an agent.
func (t *Tracker) Get(tenantID, agentID string) (*AgentVersion, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if agents, ok := t.versions[tenantID]; ok {
		if v, ok := agents[agentID]; ok {
			return v, nil
		}
	}
	return nil, fmt.Errorf("version not found for agent %s", agentID)
}
