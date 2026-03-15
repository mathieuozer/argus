package revocation

import (
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
