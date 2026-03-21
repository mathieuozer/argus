package database

import (
	"testing"
)

func TestNewPool_InvalidDSN(t *testing.T) {
	// Test with empty DSN - should fail at parse
	_, err := NewPool(t.Context(), "")
	if err == nil {
		t.Error("expected error for empty DSN")
	}
}

func TestNewPool_UnreachableHost(t *testing.T) {
	_, err := NewPool(t.Context(), "postgres://user:pass@localhost:59999/nonexistent?connect_timeout=1")
	if err == nil {
		t.Error("expected error for unreachable host")
	}
}

func TestNewPoolWithMaxConns_InvalidDSN(t *testing.T) {
	_, err := NewPoolWithMaxConns(t.Context(), "", 10)
	if err == nil {
		t.Error("expected error for empty DSN")
	}
}
