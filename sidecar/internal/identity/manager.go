package identity

import (
	"go.uber.org/zap"
)

// Manager manages the agent's certificate lifecycle.
type Manager struct {
	logger   *zap.Logger
	spiffeID string
	certPEM  []byte
	keyPEM   []byte
}

// NewManager creates a new identity manager.
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// RequestSVID requests a new SVID from the identity service.
func (m *Manager) RequestSVID(tenantID, agentID, version string) error {
	m.logger.Info("requesting SVID",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
	)
	// Stub: in production, call identity service gRPC API
	return nil
}

// RenewSVID renews the current SVID.
func (m *Manager) RenewSVID() error {
	m.logger.Info("renewing SVID", zap.String("spiffe_id", m.spiffeID))
	// Stub: in production, call identity service gRPC API
	return nil
}

// GetSpiffeID returns the current SPIFFE ID.
func (m *Manager) GetSpiffeID() string {
	return m.spiffeID
}
