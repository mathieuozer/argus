package catalog

import "context"

// Store is the interface the Handler uses to access catalog data.
// All methods accept ctx as the first argument and return errors so that
// implementations backed by a database (PGRepository) and implementations
// backed by the in-memory Repository can be used interchangeably.
type Store interface {
	// Source CRUD
	CreateSourceFull(ctx context.Context, tenantID, name, description string, srcType SourceType, owner, agentID string, tags []string, schema map[string]string, classification, domain, status, steward string, qualityScore float64, freshness *FreshnessInfo, profile *ProfileInfo, columns []*Column) (*Source, error)
	ListSources(ctx context.Context, tenantID string, srcType SourceType, tag string) ([]*Source, error)
	ListSourcesFiltered(ctx context.Context, tenantID string, srcType SourceType, domain, status, classification string) ([]*Source, error)
	GetSource(ctx context.Context, tenantID, sourceID string) (*Source, error)
	UpdateSource(ctx context.Context, tenantID, sourceID, name, description, owner string, tags []string) (*Source, error)
	UpdateSourceExtended(ctx context.Context, tenantID, sourceID, classification, domain, status, steward string, qualityScore float64) (*Source, error)
	SetSourceColumns(ctx context.Context, tenantID, sourceID string, columns []*Column) error
	DeleteSource(ctx context.Context, tenantID, sourceID string) error
	TrackPopularity(ctx context.Context, tenantID, sourceID, action string)

	// Lineage
	AddLineageEdge(ctx context.Context, tenantID, sourceID, targetID, transformType, agentID, description string) (*LineageEdge, error)
	ListEdges(ctx context.Context, tenantID string) ([]*LineageEdge, error)
	GetLineage(ctx context.Context, tenantID, sourceID string) (*LineageGraph, error)
	GetFullLineageGraph(ctx context.Context, tenantID string) (*FullLineageGraph, error)
	GetImpactAnalysis(ctx context.Context, tenantID, sourceID string) (*ImpactAnalysis, error)
	GetColumnLineage(ctx context.Context, tenantID, sourceID, columnName string) ([]*ColumnLineageEntry, error)
	RecordColumnLineage(ctx context.Context, tenantID, sourceID, sourceCol, targetID, targetCol, transform, agentID string)

	// Glossary
	CreateGlossaryTerm(ctx context.Context, tenantID, term, definition, domain, owner string, relatedTerms, linkedAssets []string) (*GlossaryTerm, error)
	ListGlossaryTerms(ctx context.Context, tenantID, domain string) ([]*GlossaryTerm, error)
	UpdateGlossaryTerm(ctx context.Context, tenantID, termID, term, definition, domain string) (*GlossaryTerm, error)
	DeleteGlossaryTerm(ctx context.Context, tenantID, termID string) error

	// Search & Stats
	SearchCatalog(ctx context.Context, tenantID, query string, filters map[string]string) ([]*SearchResult, error)
	GetCatalogStats(ctx context.Context, tenantID string) (*CatalogStats, error)
}

// memStore adapts the in-memory *Repository to satisfy the Store interface.
// It ignores ctx and wraps value-only returns with nil errors.
type memStore struct {
	r *Repository
}

// NewMemStore wraps a *Repository as a Store.
func NewMemStore(r *Repository) Store {
	return &memStore{r: r}
}

func (m *memStore) CreateSourceFull(ctx context.Context, tenantID, name, description string, srcType SourceType, owner, agentID string, tags []string, schema map[string]string, classification, domain, status, steward string, qualityScore float64, freshness *FreshnessInfo, profile *ProfileInfo, columns []*Column) (*Source, error) {
	return m.r.CreateSourceFull(tenantID, name, description, srcType, owner, agentID, tags, schema, classification, domain, status, steward, qualityScore, freshness, profile, columns), nil
}

func (m *memStore) ListSources(ctx context.Context, tenantID string, srcType SourceType, tag string) ([]*Source, error) {
	return m.r.ListSources(tenantID, srcType, tag), nil
}

func (m *memStore) ListSourcesFiltered(ctx context.Context, tenantID string, srcType SourceType, domain, status, classification string) ([]*Source, error) {
	return m.r.ListSourcesFiltered(tenantID, srcType, domain, status, classification), nil
}

