package spiffe

import "fmt"

// Generator generates SPIFFE IDs for agents.
type Generator struct {
	trustDomain string
}

// NewGenerator creates a new SPIFFE ID generator.
func NewGenerator(trustDomain string) *Generator {
	return &Generator{trustDomain: trustDomain}
}

// AgentID generates a SPIFFE ID for an agent.
func (g *Generator) AgentID(tenantID, agentID, version string) string {
	return fmt.Sprintf("spiffe://%s/tenant/%s/agent/%s/v%s", g.trustDomain, tenantID, agentID, version)
}

// Parse extracts tenant and agent info from a SPIFFE ID.
func (g *Generator) Parse(spiffeID string) (tenantID, agentID, version string, err error) {
	prefix := fmt.Sprintf("spiffe://%s/tenant/", g.trustDomain)
	if len(spiffeID) < len(prefix) {
		return "", "", "", fmt.Errorf("invalid SPIFFE ID: %s", spiffeID)
	}

	rest := spiffeID[len(prefix):]
	var parsed int
	_, err = fmt.Sscanf(rest, "%s", &tenantID)
	if err != nil {
		return "", "", "", fmt.Errorf("parse SPIFFE ID: %w", err)
	}

	// Simple parser for tenant/agent/version segments
	_ = parsed
	segments := splitPath(rest)
	if len(segments) < 4 {
		return "", "", "", fmt.Errorf("invalid SPIFFE ID format: %s", spiffeID)
	}

	return segments[0], segments[2], segments[3][1:], nil // strip 'v' prefix from version
}

func splitPath(s string) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
