package dataquality

import "context"

// Store is the interface the Handler uses to interact with data quality storage.
// All methods take ctx as the first argument to allow both in-memory and
// PostgreSQL-backed implementations.
type Store interface {
	// Rules
	CreateRule(ctx context.Context, tenantID, name, description string, ruleType RuleType, agentID, field, operator, threshold string, severity Severity) (*Rule, error)
	ListRules(ctx context.Context, tenantID, agentID string) ([]*Rule, error)
	GetRule(ctx context.Context, tenantID, ruleID string) (*Rule, error)
	UpdateRule(ctx context.Context, tenantID, ruleID string, name, description string, enabled bool) (*Rule, error)
	DeleteRule(ctx context.Context, tenantID, ruleID string) error

	// Scores
	RecordScore(ctx context.Context, tenantID, agentID string, overall, completeness, accuracy, consistency, timeliness float64, total, passed, failed int) (*Score, error)
	ListScores(ctx context.Context, tenantID, agentID string) ([]*Score, error)
	GetLatestScore(ctx context.Context, tenantID, agentID string) (*Score, error)

	// Violations
	ListViolations(ctx context.Context, tenantID, agentID, ruleID string) ([]*Violation, error)

	// Profiles
	ListProfiles(ctx context.Context, tenantID, agentID string) ([]*DataProfile, error)
	GetLatestProfile(ctx context.Context, tenantID, agentID string) (*DataProfile, error)

	// Contracts
	CreateContract(ctx context.Context, tenantID, name, desc, producer string, consumers []string, sourceID string, schema map[string]string, freshness *FreshnessSpec, quality *QualitySpec) (*DataContract, error)
	ListContracts(ctx context.Context, tenantID, agentID string) ([]*DataContract, error)
	GetContract(ctx context.Context, tenantID, contractID string) (*DataContract, error)
	UpdateContractStatus(ctx context.Context, tenantID, contractID, status string) (*DataContract, error)
	DeleteContract(ctx context.Context, tenantID, contractID string) error

	// Trends and drift
	GetQualityTrend(ctx context.Context, tenantID, agentID string, days int) (*QualityTrend, error)
	GetDrift(ctx context.Context, tenantID, agentID string) ([]*DriftPoint, error)

	// Incidents
	RecordIncident(ctx context.Context, tenantID, agentID, contractID, title, desc string, severity Severity, violationIDs []string) (*QualityIncident, error)
	ListIncidents(ctx context.Context, tenantID, status string) ([]*QualityIncident, error)
	UpdateIncidentStatus(ctx context.Context, tenantID, incidentID, status string) (*QualityIncident, error)

	// Anomalies
	ListAnomalies(ctx context.Context, tenantID, agentID string) ([]*Anomaly, error)

	// Summary
	GetSummary(ctx context.Context, tenantID string) (*DQSummary, error)
}

// memStore wraps *Repository and satisfies Store. The ctx argument is ignored
// because the in-memory Repository does not need it.
type memStore struct {
	r *Repository
}

// NewMemStore returns a Store backed by an in-memory Repository.
func NewMemStore() (Store, *Repository) {
	r := NewRepository()
	return &memStore{r: r}, r
}

// Compile-time assertion that *memStore satisfies Store.
var _ Store = (*memStore)(nil)

// --- Rules ---

func (m *memStore) CreateRule(_ context.Context, tenantID, name, description string, ruleType RuleType, agentID, field, operator, threshold string, severity Severity) (*Rule, error) {
	return m.r.CreateRule(tenantID, name, description, ruleType, agentID, field, operator, threshold, severity), nil
}

func (m *memStore) ListRules(_ context.Context, tenantID, agentID string) ([]*Rule, error) {
	return m.r.ListRules(tenantID, agentID), nil
}

func (m *memStore) GetRule(_ context.Context, tenantID, ruleID string) (*Rule, error) {
	return m.r.GetRule(tenantID, ruleID), nil
}

func (m *memStore) UpdateRule(_ context.Context, tenantID, ruleID string, name, description string, enabled bool) (*Rule, error) {
	return m.r.UpdateRule(tenantID, ruleID, name, description, enabled)
}

