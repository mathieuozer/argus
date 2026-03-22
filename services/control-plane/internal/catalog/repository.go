package catalog

import (
	"fmt"
	"sync"
	"time"
)

// SourceType defines the type of data source in the catalog.
type SourceType string

const (
	SourceTypeDatabase SourceType = "database"
	SourceTypeAPI      SourceType = "api"
	SourceTypeFile     SourceType = "file"
	SourceTypeStream   SourceType = "stream"
	SourceTypeModel    SourceType = "model"
)

// Source represents a data source registered in the catalog.
type Source struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        SourceType        `json:"type"`
	Owner       string            `json:"owner"`
	AgentID     string            `json:"agent_id,omitempty"`
	Tags        []string          `json:"tags"`
	Schema      map[string]string `json:"schema,omitempty"` // field -> type mapping
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// LineageEdge represents a data flow relationship between two sources.
type LineageEdge struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	SourceID      string    `json:"source_id"`      // upstream data source
	TargetID      string    `json:"target_id"`      // downstream data source
	TransformType string    `json:"transform_type"` // "copy", "transform", "aggregate", "filter"
	AgentID       string    `json:"agent_id"`       // agent performing the transform
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

// LineageGraph represents the full lineage graph for a source.
type LineageGraph struct {
	SourceID   string         `json:"source_id"`
	Upstream   []*LineageNode `json:"upstream"`
	Downstream []*LineageNode `json:"downstream"`
}

// LineageNode represents a node in a lineage traversal.
type LineageNode struct {
	SourceID      string `json:"source_id"`
	SourceName    string `json:"source_name"`
	TransformType string `json:"transform_type"`
	AgentID       string `json:"agent_id"`
	Depth         int    `json:"depth"`
}

// Repository provides in-memory storage for catalog sources and lineage.
type Repository struct {
	mu      sync.RWMutex
	sources []*Source
	edges   []*LineageEdge
	srcSeq  int
	edgeSeq int
}

// NewRepository creates a new catalog repository.
func NewRepository() *Repository {
	return &Repository{
		sources: make([]*Source, 0),
		edges:   make([]*LineageEdge, 0),
	}
}

