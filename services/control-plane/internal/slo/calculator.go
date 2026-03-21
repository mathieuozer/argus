package slo

import (
	"time"
)

// ComplianceStatus indicates whether an SLO is being met.
type ComplianceStatus string

const (
	ComplianceStatusMet      ComplianceStatus = "met"
	ComplianceStatusAtRisk   ComplianceStatus = "at_risk"
	ComplianceStatusBreached ComplianceStatus = "breached"
)

// SLOStatus provides the current status and compliance of an SLO.
type SLOStatus struct {
	SLO              *SLO             `json:"slo"`
	CurrentValue     float64          `json:"current_value"`
	Target           float64          `json:"target"`
	Compliance       ComplianceStatus `json:"compliance"`
	ErrorBudgetTotal float64          `json:"error_budget_total"`
	ErrorBudgetUsed  float64          `json:"error_budget_used"`
	ErrorBudgetLeft  float64          `json:"error_budget_left"`
	TotalGood        int64            `json:"total_good"`
	TotalEvents      int64            `json:"total_events"`
	MeasurementCount int              `json:"measurement_count"`
	WindowStart      time.Time        `json:"window_start"`
	WindowEnd        time.Time        `json:"window_end"`
}

// Calculator computes SLO compliance and error budgets.
type Calculator struct {
	repo *Repository
}

// NewCalculator creates a new SLO calculator.
func NewCalculator(repo *Repository) *Calculator {
	return &Calculator{repo: repo}
}

// CalculateStatus computes the current status for a single SLO.
func (c *Calculator) CalculateStatus(tenantID, sloID string) *SLOStatus {
	slo := c.repo.GetSLO(tenantID, sloID)
	if slo == nil {
		return nil
	}

	windowEnd := time.Now()
	windowStart := windowEnd.Add(-parseWindow(slo.Window))

	measurements := c.repo.GetMeasurements(tenantID, sloID, windowStart)

	return c.computeStatus(slo, measurements, windowStart, windowEnd)
}

// CalculateAllStatuses computes status for all SLOs belonging to a tenant.
func (c *Calculator) CalculateAllStatuses(tenantID, agentID string) []*SLOStatus {
	slos := c.repo.ListSLOs(tenantID, agentID)
	if len(slos) == 0 {
		return nil
	}

	windowEnd := time.Now()
	var statuses []*SLOStatus

	for _, slo := range slos {
		if !slo.Enabled {
			continue
		}
		windowStart := windowEnd.Add(-parseWindow(slo.Window))
		measurements := c.repo.GetMeasurements(tenantID, slo.ID, windowStart)
		status := c.computeStatus(slo, measurements, windowStart, windowEnd)
		statuses = append(statuses, status)
	}

	return statuses
}

func (c *Calculator) computeStatus(slo *SLO, measurements []*Measurement, windowStart, windowEnd time.Time) *SLOStatus {
	var totalGood, totalEvents int64
	for _, m := range measurements {
		totalGood += m.Good
		totalEvents += m.Total
	}

	// Calculate current value
	var currentValue float64
	if totalEvents > 0 {
		currentValue = (float64(totalGood) / float64(totalEvents)) * 100.0
	}

	// Calculate error budget
	// Error budget = (1 - target/100) * total_events
	errorBudgetTotal := (1.0 - slo.Target/100.0) * float64(totalEvents)
	errorBudgetUsed := float64(totalEvents - totalGood)
	errorBudgetLeft := errorBudgetTotal - errorBudgetUsed
	if errorBudgetLeft < 0 {
		errorBudgetLeft = 0
	}

	// Determine compliance status
	compliance := ComplianceStatusMet
	if totalEvents > 0 {
		if currentValue < slo.Target {
			compliance = ComplianceStatusBreached
		} else if errorBudgetTotal > 0 && errorBudgetUsed/errorBudgetTotal > 0.8 {
			compliance = ComplianceStatusAtRisk
		}
	}

	return &SLOStatus{
		SLO:              slo,
		CurrentValue:     currentValue,
		Target:           slo.Target,
		Compliance:       compliance,
		ErrorBudgetTotal: errorBudgetTotal,
		ErrorBudgetUsed:  errorBudgetUsed,
		ErrorBudgetLeft:  errorBudgetLeft,
		TotalGood:        totalGood,
		TotalEvents:      totalEvents,
		MeasurementCount: len(measurements),
		WindowStart:      windowStart,
		WindowEnd:        windowEnd,
	}
}

// parseWindow converts a window string like "7d", "30d", "90d" to a duration.
func parseWindow(window string) time.Duration {
	switch window {
	case "1d":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "90d":
		return 90 * 24 * time.Hour
	case "365d":
		return 365 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}
