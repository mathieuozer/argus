package catalog

import (
	"fmt"
	"strings"
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
	SourceTypeStorage  SourceType = "storage"
	SourceTypeTool     SourceType = "tool"
)

// Column represents column-level metadata for a data source.
type Column struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	Description    string   `json:"description"`
	IsPII          bool     `json:"is_pii"`
	IsNullable     bool     `json:"is_nullable"`
	Classification string   `json:"classification"`
	Tags           []string `json:"tags"`
	SampleValues   []string `json:"sample_values,omitempty"`
	NullRate       float64  `json:"null_rate"`
	UniqueCount    int      `json:"unique_count"`
	MinValue       string   `json:"min_value,omitempty"`
	MaxValue       string   `json:"max_value,omitempty"`
}

// FreshnessInfo tracks data freshness for a source.
type FreshnessInfo struct {
	LastRefreshed    time.Time  `json:"last_refreshed"`
	RefreshFrequency string     `json:"refresh_frequency"`
	SLASeconds       int        `json:"sla_seconds"`
	IsStale          bool       `json:"is_stale"`
	StaleSince       *time.Time `json:"stale_since,omitempty"`
}

// PopularityInfo tracks usage popularity for a source.
type PopularityInfo struct {
	ViewCount      int    `json:"view_count"`
	QueryCount     int    `json:"query_count"`
	UniqueUsers    int    `json:"unique_users"`
	TrendDirection string `json:"trend_direction"`
	PopularityRank int    `json:"popularity_rank"`
}

// ProfileInfo contains data profiling statistics.
type ProfileInfo struct {
	RowCount      int64     `json:"row_count"`
	ColumnCount   int       `json:"column_count"`
	SizeBytes     int64     `json:"size_bytes"`
	NullRate      float64   `json:"null_rate"`
	DuplicateRate float64   `json:"duplicate_rate"`
	Completeness  float64   `json:"completeness"`
	LastProfiled  time.Time `json:"last_profiled"`
}

// Source represents a data source registered in the catalog.
type Source struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Type           SourceType        `json:"type"`
	Owner          string            `json:"owner"`
	AgentID        string            `json:"agent_id,omitempty"`
	Tags           []string          `json:"tags"`
	Schema         map[string]string `json:"schema,omitempty"`
	Classification string            `json:"classification"`
	Domain         string            `json:"domain"`
	Status         string            `json:"status"`
	Steward        string            `json:"steward"`
	QualityScore   float64           `json:"quality_score"`
	Freshness      *FreshnessInfo    `json:"freshness,omitempty"`
	Popularity     *PopularityInfo   `json:"popularity,omitempty"`
	Profile        *ProfileInfo      `json:"profile,omitempty"`
	Columns        []*Column         `json:"columns,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// LineageEdge represents a data flow relationship between two sources.
type LineageEdge struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	SourceID      string    `json:"source_id"`
	TargetID      string    `json:"target_id"`
	TransformType string    `json:"transform_type"`
	AgentID       string    `json:"agent_id"`
	Description   string    `json:"description"`
	SpanCount     int       `json:"span_count"`
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

// FullLineageGraph is the complete lineage graph for DAG visualization.
type FullLineageGraph struct {
	Nodes []*LineageGraphNode `json:"nodes"`
	Edges []*LineageGraphEdge `json:"edges"`
}

// LineageGraphNode is a node in the full lineage graph.
type LineageGraphNode struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Type         SourceType `json:"type"`
	Domain       string     `json:"domain"`
	Status       string     `json:"status"`
	QualityScore float64    `json:"quality_score"`
}

// LineageGraphEdge is an edge in the full lineage graph.
type LineageGraphEdge struct {
	Source        string `json:"source"`
	Target        string `json:"target"`
	TransformType string `json:"transform_type"`
	AgentID       string `json:"agent_id"`
	Label         string `json:"label"`
	SpanCount     int    `json:"span_count"`
}

// GlossaryTerm defines a business glossary entry.
type GlossaryTerm struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Term         string    `json:"term"`
	Definition   string    `json:"definition"`
	Domain       string    `json:"domain"`
	Owner        string    `json:"owner"`
	RelatedTerms []string  `json:"related_terms"`
	LinkedAssets []string  `json:"linked_assets"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SearchResult represents a catalog search result.
