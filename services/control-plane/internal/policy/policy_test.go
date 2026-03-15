package policy

import (
	"testing"
)

func TestEvaluateDefaultRules(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
		subject  string
		action   Action
		resource string
		wantOK   bool
	}{
		{
			name:     "admin can read anything",
			tenantID: "tenant-1",
			subject:  "admin",
			action:   ActionRead,
			resource: "agents/agent-1",
			wantOK:   true,
		},
		{
			name:     "admin can write anything",
			tenantID: "tenant-1",
			subject:  "admin",
			action:   ActionWrite,
			resource: "agents/agent-1",
			wantOK:   true,
		},
		{
			name:     "admin can delete anything",
			tenantID: "tenant-1",
			subject:  "admin",
			action:   ActionDelete,
			resource: "telemetry/spans",
			wantOK:   true,
		},
		{
			name:     "operator can read anything",
			tenantID: "tenant-1",
			subject:  "operator",
			action:   ActionRead,
			resource: "agents/agent-1",
			wantOK:   true,
		},
		{
			name:     "operator can write agents",
			tenantID: "tenant-1",
			subject:  "operator",
			action:   ActionWrite,
			resource: "agents/agent-1",
			wantOK:   true,
		},
		{
			name:     "operator can execute tasks",
			tenantID: "tenant-1",
			subject:  "operator",
			action:   ActionExecute,
			resource: "tasks/task-1",
			wantOK:   true,
		},
		{
			name:     "viewer can read anything",
			tenantID: "tenant-1",
			subject:  "viewer",
			action:   ActionRead,
			resource: "agents/agent-1",
			wantOK:   true,
		},
		{
			name:     "viewer cannot write",
			tenantID: "tenant-1",
			subject:  "viewer",
			action:   ActionWrite,
			resource: "agents/agent-1",
			wantOK:   false,
		},
		{
			name:     "agent can write telemetry",
			tenantID: "tenant-1",
			subject:  "agent",
			action:   ActionWrite,
			resource: "telemetry/spans",
			wantOK:   true,
		},
		{
			name:     "agent can execute tasks",
			tenantID: "tenant-1",
			subject:  "agent",
			action:   ActionExecute,
			resource: "tasks/task-1",
			wantOK:   true,
		},
		{
			name:     "unknown subject is denied",
			tenantID: "tenant-1",
			subject:  "nobody",
			action:   ActionRead,
			resource: "agents/agent-1",
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := New()
			allowed, err := engine.Evaluate(tc.tenantID, tc.subject, tc.action, tc.resource)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tc.wantOK {
				t.Errorf("Evaluate(%q, %q, %q, %q) = %v, want %v",
					tc.tenantID, tc.subject, tc.action, tc.resource, allowed, tc.wantOK)
			}
		})
	}
}

func TestAddAndRemoveRule(t *testing.T) {
	engine := New()

	// By default, "custom-role" is denied
	allowed, _ := engine.Evaluate("t1", "custom-role", ActionRead, "reports/report-1")
	if allowed {
		t.Error("custom-role should be denied by default")
	}

	// Add a tenant-specific rule
	engine.AddRule("t1", &Rule{
		ID:       "custom-read",
		Subject:  "custom-role",
		Action:   ActionRead,
		Resource: "reports/*",
		Effect:   "allow",
	})

	// Now it should be allowed
	allowed, _ = engine.Evaluate("t1", "custom-role", ActionRead, "reports/report-1")
	if !allowed {
		t.Error("custom-role should be allowed after adding rule")
	}

	// Other tenants should not be affected
	allowed, _ = engine.Evaluate("t2", "custom-role", ActionRead, "reports/report-1")
	if allowed {
		t.Error("custom-role should be denied for other tenants")
	}

	// Remove the rule
	err := engine.RemoveRule("t1", "custom-read")
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}

	// Should be denied again
	allowed, _ = engine.Evaluate("t1", "custom-role", ActionRead, "reports/report-1")
	if allowed {
		t.Error("custom-role should be denied after removing rule")
	}
}

func TestListRules(t *testing.T) {
	engine := New()

	rules := engine.ListRules("t1")
	defaultCount := len(rules)
	if defaultCount == 0 {
		t.Error("expected default rules")
	}

	engine.AddRule("t1", &Rule{ID: "extra", Subject: "test", Action: ActionRead, Resource: "*", Effect: "allow"})
	rules = engine.ListRules("t1")
	if len(rules) != defaultCount+1 {
		t.Errorf("ListRules returned %d rules, want %d", len(rules), defaultCount+1)
	}
}
