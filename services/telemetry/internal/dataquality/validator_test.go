package dataquality

import (
	"encoding/json"
	"testing"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator() returned nil")
	}
	if v.rules == nil {
		t.Fatal("rules map not initialized")
	}
}

func TestValidator_AddRule(t *testing.T) {
	v := NewValidator()
	rule := &Rule{
		ID:       "r1",
		AgentID:  "agent-1",
		Name:     "test rule",
		Type:     RuleTypeSchema,
		Severity: "error",
		Enabled:  true,
		Config:   json.RawMessage(`{"required":["field_a"]}`),
	}
	v.AddRule(rule)

	v.mu.RLock()
	defer v.mu.RUnlock()
	if len(v.rules["agent-1"]) != 1 {
		t.Errorf("expected 1 rule for agent-1, got %d", len(v.rules["agent-1"]))
	}
}

func TestValidator_RemoveRule(t *testing.T) {
	tests := []struct {
		name          string
		initialRules  []*Rule
		removeAgent   string
		removeRuleID  string
		wantRemaining int
	}{
		{
			name: "remove existing rule",
			initialRules: []*Rule{
				{ID: "r1", AgentID: "agent-1", Name: "rule1", Enabled: true},
				{ID: "r2", AgentID: "agent-1", Name: "rule2", Enabled: true},
			},
			removeAgent:   "agent-1",
			removeRuleID:  "r1",
			wantRemaining: 1,
		},
		{
			name: "remove non-existent rule",
			initialRules: []*Rule{
				{ID: "r1", AgentID: "agent-1", Name: "rule1", Enabled: true},
			},
			removeAgent:   "agent-1",
			removeRuleID:  "r999",
			wantRemaining: 1,
		},
		{
			name: "remove from non-existent agent",
			initialRules: []*Rule{
				{ID: "r1", AgentID: "agent-1", Name: "rule1", Enabled: true},
			},
			removeAgent:   "agent-999",
			removeRuleID:  "r1",
			wantRemaining: 0, // no rules for agent-999
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator()
			for _, r := range tc.initialRules {
				v.AddRule(r)
			}
			v.RemoveRule(tc.removeAgent, tc.removeRuleID)

			v.mu.RLock()
			defer v.mu.RUnlock()
			got := len(v.rules[tc.removeAgent])
			if got != tc.wantRemaining {
				t.Errorf("remaining rules for %s: got %d, want %d", tc.removeAgent, got, tc.wantRemaining)
			}
		})
	}
}

func TestValidator_ValidateSchema(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		data       map[string]interface{}
		wantPassed bool
		wantMsg    string
	}{
		{
			name:       "all required fields present",
			config:     `{"required":["name","age"]}`,
			data:       map[string]interface{}{"name": "Alice", "age": 30, "extra": true},
			wantPassed: true,
		},
		{
			name:       "missing required field",
			config:     `{"required":["name","age","email"]}`,
			data:       map[string]interface{}{"name": "Bob", "age": 25},
			wantPassed: false,
			wantMsg:    "missing required field: email",
		},
		{
			name:       "empty required list passes",
			config:     `{"required":[]}`,
			data:       map[string]interface{}{},
			wantPassed: true,
		},
		{
			name:       "invalid config json",
			config:     `{invalid}`,
			data:       map[string]interface{}{},
			wantPassed: false,
			wantMsg:    "invalid schema config",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator()
			rule := &Rule{
				ID:       "schema-rule",
				AgentID:  "agent-1",
				Name:     "schema check",
				Type:     RuleTypeSchema,
				Config:   json.RawMessage(tc.config),
				Severity: "error",
				Enabled:  true,
			}
			v.AddRule(rule)

			results := v.Validate("agent-1", tc.data)
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			if results[0].Passed != tc.wantPassed {
				t.Errorf("Passed: got %v, want %v", results[0].Passed, tc.wantPassed)
			}
			if tc.wantMsg != "" && results[0].Message != tc.wantMsg {
				t.Errorf("Message: got %q, want %q", results[0].Message, tc.wantMsg)
			}
		})
	}
}

