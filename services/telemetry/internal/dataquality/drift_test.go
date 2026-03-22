package dataquality

import (
	"math"
	"testing"
	"time"
)

func TestNewDriftDetector(t *testing.T) {
	d := NewDriftDetector(10)
	if d == nil {
		t.Fatal("NewDriftDetector() returned nil")
	}
	if d.windowSize != 10 {
		t.Errorf("windowSize: got %d, want 10", d.windowSize)
	}
}

func TestDriftDetector_Detect_InsufficientBatches(t *testing.T) {
	tests := []struct {
		name    string
		records [][]map[string]interface{}
	}{
		{
			name:    "nil records",
			records: nil,
		},
		{
			name:    "empty records",
			records: [][]map[string]interface{}{},
		},
		{
			name: "single batch",
			records: [][]map[string]interface{}{
				{{"a": 1}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDriftDetector(10)
			points := d.Detect(tc.records, nil)
			if points != nil {
				t.Errorf("expected nil for insufficient batches, got %v", points)
			}
		})
	}
}

func TestDriftDetector_Detect_NoDrift(t *testing.T) {
	d := NewDriftDetector(10)
	now := time.Now()

	records := [][]map[string]interface{}{
		{{"a": 1, "b": 2}, {"a": 3, "b": 4}},
		{{"a": 5, "b": 6}, {"a": 7, "b": 8}},
		{{"a": 9, "b": 10}},
	}
	timestamps := []time.Time{now, now.Add(1 * time.Minute), now.Add(2 * time.Minute)}

	points := d.Detect(records, timestamps)
	if len(points) != 3 {
		t.Fatalf("expected 3 drift points, got %d", len(points))
	}

	// All batches have the same keys {a, b}, so consistency should be 1.0
	for i, point := range points {
		if math.Abs(point.Consistency-1.0) > 0.001 {
			t.Errorf("point[%d] consistency: got %f, want 1.0", i, point.Consistency)
		}
		if point.Baseline != 1.0 {
			t.Errorf("point[%d] baseline: got %f, want 1.0", i, point.Baseline)
		}
	}
}

func TestDriftDetector_Detect_WithDrift(t *testing.T) {
	d := NewDriftDetector(10)
	now := time.Now()

	records := [][]map[string]interface{}{
		{{"a": 1, "b": 2}},         // baseline: keys = {a, b}
		{{"a": 1, "c": 3}},         // drifted: keys = {a, c}
		{{"x": 1, "y": 2, "z": 3}}, // heavily drifted: keys = {x, y, z}
	}
	timestamps := []time.Time{now, now.Add(1 * time.Minute), now.Add(2 * time.Minute)}

	points := d.Detect(records, timestamps)
	if len(points) != 3 {
		t.Fatalf("expected 3 drift points, got %d", len(points))
	}

	// First batch vs baseline (itself): jaccard({a,b}, {a,b}) = 1.0
	if math.Abs(points[0].Consistency-1.0) > 0.001 {
		t.Errorf("point[0] consistency: got %f, want 1.0", points[0].Consistency)
	}

	// Second batch vs baseline: jaccard({a,b}, {a,c}) = 1/3
	expected1 := 1.0 / 3.0
	if math.Abs(points[1].Consistency-expected1) > 0.001 {
		t.Errorf("point[1] consistency: got %f, want %f", points[1].Consistency, expected1)
	}

	// Third batch vs baseline: jaccard({a,b}, {x,y,z}) = 0/5 = 0.0
	if math.Abs(points[2].Consistency-0.0) > 0.001 {
		t.Errorf("point[2] consistency: got %f, want 0.0", points[2].Consistency)
	}
}

func TestDriftDetector_Detect_TimestampFallback(t *testing.T) {
	d := NewDriftDetector(10)

	records := [][]map[string]interface{}{
		{{"a": 1}},
		{{"a": 2}},
		{{"a": 3}},
	}
	// Provide fewer timestamps than batches
	timestamps := []time.Time{time.Now()}

	points := d.Detect(records, timestamps)
	if len(points) != 3 {
		t.Fatalf("expected 3 drift points, got %d", len(points))
	}

	// First point should use provided timestamp
	if points[0].Timestamp.IsZero() {
		t.Error("point[0] timestamp should not be zero")
	}

	// Points beyond timestamps length should have a non-zero time (fallback to time.Now())
	if points[2].Timestamp.IsZero() {
		t.Error("point[2] timestamp should not be zero (should fall back to time.Now())")
	}
}

func TestDriftDetector_HasDrifted(t *testing.T) {
	tests := []struct {
		name      string
		baseline  float64
		current   float64
		threshold float64
		want      bool
	}{
		{
			name:      "no drift under threshold",
			baseline:  1.0,
			current:   0.95,
			threshold: 0.1,
			want:      false,
		},
		{
			name:      "drift exactly at threshold",
			baseline:  1.0,
			current:   0.9,
			threshold: 0.1,
			want:      false,
		},
		{
			name:      "drift over threshold",
			baseline:  1.0,
			current:   0.5,
			threshold: 0.1,
			want:      true,
		},
		{
			name:      "negative drift",
			baseline:  0.5,
			current:   1.0,
			threshold: 0.1,
			want:      true,
		},
		{
			name:      "zero threshold any difference drifts",
			baseline:  1.0,
			current:   0.999,
			threshold: 0.0,
			want:      true,
		},
		{
			name:      "identical values no drift",
			baseline:  0.8,
			current:   0.8,
			threshold: 0.0,
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDriftDetector(10)
			got := d.HasDrifted(tc.baseline, tc.current, tc.threshold)
			if got != tc.want {
				t.Errorf("HasDrifted(%f, %f, %f) = %v, want %v", tc.baseline, tc.current, tc.threshold, got, tc.want)
			}
		})
	}
}

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]bool
		b    map[string]bool
		want float64
	}{
		{
			name: "identical sets",
			a:    map[string]bool{"a": true, "b": true, "c": true},
			b:    map[string]bool{"a": true, "b": true, "c": true},
			want: 1.0,
		},
		{
			name: "completely disjoint",
			a:    map[string]bool{"a": true, "b": true},
			b:    map[string]bool{"c": true, "d": true},
			want: 0.0,
		},
		{
			name: "partial overlap",
			a:    map[string]bool{"a": true, "b": true, "c": true},
			b:    map[string]bool{"b": true, "c": true, "d": true},
			want: 0.5, // intersection=2, union=4
		},
		{
			name: "both empty",
			a:    map[string]bool{},
			b:    map[string]bool{},
			want: 1.0,
		},
		{
			name: "one empty one not",
			a:    map[string]bool{"a": true},
			b:    map[string]bool{},
			want: 0.0,
		},
		{
			name: "subset",
			a:    map[string]bool{"a": true, "b": true},
			b:    map[string]bool{"a": true},
			want: 0.5, // intersection=1, union=2
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := jaccardSimilarity(tc.a, tc.b)
			if math.Abs(got-tc.want) > 0.001 {
				t.Errorf("jaccardSimilarity() = %f, want %f", got, tc.want)
			}
		})
	}
}

func TestCollectKeys(t *testing.T) {
	tests := []struct {
		name    string
		records []map[string]interface{}
		want    map[string]bool
	}{
		{
			name:    "empty records",
			records: nil,
			want:    map[string]bool{},
		},
		{
			name: "single record",
			records: []map[string]interface{}{
				{"a": 1, "b": 2},
			},
			want: map[string]bool{"a": true, "b": true},
		},
		{
			name: "multiple records with overlapping keys",
			records: []map[string]interface{}{
				{"a": 1, "b": 2},
				{"b": 3, "c": 4},
			},
			want: map[string]bool{"a": true, "b": true, "c": true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := collectKeys(tc.records)
			if len(got) != len(tc.want) {
				t.Errorf("collectKeys() returned %d keys, want %d", len(got), len(tc.want))
			}
			for k := range tc.want {
				if !got[k] {
					t.Errorf("collectKeys() missing key %q", k)
				}
			}
		})
	}
}
