package dataquality

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
)

type RuleType string

const (
	RuleTypeSchema RuleType = "schema"
	RuleTypeRange  RuleType = "range"
	RuleTypeRegex  RuleType = "regex"
)

type Rule struct {
	ID       string          `json:"id"`
	AgentID  string          `json:"agent_id"`
	Name     string          `json:"name"`
	Type     RuleType        `json:"type"`
	Target   string          `json:"target"` // "input", "output", "attribute"
	Config   json.RawMessage `json:"config"`
	Severity string          `json:"severity"`
	Enabled  bool            `json:"enabled"`
}

type ValidationResult struct {
	RuleID   string `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message,omitempty"`
	Severity string `json:"severity"`
}

type Validator struct {
	mu    sync.RWMutex
	rules map[string][]*Rule // agentID -> rules
}

func NewValidator() *Validator {
	return &Validator{
		rules: make(map[string][]*Rule),
	}
}

func (v *Validator) AddRule(rule *Rule) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rules[rule.AgentID] = append(v.rules[rule.AgentID], rule)
}

func (v *Validator) RemoveRule(agentID, ruleID string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	rules := v.rules[agentID]
	for i, r := range rules {
		if r.ID == ruleID {
			v.rules[agentID] = append(rules[:i], rules[i+1:]...)
			return
		}
	}
}

func (v *Validator) Validate(agentID string, data map[string]interface{}) []ValidationResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	rules := v.rules[agentID]
	var results []ValidationResult

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		result := v.validateRule(rule, data)
		results = append(results, result)
	}
	return results
}

func (v *Validator) validateRule(rule *Rule, data map[string]interface{}) ValidationResult {
	switch rule.Type {
	case RuleTypeSchema:
		return v.validateSchema(rule, data)
	case RuleTypeRange:
		return v.validateRange(rule, data)
	case RuleTypeRegex:
		return v.validateRegex(rule, data)
	default:
		return ValidationResult{
			RuleID:   rule.ID,
			RuleName: rule.Name,
			Passed:   false,
			Message:  fmt.Sprintf("unknown rule type: %s", rule.Type),
			Severity: rule.Severity,
		}
	}
}

func (v *Validator) validateSchema(rule *Rule, data map[string]interface{}) ValidationResult {
	var config struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(rule.Config, &config); err != nil {
		return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: false, Message: "invalid schema config", Severity: rule.Severity}
	}

	for _, field := range config.Required {
		if _, ok := data[field]; !ok {
			return ValidationResult{
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Passed:   false,
				Message:  fmt.Sprintf("missing required field: %s", field),
				Severity: rule.Severity,
			}
		}
	}
	return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: true, Severity: rule.Severity}
}

func (v *Validator) validateRange(rule *Rule, data map[string]interface{}) ValidationResult {
	var config struct {
		Field string  `json:"field"`
		Min   float64 `json:"min"`
		Max   float64 `json:"max"`
	}
	if err := json.Unmarshal(rule.Config, &config); err != nil {
		return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: false, Message: "invalid range config", Severity: rule.Severity}
	}

	val, ok := data[config.Field]
	if !ok {
		return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: false, Message: fmt.Sprintf("field %s not found", config.Field), Severity: rule.Severity}
	}

	numVal, ok := toFloat64(val)
	if !ok {
		return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: false, Message: fmt.Sprintf("field %s is not numeric", config.Field), Severity: rule.Severity}
	}

	if numVal < config.Min || numVal > config.Max {
		return ValidationResult{
			RuleID:   rule.ID,
			RuleName: rule.Name,
			Passed:   false,
			Message:  fmt.Sprintf("field %s value %.2f out of range [%.2f, %.2f]", config.Field, numVal, config.Min, config.Max),
			Severity: rule.Severity,
		}
	}
	return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: true, Severity: rule.Severity}
}

func (v *Validator) validateRegex(rule *Rule, data map[string]interface{}) ValidationResult {
	var config struct {
		Field   string `json:"field"`
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(rule.Config, &config); err != nil {
		return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: false, Message: "invalid regex config", Severity: rule.Severity}
	}

	val, ok := data[config.Field]
	if !ok {
		return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: false, Message: fmt.Sprintf("field %s not found", config.Field), Severity: rule.Severity}
	}

	strVal, ok := val.(string)
	if !ok {
		return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: false, Message: fmt.Sprintf("field %s is not a string", config.Field), Severity: rule.Severity}
	}

	matched, err := regexp.MatchString(config.Pattern, strVal)
	if err != nil {
		return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: false, Message: fmt.Sprintf("invalid regex pattern: %v", err), Severity: rule.Severity}
	}

	if !matched {
		return ValidationResult{
			RuleID:   rule.ID,
			RuleName: rule.Name,
			Passed:   false,
			Message:  fmt.Sprintf("field %s does not match pattern %s", config.Field, config.Pattern),
			Severity: rule.Severity,
		}
	}
	return ValidationResult{RuleID: rule.ID, RuleName: rule.Name, Passed: true, Severity: rule.Severity}
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
