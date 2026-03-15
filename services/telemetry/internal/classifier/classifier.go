package classifier

// DataTier represents the classification level of telemetry data.
type DataTier int

const (
	TierStructural DataTier = 1 // exits node freely
	TierSensitive  DataTier = 2 // stays within tenant boundary
	TierRestricted DataTier = 3 // never leaves node
)

// FieldRules defines which fields map to which tier.
var FieldRules = map[string]DataTier{
	"latency_ms":   TierStructural,
	"token_count":  TierStructural,
	"error_code":   TierStructural,
	"agent_id":     TierStructural,
	"timestamp":    TierStructural,
	"task_desc":    TierSensitive,
	"tool_params":  TierSensitive,
	"partial_out":  TierSensitive,
	"full_input":   TierRestricted,
	"full_output":  TierRestricted,
	"user_context": TierRestricted,
}

// Classify determines the data tier for a field name.
func Classify(fieldName string) DataTier {
	if tier, ok := FieldRules[fieldName]; ok {
		return tier
	}
	return TierSensitive // default to sensitive if unknown
}

// ClassifyAttributes classifies a set of attributes and returns them grouped by tier.
func ClassifyAttributes(attrs map[string]string) map[DataTier]map[string]string {
	result := map[DataTier]map[string]string{
		TierStructural: {},
		TierSensitive:  {},
		TierRestricted: {},
	}

	for key, value := range attrs {
		tier := Classify(key)
		result[tier][key] = value
	}

	return result
}
