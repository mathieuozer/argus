package dataquality

import (
	"fmt"
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
	ID          string   `json:"id"`
	TenantID    string   `json:"tenant_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        RuleType `json:"type"`
	AgentID     string   `json:"agent_id,omitempty"` // empty means all agents
	Field       string   `json:"field"`              // field to check (e.g. "latency_ms")
	Operator    string   `json:"operator"`           // "gt", "lt", "eq", "not_null", "regex"
	Threshold   string   `json:"threshold"`          // comparison value
	Severity    Severity `json:"severity"`
	Enabled     bool     `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Score represents an aggregated data quality score for an agent.
type Score struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	AgentID         string    `json:"agent_id"`
	OverallScore    float64   `json:"overall_score"`    // 0.0 to 100.0
	CompletenessScore float64 `json:"completeness_score"`
	AccuracyScore   float64   `json:"accuracy_score"`
	ConsistencyScore float64  `json:"consistency_score"`
	TimelinessScore float64   `json:"timeliness_score"`
	TotalChecks     int       `json:"total_checks"`
	PassedChecks    int       `json:"passed_checks"`
	FailedChecks    int       `json:"failed_checks"`
	EvaluatedAt     time.Time `json:"evaluated_at"`
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

// Repository provides in-memory storage for data quality rules, scores, and violations.
type Repository struct {
	mu         sync.RWMutex
	rules      []*Rule
	scores     []*Score
	violations []*Violation
	ruleSeq    int
	scoreSeq   int
	violSeq    int
}

// NewRepository creates a new data quality repository with demo data.
func NewRepository() *Repository {
	r := &Repository{
		rules:      make([]*Rule, 0),
		scores:     make([]*Score, 0),
		violations: make([]*Violation, 0),
	}
	return r
}

// CreateRule adds a new data quality rule.
func (r *Repository) CreateRule(tenantID, name, description string, ruleType RuleType, agentID, field, operator, threshold string, severity Severity) *Rule {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ruleSeq++
	now := time.Now()
	rule := &Rule{
		ID:          fmt.Sprintf("dqr-%d", r.ruleSeq),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Type:        ruleType,
		AgentID:     agentID,
		Field:       field,
		Operator:    operator,
		Threshold:   threshold,
		Severity:    severity,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.rules = append(r.rules, rule)
	return rule
}

// ListRules returns all rules for a tenant, optionally filtered by agent ID.
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

// GetRule returns a specific rule by ID within a tenant.
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

// UpdateRule updates a rule's fields.
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

// RecordScore records a data quality score for an agent.
func (r *Repository) RecordScore(tenantID, agentID string, overall, completeness, accuracy, consistency, timeliness float64, total, passed, failed int) *Score {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.scoreSeq++
	score := &Score{
		ID:                fmt.Sprintf("dqs-%d", r.scoreSeq),
		TenantID:          tenantID,
		AgentID:           agentID,
		OverallScore:      overall,
		CompletenessScore: completeness,
		AccuracyScore:     accuracy,
		ConsistencyScore:  consistency,
		TimelinessScore:   timeliness,
		TotalChecks:       total,
		PassedChecks:      passed,
		FailedChecks:      failed,
		EvaluatedAt:       time.Now(),
	}
	r.scores = append(r.scores, score)
	return score
}

// ListScores returns data quality scores for a tenant, optionally filtered by agent.
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

// GetLatestScore returns the most recent score for an agent in a tenant.
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

// RecordViolation records a data quality violation.
func (r *Repository) RecordViolation(tenantID, ruleID, ruleName, agentID, field, value, expected string, severity Severity, message string) *Violation {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.violSeq++
	v := &Violation{
		ID:         fmt.Sprintf("dqv-%d", r.violSeq),
		TenantID:   tenantID,
		RuleID:     ruleID,
		RuleName:   ruleName,
		AgentID:    agentID,
		Field:      field,
		Value:      value,
		Expected:   expected,
		Severity:   severity,
		Message:    message,
		OccurredAt: time.Now(),
	}
	r.violations = append(r.violations, v)
	return v
}

// ListViolations returns violations for a tenant, optionally filtered by agent and/or rule.
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
