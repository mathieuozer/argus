package predictor

import (
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

// Client calls the ONNX predictor service.
type Client struct {
	endpoint string
}

// NewClient creates a new predictor client.
func NewClient(endpoint string) *Client {
	return &Client{endpoint: endpoint}
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

// Prediction is the result from the ONNX model.
type Prediction struct {
	FailureProbability float64 `json:"failure_probability"`
	TTFSeconds         int     `json:"ttf_seconds"`
	PrecursorType      string  `json:"precursor_type"`
}

// Predict calls the ONNX predictor service with the given features.
// In development, this returns a stub prediction.
func (c *Client) Predict(features *Features) (*Prediction, error) {
	// Stub: in production, this would call the ONNX predictor service
	return &Prediction{
		FailureProbability: 0.0,
		TTFSeconds:         0,
		PrecursorType:      "none",
	}, nil
}
