package policy

// Engine evaluates access policies.
type Engine struct {
	// In production, this would integrate with OPA
}

// New creates a new policy engine.
func New() *Engine {
	return &Engine{}
}

// Evaluate checks if an action is allowed for the given context.
func (e *Engine) Evaluate(tenantID, subject, action, resource string) (bool, error) {
	// Stub: allow all actions in development
	return true, nil
}
