package identity

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"sync"
	"time"

	identityv1 "github.com/argus-platform/argus/gen/go/identity"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Manager manages the agent's certificate lifecycle via the identity service.
type Manager struct {
	logger   *zap.Logger
	spiffeID string
	certPEM  []byte
	keyPEM   []byte
	mu       sync.RWMutex

	identityAddr string
	client       identityv1.IdentityServiceClient
	conn         *grpc.ClientConn
	renewTicker  *time.Ticker
	stopCh       chan struct{}
}

// NewManager creates a new identity manager.
func NewManager(logger *zap.Logger) *Manager {
	identityAddr := os.Getenv("ARGUS_IDENTITY_ADDR")
	if identityAddr == "" {
		identityAddr = "localhost:9081" // default identity service gRPC port
	}

	return &Manager{
		logger:       logger,
		identityAddr: identityAddr,
		stopCh:       make(chan struct{}),
	}
}

// Connect establishes a gRPC connection to the identity service.
func (m *Manager) Connect() error {
	conn, err := grpc.NewClient(m.identityAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		m.logger.Warn("failed to connect to identity service, running without SVID",
			zap.String("addr", m.identityAddr),
			zap.Error(err),
		)
		return nil // graceful degradation
	}

	m.conn = conn
	m.client = identityv1.NewIdentityServiceClient(conn)
	m.logger.Info("connected to identity service", zap.String("addr", m.identityAddr))
	return nil
}

// RequestSVID requests a new SVID from the identity service.
func (m *Manager) RequestSVID(tenantID, agentID, version string) error {
	m.logger.Info("requesting SVID",
		zap.String("tenant_id", tenantID),
		zap.String("agent_id", agentID),
		zap.String("version", version),
	)

	if m.client == nil {
		m.logger.Warn("identity service not connected, skipping SVID request")
		return nil
	}

	// Generate a CSR for the SVID request
	csr, err := m.generateCSR(tenantID, agentID)
	if err != nil {
		return fmt.Errorf("generate CSR: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := m.client.CreateSVID(ctx, &identityv1.CreateSVIDRequest{
		TenantId: tenantID,
		AgentId:  agentID,
		Version:  version,
		Csr:      csr,
	})
	if err != nil {
		return fmt.Errorf("create SVID: %w", err)
	}

	m.mu.Lock()
	m.spiffeID = resp.Svid.SpiffeId
	m.certPEM = resp.Svid.Certificate
	m.keyPEM = resp.Svid.PrivateKey
	m.mu.Unlock()

	m.logger.Info("SVID issued successfully",
		zap.String("spiffe_id", resp.Svid.SpiffeId),
		zap.Time("expires_at", resp.Svid.ExpiresAt.AsTime()),
	)

	return nil
}

// RenewSVID renews the current SVID before it expires.
func (m *Manager) RenewSVID() error {
	m.mu.RLock()
	spiffeID := m.spiffeID
	m.mu.RUnlock()

	m.logger.Info("renewing SVID", zap.String("spiffe_id", spiffeID))

	if m.client == nil || spiffeID == "" {
		m.logger.Warn("identity service not connected or no SVID to renew")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := m.client.RenewSVID(ctx, &identityv1.RenewSVIDRequest{
		SpiffeId: spiffeID,
	})
	if err != nil {
		return fmt.Errorf("renew SVID: %w", err)
	}

	m.mu.Lock()
	m.spiffeID = resp.Svid.SpiffeId
	m.certPEM = resp.Svid.Certificate
	m.keyPEM = resp.Svid.PrivateKey
	m.mu.Unlock()

	m.logger.Info("SVID renewed successfully",
		zap.String("spiffe_id", resp.Svid.SpiffeId),
		zap.Time("expires_at", resp.Svid.ExpiresAt.AsTime()),
	)

	return nil
}

// StartRenewalLoop starts a background goroutine that renews the SVID
// before expiry. The SVID has a 1-hour TTL; renew at 45 minutes.
func (m *Manager) StartRenewalLoop() {
	m.renewTicker = time.NewTicker(45 * time.Minute)
	go func() {
		for {
			select {
			case <-m.renewTicker.C:
				if err := m.RenewSVID(); err != nil {
					m.logger.Error("SVID renewal failed", zap.Error(err))
				}
			case <-m.stopCh:
				return
			}
		}
	}()
}

// GetSpiffeID returns the current SPIFFE ID.
func (m *Manager) GetSpiffeID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.spiffeID
}

// GetCertPEM returns the current certificate PEM.
func (m *Manager) GetCertPEM() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.certPEM
}

// GetKeyPEM returns the current private key PEM.
func (m *Manager) GetKeyPEM() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.keyPEM
}

// Close stops the renewal loop and closes the gRPC connection.
func (m *Manager) Close() {
	close(m.stopCh)
	if m.renewTicker != nil {
		m.renewTicker.Stop()
	}
	if m.conn != nil {
		m.conn.Close()
	}
}

// generateCSR creates a Certificate Signing Request for the agent.
func (m *Manager) generateCSR(tenantID, agentID string) ([]byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("argus-agent-%s-%s", tenantID, agentID),
			Organization: []string{"argus"},
		},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return nil, fmt.Errorf("create CSR: %w", err)
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})

	return csrPEM, nil
}
