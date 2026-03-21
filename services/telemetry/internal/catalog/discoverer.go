package catalog

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type SourceType string

const (
	SourceDatabase SourceType = "database"
	SourceAPI      SourceType = "api"
	SourceStorage  SourceType = "storage"
	SourceTool     SourceType = "tool"
)

type CatalogEntry struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Type        SourceType `json:"type"`
	Name        string     `json:"name"`
	Identifier  string     `json:"identifier"`
	Agents      []string   `json:"agents"`
	AccessTypes []string   `json:"access_types"`
	DataTier    int        `json:"tier"`
	FirstSeen   time.Time  `json:"first_seen"`
	LastSeen    time.Time  `json:"last_seen"`
	SpanCount   int64      `json:"span_count"`
}

type Discoverer struct {
	mu      sync.RWMutex
	entries map[string]*CatalogEntry // key: tenantID:type:identifier
	counter int
}

func NewDiscoverer() *Discoverer {
	return &Discoverer{
		entries: make(map[string]*CatalogEntry),
	}
}

func (d *Discoverer) DiscoverFromSpan(tenantID, agentID, operationName string, attributes map[string]string) *CatalogEntry {
	sourceType, name, identifier := classifyOperation(operationName, attributes)
	if identifier == "" {
		return nil
	}

	key := tenantID + ":" + string(sourceType) + ":" + identifier

	d.mu.Lock()
	defer d.mu.Unlock()

	entry, exists := d.entries[key]
	if exists {
		entry.LastSeen = time.Now()
		entry.SpanCount++
		if !containsString(entry.Agents, agentID) {
			entry.Agents = append(entry.Agents, agentID)
		}
		accessType := inferAccessType(operationName)
		if accessType != "" && !containsString(entry.AccessTypes, accessType) {
			entry.AccessTypes = append(entry.AccessTypes, accessType)
		}
		return entry
	}

	d.counter++
	accessType := inferAccessType(operationName)
	accessTypes := []string{}
	if accessType != "" {
		accessTypes = []string{accessType}
	}

	entry = &CatalogEntry{
		ID:          fmt.Sprintf("src-%03d", d.counter),
		TenantID:    tenantID,
		Type:        sourceType,
		Name:        name,
		Identifier:  identifier,
		Agents:      []string{agentID},
		AccessTypes: accessTypes,
		DataTier:    1,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		SpanCount:   1,
	}
	d.entries[key] = entry
	return entry
}

func (d *Discoverer) ListSources(tenantID string) []*CatalogEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []*CatalogEntry
	for _, entry := range d.entries {
		if entry.TenantID == tenantID {
			result = append(result, entry)
		}
	}
	return result
}

func (d *Discoverer) ListSourcesByAgent(tenantID, agentID string) []*CatalogEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []*CatalogEntry
	for _, entry := range d.entries {
		if entry.TenantID == tenantID && containsString(entry.Agents, agentID) {
			result = append(result, entry)
		}
	}
	return result
}

func classifyOperation(opName string, attrs map[string]string) (SourceType, string, string) {
	lower := strings.ToLower(opName)

	if strings.Contains(lower, "db") || strings.Contains(lower, "sql") || strings.Contains(lower, "query") {
		if dsn, ok := attrs["db.connection_string"]; ok {
			name := extractDBName(dsn)
			return SourceDatabase, name, dsn
		}
		if name, ok := attrs["db.name"]; ok {
			return SourceDatabase, name, name
		}
	}

	if strings.Contains(lower, "http") || strings.Contains(lower, "api") || strings.Contains(lower, "llm") {
		if url, ok := attrs["http.url"]; ok {
			name := extractHostname(url)
			return SourceAPI, name, url
		}
		if model, ok := attrs["model"]; ok {
			return SourceAPI, model+"_api", model
		}
	}

	if strings.Contains(lower, "storage") || strings.Contains(lower, "s3") || strings.Contains(lower, "blob") {
		if bucket, ok := attrs["storage.bucket"]; ok {
			return SourceStorage, bucket, bucket
		}
	}

	if strings.Contains(lower, "tool") {
		name := opName
		if toolName, ok := attrs["tool.name"]; ok {
			name = toolName
		}
		return SourceTool, name, name
	}

	return SourceTool, opName, opName
}

func inferAccessType(opName string) string {
	lower := strings.ToLower(opName)
	// Check write before read — "db.query.write" should be "write", not "read"
	if strings.Contains(lower, "write") || strings.Contains(lower, "put") || strings.Contains(lower, "insert") || strings.Contains(lower, "create") {
		return "write"
	}
	if strings.Contains(lower, "read") || strings.Contains(lower, "get") || strings.Contains(lower, "query") || strings.Contains(lower, "select") {
		return "read"
	}
	if strings.Contains(lower, "call") || strings.Contains(lower, "invoke") || strings.Contains(lower, "completion") {
		return "call"
	}
	return ""
}

func extractDBName(dsn string) string {
	parts := strings.Split(dsn, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		if idx := strings.Index(name, "?"); idx > 0 {
			name = name[:idx]
		}
		if name != "" {
			return name
		}
	}
	return "unknown_db"
}

func extractHostname(url string) string {
	u := strings.TrimPrefix(url, "https://")
	u = strings.TrimPrefix(u, "http://")
	if idx := strings.Index(u, "/"); idx > 0 {
		u = u[:idx]
	}
	if idx := strings.Index(u, ":"); idx > 0 {
		u = u[:idx]
	}
	return u
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
