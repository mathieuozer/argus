package discovery

import (
	"go.uber.org/zap"
)

// Discovery handles auto-registration with the orchestrator.
type Discovery struct {
	logger         *zap.Logger
	orchestratorAddr string
	registered     bool
}

// New creates a new discovery component.
func New(logger *zap.Logger) *Discovery {
	return &Discovery{
		logger:         logger,
		orchestratorAddr: "localhost:8082",
	}
}

// Register registers this sidecar's agent with the orchestrator.
func (d *Discovery) Register(agentID, version, framework string, capabilities []string) error {
	d.logger.Info("registering agent",
		zap.String("agent_id", agentID),
		zap.String("version", version),
	)
	// Stub: in production, call orchestrator gRPC API
	d.registered = true
	return nil
}

// IsRegistered returns whether the agent has been registered.
func (d *Discovery) IsRegistered() bool {
	return d.registered
}
