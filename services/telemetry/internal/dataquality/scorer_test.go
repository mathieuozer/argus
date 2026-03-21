package dataquality

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestNewScorer(t *testing.T) {
	v := NewValidator()
	s := NewScorer(v, 5*time.Minute)
	if s == nil {
		t.Fatal("NewScorer() returned nil")
	}
}

func TestScorer_Score_EmptyRecords(t *testing.T) {
	v := NewValidator()
	s := NewScorer(v, 5*time.Minute)
	score := s.Score("agent-1", nil, []string{"a"}, nil)

	if score.AgentID != "agent-1" {
		t.Errorf("AgentID: got %q, want %q", score.AgentID, "agent-1")
	}
	if score.SampleSize != 0 {
		t.Errorf("SampleSize: got %d, want 0", score.SampleSize)
	}
	if score.Overall != 0 {
		t.Errorf("Overall: got %f, want 0", score.Overall)
	}
}

func TestScorer_Completeness(t *testing.T) {
	tests := []struct {
		name           string
		records        []map[string]interface{}
		requiredFields []string
		wantScore      float64
	}{
		{
			name: "all fields present",
			records: []map[string]interface{}{
				{"a": 1, "b": 2},
				{"a": 3, "b": 4},
			},
			requiredFields: []string{"a", "b"},
			wantScore:      1.0,
		},
		{
			name: "half fields missing",
			records: []map[string]interface{}{
				{"a": 1},
				{"a": 3},
			},
			requiredFields: []string{"a", "b"},
			wantScore:      0.5,
		},
		{
			name: "no required fields specified",
			records: []map[string]interface{}{
				{"a": 1},
			},
			requiredFields: []string{},
			wantScore:      1.0,
		},
		{
			name: "all fields missing",
			records: []map[string]interface{}{
				{"x": 1},
				{"y": 2},
			},
			requiredFields: []string{"a", "b"},
			wantScore:      0.0,
		},
		{
			name: "partial presence across records",
			records: []map[string]interface{}{
				{"a": 1, "b": 2, "c": 3},
				{"a": 1},
				{"a": 1, "c": 3},
			},
			requiredFields: []string{"a", "b", "c"},
			wantScore:      float64(6) / float64(9), // 6 present out of 9
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator()
			s := NewScorer(v, 5*time.Minute)

			now := time.Now()
			timestamps := make([]time.Time, len(tc.records))
			for i := range timestamps {
				timestamps[i] = now
			}

			score := s.Score("agent-1", tc.records, tc.requiredFields, timestamps)
			if math.Abs(score.Completeness-tc.wantScore) > 0.001 {
				t.Errorf("Completeness: got %f, want %f", score.Completeness, tc.wantScore)
			}
		})
	}
}

func TestScorer_Conformance(t *testing.T) {
	tests := []struct {
		name      string
		rules     []*Rule
		records   []map[string]interface{}
		wantScore float64
	}{
		{
			name:  "no rules means full conformance",
			rules: nil,
			records: []map[string]interface{}{
				{"a": 1},
			},
			wantScore: 1.0,
		},
		{
			name: "all records pass",
			rules: []*Rule{
				{
					ID: "r1", AgentID: "agent-1", Name: "require a",
					Type: RuleTypeSchema, Config: json.RawMessage(`{"required":["a"]}`),
					Severity: "error", Enabled: true,
				},
			},
			records: []map[string]interface{}{
				{"a": 1},
				{"a": 2},
			},
			wantScore: 1.0,
		},
		{
			name: "half records fail",
			rules: []*Rule{
				{
					ID: "r1", AgentID: "agent-1", Name: "require a",
					Type: RuleTypeSchema, Config: json.RawMessage(`{"required":["a"]}`),
					Severity: "error", Enabled: true,
				},
			},
			records: []map[string]interface{}{
				{"a": 1},
				{"b": 2},
			},
			wantScore: 0.5,
		},
		{
			name: "all records fail",
			rules: []*Rule{
				{
					ID: "r1", AgentID: "agent-1", Name: "require x",
					Type: RuleTypeSchema, Config: json.RawMessage(`{"required":["x"]}`),
					Severity: "error", Enabled: true,
				},
			},
			records: []map[string]interface{}{
				{"a": 1},
				{"b": 2},
			},
			wantScore: 0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator()
			for _, r := range tc.rules {
				v.AddRule(r)
			}
			s := NewScorer(v, 5*time.Minute)

			now := time.Now()
			timestamps := make([]time.Time, len(tc.records))
			for i := range timestamps {
				timestamps[i] = now
			}

			score := s.Score("agent-1", tc.records, nil, timestamps)
			if math.Abs(score.Conformance-tc.wantScore) > 0.001 {
				t.Errorf("Conformance: got %f, want %f", score.Conformance, tc.wantScore)
			}
		})
	}
}

