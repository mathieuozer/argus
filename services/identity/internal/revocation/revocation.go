package revocation

import (
	"fmt"
	"sync"
	"time"
)

// Entry represents a revoked certificate.
type Entry struct {
	SpiffeID  string
	Reason    string
	RevokedAt time.Time
}

// Store manages certificate revocation.
type Store struct {
	mu      sync.RWMutex
	revoked map[string]*Entry
}

// NewStore creates a new revocation store.
func NewStore() *Store {
	return &Store{
		revoked: make(map[string]*Entry),
	}
}

// Revoke adds a certificate to the revocation list.
func (s *Store) Revoke(spiffeID, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[spiffeID] = &Entry{
		SpiffeID:  spiffeID,
		Reason:    reason,
		RevokedAt: time.Now(),
	}
}

// RevokeAgentCert constructs the SPIFFE ID for the given tenant and agent,
// then adds it to the revocation list. This is used by the auto-quarantine
// pipeline to revoke an agent's certificate so it can no longer initiate
// mTLS connections, while still being inspectable via the dashboard.
func (s *Store) RevokeAgentCert(tenantID, agentID string) {
	spiffeID := fmt.Sprintf("spiffe://argus.local/tenant/%s/agent/%s", tenantID, agentID)
	s.Revoke(spiffeID, "auto-quarantine: predictive failure probability exceeded threshold")
}

// IsRevoked checks if a certificate is revoked.
func (s *Store) IsRevoked(spiffeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.revoked[spiffeID]
	return ok
}

// List returns all revoked certificates.
func (s *Store) List() []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]*Entry, 0, len(s.revoked))
	for _, e := range s.revoked {
		entries = append(entries, e)
	}
	return entries
}
