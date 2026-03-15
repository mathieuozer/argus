package router

import (
	"fmt"

	"github.com/argus-platform/argus/services/orchestrator/internal/registry"
)

// Router routes tasks to agents based on capabilities.
type Router struct {
	registry *registry.Registry
}

// New creates a new task router.
func New(reg *registry.Registry) *Router {
	return &Router{registry: reg}
}

// Route finds the best agent for a task based on required capabilities.
func (r *Router) Route(tenantID string, capabilities []string, preferredAgentID string) (*registry.Agent, error) {
	// If a preferred agent is specified, try it first
	if preferredAgentID != "" {
		agent, err := r.registry.Get(tenantID, preferredAgentID)
		if err == nil && agent.Status == registry.StatusHealthy {
			if hasAllCapabilities(agent.Capabilities, capabilities) {
				return agent, nil
			}
		}
	}

	// Find agents with matching capabilities
	agents := r.registry.FindByCapabilities(tenantID, capabilities)
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents found with capabilities: %v", capabilities)
	}

	// Simple selection: pick the first healthy agent
	for _, a := range agents {
		if a.Status == registry.StatusHealthy {
			return a, nil
		}
	}

	// Fall back to any available agent
	return agents[0], nil
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