// CreateSource registers a new data source in the catalog.
func (r *Repository) CreateSource(tenantID, name, description string, srcType SourceType, owner, agentID string, tags []string, schema map[string]string) *Source {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.srcSeq++
	now := time.Now()
	if tags == nil {
		tags = []string{}
	}
	src := &Source{
		ID:          fmt.Sprintf("src-%d", r.srcSeq),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Type:        srcType,
		Owner:       owner,
		AgentID:     agentID,
		Tags:        tags,
		Schema:      schema,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.sources = append(r.sources, src)
	return src
}

// ListSources returns all catalog sources for a tenant, optionally filtered by type or tag.
func (r *Repository) ListSources(tenantID string, srcType SourceType, tag string) []*Source {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Source
	for _, src := range r.sources {
		if src.TenantID != tenantID {
			continue
		}
		if srcType != "" && src.Type != srcType {
			continue
		}
		if tag != "" && !containsTag(src.Tags, tag) {
			continue
		}
		result = append(result, src)
	}
	return result
}

// GetSource returns a specific source by ID within a tenant.
func (r *Repository) GetSource(tenantID, sourceID string) *Source {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, src := range r.sources {
		if src.TenantID == tenantID && src.ID == sourceID {
			return src
		}
	}
	return nil
}

// UpdateSource updates a source's metadata.
func (r *Repository) UpdateSource(tenantID, sourceID, name, description, owner string, tags []string) (*Source, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, src := range r.sources {
		if src.TenantID == tenantID && src.ID == sourceID {
			if name != "" {
				src.Name = name
			}
			if description != "" {
				src.Description = description
			}
			if owner != "" {
				src.Owner = owner
			}
			if tags != nil {
				src.Tags = tags
			}
			src.UpdatedAt = time.Now()
			return src, nil
		}
	}
	return nil, fmt.Errorf("source %s not found", sourceID)
}

// DeleteSource removes a source and its associated lineage edges.
func (r *Repository) DeleteSource(tenantID, sourceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, src := range r.sources {
		if src.TenantID == tenantID && src.ID == sourceID {
			r.sources = append(r.sources[:i], r.sources[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("source %s not found", sourceID)
	}

	// Remove associated edges
	filtered := make([]*LineageEdge, 0, len(r.edges))
	for _, edge := range r.edges {
		if edge.SourceID != sourceID && edge.TargetID != sourceID {
			filtered = append(filtered, edge)
		}
	}
	r.edges = filtered

	return nil
}

// AddLineageEdge creates a data flow relationship between two sources.
func (r *Repository) AddLineageEdge(tenantID, sourceID, targetID, transformType, agentID, description string) (*LineageEdge, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Verify both sources exist and belong to the tenant
	srcExists, tgtExists := false, false
	for _, src := range r.sources {
		if src.TenantID == tenantID {
			if src.ID == sourceID {
				srcExists = true
			}
			if src.ID == targetID {
				tgtExists = true
			}
		}
	}
	if !srcExists {
		return nil, fmt.Errorf("source %s not found", sourceID)
	}
	if !tgtExists {
		return nil, fmt.Errorf("target %s not found", targetID)
	}

	r.edgeSeq++
	edge := &LineageEdge{
		ID:            fmt.Sprintf("edge-%d", r.edgeSeq),
		TenantID:      tenantID,
		SourceID:      sourceID,
		TargetID:      targetID,
		TransformType: transformType,
		AgentID:       agentID,
		Description:   description,
		CreatedAt:     time.Now(),
	}
	r.edges = append(r.edges, edge)
	return edge, nil
}

// GetLineage returns the lineage graph (upstream and downstream) for a source.
func (r *Repository) GetLineage(tenantID, sourceID string) *LineageGraph {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Verify source exists
	src := r.findSource(tenantID, sourceID)
	if src == nil {
		return nil
	}

	graph := &LineageGraph{
		SourceID:   sourceID,
		Upstream:   make([]*LineageNode, 0),
		Downstream: make([]*LineageNode, 0),
	}

	// Collect upstream (sources that feed into this source)
	visited := make(map[string]bool)
	r.collectUpstream(tenantID, sourceID, visited, graph, 1)

	// Collect downstream (sources that this source feeds)
	visited = make(map[string]bool)
	r.collectDownstream(tenantID, sourceID, visited, graph, 1)

	return graph
}

func (r *Repository) collectUpstream(tenantID, targetID string, visited map[string]bool, graph *LineageGraph, depth int) {
	if depth > 10 { // prevent infinite recursion
		return
	}
	for _, edge := range r.edges {
		if edge.TenantID == tenantID && edge.TargetID == targetID && !visited[edge.SourceID] {
			visited[edge.SourceID] = true
			src := r.findSource(tenantID, edge.SourceID)
			srcName := ""
			if src != nil {
				srcName = src.Name
			}
			graph.Upstream = append(graph.Upstream, &LineageNode{
				SourceID:      edge.SourceID,
				SourceName:    srcName,
				TransformType: edge.TransformType,
				AgentID:       edge.AgentID,
				Depth:         depth,
			})
			r.collectUpstream(tenantID, edge.SourceID, visited, graph, depth+1)
		}
	}
}

func (r *Repository) collectDownstream(tenantID, sourceID string, visited map[string]bool, graph *LineageGraph, depth int) {
	if depth > 10 {
		return
	}
	for _, edge := range r.edges {
		if edge.TenantID == tenantID && edge.SourceID == sourceID && !visited[edge.TargetID] {
			visited[edge.TargetID] = true
			src := r.findSource(tenantID, edge.TargetID)
			srcName := ""
			if src != nil {
				srcName = src.Name
			}
			graph.Downstream = append(graph.Downstream, &LineageNode{
				SourceID:      edge.TargetID,
				SourceName:    srcName,
				TransformType: edge.TransformType,
				AgentID:       edge.AgentID,
				Depth:         depth,
			})
			r.collectDownstream(tenantID, edge.TargetID, visited, graph, depth+1)
		}
	}
}

func (r *Repository) findSource(tenantID, sourceID string) *Source {
	for _, src := range r.sources {
		if src.TenantID == tenantID && src.ID == sourceID {
			return src
		}
	}
	return nil
}

// ListEdges returns all lineage edges for a tenant.
func (r *Repository) ListEdges(tenantID string) []*LineageEdge {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*LineageEdge
	for _, edge := range r.edges {
		if edge.TenantID == tenantID {
			result = append(result, edge)
		}
	}
	return result
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