func (m *memStore) DeleteRule(_ context.Context, tenantID, ruleID string) error {
	return m.r.DeleteRule(tenantID, ruleID)
}

// --- Scores ---

func (m *memStore) RecordScore(_ context.Context, tenantID, agentID string, overall, completeness, accuracy, consistency, timeliness float64, total, passed, failed int) (*Score, error) {
	return m.r.RecordScore(tenantID, agentID, overall, completeness, accuracy, consistency, timeliness, total, passed, failed), nil
}

func (m *memStore) ListScores(_ context.Context, tenantID, agentID string) ([]*Score, error) {
	return m.r.ListScores(tenantID, agentID), nil
}

func (m *memStore) GetLatestScore(_ context.Context, tenantID, agentID string) (*Score, error) {
	return m.r.GetLatestScore(tenantID, agentID), nil
}

// --- Violations ---

func (m *memStore) ListViolations(_ context.Context, tenantID, agentID, ruleID string) ([]*Violation, error) {
	return m.r.ListViolations(tenantID, agentID, ruleID), nil
}

// --- Profiles ---

func (m *memStore) ListProfiles(_ context.Context, tenantID, agentID string) ([]*DataProfile, error) {
	return m.r.ListProfiles(tenantID, agentID), nil
}

func (m *memStore) GetLatestProfile(_ context.Context, tenantID, agentID string) (*DataProfile, error) {
	return m.r.GetLatestProfile(tenantID, agentID), nil
}

// --- Contracts ---

func (m *memStore) CreateContract(_ context.Context, tenantID, name, desc, producer string, consumers []string, sourceID string, schema map[string]string, freshness *FreshnessSpec, quality *QualitySpec) (*DataContract, error) {
	return m.r.CreateContract(tenantID, name, desc, producer, consumers, sourceID, schema, freshness, quality), nil
}

func (m *memStore) ListContracts(_ context.Context, tenantID, agentID string) ([]*DataContract, error) {
	return m.r.ListContracts(tenantID, agentID), nil
}

func (m *memStore) GetContract(_ context.Context, tenantID, contractID string) (*DataContract, error) {
	return m.r.GetContract(tenantID, contractID), nil
}

func (m *memStore) UpdateContractStatus(_ context.Context, tenantID, contractID, status string) (*DataContract, error) {
	return m.r.UpdateContractStatus(tenantID, contractID, status)
}

func (m *memStore) DeleteContract(_ context.Context, tenantID, contractID string) error {
	return m.r.DeleteContract(tenantID, contractID)
}

// --- Trends and drift ---

func (m *memStore) GetQualityTrend(_ context.Context, tenantID, agentID string, days int) (*QualityTrend, error) {
	return m.r.GetQualityTrend(tenantID, agentID, days), nil
}

func (m *memStore) GetDrift(_ context.Context, tenantID, agentID string) ([]*DriftPoint, error) {
	return m.r.GetDrift(tenantID, agentID), nil
}

// --- Incidents ---

func (m *memStore) RecordIncident(_ context.Context, tenantID, agentID, contractID, title, desc string, severity Severity, violationIDs []string) (*QualityIncident, error) {
	return m.r.RecordIncident(tenantID, agentID, contractID, title, desc, severity, violationIDs), nil
}

func (m *memStore) ListIncidents(_ context.Context, tenantID, status string) ([]*QualityIncident, error) {
	return m.r.ListIncidents(tenantID, status), nil
}

func (m *memStore) UpdateIncidentStatus(_ context.Context, tenantID, incidentID, status string) (*QualityIncident, error) {
	return m.r.UpdateIncidentStatus(tenantID, incidentID, status)
}

// --- Anomalies ---

func (m *memStore) ListAnomalies(_ context.Context, tenantID, agentID string) ([]*Anomaly, error) {
	return m.r.ListAnomalies(tenantID, agentID), nil
}

// --- Summary ---

func (m *memStore) GetSummary(_ context.Context, tenantID string) (*DQSummary, error) {
	return m.r.GetSummary(tenantID), nil
}
