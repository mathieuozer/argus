package governance

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ClassificationPolicy defines auto-classification rules for data sources.
type ClassificationPolicy struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	MatchPattern   string    `json:"match_pattern"`
	MatchType      string    `json:"match_type"` // "field_name", "source_name", "tag", "content"
	Classification string    `json:"classification"`
	AutoApply      bool      `json:"auto_apply"`
	AppliedCount   int       `json:"applied_count"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RetentionPolicy defines data retention rules per classification level.
type RetentionPolicy struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Classification string     `json:"classification"`
	RetentionDays  int        `json:"retention_days"`
	Action         string     `json:"action"` // "archive", "delete", "anonymize"
	Enabled        bool       `json:"enabled"`
	LastExecutedAt *time.Time `json:"last_executed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// AccessLog records a data access event.
type AccessLog struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	SourceID   string    `json:"source_id"`
	SourceName string    `json:"source_name"`
	AgentID    string    `json:"agent_id"`
	AccessType string    `json:"access_type"`
	UserID     string    `json:"user_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// PIIField represents a single PII detection in a data source field.
type PIIField struct {
	FieldName      string  `json:"field_name"`
	PIIType        string  `json:"pii_type"`
	Confidence     float64 `json:"confidence"`
	SampleCount    int     `json:"sample_count"`
	Recommendation string  `json:"recommendation"`
}

// PIIScanResult captures PII detection results for a data source.
type PIIScanResult struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	SourceID    string      `json:"source_id"`
	SourceName  string      `json:"source_name"`
	PIIFields   []*PIIField `json:"pii_fields"`
	TotalFields int         `json:"total_fields"`
	PIICount    int         `json:"pii_count"`
	RiskLevel   string      `json:"risk_level"`
	ScannedAt   time.Time   `json:"scanned_at"`
}

// ComplianceMapping maps a data source to a regulatory requirement.
type ComplianceMapping struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	SourceID       string    `json:"source_id"`
	SourceName     string    `json:"source_name"`
	Framework      string    `json:"framework"`
	ArticleRef     string    `json:"article_ref"`
	Requirement    string    `json:"requirement"`
	Status         string    `json:"status"`
	Evidence       []string  `json:"evidence"`
	LastAssessedAt time.Time `json:"last_assessed_at"`
}

// DataSteward represents a person responsible for data governance.
type DataSteward struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Domains   []string  `json:"domains"`
	SourceIDs []string  `json:"source_ids"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// GovernanceSummary provides an overview of governance status.
type GovernanceSummary struct {
	TotalPolicies          int            `json:"total_policies"`
	ActivePolicies         int            `json:"active_policies"`
	ClassificationCoverage map[string]int `json:"classification_coverage"`
	RetentionPolicies      int            `json:"retention_policies"`
	PIIScanResults         int            `json:"pii_scan_results"`
	HighRiskSources        int            `json:"high_risk_sources"`
	ComplianceMappings     int            `json:"compliance_mappings"`
	CompliantCount         int            `json:"compliant_count"`
	NonCompliantCount      int            `json:"non_compliant_count"`
	DataStewards           int            `json:"data_stewards"`
	UnownedSources         int            `json:"unowned_sources"`
	RecentAccessLogs       int            `json:"recent_access_logs"`
}

// Repository provides in-memory storage for governance data.
type Repository struct {
	mu         sync.RWMutex
	classPols  []*ClassificationPolicy
	retPols    []*RetentionPolicy
	accessLogs []*AccessLog
	piiScans   []*PIIScanResult
	mappings   []*ComplianceMapping
	stewards   []*DataSteward
	classSeq   int
	retSeq     int
	alSeq      int
	piiSeq     int
	cmSeq      int
	dsSeq      int
}

// NewRepository creates a new governance repository.
func NewRepository() *Repository {
	return &Repository{
		classPols:  make([]*ClassificationPolicy, 0),
		retPols:    make([]*RetentionPolicy, 0),
		accessLogs: make([]*AccessLog, 0),
		piiScans:   make([]*PIIScanResult, 0),
		mappings:   make([]*ComplianceMapping, 0),
		stewards:   make([]*DataSteward, 0),
	}
}