type SearchResult struct {
	Type        string            `json:"type"`
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Domain      string            `json:"domain,omitempty"`
	Score       float64           `json:"score"`
	Highlights  []string          `json:"highlights"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CatalogStats provides aggregate catalog statistics.
type CatalogStats struct {
	TotalSources       int            `json:"total_sources"`
	SourcesByType      map[string]int `json:"sources_by_type"`
	SourcesByDomain    map[string]int `json:"sources_by_domain"`
	SourcesByStatus    map[string]int `json:"sources_by_status"`
	TotalColumns       int            `json:"total_columns"`
	PIIColumns         int            `json:"pii_columns"`
	StaleCount         int            `json:"stale_count"`
	AvgQualityScore    float64        `json:"avg_quality_score"`
	TotalGlossaryTerms int            `json:"total_glossary_terms"`
	TotalLineageEdges  int            `json:"total_lineage_edges"`
	RecentlyUpdated    int            `json:"recently_updated"`
	TopDomains         []*DomainStat  `json:"top_domains"`
}

// DomainStat summarizes stats for a business domain.
type DomainStat struct {
	Domain      string  `json:"domain"`
	SourceCount int     `json:"source_count"`
	AvgQuality  float64 `json:"avg_quality"`
}

// ImpactAnalysis shows downstream impact of a source change.
type ImpactAnalysis struct {
	SourceID        string        `json:"source_id"`
	SourceName      string        `json:"source_name"`
	DownstreamCount int           `json:"downstream_count"`
	AffectedAgents  []string      `json:"affected_agents"`
	ImpactPaths     []*ImpactPath `json:"impact_paths"`
	RiskLevel       string        `json:"risk_level"`
}

// ImpactPath represents a single downstream impact chain.
type ImpactPath struct {
	Path      []ImpactNode `json:"path"`
	AgentID   string       `json:"agent_id"`
	Transform string       `json:"transform"`
}

// ImpactNode is a node in an impact path.
type ImpactNode struct {
	SourceID   string     `json:"source_id"`
	SourceName string     `json:"source_name"`
	SourceType SourceType `json:"source_type"`
}

// ColumnLineageEntry records column-level lineage.
type ColumnLineageEntry struct {
	SourceID     string `json:"source_id"`
	SourceName   string `json:"source_name"`
	SourceColumn string `json:"source_column"`
	TargetID     string `json:"target_id"`
	TargetName   string `json:"target_name"`
	TargetColumn string `json:"target_column"`
	Transform    string `json:"transform"`
	AgentID      string `json:"agent_id"`
}

// Repository provides in-memory storage for catalog sources and lineage.
type Repository struct {
	mu          sync.RWMutex
	sources     []*Source
	edges       []*LineageEdge
	glossary    []*GlossaryTerm
	colLineage  []*ColumnLineageEntry
	srcSeq      int
	edgeSeq     int
	glossarySeq int
}

// NewRepository creates a new catalog repository.
func NewRepository() *Repository {
	return &Repository{
		sources:    make([]*Source, 0),
		edges:      make([]*LineageEdge, 0),
		glossary:   make([]*GlossaryTerm, 0),
		colLineage: make([]*ColumnLineageEntry, 0),
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
		ID: fmt.Sprintf("src-%d", r.srcSeq), TenantID: tenantID,
		Name: name, Description: description, Type: srcType,
		Owner: owner, AgentID: agentID, Tags: tags, Schema: schema,
		Classification: "internal", Domain: "", Status: "active",
		QualityScore: 0, CreatedAt: now, UpdatedAt: now,
	}
	r.sources = append(r.sources, src)
	return src
}

// CreateSourceFull registers a source with all extended fields.
func (r *Repository) CreateSourceFull(tenantID, name, description string, srcType SourceType, owner, agentID string, tags []string, schema map[string]string, classification, domain, status, steward string, qualityScore float64, freshness *FreshnessInfo, profile *ProfileInfo, columns []*Column) *Source {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.srcSeq++
	now := time.Now()
	if tags == nil {
		tags = []string{}
	}
	if classification == "" {
		classification = "internal"
	}
	if status == "" {
		status = "active"
	}
	src := &Source{
		ID: fmt.Sprintf("src-%d", r.srcSeq), TenantID: tenantID,
		Name: name, Description: description, Type: srcType,
		Owner: owner, AgentID: agentID, Tags: tags, Schema: schema,
		Classification: classification, Domain: domain, Status: status,
		Steward: steward, QualityScore: qualityScore,
		Freshness: freshness, Profile: profile, Columns: columns,
		Popularity: &PopularityInfo{TrendDirection: "stable"},
		CreatedAt:  now, UpdatedAt: now,
	}
	r.sources = append(r.sources, src)
	return src
}

// ListSources returns catalog sources for a tenant with optional filters.
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

// ListSourcesFiltered returns sources with extended filters.
func (r *Repository) ListSourcesFiltered(tenantID string, srcType SourceType, domain, status, classification string) []*Source {
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
		if domain != "" && src.Domain != domain {
			continue
		}
		if status != "" && src.Status != status {
			continue
		}
		if classification != "" && src.Classification != classification {
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

// UpdateSourceExtended updates extended source fields.
func (r *Repository) UpdateSourceExtended(tenantID, sourceID, classification, domain, status, steward string, qualityScore float64) (*Source, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, src := range r.sources {
		if src.TenantID == tenantID && src.ID == sourceID {
			if classification != "" {
				src.Classification = classification
			}
			if domain != "" {
				src.Domain = domain
			}
			if status != "" {
				src.Status = status
			}
			if steward != "" {
				src.Steward = steward
			}
			if qualityScore > 0 {
				src.QualityScore = qualityScore
			}
			src.UpdatedAt = time.Now()
			return src, nil
		}
	}
	return nil, fmt.Errorf("source %s not found", sourceID)
}

// SetSourceColumns sets column metadata for a source.
func (r *Repository) SetSourceColumns(tenantID, sourceID string, columns []*Column) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, src := range r.sources {
		if src.TenantID == tenantID && src.ID == sourceID {
			src.Columns = columns
			src.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("source %s not found", sourceID)
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
		ID: fmt.Sprintf("edge-%d", r.edgeSeq), TenantID: tenantID,
		SourceID: sourceID, TargetID: targetID, TransformType: transformType,
		AgentID: agentID, Description: description, SpanCount: 1,
		CreatedAt: time.Now(),
	}
	r.edges = append(r.edges, edge)
	return edge, nil
}

// GetLineage returns the lineage graph (upstream and downstream) for a source.
func (r *Repository) GetLineage(tenantID, sourceID string) *LineageGraph {
	r.mu.RLock()
	defer r.mu.RUnlock()

	src := r.findSource(tenantID, sourceID)
	if src == nil {
		return nil
	}
	graph := &LineageGraph{
		SourceID:   sourceID,
		Upstream:   make([]*LineageNode, 0),
		Downstream: make([]*LineageNode, 0),
	}
	visited := make(map[string]bool)
	r.collectUpstream(tenantID, sourceID, visited, graph, 1)
	visited = make(map[string]bool)
	r.collectDownstream(tenantID, sourceID, visited, graph, 1)
	return graph
}

// GetFullLineageGraph returns the complete lineage graph for DAG visualization.
func (r *Repository) GetFullLineageGraph(tenantID string) *FullLineageGraph {
	r.mu.RLock()
	defer r.mu.RUnlock()

	graph := &FullLineageGraph{
		Nodes: make([]*LineageGraphNode, 0),
		Edges: make([]*LineageGraphEdge, 0),
	}
	nodeSet := make(map[string]bool)

	for _, edge := range r.edges {
		if edge.TenantID != tenantID {
			continue
		}
		// Add source node
		if !nodeSet[edge.SourceID] {
			if src := r.findSource(tenantID, edge.SourceID); src != nil {
				graph.Nodes = append(graph.Nodes, &LineageGraphNode{
					ID: src.ID, Name: src.Name, Type: src.Type,
					Domain: src.Domain, Status: src.Status, QualityScore: src.QualityScore,
				})
				nodeSet[edge.SourceID] = true
			}
		}
		// Add target node
		if !nodeSet[edge.TargetID] {
			if src := r.findSource(tenantID, edge.TargetID); src != nil {
				graph.Nodes = append(graph.Nodes, &LineageGraphNode{
					ID: src.ID, Name: src.Name, Type: src.Type,
					Domain: src.Domain, Status: src.Status, QualityScore: src.QualityScore,
				})
				nodeSet[edge.TargetID] = true
			}
		}
		graph.Edges = append(graph.Edges, &LineageGraphEdge{
			Source: edge.SourceID, Target: edge.TargetID,
			TransformType: edge.TransformType, AgentID: edge.AgentID,
			Label: edge.Description, SpanCount: edge.SpanCount,
		})
	}

	// Add orphan nodes (sources with no edges)
	for _, src := range r.sources {
		if src.TenantID == tenantID && !nodeSet[src.ID] {
			graph.Nodes = append(graph.Nodes, &LineageGraphNode{
				ID: src.ID, Name: src.Name, Type: src.Type,
				Domain: src.Domain, Status: src.Status, QualityScore: src.QualityScore,
			})
		}
	}
	return graph
}

// GetImpactAnalysis computes downstream impact for a source.
func (r *Repository) GetImpactAnalysis(tenantID, sourceID string) *ImpactAnalysis {
	r.mu.RLock()
	defer r.mu.RUnlock()

	src := r.findSource(tenantID, sourceID)
	if src == nil {
		return nil
	}

	ia := &ImpactAnalysis{
		SourceID:       sourceID,
		SourceName:     src.Name,
		AffectedAgents: make([]string, 0),
		ImpactPaths:    make([]*ImpactPath, 0),
	}

	visited := make(map[string]bool)
	agentSet := make(map[string]bool)
	r.collectImpact(tenantID, sourceID, visited, agentSet, ia, []ImpactNode{{SourceID: src.ID, SourceName: src.Name, SourceType: src.Type}})

	ia.DownstreamCount = len(visited)
	for agent := range agentSet {
		ia.AffectedAgents = append(ia.AffectedAgents, agent)
	}

	if ia.DownstreamCount > 5 {
		ia.RiskLevel = "critical"
	} else if ia.DownstreamCount > 3 {
		ia.RiskLevel = "high"
	} else if ia.DownstreamCount > 0 {
		ia.RiskLevel = "medium"
	} else {
		ia.RiskLevel = "low"
	}
	return ia
}

func (r *Repository) collectImpact(tenantID, sourceID string, visited, agentSet map[string]bool, ia *ImpactAnalysis, currentPath []ImpactNode) {
	for _, edge := range r.edges {
		if edge.TenantID == tenantID && edge.SourceID == sourceID && !visited[edge.TargetID] {
			visited[edge.TargetID] = true
			if edge.AgentID != "" {
				agentSet[edge.AgentID] = true
			}
			target := r.findSource(tenantID, edge.TargetID)
			targetName := ""
			var targetType SourceType
			if target != nil {
				targetName = target.Name
				targetType = target.Type
			}
			newPath := make([]ImpactNode, len(currentPath))
			copy(newPath, currentPath)
			newPath = append(newPath, ImpactNode{SourceID: edge.TargetID, SourceName: targetName, SourceType: targetType})
			ia.ImpactPaths = append(ia.ImpactPaths, &ImpactPath{Path: newPath, AgentID: edge.AgentID, Transform: edge.TransformType})
			r.collectImpact(tenantID, edge.TargetID, visited, agentSet, ia, newPath)
		}
	}
}

// SearchCatalog performs full-text search across sources, columns, and glossary.
func (r *Repository) SearchCatalog(tenantID, query string, filters map[string]string) []*SearchResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*SearchResult
	q := strings.ToLower(query)

	for _, src := range r.sources {
		if src.TenantID != tenantID {
			continue
		}
		if ft, ok := filters["type"]; ok && ft != "" && string(src.Type) != ft {
			continue
		}
		if fd, ok := filters["domain"]; ok && fd != "" && src.Domain != fd {
			continue
		}
		if fc, ok := filters["classification"]; ok && fc != "" && src.Classification != fc {
			continue
		}

		score := 0.0
		var highlights []string
		nameLower := strings.ToLower(src.Name)
		descLower := strings.ToLower(src.Description)

		if strings.Contains(nameLower, q) {
			score += 10.0
			highlights = append(highlights, "Name: "+src.Name)
		}
		if strings.Contains(descLower, q) {
			score += 5.0
			highlights = append(highlights, "Description match")
		}
		for _, tag := range src.Tags {
			if strings.Contains(strings.ToLower(tag), q) {
				score += 3.0
				highlights = append(highlights, "Tag: "+tag)
			}
		}
		if strings.Contains(strings.ToLower(src.Owner), q) {
			score += 2.0
			highlights = append(highlights, "Owner: "+src.Owner)
		}

		// Search columns
		for _, col := range src.Columns {
			if strings.Contains(strings.ToLower(col.Name), q) {
				score += 4.0
				highlights = append(highlights, "Column: "+col.Name)
			}
		}

		if score > 0 {
			results = append(results, &SearchResult{
				Type: "source", ID: src.ID, Name: src.Name,
				Description: src.Description, Domain: src.Domain,
				Score: score, Highlights: highlights,
				Metadata: map[string]string{"type": string(src.Type), "status": src.Status},
			})
		}
	}

	// Search glossary terms
	for _, term := range r.glossary {
		if term.TenantID != tenantID {
			continue
		}
		score := 0.0
		var highlights []string
		if strings.Contains(strings.ToLower(term.Term), q) {
			score += 8.0
			highlights = append(highlights, "Term: "+term.Term)
		}
		if strings.Contains(strings.ToLower(term.Definition), q) {
			score += 4.0
			highlights = append(highlights, "Definition match")
		}
		if score > 0 {
			results = append(results, &SearchResult{
				Type: "glossary", ID: term.ID, Name: term.Term,
				Description: term.Definition, Domain: term.Domain,
				Score: score, Highlights: highlights,
			})
		}
	}
	return results
}

// GetCatalogStats computes aggregate statistics.
func (r *Repository) GetCatalogStats(tenantID string) *CatalogStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &CatalogStats{
		SourcesByType:   make(map[string]int),
		SourcesByDomain: make(map[string]int),
		SourcesByStatus: make(map[string]int),
		TopDomains:      make([]*DomainStat, 0),
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	var totalQuality float64
	domainQuality := make(map[string][]float64)

	for _, src := range r.sources {
		if src.TenantID != tenantID {
			continue
		}
		stats.TotalSources++
		stats.SourcesByType[string(src.Type)]++
		if src.Domain != "" {
			stats.SourcesByDomain[src.Domain]++
			domainQuality[src.Domain] = append(domainQuality[src.Domain], src.QualityScore)
		}
		stats.SourcesByStatus[src.Status]++
		totalQuality += src.QualityScore

		stats.TotalColumns += len(src.Columns)
		for _, col := range src.Columns {
			if col.IsPII {
				stats.PIIColumns++
			}
		}
		if src.Freshness != nil && src.Freshness.IsStale {
			stats.StaleCount++
		}
		if src.UpdatedAt.After(cutoff) {
			stats.RecentlyUpdated++
		}
	}

	if stats.TotalSources > 0 {
		stats.AvgQualityScore = totalQuality / float64(stats.TotalSources)
	}

	for domain, scores := range domainQuality {
		avg := 0.0
		for _, s := range scores {
			avg += s
		}
		if len(scores) > 0 {
			avg /= float64(len(scores))
		}
		stats.TopDomains = append(stats.TopDomains, &DomainStat{
			Domain: domain, SourceCount: len(scores), AvgQuality: avg,
		})
	}

	for _, term := range r.glossary {
		if term.TenantID == tenantID {
			stats.TotalGlossaryTerms++
		}
	}
	for _, edge := range r.edges {
		if edge.TenantID == tenantID {
			stats.TotalLineageEdges++
		}
	}
	return stats
}

// TrackPopularity increments usage counters for a source.
func (r *Repository) TrackPopularity(tenantID, sourceID, action string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, src := range r.sources {
		if src.TenantID == tenantID && src.ID == sourceID {
			if src.Popularity == nil {
				src.Popularity = &PopularityInfo{TrendDirection: "stable"}
			}
			switch action {
			case "view":
				src.Popularity.ViewCount++
			case "query":
				src.Popularity.QueryCount++
			}
		}
	}
}

// CreateGlossaryTerm adds a new business glossary term.
func (r *Repository) CreateGlossaryTerm(tenantID, term, definition, domain, owner string, relatedTerms, linkedAssets []string) *GlossaryTerm {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.glossarySeq++
	now := time.Now()
	if relatedTerms == nil {
		relatedTerms = []string{}
	}
	if linkedAssets == nil {
		linkedAssets = []string{}
	}
	gt := &GlossaryTerm{
		ID: fmt.Sprintf("gt-%d", r.glossarySeq), TenantID: tenantID,
		Term: term, Definition: definition, Domain: domain, Owner: owner,
		RelatedTerms: relatedTerms, LinkedAssets: linkedAssets,
		CreatedAt: now, UpdatedAt: now,
	}
	r.glossary = append(r.glossary, gt)
	return gt
}

// ListGlossaryTerms returns all glossary terms for a tenant with optional domain filter.
func (r *Repository) ListGlossaryTerms(tenantID, domain string) []*GlossaryTerm {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*GlossaryTerm
	for _, gt := range r.glossary {
		if gt.TenantID != tenantID {
			continue
		}
		if domain != "" && gt.Domain != domain {
			continue
		}
		result = append(result, gt)
	}
	return result
}

// UpdateGlossaryTerm updates a glossary term.
func (r *Repository) UpdateGlossaryTerm(tenantID, termID, term, definition, domain string) (*GlossaryTerm, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, gt := range r.glossary {
		if gt.TenantID == tenantID && gt.ID == termID {
			if term != "" {
				gt.Term = term
			}
			if definition != "" {
				gt.Definition = definition
			}
			if domain != "" {
				gt.Domain = domain
			}
			gt.UpdatedAt = time.Now()
			return gt, nil
		}
	}
	return nil, fmt.Errorf("glossary term %s not found", termID)
}

// DeleteGlossaryTerm removes a glossary term.
func (r *Repository) DeleteGlossaryTerm(tenantID, termID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, gt := range r.glossary {
		if gt.TenantID == tenantID && gt.ID == termID {
			r.glossary = append(r.glossary[:i], r.glossary[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("glossary term %s not found", termID)
}

// RecordColumnLineage records a column-level lineage entry.
func (r *Repository) RecordColumnLineage(tenantID, sourceID, sourceCol, targetID, targetCol, transform, agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	srcName, tgtName := "", ""
	for _, src := range r.sources {
		if src.TenantID == tenantID {
			if src.ID == sourceID {
				srcName = src.Name
			}
			if src.ID == targetID {
				tgtName = src.Name
			}
		}
	}
	r.colLineage = append(r.colLineage, &ColumnLineageEntry{
		SourceID: sourceID, SourceName: srcName, SourceColumn: sourceCol,
		TargetID: targetID, TargetName: tgtName, TargetColumn: targetCol,
		Transform: transform, AgentID: agentID,
	})
}

// GetColumnLineage returns column-level lineage for a source.
func (r *Repository) GetColumnLineage(tenantID, sourceID, columnName string) []*ColumnLineageEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*ColumnLineageEntry
	for _, cl := range r.colLineage {
		if cl.SourceID == sourceID || cl.TargetID == sourceID {
			if columnName != "" && cl.SourceColumn != columnName && cl.TargetColumn != columnName {
				continue
			}
			result = append(result, cl)
		}
	}
	return result
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

func (r *Repository) collectUpstream(tenantID, targetID string, visited map[string]bool, graph *LineageGraph, depth int) {
	if depth > 10 {
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
				SourceID: edge.SourceID, SourceName: srcName,
				TransformType: edge.TransformType, AgentID: edge.AgentID, Depth: depth,
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
				SourceID: edge.TargetID, SourceName: srcName,
				TransformType: edge.TransformType, AgentID: edge.AgentID, Depth: depth,
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

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
