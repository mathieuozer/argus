package dataquality

import (
	"math"
	"time"
)

type DriftPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	Consistency float64   `json:"consistency"`
	Baseline    float64   `json:"baseline"`
}

type DriftDetector struct {
	windowSize int
}

func NewDriftDetector(windowSize int) *DriftDetector {
	return &DriftDetector{windowSize: windowSize}
}

func (d *DriftDetector) Detect(records [][]map[string]interface{}, timestamps []time.Time) []DriftPoint {
	if len(records) < 2 {
		return nil
	}

	// Use first window as baseline
	baselineKeys := collectKeys(records[0])
	baselineScore := 1.0

	var points []DriftPoint

	for i, batch := range records {
		currentKeys := collectKeys(batch)
		similarity := jaccardSimilarity(baselineKeys, currentKeys)

		var ts time.Time
		if i < len(timestamps) {
			ts = timestamps[i]
		} else {
			ts = time.Now()
		}

		points = append(points, DriftPoint{
			Timestamp:   ts,
			Consistency: math.Round(similarity*10000) / 10000,
			Baseline:    baselineScore,
		})
	}

	return points
}

func (d *DriftDetector) HasDrifted(baseline, current float64, threshold float64) bool {
	return math.Abs(baseline-current) > threshold
}

func collectKeys(records []map[string]interface{}) map[string]bool {
	keys := make(map[string]bool)
	for _, r := range records {
		for k := range r {
			keys[k] = true
		}
	}
	return keys
}

func jaccardSimilarity(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}

	intersection := 0
	union := make(map[string]bool)

	for k := range a {
		union[k] = true
		if b[k] {
			intersection++
		}
	}
	for k := range b {
		union[k] = true
	}

	if len(union) == 0 {
		return 1.0
	}

	return float64(intersection) / float64(len(union))
}