func TestScorer_Consistency(t *testing.T) {
	tests := []struct {
		name      string
		records   []map[string]interface{}
		wantScore float64
	}{
		{
			name: "single record is fully consistent",
			records: []map[string]interface{}{
				{"a": 1, "b": 2},
			},
			wantScore: 1.0,
		},
		{
			name: "identical structure is consistent",
			records: []map[string]interface{}{
				{"a": 1, "b": 2},
				{"a": 3, "b": 4},
				{"a": 5, "b": 6},
			},
			wantScore: 1.0,
		},
		{
			name: "completely different keys",
			records: []map[string]interface{}{
				{"a": 1},
				{"b": 2},
			},
			wantScore: 0.0,
		},
		{
			name: "partial overlap",
			records: []map[string]interface{}{
				{"a": 1, "b": 2, "c": 3},
				{"a": 1, "b": 2, "d": 4},
			},
			// a and b are consistent (2/4 total distinct keys)
			wantScore: 0.5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator()
			s := NewScorer(v, 5*time.Minute)

			now := time.Now()
			timestamps := make([]time.Time, len(tc.records))
			for i := range timestamps {
				timestamps[i] = now
			}

			score := s.Score("agent-1", tc.records, nil, timestamps)
			if math.Abs(score.Consistency-tc.wantScore) > 0.001 {
				t.Errorf("Consistency: got %f, want %f", score.Consistency, tc.wantScore)
			}
		})
	}
}

func TestScorer_Freshness(t *testing.T) {
	tests := []struct {
		name           string
		freshnessLimit time.Duration
		timestamps     []time.Time
		wantScore      float64
	}{
		{
			name:           "all timestamps fresh",
			freshnessLimit: 5 * time.Minute,
			timestamps:     []time.Time{time.Now(), time.Now().Add(-1 * time.Minute), time.Now().Add(-2 * time.Minute)},
			wantScore:      1.0,
		},
		{
			name:           "all timestamps stale",
			freshnessLimit: 5 * time.Minute,
			timestamps:     []time.Time{time.Now().Add(-10 * time.Minute), time.Now().Add(-20 * time.Minute)},
			wantScore:      0.0,
		},
		{
			name:           "half fresh",
			freshnessLimit: 5 * time.Minute,
			timestamps:     []time.Time{time.Now(), time.Now().Add(-10 * time.Minute)},
			wantScore:      0.5,
		},
		{
			name:           "no timestamps",
			freshnessLimit: 5 * time.Minute,
			timestamps:     []time.Time{},
			wantScore:      0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator()
			s := NewScorer(v, tc.freshnessLimit)

			records := make([]map[string]interface{}, len(tc.timestamps))
			for i := range records {
				records[i] = map[string]interface{}{"key": i}
			}

			// Use empty records if no timestamps
			if len(tc.timestamps) == 0 {
				records = []map[string]interface{}{{"key": 1}}
			}

			score := s.Score("agent-1", records, nil, tc.timestamps)
			if math.Abs(score.Freshness-tc.wantScore) > 0.001 {
				t.Errorf("Freshness: got %f, want %f", score.Freshness, tc.wantScore)
			}
		})
	}
}

func TestScorer_Overall(t *testing.T) {
	// With perfect completeness, conformance, consistency, and freshness, overall should be 1.0
	v := NewValidator()
	s := NewScorer(v, 5*time.Minute)

	records := []map[string]interface{}{
		{"a": 1, "b": 2},
		{"a": 3, "b": 4},
	}
	requiredFields := []string{"a", "b"}
	timestamps := []time.Time{time.Now(), time.Now()}

	score := s.Score("agent-1", records, requiredFields, timestamps)

	// overall = completeness*0.3 + conformance*0.3 + consistency*0.2 + freshness*0.2
	// = 1.0*0.3 + 1.0*0.3 + 1.0*0.2 + 1.0*0.2 = 1.0
	if math.Abs(score.Overall-1.0) > 0.001 {
		t.Errorf("Overall: got %f, want 1.0", score.Overall)
	}
	if score.SampleSize != 2 {
		t.Errorf("SampleSize: got %d, want 2", score.SampleSize)
	}
}

func TestScorer_OverallWeighting(t *testing.T) {
	// Test that overall score correctly weights the dimensions
	// With zero freshness (stale timestamps), the score should be reduced
	v := NewValidator()
	s := NewScorer(v, 5*time.Minute)

	records := []map[string]interface{}{
		{"a": 1, "b": 2},
		{"a": 3, "b": 4},
	}
	requiredFields := []string{"a", "b"}
	staleTimestamps := []time.Time{
		time.Now().Add(-1 * time.Hour),
		time.Now().Add(-1 * time.Hour),
	}

	score := s.Score("agent-1", records, requiredFields, staleTimestamps)

	// Freshness = 0 (all stale), others = 1.0
	// overall = 1.0*0.3 + 1.0*0.3 + 1.0*0.2 + 0.0*0.2 = 0.8
	if math.Abs(score.Overall-0.8) > 0.001 {
		t.Errorf("Overall: got %f, want 0.8", score.Overall)
	}
}
