package audit

import (
	"fmt"
	"sync"
	"time"
)

// Entry represents an immutable audit log entry.
type Entry struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

// Writer writes audit log entries.
type Writer struct {
	mu      sync.RWMutex
	entries []*Entry
	counter int
}

// NewWriter creates a new audit log writer.
func NewWriter() *Writer {
	return &Writer{
		entries: make([]*Entry, 0),
	}
}

// Write appends an entry to the audit log.
func (w *Writer) Write(tenantID, actor, action, resource, details string) *Entry {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.counter++
	entry := &Entry{
		ID:        fmt.Sprintf("audit-%d", w.counter),
		TenantID:  tenantID,
		Actor:     actor,
		Action:    action,
		Resource:  resource,
		Details:   details,
		Timestamp: time.Now(),
	}

	w.entries = append(w.entries, entry)
	return entry
}

// List returns all audit entries for a tenant.
func (w *Writer) List(tenantID string) []*Entry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var result []*Entry
	for _, e := range w.entries {
		if e.TenantID == tenantID {
			result = append(result, e)
		}
	}
	return result
}