func TestValidator_ValidateRange(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		data       map[string]interface{}
		wantPassed bool
		wantMsg    string
	}{
		{
			name:       "value within range",
			config:     `{"field":"latency","min":0,"max":1000}`,
			data:       map[string]interface{}{"latency": 500.0},
			wantPassed: true,
		},
		{
			name:       "value at lower bound",
			config:     `{"field":"latency","min":0,"max":1000}`,
			data:       map[string]interface{}{"latency": 0.0},
			wantPassed: true,
		},
		{
			name:       "value at upper bound",
			config:     `{"field":"latency","min":0,"max":1000}`,
			data:       map[string]interface{}{"latency": 1000.0},
			wantPassed: true,
		},
		{
			name:       "value below range",
			config:     `{"field":"latency","min":10,"max":1000}`,
			data:       map[string]interface{}{"latency": 5.0},
			wantPassed: false,
			wantMsg:    "field latency value 5.00 out of range [10.00, 1000.00]",
		},
		{
			name:       "value above range",
			config:     `{"field":"latency","min":0,"max":100}`,
			data:       map[string]interface{}{"latency": 200.0},
			wantPassed: false,
			wantMsg:    "field latency value 200.00 out of range [0.00, 100.00]",
		},
		{
			name:       "field not found",
			config:     `{"field":"latency","min":0,"max":100}`,
			data:       map[string]interface{}{"other": 50.0},
			wantPassed: false,
			wantMsg:    "field latency not found",
		},
		{
			name:       "field not numeric",
			config:     `{"field":"latency","min":0,"max":100}`,
			data:       map[string]interface{}{"latency": "not a number"},
			wantPassed: false,
			wantMsg:    "field latency is not numeric",
		},
		{
			name:       "integer value within range",
			config:     `{"field":"count","min":0,"max":10}`,
			data:       map[string]interface{}{"count": 5},
			wantPassed: true,
		},
		{
			name:       "int64 value within range",
			config:     `{"field":"tokens","min":0,"max":10000}`,
			data:       map[string]interface{}{"tokens": int64(5000)},
			wantPassed: true,
		},
		{
			name:       "invalid config",
			config:     `{bad_json}`,
			data:       map[string]interface{}{},
			wantPassed: false,
			wantMsg:    "invalid range config",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator()
			rule := &Rule{
				ID:       "range-rule",
				AgentID:  "agent-1",
				Name:     "range check",
				Type:     RuleTypeRange,
				Config:   json.RawMessage(tc.config),
				Severity: "warning",
				Enabled:  true,
			}
			v.AddRule(rule)

			results := v.Validate("agent-1", tc.data)
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			if results[0].Passed != tc.wantPassed {
				t.Errorf("Passed: got %v, want %v", results[0].Passed, tc.wantPassed)
			}
			if tc.wantMsg != "" && results[0].Message != tc.wantMsg {
				t.Errorf("Message: got %q, want %q", results[0].Message, tc.wantMsg)
			}
		})
	}
}

func TestValidator_ValidateRegex(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		data       map[string]interface{}
		wantPassed bool
		wantMsg    string
	}{
		{
			name:       "value matches pattern",
			config:     `{"field":"agent_id","pattern":"^agent-[0-9]+$"}`,
			data:       map[string]interface{}{"agent_id": "agent-42"},
			wantPassed: true,
		},
		{
			name:       "value does not match pattern",
			config:     `{"field":"agent_id","pattern":"^agent-[0-9]+$"}`,
			data:       map[string]interface{}{"agent_id": "invalid-name"},
			wantPassed: false,
			wantMsg:    "field agent_id does not match pattern ^agent-[0-9]+$",
		},
		{
			name:       "field not found",
			config:     `{"field":"agent_id","pattern":"^agent-[0-9]+$"}`,
			data:       map[string]interface{}{"other": "value"},
			wantPassed: false,
			wantMsg:    "field agent_id not found",
		},
		{
			name:       "field not a string",
			config:     `{"field":"agent_id","pattern":"^agent-[0-9]+$"}`,
			data:       map[string]interface{}{"agent_id": 123},
			wantPassed: false,
			wantMsg:    "field agent_id is not a string",
		},
		{
			name:       "invalid regex pattern",
			config:     `{"field":"value","pattern":"[invalid"}`,
			data:       map[string]interface{}{"value": "test"},
			wantPassed: false,
			wantMsg:    "invalid regex pattern: error parsing regexp: missing closing ]: `[invalid`",
		},
		{
			name:       "invalid config",
			config:     `{bad}`,
			data:       map[string]interface{}{},
			wantPassed: false,
			wantMsg:    "invalid regex config",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator()
			rule := &Rule{
				ID:       "regex-rule",
				AgentID:  "agent-1",
				Name:     "regex check",
				Type:     RuleTypeRegex,
				Config:   json.RawMessage(tc.config),
				Severity: "info",
				Enabled:  true,
			}
			v.AddRule(rule)

			results := v.Validate("agent-1", tc.data)
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			if results[0].Passed != tc.wantPassed {
				t.Errorf("Passed: got %v, want %v", results[0].Passed, tc.wantPassed)
			}
			if tc.wantMsg != "" && results[0].Message != tc.wantMsg {
				t.Errorf("Message: got %q, want %q", results[0].Message, tc.wantMsg)
			}
		})
	}
}

