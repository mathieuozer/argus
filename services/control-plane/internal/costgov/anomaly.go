package costgov

import (
	"fmt"
	"math"
	"time"
)

// AnomalyType classifies the kind of cost anomaly detected.
type AnomalyType string

const (
	AnomalyTypeSpikeAbsolute   AnomalyType = "spike_absolute"
	AnomalyTypeSpikePercentage AnomalyType = "spike_percentage"
	AnomalyTypeUnusualAgent    AnomalyType = "unusual_agent"
	AnomalyTypeUnusualCategory AnomalyType = "unusual_category"
)

// AnomalySeverity indicates how critical the anomaly is.
type AnomalySeverity string

const (
	AnomalySeverityCritical AnomalySeverity = "critical"
	AnomalySeverityWarning  AnomalySeverity = "warning"
	AnomalySeverityInfo     AnomalySeverity = "info"
)

// Anomaly represents a detected cost anomaly.
type Anomaly struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	AgentID      string          `json:"agent_id,omitempty"`
	Type         AnomalyType     `json:"type"`
	Severity     AnomalySeverity `json:"severity"`
	Description  string          `json:"description"`
	ExpectedCost float64         `json:"expected_cost"`
	ActualCost   float64         `json:"actual_cost"`
	Deviation    float64         `json:"deviation"` // percentage deviation from expected
	DetectedAt   time.Time       `json:"detected_at"`
}

// AnomalyDetector analyzes cost data for anomalies.
type AnomalyDetector struct {
	// SpikeThreshold defines the percentage increase over average that triggers a spike alert.
	// Default is 2.0 (200% increase).
	SpikeThreshold float64

	// MinDataPoints is the minimum number of historical data points needed for detection.
	// Default is 3.
	MinDataPoints int
}

// NewAnomalyDetector creates a new anomaly detector with default settings.
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		SpikeThreshold: 2.0,
		MinDataPoints:  3,
	}
}

// DetectAnomalies analyzes cost entries for a tenant and returns any detected anomalies.
func (d *AnomalyDetector) DetectAnomalies(repo *Repository, tenantID string) []Anomaly {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	var anomalies []Anomaly
	anomalyID := 0

	// Collect entries for this tenant
	var tenantEntries []*CostEntry
	for _, e := range repo.entries {
		if e.TenantID == tenantID {
			tenantEntries = append(tenantEntries, e)
		}
	}

	if len(tenantEntries) < d.MinDataPoints {
		return anomalies
	}

	// Detect per-agent cost spikes
	agentCosts := make(map[string][]float64)
	for _, e := range tenantEntries {
		agentCosts[e.AgentID] = append(agentCosts[e.AgentID], e.CostUSD)
	}

	for agentID, costs := range agentCosts {
		if len(costs) < d.MinDataPoints {
			continue
		}

		// Calculate baseline (mean of all entries except the last)
		baseline := costs[:len(costs)-1]
		latest := costs[len(costs)-1]

		mean := calculateMean(baseline)
		stddev := calculateStdDev(baseline, mean)

		if mean == 0 {
			continue
		}

		deviation := (latest - mean) / mean

		// Check for absolute spike using standard deviation
		if stddev > 0 && latest > mean+2*stddev {
			anomalyID++
			severity := AnomalySeverityWarning
			if latest > mean+3*stddev {
				severity = AnomalySeverityCritical
			}

			anomalies = append(anomalies, Anomaly{
				ID:           fmt.Sprintf("anomaly-%d", anomalyID),
				TenantID:     tenantID,
				AgentID:      agentID,
				Type:         AnomalyTypeSpikeAbsolute,
				Severity:     severity,
				Description:  fmt.Sprintf("Agent %s cost $%.4f is %.1f standard deviations above mean $%.4f", agentID, latest, (latest-mean)/stddev, mean),
				ExpectedCost: mean,
				ActualCost:   latest,
				Deviation:    deviation * 100,
				DetectedAt:   time.Now(),
			})
		}

		// Check for percentage spike
		if deviation > d.SpikeThreshold {
			anomalyID++
			severity := AnomalySeverityWarning
			if deviation > d.SpikeThreshold*2 {
				severity = AnomalySeverityCritical
			}

			anomalies = append(anomalies, Anomaly{
				ID:           fmt.Sprintf("anomaly-%d", anomalyID),
				TenantID:     tenantID,
				AgentID:      agentID,
				Type:         AnomalyTypeSpikePercentage,
				Severity:     severity,
				Description:  fmt.Sprintf("Agent %s cost increased %.0f%% over baseline (expected $%.4f, got $%.4f)", agentID, deviation*100, mean, latest),
				ExpectedCost: mean,
				ActualCost:   latest,
				Deviation:    deviation * 100,
				DetectedAt:   time.Now(),
			})
		}
	}

	// Detect unusual category distribution
	categoryCosts := make(map[string]float64)
	var totalCost float64
	for _, e := range tenantEntries {
		categoryCosts[e.Category] += e.CostUSD
		totalCost += e.CostUSD
	}

	if totalCost > 0 {
		for category, cost := range categoryCosts {
			proportion := cost / totalCost
			// Flag if any single category exceeds 90% of total spend
			if proportion > 0.9 && len(categoryCosts) > 1 {
				anomalyID++
				anomalies = append(anomalies, Anomaly{
					ID:           fmt.Sprintf("anomaly-%d", anomalyID),
					TenantID:     tenantID,
					Type:         AnomalyTypeUnusualCategory,
					Severity:     AnomalySeverityInfo,
					Description:  fmt.Sprintf("Category %q accounts for %.0f%% of total spend", category, proportion*100),
					ExpectedCost: totalCost / float64(len(categoryCosts)),
					ActualCost:   cost,
					Deviation:    proportion * 100,
					DetectedAt:   time.Now(),
				})
			}
		}
	}

	return anomalies
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	var sumSquares float64
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)-1))
}