// CreateClassificationPolicy adds a new classification policy.
func (r *Repository) CreateClassificationPolicy(tenantID, name, desc, matchPattern, matchType, classification string, autoApply bool) *ClassificationPolicy {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.classSeq++
	now := time.Now()
	p := &ClassificationPolicy{
		ID: fmt.Sprintf("cp-%d", r.classSeq), TenantID: tenantID,
		Name: name, Description: desc, MatchPattern: matchPattern,
		MatchType: matchType, Classification: classification,
		AutoApply: autoApply, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	r.classPols = append(r.classPols, p)
	return p
}

// ListClassificationPolicies returns all classification policies for a tenant.
func (r *Repository) ListClassificationPolicies(tenantID string) []*ClassificationPolicy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*ClassificationPolicy
	for _, p := range r.classPols {
		if p.TenantID == tenantID {
			result = append(result, p)
		}
	}
	return result
}

// GetClassificationPolicy returns a specific policy by ID.
func (r *Repository) GetClassificationPolicy(tenantID, policyID string) *ClassificationPolicy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.classPols {
		if p.TenantID == tenantID && p.ID == policyID {
			return p
		}
	}
	return nil
}

// DeleteClassificationPolicy removes a classification policy.
func (r *Repository) DeleteClassificationPolicy(tenantID, policyID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.classPols {
		if p.TenantID == tenantID && p.ID == policyID {
			r.classPols = append(r.classPols[:i], r.classPols[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("policy %s not found", policyID)
}

// CreateRetentionPolicy adds a new retention policy.
func (r *Repository) CreateRetentionPolicy(tenantID, name, desc, classification string, retentionDays int, action string) *RetentionPolicy {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retSeq++
	now := time.Now()
	p := &RetentionPolicy{
		ID: fmt.Sprintf("rp-%d", r.retSeq), TenantID: tenantID,
		Name: name, Description: desc, Classification: classification,
		RetentionDays: retentionDays, Action: action, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	r.retPols = append(r.retPols, p)
	return p
}

// ListRetentionPolicies returns all retention policies for a tenant.
func (r *Repository) ListRetentionPolicies(tenantID string) []*RetentionPolicy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*RetentionPolicy
	for _, p := range r.retPols {
		if p.TenantID == tenantID {
			result = append(result, p)
		}
	}
	return result
}

// DeleteRetentionPolicy removes a retention policy.
func (r *Repository) DeleteRetentionPolicy(tenantID, policyID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.retPols {
		if p.TenantID == tenantID && p.ID == policyID {
			r.retPols = append(r.retPols[:i], r.retPols[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("policy %s not found", policyID)
}

// RecordAccessLog records a data access event.
func (r *Repository) RecordAccessLog(tenantID, sourceID, sourceName, agentID, accessType, userID string) *AccessLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alSeq++
	al := &AccessLog{
		ID: fmt.Sprintf("al-%d", r.alSeq), TenantID: tenantID,
		SourceID: sourceID, SourceName: sourceName, AgentID: agentID,
		AccessType: accessType, UserID: userID, Timestamp: time.Now(),
	}
	r.accessLogs = append(r.accessLogs, al)
	return al
}

// ListAccessLogs returns access logs for a tenant with optional filters.
func (r *Repository) ListAccessLogs(tenantID, sourceID, agentID string) []*AccessLog {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*AccessLog
	for _, al := range r.accessLogs {
		if al.TenantID != tenantID {
			continue
		}
		if sourceID != "" && al.SourceID != sourceID {
			continue
		}
		if agentID != "" && al.AgentID != agentID {
			continue
		}
		result = append(result, al)
	}
	return result
}

// RecordPIIScan records a PII scan result.
func (r *Repository) RecordPIIScan(tenantID, sourceID, sourceName string, piiFields []*PIIField, totalFields int) *PIIScanResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.piiSeq++
	piiCount := len(piiFields)
	risk := "low"
	if piiCount > 5 {
		risk = "critical"
	} else if piiCount > 3 {
		risk = "high"
	} else if piiCount > 1 {
		risk = "medium"
	}
	scan := &PIIScanResult{
		ID: fmt.Sprintf("pii-%d", r.piiSeq), TenantID: tenantID,
		SourceID: sourceID, SourceName: sourceName, PIIFields: piiFields,
		TotalFields: totalFields, PIICount: piiCount, RiskLevel: risk,
		ScannedAt: time.Now(),
	}
	r.piiScans = append(r.piiScans, scan)
	return scan
}

// ListPIIScans returns all PII scan results for a tenant.
func (r *Repository) ListPIIScans(tenantID string) []*PIIScanResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*PIIScanResult
	for _, s := range r.piiScans {
		if s.TenantID == tenantID {
			result = append(result, s)
		}
	}
	return result
}

// GetPIIScan returns a specific PII scan by ID.
func (r *Repository) GetPIIScan(tenantID, scanID string) *PIIScanResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.piiScans {
		if s.TenantID == tenantID && s.ID == scanID {
			return s
		}
	}
	return nil
}

// CreateComplianceMapping creates a new compliance mapping.
func (r *Repository) CreateComplianceMapping(tenantID, sourceID, sourceName, framework, articleRef, requirement, status string, evidence []string) *ComplianceMapping {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmSeq++
	if evidence == nil {
		evidence = []string{}
	}
	cm := &ComplianceMapping{
		ID: fmt.Sprintf("cm-%d", r.cmSeq), TenantID: tenantID,
		SourceID: sourceID, SourceName: sourceName, Framework: framework,
		ArticleRef: articleRef, Requirement: requirement, Status: status,
		Evidence: evidence, LastAssessedAt: time.Now(),
	}
	r.mappings = append(r.mappings, cm)
	return cm
}

// ListComplianceMappings returns compliance mappings with optional filters.
func (r *Repository) ListComplianceMappings(tenantID, framework, status string) []*ComplianceMapping {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*ComplianceMapping
	for _, cm := range r.mappings {
		if cm.TenantID != tenantID {
			continue
		}
		if framework != "" && cm.Framework != framework {
			continue
		}
		if status != "" && cm.Status != status {
			continue
		}
		result = append(result, cm)
	}
	return result
}

// UpdateComplianceMappingStatus updates the status of a compliance mapping.
func (r *Repository) UpdateComplianceMappingStatus(tenantID, mappingID, status string, evidence []string) (*ComplianceMapping, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, cm := range r.mappings {
		if cm.TenantID == tenantID && cm.ID == mappingID {
			cm.Status = status
			if evidence != nil {
				cm.Evidence = evidence
			}
			cm.LastAssessedAt = time.Now()
			return cm, nil
		}
	}
	return nil, fmt.Errorf("mapping %s not found", mappingID)
}

// CreateSteward adds a new data steward.
func (r *Repository) CreateSteward(tenantID, userID, name, email string, domains, sourceIDs []string, role string) *DataSteward {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dsSeq++
	if domains == nil {
		domains = []string{}
	}
	if sourceIDs == nil {
		sourceIDs = []string{}
	}
	ds := &DataSteward{
		ID: fmt.Sprintf("ds-%d", r.dsSeq), TenantID: tenantID,
		UserID: userID, Name: name, Email: email,
		Domains: domains, SourceIDs: sourceIDs, Role: role,
		CreatedAt: time.Now(),
	}
	r.stewards = append(r.stewards, ds)
	return ds
}

// ListStewards returns all data stewards for a tenant.
func (r *Repository) ListStewards(tenantID string) []*DataSteward {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*DataSteward
	for _, ds := range r.stewards {
		if ds.TenantID == tenantID {
			result = append(result, ds)
		}
	}
	return result
}

// DeleteSteward removes a data steward.
func (r *Repository) DeleteSteward(tenantID, stewardID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, ds := range r.stewards {
		if ds.TenantID == tenantID && ds.ID == stewardID {
			r.stewards = append(r.stewards[:i], r.stewards[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("steward %s not found", stewardID)
}

// GetSummary computes a governance summary for a tenant.
func (r *Repository) GetSummary(tenantID string) *GovernanceSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s := &GovernanceSummary{
		ClassificationCoverage: make(map[string]int),
	}
	for _, p := range r.classPols {
		if p.TenantID == tenantID {
			s.TotalPolicies++
			if p.Enabled {
				s.ActivePolicies++
			}
			s.ClassificationCoverage[p.Classification] += p.AppliedCount
		}
	}
	for _, p := range r.retPols {
		if p.TenantID == tenantID {
			s.RetentionPolicies++
		}
	}
	for _, scan := range r.piiScans {
		if scan.TenantID == tenantID {
			s.PIIScanResults++
			if scan.RiskLevel == "high" || scan.RiskLevel == "critical" {
				s.HighRiskSources++
			}
		}
	}
	for _, cm := range r.mappings {
		if cm.TenantID == tenantID {
			s.ComplianceMappings++
			if cm.Status == "compliant" {
				s.CompliantCount++
			} else if cm.Status == "non_compliant" {
				s.NonCompliantCount++
			}
		}
	}
	for _, ds := range r.stewards {
		if ds.TenantID == tenantID {
			s.DataStewards++
		}
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, al := range r.accessLogs {
		if al.TenantID == tenantID && al.Timestamp.After(cutoff) {
			s.RecentAccessLogs++
		}
	}

	_ = strings.ToLower // keep import
	return s
}
