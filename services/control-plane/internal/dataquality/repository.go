package dataquality

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// RuleType defines the type of data quality check.
type RuleType string

const (
	RuleTypeCompleteness RuleType = "completeness"
	RuleTypeAccuracy     RuleType = "accuracy"
	RuleTypeConsistency  RuleType = "consistency"
	RuleTypeTimeliness   RuleType = "timeliness"
	RuleTypeUniqueness   RuleType = "uniqueness"
)

// Severity indicates how critical a data quality violation is.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Rule defines a data quality check to be applied to agent telemetry.
type Rule struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        RuleType  `json:"type"`
	AgentID     string    `json:"agent_id,omitempty"`
	Field       string    `json:"field"`
	Operator    string    `json:"operator"`
	Threshold   string    `json:"threshold"`
	Severity    Severity  `json:"severity"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Score represents an aggregated data quality score for an agent.
type Score struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	AgentID           string    `json:"agent_id"`
	OverallScore      float64   `json:"overall_score"`
	CompletenessScore float64   `json:"completeness_score"`
	AccuracyScore     float64   `json:"accuracy_score"`
	ConsistencyScore  float64   `json:"consistency_score"`
	TimelinessScore   float64   `json:"timeliness_score"`
	TotalChecks       int       `json:"total_checks"`
	PassedChecks      int       `json:"passed_checks"`
	FailedChecks      int       `json:"failed_checks"`
	EvaluatedAt       time.Time `json:"evaluated_at"`
}

// Violation records a single data quality rule violation.
type Violation struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	RuleID     string    `json:"rule_id"`
	RuleName   string    `json:"rule_name"`
	AgentID    string    `json:"agent_id"`
	Field      string    `json:"field"`
	Value      string    `json:"value"`
	Expected   string    `json:"expected"`
	Severity   Severity  `json:"severity"`
	Message    string    `json:"message"`
	OccurredAt time.Time `json:"occurred_at"`
}

// ColumnProfile captures profiling results for a single column.
type ColumnProfile struct {
	Name         string       `json:"name"`
	Type         string       `json:"type"`
	NullRate     float64      `json:"null_rate"`
	UniqueRate   float64      `json:"unique_rate"`
	MinValue     string       `json:"min_value,omitempty"`
	MaxValue     string       `json:"max_value,omitempty"`
	MeanValue    string       `json:"mean_value,omitempty"`
	TopValues    []string     `json:"top_values,omitempty"`
	Distribution []DistBucket `json:"distribution,omitempty"`
}

// DistBucket is a distribution histogram bucket.
type DistBucket struct {
	Label string  `json:"label"`
	Count int     `json:"count"`
	Pct   float64 `json:"pct"`
}

// DataProfile captures automated profiling results.
type DataProfile struct {
	ID             string           `json:"id"`
	TenantID       string           `json:"tenant_id"`
	AgentID        string           `json:"agent_id"`
	SourceID       string           `json:"source_id,omitempty"`
	RowCount       int64            `json:"row_count"`
	ColumnCount    int              `json:"column_count"`
	NullRate       float64          `json:"null_rate"`
	DuplicateRate  float64          `json:"duplicate_rate"`
	Completeness   float64          `json:"completeness"`
	ColumnProfiles []*ColumnProfile `json:"column_profiles"`
	ProfiledAt     time.Time        `json:"profiled_at"`
}

// FreshnessSpec defines freshness requirements in a data contract.
type FreshnessSpec struct {
	MaxStalenessSeconds int    `json:"max_staleness_seconds"`
	RefreshSchedule     string `json:"refresh_schedule"`
}

// QualitySpec defines quality requirements in a data contract.
type QualitySpec struct {
	MinCompleteness float64 `json:"min_completeness"`
	MinAccuracy     float64 `json:"min_accuracy"`
	MaxNullRate     float64 `json:"max_null_rate"`
}

// DataContract defines a producer-consumer data agreement.
type DataContract struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	ProducerAgent  string            `json:"producer_agent"`
	ConsumerAgents []string          `json:"consumer_agents"`
	SourceID       string            `json:"source_id"`
	SchemaSpec     map[string]string `json:"schema_spec"`
	FreshnessSpec  *FreshnessSpec    `json:"freshness_spec,omitempty"`
	QualitySpec    *QualitySpec      `json:"quality_spec,omitempty"`
	Status         string            `json:"status"`
	LastCheckedAt  time.Time         `json:"last_checked_at"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// TrendPoint is a single point in a quality trend.
type TrendPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	Overall      float64   `json:"overall"`
	Completeness float64   `json:"completeness"`
	Accuracy     float64   `json:"accuracy"`
	Consistency  float64   `json:"consistency"`
	Timeliness   float64   `json:"timeliness"`
}

// QualityTrend shows quality over time for an agent.
type QualityTrend struct {
	AgentID string        `json:"agent_id"`
	Points  []*TrendPoint `json:"points"`
}

// QualityIncident represents a data quality incident.
type QualityIncident struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	AgentID      string     `json:"agent_id"`
	ContractID   string     `json:"contract_id,omitempty"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Severity     Severity   `json:"severity"`
	Status       string     `json:"status"`
	ViolationIDs []string   `json:"violation_ids"`
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

// Anomaly records a detected quality anomaly.
type Anomaly struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	AgentID    string    `json:"agent_id"`
	MetricName string    `json:"metric_name"`
	Expected   float64   `json:"expected"`
	Actual     float64   `json:"actual"`
	Deviation  float64   `json:"deviation"`
	Severity   Severity  `json:"severity"`
	DetectedAt time.Time `json:"detected_at"`
}

// DriftPoint captures a drift measurement.
type DriftPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Baseline  float64   `json:"baseline"`
	IsAnomaly bool      `json:"is_anomaly"`
}

// DQSummary provides an overview of data quality status.
type DQSummary struct {
	TotalRules         int     `json:"total_rules"`
	ActiveRules        int     `json:"active_rules"`
	TotalViolations    int     `json:"total_violations"`
	CriticalViolations int     `json:"critical_violations"`
	AvgQualityScore    float64 `json:"avg_quality_score"`
	TotalContracts     int     `json:"total_contracts"`
	ActiveContracts    int     `json:"active_contracts"`
	ViolatedContracts  int     `json:"violated_contracts"`
	TotalIncidents     int     `json:"total_incidents"`
	OpenIncidents      int     `json:"open_incidents"`
	TotalAnomalies     int     `json:"total_anomalies"`
	AgentCount         int     `json:"agent_count"`
}

// Repository provides in-memory storage for data quality.
type Repository struct {
	mu          sync.RWMutex
	rules       []*Rule
	scores      []*Score
	violations  []*Violation
	profiles    []*DataProfile
	contracts   []*DataContract
	incidents   []*QualityIncident
	anomalies   []*Anomaly
	ruleSeq     int
	scoreSeq    int
	violSeq     int
	profileSeq  int
	contractSeq int
	incidentSeq int
	anomalySeq  int
}

// NewRepository creates a new data quality repository.
func NewRepository() *Repository {
	return &Repository{
		rules:      make([]*Rule, 0),
		scores:     make([]*Score, 0),
		violations: make([]*Violation, 0),
		profiles:   make([]*DataProfile, 0),
		contracts:  make([]*DataContract, 0),
		incidents:  make([]*QualityIncident, 0),
		anomalies:  make([]*Anomaly, 0),
	}
}

// CreateRule adds a new data quality rule.
func (r *Repository) CreateRule(tenantID, name, description string, ruleType RuleType, agentID, field, operator, threshold string, severity Severity) *Rule {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ruleSeq++
	now := time.Now()
	rule := &Rule{
		ID: fmt.Sprintf("dqr-%d", r.ruleSeq), TenantID: tenantID,
		Name: name, Description: description, Type: ruleType,
		AgentID: agentID, Field: field, Operator: operator,
		Threshold: threshold, Severity: severity, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	r.rules = append(r.rules, rule)
	return rule
}

// ListRules returns rules for a tenant, optionally filtered by agent ID.
func (r *Repository) ListRules(tenantID, agentID string) []*Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Rule
	for _, rule := range r.rules {
		if rule.TenantID != tenantID {
			continue
		}
		if agentID != "" && rule.AgentID != "" && rule.AgentID != agentID {
			continue
		}
		result = append(result, rule)
	}
	return result
}

// GetRule returns a specific rule.
func (r *Repository) GetRule(tenantID, ruleID string) *Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rule := range r.rules {
		if rule.TenantID == tenantID && rule.ID == ruleID {
			return rule
		}
	}
	return nil
}

// UpdateRule updates a rule.
func (r *Repository) UpdateRule(tenantID, ruleID string, name, description string, enabled bool) (*Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rule := range r.rules {
		if rule.TenantID == tenantID && rule.ID == ruleID {
			if name != "" {
				rule.Name = name
			}
			if description != "" {
				rule.Description = description
			}
			rule.Enabled = enabled
			rule.UpdatedAt = time.Now()
			return rule, nil
		}
	}
	return nil, fmt.Errorf("rule %s not found", ruleID)
}

// DeleteRule removes a rule.
func (r *Repository) DeleteRule(tenantID, ruleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, rule := range r.rules {
		if rule.TenantID == tenantID && rule.ID == ruleID {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule %s not found", ruleID)
}

// RecordScore records a data quality score.
func (r *Repository) RecordScore(tenantID, agentID string, overall, completeness, accuracy, consistency, timeliness float64, total, passed, failed int) *Score {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scoreSeq++
	score := &Score{
		ID: fmt.Sprintf("dqs-%d", r.scoreSeq), TenantID: tenantID,
		AgentID: agentID, OverallScore: overall,
		CompletenessScore: completeness, AccuracyScore: accuracy,
		ConsistencyScore: consistency, TimelinessScore: timeliness,
		TotalChecks: total, PassedChecks: passed, FailedChecks: failed,
		EvaluatedAt: time.Now(),
	}
	r.scores = append(r.scores, score)
	return score
}

// ListScores returns quality scores for a tenant.
func (r *Repository) ListScores(tenantID, agentID string) []*Score {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Score
	for _, score := range r.scores {
		if score.TenantID != tenantID {
			continue
		}
		if agentID != "" && score.AgentID != agentID {
			continue
		}
		result = append(result, score)
	}
	return result
}

// GetLatestScore returns the latest score for an agent.
func (r *Repository) GetLatestScore(tenantID, agentID string) *Score {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var latest *Score
	for _, score := range r.scores {
		if score.TenantID == tenantID && score.AgentID == agentID {
			if latest == nil || score.EvaluatedAt.After(latest.EvaluatedAt) {
				latest = score
			}
		}
	}
	return latest
}

// RecordViolation records a quality violation.
func (r *Repository) RecordViolation(tenantID, ruleID, ruleName, agentID, field, value, expected string, severity Severity, message string) *Violation {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.violSeq++
	v := &Violation{
		ID: fmt.Sprintf("dqv-%d", r.violSeq), TenantID: tenantID,
		RuleID: ruleID, RuleName: ruleName, AgentID: agentID,
		Field: field, Value: value, Expected: expected,
		Severity: severity, Message: message, OccurredAt: time.Now(),
	}
	r.violations = append(r.violations, v)
	return v
}

// ListViolations returns violations with optional filters.
func (r *Repository) ListViolations(tenantID, agentID, ruleID string) []*Violation {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Violation
	for _, v := range r.violations {
		if v.TenantID != tenantID {
			continue
		}
		if agentID != "" && v.AgentID != agentID {
			continue
		}
		if ruleID != "" && v.RuleID != ruleID {
			continue
		}
		result = append(result, v)
	}
	return result
}

// RecordProfile records a data profiling result.
func (r *Repository) RecordProfile(tenantID, agentID, sourceID string, rowCount int64, colCount int, nullRate, dupRate, completeness float64, colProfiles []*ColumnProfile) *DataProfile {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profileSeq++
	if colProfiles == nil {
		colProfiles = []*ColumnProfile{}
	}
	p := &DataProfile{
		ID: fmt.Sprintf("dqp-%d", r.profileSeq), TenantID: tenantID,
		AgentID: agentID, SourceID: sourceID, RowCount: rowCount,
		ColumnCount: colCount, NullRate: nullRate, DuplicateRate: dupRate,
		Completeness: completeness, ColumnProfiles: colProfiles,
		ProfiledAt: time.Now(),
	}
	r.profiles = append(r.profiles, p)
	return p
}

// ListProfiles returns profiles for a tenant.
func (r *Repository) ListProfiles(tenantID, agentID string) []*DataProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*DataProfile
	for _, p := range r.profiles {
		if p.TenantID != tenantID {
			continue
		}
		if agentID != "" && p.AgentID != agentID {
			continue
		}
		result = append(result, p)
	}
	return result
}

// GetLatestProfile returns the most recent profile for an agent.
func (r *Repository) GetLatestProfile(tenantID, agentID string) *DataProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var latest *DataProfile
	for _, p := range r.profiles {
		if p.TenantID == tenantID && p.AgentID == agentID {
			if latest == nil || p.ProfiledAt.After(latest.ProfiledAt) {
				latest = p
			}
		}
	}
	return latest
}

// CreateContract creates a new data contract.
func (r *Repository) CreateContract(tenantID, name, desc, producer string, consumers []string, sourceID string, schema map[string]string, freshness *FreshnessSpec, quality *QualitySpec) *DataContract {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contractSeq++
	if consumers == nil {
		consumers = []string{}
	}
	if schema == nil {
		schema = make(map[string]string)
	}
	now := time.Now()
	c := &DataContract{
		ID: fmt.Sprintf("dqc-%d", r.contractSeq), TenantID: tenantID,
		Name: name, Description: desc, ProducerAgent: producer,
		ConsumerAgents: consumers, SourceID: sourceID, SchemaSpec: schema,
		FreshnessSpec: freshness, QualitySpec: quality, Status: "active",
		LastCheckedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	r.contracts = append(r.contracts, c)
	return c
}

// ListContracts returns contracts for a tenant.
func (r *Repository) ListContracts(tenantID, agentID string) []*DataContract {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*DataContract
	for _, c := range r.contracts {
		if c.TenantID != tenantID {
			continue
		}
		if agentID != "" && c.ProducerAgent != agentID {
			match := false
			for _, ca := range c.ConsumerAgents {
				if ca == agentID {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		result = append(result, c)
	}
	return result
}

// GetContract returns a specific contract.
func (r *Repository) GetContract(tenantID, contractID string) *DataContract {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.contracts {
		if c.TenantID == tenantID && c.ID == contractID {
			return c
		}
	}
	return nil
}

// UpdateContractStatus updates contract status.
func (r *Repository) UpdateContractStatus(tenantID, contractID, status string) (*DataContract, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.contracts {
		if c.TenantID == tenantID && c.ID == contractID {
			c.Status = status
			c.UpdatedAt = time.Now()
			return c, nil
		}
	}
	return nil, fmt.Errorf("contract %s not found", contractID)
}

// DeleteContract removes a contract.
func (r *Repository) DeleteContract(tenantID, contractID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, c := range r.contracts {
		if c.TenantID == tenantID && c.ID == contractID {
			r.contracts = append(r.contracts[:i], r.contracts[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("contract %s not found", contractID)
}

// GetQualityTrend builds quality trend data for an agent.
func (r *Repository) GetQualityTrend(tenantID, agentID string, days int) *QualityTrend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	trend := &QualityTrend{AgentID: agentID, Points: make([]*TrendPoint, 0)}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	for _, score := range r.scores {
		if score.TenantID == tenantID && score.AgentID == agentID && score.EvaluatedAt.After(cutoff) {
			trend.Points = append(trend.Points, &TrendPoint{
				Timestamp: score.EvaluatedAt, Overall: score.OverallScore,
				Completeness: score.CompletenessScore, Accuracy: score.AccuracyScore,
				Consistency: score.ConsistencyScore, Timeliness: score.TimelinessScore,
			})
		}
	}

	// Generate synthetic trend if no real data
	if len(trend.Points) == 0 {
		now := time.Now()
		for i := days; i >= 0; i-- {
			ts := now.Add(-time.Duration(i) * 24 * time.Hour)
			trend.Points = append(trend.Points, &TrendPoint{
				Timestamp: ts, Overall: 85 + float64(i%5)*2,
				Completeness: 90 + float64(i%4), Accuracy: 88 + float64(i%3),
				Consistency: 87 + float64(i%6), Timeliness: 92 - float64(i%5),
			})
		}
	}
	return trend
}

// RecordIncident creates a quality incident.
func (r *Repository) RecordIncident(tenantID, agentID, contractID, title, desc string, severity Severity, violationIDs []string) *QualityIncident {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.incidentSeq++
	if violationIDs == nil {
		violationIDs = []string{}
	}
	inc := &QualityIncident{
		ID: fmt.Sprintf("dqi-%d", r.incidentSeq), TenantID: tenantID,
		AgentID: agentID, ContractID: contractID, Title: title,
		Description: desc, Severity: severity, Status: "open",
		ViolationIDs: violationIDs, CreatedAt: time.Now(),
	}
	r.incidents = append(r.incidents, inc)
	return inc
}

// ListIncidents returns incidents with optional status filter.
func (r *Repository) ListIncidents(tenantID, status string) []*QualityIncident {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*QualityIncident
	for _, inc := range r.incidents {
		if inc.TenantID != tenantID {
			continue
		}
		if status != "" && inc.Status != status {
			continue
		}
		result = append(result, inc)
	}
	return result
}

// UpdateIncidentStatus updates incident status.
func (r *Repository) UpdateIncidentStatus(tenantID, incidentID, status string) (*QualityIncident, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inc := range r.incidents {
		if inc.TenantID == tenantID && inc.ID == incidentID {
			inc.Status = status
			if status == "resolved" {
				now := time.Now()
				inc.ResolvedAt = &now
			}
			return inc, nil
		}
	}
	return nil, fmt.Errorf("incident %s not found", incidentID)
}

// RecordAnomaly records a detected quality anomaly.
func (r *Repository) RecordAnomaly(tenantID, agentID, metric string, expected, actual, deviation float64, severity Severity) *Anomaly {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.anomalySeq++
	a := &Anomaly{
		ID: fmt.Sprintf("dqa-%d", r.anomalySeq), TenantID: tenantID,
		AgentID: agentID, MetricName: metric, Expected: expected,
		Actual: actual, Deviation: deviation, Severity: severity,
		DetectedAt: time.Now(),
	}
	r.anomalies = append(r.anomalies, a)
	return a
}

// ListAnomalies returns anomalies for a tenant.
func (r *Repository) ListAnomalies(tenantID, agentID string) []*Anomaly {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Anomaly
	for _, a := range r.anomalies {
		if a.TenantID != tenantID {
			continue
		}
		if agentID != "" && a.AgentID != agentID {
			continue
		}
		result = append(result, a)
	}
	return result
}

// GetDrift generates drift data for an agent.
func (r *Repository) GetDrift(tenantID, agentID string) []*DriftPoint {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var points []*DriftPoint
	now := time.Now()
	metrics := []string{"completeness", "consistency", "accuracy", "timeliness"}

	for _, metric := range metrics {
		for i := 7; i >= 0; i-- {
			ts := now.Add(-time.Duration(i) * 24 * time.Hour)
			baseline := 90.0
			value := baseline + float64(i%4)*2 - 3
			deviation := math.Abs(value - baseline)
			points = append(points, &DriftPoint{
				Timestamp: ts, Metric: metric,
				Value: value, Baseline: baseline,
				IsAnomaly: deviation > 5,
			})
		}
	}
	return points
}

// GetSummary computes a DQ summary for a tenant.
func (r *Repository) GetSummary(tenantID string) *DQSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s := &DQSummary{}
	agentSet := make(map[string]bool)

	for _, rule := range r.rules {
		if rule.TenantID == tenantID {
			s.TotalRules++
			if rule.Enabled {
				s.ActiveRules++
			}
		}
	}
	for _, v := range r.violations {
		if v.TenantID == tenantID {
			s.TotalViolations++
			if v.Severity == SeverityCritical {
				s.CriticalViolations++
			}
		}
	}
	var totalScore float64
	scoreCount := 0
	for _, score := range r.scores {
		if score.TenantID == tenantID {
			totalScore += score.OverallScore
			scoreCount++
			agentSet[score.AgentID] = true
		}
	}
	if scoreCount > 0 {
		s.AvgQualityScore = totalScore / float64(scoreCount)
	}
	for _, c := range r.contracts {
		if c.TenantID == tenantID {
			s.TotalContracts++
			if c.Status == "active" {
				s.ActiveContracts++
			}
			if c.Status == "violated" {
				s.ViolatedContracts++
			}
		}
	}
	for _, inc := range r.incidents {
		if inc.TenantID == tenantID {
			s.TotalIncidents++
			if inc.Status == "open" {
				s.OpenIncidents++
			}
		}
	}
	for _, a := range r.anomalies {
		if a.TenantID == tenantID {
			s.TotalAnomalies++
		}
	}
	s.AgentCount = len(agentSet)
	return s
}
