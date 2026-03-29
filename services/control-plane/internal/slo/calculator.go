package slo

import (
	"context"
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
	store Store
}

// NewCalculator creates a new SLO calculator.
func NewCalculator(store Store) *Calculator {
	return &Calculator{store: store}
}

// CalculateStatus computes the current status for a single SLO.
// Returns (nil, nil) when the SLO does not exist.
func (c *Calculator) CalculateStatus(ctx context.Context, tenantID, sloID string) (*SLOStatus, error) {
	slo, err := c.store.GetSLO(ctx, tenantID, sloID)
	if err != nil {
		return nil, err
	}
	if slo == nil {
		return nil, nil
	}

	windowEnd := time.Now()
	windowStart := windowEnd.Add(-parseWindow(slo.Window))

	measurements, err := c.store.GetMeasurements(ctx, tenantID, sloID, windowStart)
	if err != nil {
		return nil, err
	}

	return c.computeStatus(slo, measurements, windowStart, windowEnd), nil
}

// CalculateAllStatuses computes status for all SLOs belonging to a tenant.
func (c *Calculator) CalculateAllStatuses(ctx context.Context, tenantID, agentID string) ([]*SLOStatus, error) {
	slos, err := c.store.ListSLOs(ctx, tenantID, agentID)
	if err != nil {
		return nil, err
	}
	if len(slos) == 0 {
		return nil, nil
	}

	windowEnd := time.Now()
	var statuses []*SLOStatus

	for _, slo := range slos {
		if !slo.Enabled {
			continue
		}
		windowStart := windowEnd.Add(-parseWindow(slo.Window))
		measurements, err := c.store.GetMeasurements(ctx, tenantID, slo.ID, windowStart)
		if err != nil {
			return nil, err
		}
		status := c.computeStatus(slo, measurements, windowStart, windowEnd)
		statuses = append(statuses, status)
	}

	return statuses, nil
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

// ErrorBudgetPoint represents a single data point in an error budget time series.
type ErrorBudgetPoint struct {
	Timestamp string  `json:"timestamp"`
	Remaining float64 `json:"remaining"`
}

// CalculateErrorBudget returns the error budget time series for an SLO.
func (c *Calculator) CalculateErrorBudget(ctx context.Context, tenantID, sloID string) ([]ErrorBudgetPoint, error) {
	slo, err := c.store.GetSLO(ctx, tenantID, sloID)
	if err != nil {
		return nil, err
	}
	if slo == nil {
		return []ErrorBudgetPoint{}, nil
	}

	windowDuration := parseWindow(slo.Window)
	windowEnd := time.Now()
	windowStart := windowEnd.Add(-windowDuration)
	measurements, err := c.store.GetMeasurements(ctx, tenantID, sloID, windowStart)
	if err != nil {
		return nil, err
	}

	if len(measurements) == 0 {
		// Return a flat budget line at 100% remaining
		points := make([]ErrorBudgetPoint, 0, 7)
		step := windowDuration / 7
		for i := 0; i < 7; i++ {
			t := windowStart.Add(step * time.Duration(i))
			points = append(points, ErrorBudgetPoint{
				Timestamp: t.Format(time.RFC3339),
				Remaining: 100.0,
			})
		}
		return points, nil
	}

	// Build cumulative error budget from measurements
	errorBudgetTotal := (1.0 - slo.Target/100.0) * float64(len(measurements)) * 100
	if errorBudgetTotal <= 0 {
		errorBudgetTotal = 1
	}

	points := make([]ErrorBudgetPoint, 0, len(measurements))
	var cumulativeBad float64
	for _, m := range measurements {
		cumulativeBad += float64(m.Total - m.Good)
		remaining := ((errorBudgetTotal - cumulativeBad) / errorBudgetTotal) * 100
		if remaining < 0 {
			remaining = 0
		}
		points = append(points, ErrorBudgetPoint{
			Timestamp: m.Timestamp.Format(time.RFC3339),
			Remaining: remaining,
		})
	}

	return points, nil
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
