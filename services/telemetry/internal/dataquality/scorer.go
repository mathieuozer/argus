package dataquality

import (
	"math"
	"time"
)

type QualityScore struct {
	AgentID      string    `json:"agent_id"`
	Completeness float64   `json:"completeness"`
	Conformance  float64   `json:"conformance"`
	Consistency  float64   `json:"consistency"`
	Freshness    float64   `json:"freshness"`
	Overall      float64   `json:"overall"`
	SampleSize   int       `json:"sample_size"`
	ComputedAt   time.Time `json:"computed_at"`
}

type Scorer struct {
	validator      *Validator
	freshnessLimit time.Duration
}

func NewScorer(validator *Validator, freshnessLimit time.Duration) *Scorer {
	return &Scorer{
		validator:      validator,
		freshnessLimit: freshnessLimit,
	}
}

func (s *Scorer) Score(agentID string, records []map[string]interface{}, requiredFields []string, timestamps []time.Time) QualityScore {
	if len(records) == 0 {
		return QualityScore{AgentID: agentID, ComputedAt: time.Now()}
	}

	completeness := s.computeCompleteness(records, requiredFields)
	conformance := s.computeConformance(agentID, records)
	consistency := s.computeConsistency(records)
	freshness := s.computeFreshness(timestamps)

	overall := (completeness*0.3 + conformance*0.3 + consistency*0.2 + freshness*0.2)

	return QualityScore{
		AgentID:      agentID,
		Completeness: math.Round(completeness*10000) / 10000,
		Conformance:  math.Round(conformance*10000) / 10000,
		Consistency:  math.Round(consistency*10000) / 10000,
		Freshness:    math.Round(freshness*10000) / 10000,
		Overall:      math.Round(overall*10000) / 10000,
		SampleSize:   len(records),
		ComputedAt:   time.Now(),
	}
}

func (s *Scorer) computeCompleteness(records []map[string]interface{}, requiredFields []string) float64 {
	if len(requiredFields) == 0 || len(records) == 0 {
		return 1.0
	}

	totalFields := len(records) * len(requiredFields)
	presentFields := 0

	for _, record := range records {
		for _, field := range requiredFields {
			if _, ok := record[field]; ok {
				presentFields++
			}
		}
	}

	return float64(presentFields) / float64(totalFields)
}

func (s *Scorer) computeConformance(agentID string, records []map[string]interface{}) float64 {
	if len(records) == 0 {
		return 1.0
	}

	passing := 0
	total := 0

	for _, record := range records {
		results := s.validator.Validate(agentID, record)
		if len(results) == 0 {
			passing++
			total++
			continue
		}
		total++
		allPassed := true
		for _, r := range results {
			if !r.Passed {
				allPassed = false
				break
			}
		}
		if allPassed {
			passing++
		}
	}

	if total == 0 {
		return 1.0
	}
	return float64(passing) / float64(total)
}

func (s *Scorer) computeConsistency(records []map[string]interface{}) float64 {
	if len(records) < 2 {
		return 1.0
	}

	// Count key frequency across records
	keyCounts := make(map[string]int)
	for _, record := range records {
		for k := range record {
			keyCounts[k]++
		}
	}

	// Keys present in all records = consistent
	totalKeys := len(keyCounts)
	if totalKeys == 0 {
		return 1.0
	}

	consistentKeys := 0
	for _, count := range keyCounts {
		if count == len(records) {
			consistentKeys++
		}
	}

	return float64(consistentKeys) / float64(totalKeys)
}

func (s *Scorer) computeFreshness(timestamps []time.Time) float64 {
	if len(timestamps) == 0 {
		return 0.0
	}

	now := time.Now()
	fresh := 0
	for _, ts := range timestamps {
		if now.Sub(ts) <= s.freshnessLimit {
			fresh++
		}
	}

	return float64(fresh) / float64(len(timestamps))
}
