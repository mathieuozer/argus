package classifier

import (
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		wantTier  DataTier
	}{
		// Structural (Tier 1)
		{name: "latency_ms is structural", fieldName: "latency_ms", wantTier: TierStructural},
		{name: "token_count is structural", fieldName: "token_count", wantTier: TierStructural},
		{name: "error_code is structural", fieldName: "error_code", wantTier: TierStructural},
		{name: "agent_id is structural", fieldName: "agent_id", wantTier: TierStructural},
		{name: "timestamp is structural", fieldName: "timestamp", wantTier: TierStructural},

		// Sensitive (Tier 2)
		{name: "task_desc is sensitive", fieldName: "task_desc", wantTier: TierSensitive},
		{name: "tool_params is sensitive", fieldName: "tool_params", wantTier: TierSensitive},
		{name: "partial_out is sensitive", fieldName: "partial_out", wantTier: TierSensitive},

		// Restricted (Tier 3)
		{name: "full_input is restricted", fieldName: "full_input", wantTier: TierRestricted},
		{name: "full_output is restricted", fieldName: "full_output", wantTier: TierRestricted},
		{name: "user_context is restricted", fieldName: "user_context", wantTier: TierRestricted},

		// Unknown defaults to sensitive
		{name: "unknown field defaults to sensitive", fieldName: "some_random_field", wantTier: TierSensitive},
		{name: "empty string defaults to sensitive", fieldName: "", wantTier: TierSensitive},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.fieldName)
			if got != tc.wantTier {
				t.Errorf("Classify(%q) = %d, want %d", tc.fieldName, got, tc.wantTier)
			}
		})
	}
}

func TestClassifyAttributes(t *testing.T) {
	tests := []struct {
		name            string
		attrs           map[string]string
		wantStructCount int
		wantSensCount   int
		wantRestCount   int
	}{
		{
			name: "mixed attributes are grouped correctly",
			attrs: map[string]string{
				"latency_ms":   "42",
				"token_count":  "1500",
				"task_desc":    "summarize document",
				"full_input":   "the entire user prompt",
				"unknown_field": "value",
			},
			wantStructCount: 2,
			wantSensCount:   2, // task_desc + unknown_field
			wantRestCount:   1,
		},
		{
			name:            "empty attributes",
			attrs:           map[string]string{},
			wantStructCount: 0,
			wantSensCount:   0,
			wantRestCount:   0,
		},
		{
			name: "all structural",
			attrs: map[string]string{
				"latency_ms":  "100",
				"error_code":  "TIMEOUT",
				"agent_id":    "agent-1",
			},
			wantStructCount: 3,
			wantSensCount:   0,
			wantRestCount:   0,
		},
		{
			name: "all restricted",
			attrs: map[string]string{
				"full_input":   "secret data",
				"full_output":  "secret response",
				"user_context": "classified info",
			},
			wantStructCount: 0,
			wantSensCount:   0,
			wantRestCount:   3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ClassifyAttributes(tc.attrs)

			gotStruct := len(result[TierStructural])
			if gotStruct != tc.wantStructCount {
				t.Errorf("structural count: got %d, want %d", gotStruct, tc.wantStructCount)
			}

			gotSens := len(result[TierSensitive])
			if gotSens != tc.wantSensCount {
				t.Errorf("sensitive count: got %d, want %d", gotSens, tc.wantSensCount)
			}

			gotRest := len(result[TierRestricted])
			if gotRest != tc.wantRestCount {
				t.Errorf("restricted count: got %d, want %d", gotRest, tc.wantRestCount)
			}

			// Verify values are preserved
			for key, value := range tc.attrs {
				tier := Classify(key)
				got, ok := result[tier][key]
				if !ok {
					t.Errorf("key %q not found in tier %d", key, tier)
					continue
				}
				if got != value {
					t.Errorf("value for key %q: got %q, want %q", key, got, value)
				}
			}
		})
	}
}
