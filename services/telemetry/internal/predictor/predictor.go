package predictor

import (
	"math"
	"time"
)

// Alert represents a predictive failure alert.
type Alert struct {
	ID            string        `json:"id"`
	TenantID      string        `json:"tenant_id"`
	AgentID       string        `json:"agent_id"`
	Probability   float64       `json:"probability"`
	EstimatedTTF  time.Duration `json:"estimated_ttf"`
	PrecursorType string        `json:"precursor_type"`
	Evidence      []string      `json:"evidence"`
	CreatedAt     time.Time     `json:"created_at"`
}

// QuarantineCallback is called when a prediction exceeds the auto-quarantine
// threshold (probability > 0.9). The callback receives the agentID and tenantID
// of the agent that should be quarantined.
type QuarantineCallback func(agentID, tenantID string)

// Client calls the ONNX predictor service or uses built-in heuristics.
type Client struct {
	endpoint           string
	useLocal           bool // when true, use local heuristic model instead of ONNX
	quarantineCallback QuarantineCallback
}

// NewClient creates a new predictor client.
func NewClient(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		useLocal: true, // default to local heuristics for dev/air-gap
	}
}

// Features represents the input features for the prediction model.
type Features struct {
	LatencyP99Ratio  float64 `json:"latency_p99_ratio"`
	TokenVelocity    float64 `json:"token_velocity"`
	RetryRate        float64 `json:"retry_rate"`
	ErrorRateDelta   float64 `json:"error_rate_delta"`
	ContextFillPct   float64 `json:"context_fill_pct"`
	ToolCallDepth    float64 `json:"tool_call_depth"`
	ConsecutiveSlow  float64 `json:"consecutive_slow"`
	CostAcceleration float64 `json:"cost_acceleration"`
}

// Prediction is the result from the model.
type Prediction struct {
	FailureProbability float64 `json:"failure_probability"`
	TTFSeconds         int     `json:"ttf_seconds"`
	PrecursorType      string  `json:"precursor_type"`
}

// Predict calls the predictor with the given features.
func (c *Client) Predict(features *Features) (*Prediction, error) {
	if c.useLocal {
		return c.predictLocal(features)
	}
	// In production with ONNX endpoint, would make HTTP call here
	return c.predictLocal(features)
}

// predictLocal uses heuristic rules to predict failures.
// This is the built-in model for dev mode and air-gapped deployments.
func (c *Client) predictLocal(f *Features) (*Prediction, error) {
	var probability float64
	var precursorType string
	var ttfSeconds int

	// Check each precursor type and compute probability
	checks := []struct {
		name        string
		probability float64
		ttf         int
	}{
		{
			name:        "latency_spike",
			probability: computeLatencySpikeProbability(f.LatencyP99Ratio, f.ConsecutiveSlow),
			ttf:         estimateLatencyTTF(f.LatencyP99Ratio),
		},
		{
			name:        "token_escalation",
			probability: computeTokenEscalationProbability(f.TokenVelocity, f.ContextFillPct),
			ttf:         estimateTokenTTF(f.ContextFillPct, f.TokenVelocity),
		},
		{
			name:        "retry_storm",
			probability: computeRetryStormProbability(f.RetryRate, f.ErrorRateDelta),
			ttf:         estimateRetryTTF(f.RetryRate),
		},
		{
			name:        "cost_runaway",
			probability: computeCostRunawayProbability(f.CostAcceleration, f.TokenVelocity),
			ttf:         estimateCostTTF(f.CostAcceleration),
		},
	}

	// Pick the highest probability precursor
	for _, check := range checks {
		if check.probability > probability {
			probability = check.probability
			precursorType = check.name
			ttfSeconds = check.ttf
		}
	}

	// Clamp probability to [0, 1]
	probability = math.Max(0, math.Min(1, probability))

	if probability < 0.1 {
		precursorType = "none"
		ttfSeconds = 0
	}

	return &Prediction{
		FailureProbability: math.Round(probability*10000) / 10000,
		TTFSeconds:         ttfSeconds,
		PrecursorType:      precursorType,
	}, nil
}

