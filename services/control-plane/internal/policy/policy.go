package policy

import (
	"fmt"
	"strings"
	"sync"
)

// Action represents a policy action.
type Action string

const (
	ActionRead    Action = "read"
	ActionWrite   Action = "write"
	ActionDelete  Action = "delete"
	ActionExecute Action = "execute"
	ActionAdmin   Action = "admin"
)

// Rule defines a single policy rule.
type Rule struct {
	ID       string `json:"id"`
	Subject  string `json:"subject"` // role or specific user
	Action   Action `json:"action"`
	Resource string `json:"resource"` // resource pattern (supports wildcards)
	Effect   string `json:"effect"`   // "allow" or "deny"
}

// Engine evaluates access policies.
type Engine struct {
	mu    sync.RWMutex
	rules map[string][]*Rule // tenant_id -> rules
}

// New creates a new policy engine with default rules.
func New() *Engine {
	e := &Engine{
		rules: make(map[string][]*Rule),
	}
	return e
}

// defaultRules returns the built-in RBAC rules.
func defaultRules() []*Rule {
	return []*Rule{
		{ID: "admin-all", Subject: "admin", Action: "*", Resource: "*", Effect: "allow"},
		{ID: "operator-read", Subject: "operator", Action: "read", Resource: "*", Effect: "allow"},
		{ID: "operator-write-agents", Subject: "operator", Action: "write", Resource: "agents/*", Effect: "allow"},
		{ID: "operator-execute-tasks", Subject: "operator", Action: "execute", Resource: "tasks/*", Effect: "allow"},
		{ID: "operator-write-alerts", Subject: "operator", Action: "write", Resource: "alerts/*", Effect: "allow"},
		{ID: "viewer-read", Subject: "viewer", Action: "read", Resource: "*", Effect: "allow"},
		{ID: "agent-read-self", Subject: "agent", Action: "read", Resource: "agents/self", Effect: "allow"},
		{ID: "agent-write-telemetry", Subject: "agent", Action: "write", Resource: "telemetry/*", Effect: "allow"},
		{ID: "agent-execute-tasks", Subject: "agent", Action: "execute", Resource: "tasks/*", Effect: "allow"},
	}
}

// Evaluate checks if an action is allowed for the given context.
func (e *Engine) Evaluate(tenantID, subject string, action Action, resource string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check tenant-specific rules first
	if tenantRules, ok := e.rules[tenantID]; ok {
		if result, matched := evaluateRules(tenantRules, subject, action, resource); matched {
			return result, nil
		}
	}

	// Fall back to default rules
	if result, matched := evaluateRules(defaultRules(), subject, action, resource); matched {
		return result, nil
	}

	// Default deny
	return false, nil
}

// AddRule adds a tenant-specific policy rule.
func (e *Engine) AddRule(tenantID string, rule *Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules[tenantID] = append(e.rules[tenantID], rule)
}

// RemoveRule removes a tenant-specific policy rule by ID.
func (e *Engine) RemoveRule(tenantID, ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rules, ok := e.rules[tenantID]
	if !ok {
		return fmt.Errorf("no rules for tenant %s", tenantID)
	}

	for i, r := range rules {
		if r.ID == ruleID {
			e.rules[tenantID] = append(rules[:i], rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule not found: %s", ruleID)
}

// ListRules returns all rules for a tenant (including defaults).
func (e *Engine) ListRules(tenantID string) []*Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []*Rule
	result = append(result, defaultRules()...)
	if tenantRules, ok := e.rules[tenantID]; ok {
		result = append(result, tenantRules...)
	}
	return result
}

func evaluateRules(rules []*Rule, subject string, action Action, resource string) (bool, bool) {
	for _, rule := range rules {
		if matchSubject(rule.Subject, subject) &&
			matchAction(rule.Action, action) &&
			matchResource(rule.Resource, resource) {
			return rule.Effect == "allow", true
		}
	}
	return false, false
}

func matchSubject(pattern, subject string) bool {
	return pattern == "*" || pattern == subject
}

func matchAction(pattern Action, action Action) bool {
	return pattern == "*" || pattern == action
}

func matchResource(pattern, resource string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(resource, prefix)
	}
	return pattern == resource
}
