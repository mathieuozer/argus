package guardrails

import (
	"regexp"
	"strings"
	"time"
)

// RuleType defines the type of guardrail check.
type RuleType string

const (
	RuleTypePII         RuleType = "pii_detection"
	RuleTypeInjection   RuleType = "prompt_injection"
	RuleTypeToxicity    RuleType = "toxicity"
	RuleTypeBlocklist   RuleType = "blocklist"
	RuleTypeSchema      RuleType = "schema_enforcement"
	RuleTypeCustomRegex RuleType = "custom_regex"
)

// Rule defines a guardrail rule.
type Rule struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        RuleType  `json:"type"`
	Pattern     string    `json:"pattern"`
	Action      string    `json:"action"` // block, warn, log
	Enabled     bool      `json:"enabled"`
	AgentIDs    []string  `json:"agent_ids"` // empty = all agents
	CreatedAt   time.Time `json:"created_at"`
}

// Violation records a guardrail violation.
type Violation struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	RuleType  RuleType  `json:"rule_type"`
	AgentID   string    `json:"agent_id"`
	SpanID    string    `json:"span_id"`
	Action    string    `json:"action"`
	Content   string    `json:"content"` // redacted excerpt
	CreatedAt time.Time `json:"created_at"`
}

// Engine evaluates content against guardrail rules.
type Engine struct {
	rules []*Rule
}

// NewEngine creates a new guardrail engine.
func NewEngine(rules []*Rule) *Engine {
	return &Engine{rules: rules}
}

// CheckResult is the result of a guardrail check.
type CheckResult struct {
	Passed     bool         `json:"passed"`
	Violations []*Violation `json:"violations"`
}

// Check evaluates input content against all enabled rules for a given agent.
func (e *Engine) Check(tenantID, agentID, spanID, content string) *CheckResult {
	result := &CheckResult{Passed: true}

	for _, rule := range e.rules {
		if !rule.Enabled || rule.TenantID != tenantID {
			continue
		}
		if len(rule.AgentIDs) > 0 && !contains(rule.AgentIDs, agentID) {
			continue
		}

		violated := false
		switch rule.Type {
		case RuleTypeInjection:
			violated = checkInjection(content)
		case RuleTypeBlocklist:
			violated = checkBlocklist(content, rule.Pattern)
		case RuleTypeCustomRegex:
			violated = checkRegex(content, rule.Pattern)
		case RuleTypeToxicity:
			violated = checkToxicity(content)
		case RuleTypePII:
			violated = checkPII(content)
		}

		if violated {
			result.Passed = false
			excerpt := content
			if len(excerpt) > 100 {
				excerpt = excerpt[:100] + "..."
			}
			result.Violations = append(result.Violations, &Violation{
				ID:        "viol-" + time.Now().Format("20060102150405"),
				TenantID:  tenantID,
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				RuleType:  rule.Type,
				AgentID:   agentID,
				SpanID:    spanID,
				Action:    rule.Action,
				Content:   excerpt,
				CreatedAt: time.Now(),
			})

			if rule.Action == "block" {
				break
			}
		}
	}

	return result
}

func checkInjection(content string) bool {
	lower := strings.ToLower(content)
	patterns := []string{
		"ignore previous instructions",
		"ignore all previous",
		"disregard your instructions",
		"forget your instructions",
		"you are now",
		"act as if you",
		"pretend you are",
		"system prompt",
		"reveal your prompt",
		"show me your instructions",
		"override your",
		"bypass your",
		"jailbreak",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func checkBlocklist(content, pattern string) bool {
	words := strings.Split(pattern, ",")
	lower := strings.ToLower(content)
	for _, word := range words {
		if strings.Contains(lower, strings.TrimSpace(strings.ToLower(word))) {
			return true
		}
	}
	return false
}

func checkRegex(content, pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(content)
}

func checkToxicity(content string) bool {
	lower := strings.ToLower(content)
	toxicPatterns := []string{
		"kill", "murder", "weapon", "bomb", "terrorist",
		"suicide", "self-harm", "hack into", "exploit vulnerability",
	}
	for _, p := range toxicPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func checkPII(content string) bool {
	piiPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`),
		regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		regexp.MustCompile(`\b\d{16}\b`),
	}
	for _, re := range piiPatterns {
		if re.MatchString(content) {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