// computeLatencySpikeProbability detects p99 latency spikes.
// When p99/p50 ratio > 3 for sustained periods, failure is likely.
func computeLatencySpikeProbability(p99Ratio, consecutiveSlow float64) float64 {
	if p99Ratio < 2 {
		return 0
	}
	// Sigmoid function centered at ratio=4 with slope from consecutive slow calls
	base := sigmoid((p99Ratio - 3) * 2)
	slowBoost := math.Min(consecutiveSlow/10, 0.3) // up to 30% boost from slow calls
	return math.Min(base+slowBoost, 1.0)
}

// computeTokenEscalationProbability detects token usage spiraling.
func computeTokenEscalationProbability(tokenVelocity, contextFillPct float64) float64 {
	if contextFillPct < 0.5 {
		return 0
	}
	// Higher context fill = higher risk, especially with high token velocity
	fillRisk := sigmoid((contextFillPct - 0.7) * 8)
	velocityFactor := math.Min(tokenVelocity/100, 1.0) * 0.3
	return math.Min(fillRisk+velocityFactor, 1.0)
}

// computeRetryStormProbability detects retry storms.
func computeRetryStormProbability(retryRate, errorRateDelta float64) float64 {
	if retryRate < 0.1 {
		return 0
	}
	base := sigmoid((retryRate - 0.3) * 5)
	errorBoost := math.Min(errorRateDelta*2, 0.3)
	return math.Min(base+errorBoost, 1.0)
}

// computeCostRunawayProbability detects cost acceleration.
func computeCostRunawayProbability(costAcceleration, tokenVelocity float64) float64 {
	if costAcceleration <= 1.0 {
		return 0
	}
	// Cost doubling or more is concerning
	base := sigmoid((costAcceleration - 2) * 2)
	tokenFactor := math.Min(tokenVelocity/200, 0.2)
	return math.Min(base+tokenFactor, 1.0)
}

func estimateLatencyTTF(ratio float64) int {
	// Higher ratio = sooner failure
	if ratio > 5 {
		return 60
	}
	return int(600 / math.Max(ratio, 1))
}

func estimateTokenTTF(fillPct, velocity float64) int {
	remaining := 1.0 - fillPct
	if velocity <= 0 || remaining <= 0 {
		return 30
	}
	// Rough estimate: remaining context / velocity
	return int(math.Max(remaining*1000/velocity, 30))
}

func estimateRetryTTF(retryRate float64) int {
	if retryRate > 0.8 {
		return 30
	}
	return int(300 / math.Max(retryRate*5, 1))
}

func estimateCostTTF(acceleration float64) int {
	if acceleration > 5 {
		return 120
	}
	return int(600 / math.Max(acceleration, 1))
}

// sigmoid returns the sigmoid function value.
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// SetQuarantineCallback registers a callback that is invoked when a prediction
// result has a failure probability greater than 0.9. This enables the
// auto-quarantine pipeline: the predictor detects imminent failure and triggers
// agent quarantine via the orchestrator registry and identity revocation.
func (c *Client) SetQuarantineCallback(cb QuarantineCallback) {
	c.quarantineCallback = cb
}

// PredictAndEvaluate calls Predict and, if the failure probability exceeds 0.9,
// invokes the quarantine callback (if set) for the specified agent.
func (c *Client) PredictAndEvaluate(tenantID, agentID string, features *Features) (*Prediction, error) {
	pred, err := c.Predict(features)
	if err != nil {
		return nil, err
	}

	if pred.FailureProbability > 0.9 && c.quarantineCallback != nil {
		c.quarantineCallback(agentID, tenantID)
	}

	return pred, nil
}

// AnalyzeFeatures is a convenience function that analyzes features and returns
// whether an alert should be fired.
func AnalyzeFeatures(f *Features) (shouldAlert bool, prediction *Prediction) {
	client := NewClient("")
	pred, _ := client.Predict(f)
	return pred.FailureProbability >= 0.5, pred
}