func TestValidator_DisabledRuleSkipped(t *testing.T) {
	v := NewValidator()
	v.AddRule(&Rule{
		ID:       "disabled-rule",
		AgentID:  "agent-1",
		Name:     "disabled",
		Type:     RuleTypeSchema,
		Config:   json.RawMessage(`{"required":["missing_field"]}`),
		Severity: "error",
		Enabled:  false,
	})

	results := v.Validate("agent-1", map[string]interface{}{})
	if len(results) != 0 {
		t.Errorf("expected 0 results for disabled rule, got %d", len(results))
	}
}

func TestValidator_UnknownRuleType(t *testing.T) {
	v := NewValidator()
	v.AddRule(&Rule{
		ID:       "unknown-rule",
		AgentID:  "agent-1",
		Name:     "unknown type",
		Type:     RuleType("custom"),
		Config:   json.RawMessage(`{}`),
		Severity: "error",
		Enabled:  true,
	})

	results := v.Validate("agent-1", map[string]interface{}{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("expected unknown rule type to fail")
	}
	if results[0].Message != "unknown rule type: custom" {
		t.Errorf("unexpected message: %q", results[0].Message)
	}
}

func TestValidator_NoRulesForAgent(t *testing.T) {
	v := NewValidator()
	v.AddRule(&Rule{
		ID:       "r1",
		AgentID:  "agent-1",
		Name:     "rule for agent-1",
		Type:     RuleTypeSchema,
		Config:   json.RawMessage(`{"required":["x"]}`),
		Severity: "error",
		Enabled:  true,
	})

	results := v.Validate("agent-other", map[string]interface{}{})
	if len(results) != 0 {
		t.Errorf("expected 0 results for agent with no rules, got %d", len(results))
	}
}

func TestValidator_MultipleRules(t *testing.T) {
	v := NewValidator()
	v.AddRule(&Rule{
		ID:       "r1",
		AgentID:  "agent-1",
		Name:     "schema check",
		Type:     RuleTypeSchema,
		Config:   json.RawMessage(`{"required":["name"]}`),
		Severity: "error",
		Enabled:  true,
	})
	v.AddRule(&Rule{
		ID:       "r2",
		AgentID:  "agent-1",
		Name:     "range check",
		Type:     RuleTypeRange,
		Config:   json.RawMessage(`{"field":"score","min":0,"max":100}`),
		Severity: "warning",
		Enabled:  true,
	})

	data := map[string]interface{}{
		"name":  "test",
		"score": 50.0,
	}
	results := v.Validate("agent-1", data)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("rule %s (%s) should have passed", r.RuleID, r.RuleName)
		}
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantVal float64
		wantOK  bool
	}{
		{name: "float64", input: float64(3.14), wantVal: 3.14, wantOK: true},
		{name: "float32", input: float32(2.5), wantVal: float64(float32(2.5)), wantOK: true},
		{name: "int", input: 42, wantVal: 42.0, wantOK: true},
		{name: "int64", input: int64(100), wantVal: 100.0, wantOK: true},
		{name: "string", input: "nope", wantVal: 0, wantOK: false},
		{name: "bool", input: true, wantVal: 0, wantOK: false},
		{name: "nil", input: nil, wantVal: 0, wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, ok := toFloat64(tc.input)
			if ok != tc.wantOK {
				t.Errorf("toFloat64(%v) ok = %v, want %v", tc.input, ok, tc.wantOK)
			}
			if ok && val != tc.wantVal {
				t.Errorf("toFloat64(%v) = %v, want %v", tc.input, val, tc.wantVal)
			}
		})
	}
}