func (m *memStore) GetSource(ctx context.Context, tenantID, sourceID string) (*Source, error) {
	return m.r.GetSource(tenantID, sourceID), nil
}

func (m *memStore) UpdateSource(ctx context.Context, tenantID, sourceID, name, description, owner string, tags []string) (*Source, error) {
	return m.r.UpdateSource(tenantID, sourceID, name, description, owner, tags)
}

func (m *memStore) UpdateSourceExtended(ctx context.Context, tenantID, sourceID, classification, domain, status, steward string, qualityScore float64) (*Source, error) {
	return m.r.UpdateSourceExtended(tenantID, sourceID, classification, domain, status, steward, qualityScore)
}

func (m *memStore) SetSourceColumns(ctx context.Context, tenantID, sourceID string, columns []*Column) error {
	return m.r.SetSourceColumns(tenantID, sourceID, columns)
}

func (m *memStore) DeleteSource(ctx context.Context, tenantID, sourceID string) error {
	return m.r.DeleteSource(tenantID, sourceID)
}

func (m *memStore) TrackPopularity(ctx context.Context, tenantID, sourceID, action string) {
	m.r.TrackPopularity(tenantID, sourceID, action)
}

func (m *memStore) AddLineageEdge(ctx context.Context, tenantID, sourceID, targetID, transformType, agentID, description string) (*LineageEdge, error) {
	return m.r.AddLineageEdge(tenantID, sourceID, targetID, transformType, agentID, description)
}

func (m *memStore) ListEdges(ctx context.Context, tenantID string) ([]*LineageEdge, error) {
	return m.r.ListEdges(tenantID), nil
}

func (m *memStore) GetLineage(ctx context.Context, tenantID, sourceID string) (*LineageGraph, error) {
	return m.r.GetLineage(tenantID, sourceID), nil
}

func (m *memStore) GetFullLineageGraph(ctx context.Context, tenantID string) (*FullLineageGraph, error) {
	return m.r.GetFullLineageGraph(tenantID), nil
}

func (m *memStore) GetImpactAnalysis(ctx context.Context, tenantID, sourceID string) (*ImpactAnalysis, error) {
	return m.r.GetImpactAnalysis(tenantID, sourceID), nil
}

func (m *memStore) GetColumnLineage(ctx context.Context, tenantID, sourceID, columnName string) ([]*ColumnLineageEntry, error) {
	return m.r.GetColumnLineage(tenantID, sourceID, columnName), nil
}

func (m *memStore) RecordColumnLineage(ctx context.Context, tenantID, sourceID, sourceCol, targetID, targetCol, transform, agentID string) {
	m.r.RecordColumnLineage(tenantID, sourceID, sourceCol, targetID, targetCol, transform, agentID)
}

func (m *memStore) CreateGlossaryTerm(ctx context.Context, tenantID, term, definition, domain, owner string, relatedTerms, linkedAssets []string) (*GlossaryTerm, error) {
	return m.r.CreateGlossaryTerm(tenantID, term, definition, domain, owner, relatedTerms, linkedAssets), nil
}

func (m *memStore) ListGlossaryTerms(ctx context.Context, tenantID, domain string) ([]*GlossaryTerm, error) {
	return m.r.ListGlossaryTerms(tenantID, domain), nil
}

func (m *memStore) UpdateGlossaryTerm(ctx context.Context, tenantID, termID, term, definition, domain string) (*GlossaryTerm, error) {
	return m.r.UpdateGlossaryTerm(tenantID, termID, term, definition, domain)
}

func (m *memStore) DeleteGlossaryTerm(ctx context.Context, tenantID, termID string) error {
	return m.r.DeleteGlossaryTerm(tenantID, termID)
}

func (m *memStore) SearchCatalog(ctx context.Context, tenantID, query string, filters map[string]string) ([]*SearchResult, error) {
	return m.r.SearchCatalog(tenantID, query, filters), nil
}

func (m *memStore) GetCatalogStats(ctx context.Context, tenantID string) (*CatalogStats, error) {
	return m.r.GetCatalogStats(tenantID), nil
}
