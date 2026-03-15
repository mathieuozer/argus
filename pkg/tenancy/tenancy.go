package tenancy

import (
	"context"
	"errors"
)

type contextKey struct{}

var ErrTenantNotFound = errors.New("tenant ID not found in context")

// FromContext extracts the tenant ID from the context.
func FromContext(ctx context.Context) (string, error) {
	v, ok := ctx.Value(contextKey{}).(string)
	if !ok || v == "" {
		return "", ErrTenantNotFound
	}
	return v, nil
}

// MustFromContext extracts the tenant ID or panics.
func MustFromContext(ctx context.Context) string {
	id, err := FromContext(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// WithTenant returns a new context with the tenant ID set.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, contextKey{}, tenantID)
}
