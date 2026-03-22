package predictor

import (
	"testing"
)

func TestPredictHealthyAgent(t *testing.T) {
	client := NewClient("")

	features := &Features{
		LatencyP99Ratio:  1.2,
		TokenVelocity:    10,
		RetryRate:        0.01,
		ErrorRateDelta:   0,
		ContextFillPct:   0.3,
		ToolCallDepth:    2,
		ConsecutiveSlow:  0,
		CostAcceleration: 1.0,
	}

	pred, err := client.Predict(features)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if pred.FailureProbability > 0.1 {
		t.Errorf("healthy agent should have low probability, got %f", pred.FailureProbability)
	}
	if pred.PrecursorType != "none" {
		t.Errorf("healthy agent precursor should be 'none', got %q", pred.PrecursorType)
	}
}

func TestPredictLatencySpike(t *testing.T) {
	client := NewClient("")

	features := &Features{
		LatencyP99Ratio: 5.0,
		ConsecutiveSlow: 8,
	}

	pred, err := client.Predict(features)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if pred.FailureProbability < 0.5 {
		t.Errorf("latency spike should have high probability, got %f", pred.FailureProbability)
	}
	if pred.PrecursorType != "latency_spike" {
		t.Errorf("precursor should be 'latency_spike', got %q", pred.PrecursorType)
	}
	if pred.TTFSeconds <= 0 {
		t.Error("TTF should be positive")
	}
}

func TestPredictTokenEscalation(t *testing.T) {
	client := NewClient("")

	features := &Features{
		ContextFillPct: 0.9,
		TokenVelocity:  150,
	}

	pred, err := client.Predict(features)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if pred.FailureProbability < 0.5 {
		t.Errorf("token escalation should have high probability, got %f", pred.FailureProbability)
	}
	if pred.PrecursorType != "token_escalation" {
		t.Errorf("precursor should be 'token_escalation', got %q", pred.PrecursorType)
	}
}

func TestPredictRetryStorm(t *testing.T) {
	client := NewClient("")

	features := &Features{
		RetryRate:      0.7,
		ErrorRateDelta: 0.5,
	}

	pred, err := client.Predict(features)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if pred.FailureProbability < 0.5 {
		t.Errorf("retry storm should have high probability, got %f", pred.FailureProbability)
	}
	if pred.PrecursorType != "retry_storm" {
		t.Errorf("precursor should be 'retry_storm', got %q", pred.PrecursorType)
	}
}

func TestPredictCostRunaway(t *testing.T) {
	client := NewClient("")

	features := &Features{
		CostAcceleration: 4.0,
		TokenVelocity:    100,
	}

	pred, err := client.Predict(features)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if pred.FailureProbability < 0.3 {
		t.Errorf("cost runaway should have moderate+ probability, got %f", pred.FailureProbability)
	}
	if pred.PrecursorType != "cost_runaway" {
		t.Errorf("precursor should be 'cost_runaway', got %q", pred.PrecursorType)
	}
}

func TestPredictProbabilityBounds(t *testing.T) {
	client := NewClient("")

	tests := []struct {
		name     string
		features *Features
	}{
		{
			name:     "all zeros",
			features: &Features{},
		},
		{
			name: "extreme values",
			features: &Features{
				LatencyP99Ratio:  100,
				TokenVelocity:    10000,
				RetryRate:        1.0,
				ErrorRateDelta:   1.0,
				ContextFillPct:   1.0,
				ToolCallDepth:    50,
				ConsecutiveSlow:  100,
				CostAcceleration: 100,
			},
		},
		{
			name: "negative values",
			features: &Features{
				LatencyP99Ratio:  -1,
				TokenVelocity:    -10,
				RetryRate:        -0.5,
				ErrorRateDelta:   -1,
				ContextFillPct:   -0.5,
				CostAcceleration: -2,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pred, err := client.Predict(tc.features)
			if err != nil {
				t.Fatalf("Predict failed: %v", err)
			}

			if pred.FailureProbability < 0 || pred.FailureProbability > 1 {
				t.Errorf("probability %f out of bounds [0, 1]", pred.FailureProbability)
			}
			if pred.TTFSeconds < 0 {
				t.Errorf("TTF %d should not be negative", pred.TTFSeconds)
			}
		})
	}
}

func TestAnalyzeFeatures(t *testing.T) {
	t.Run("healthy should not alert", func(t *testing.T) {
		shouldAlert, pred := AnalyzeFeatures(&Features{
			LatencyP99Ratio: 1.1,
			RetryRate:       0.01,
			ContextFillPct:  0.2,
		})
		if shouldAlert {
			t.Errorf("healthy features should not trigger alert, probability=%f", pred.FailureProbability)
		}
	})

	t.Run("failing should alert", func(t *testing.T) {
		shouldAlert, pred := AnalyzeFeatures(&Features{
			LatencyP99Ratio: 6.0,
			ConsecutiveSlow: 15,
			RetryRate:       0.6,
			ErrorRateDelta:  0.3,
		})
		if !shouldAlert {
			t.Errorf("failing features should trigger alert, probability=%f", pred.FailureProbability)
		}
	})
}

func TestSigmoid(t *testing.T) {
	tests := []struct {
		name string
		x    float64
		want float64
		tol  float64
	}{
		{name: "zero", x: 0, want: 0.5, tol: 0.001},
		{name: "large positive", x: 10, want: 1.0, tol: 0.001},
		{name: "large negative", x: -10, want: 0.0, tol: 0.001},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sigmoid(tc.x)
			if got < tc.want-tc.tol || got > tc.want+tc.tol {
				t.Errorf("sigmoid(%f) = %f, want ~%f", tc.x, got, tc.want)
			}
		})
	}
}
