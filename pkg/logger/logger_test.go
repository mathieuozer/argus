package logger

import (
	"context"
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		format  string
		wantErr bool
	}{
		{
			name:   "json format with info level",
			level:  "info",
			format: "json",
		},
		{
			name:   "json format with debug level",
			level:  "debug",
			format: "json",
		},
		{
			name:   "json format with warn level",
			level:  "warn",
			format: "json",
		},
		{
			name:   "json format with error level",
			level:  "error",
			format: "json",
		},
		{
			name:   "text format with info level",
			level:  "info",
			format: "text",
		},
		{
			name:   "text format with debug level",
			level:  "debug",
			format: "text",
		},
		{
			name:   "invalid level falls back to info",
			level:  "not-a-level",
			format: "json",
		},
		{
			name:   "empty level falls back to info",
			level:  "",
			format: "json",
		},
		{
			name:   "non-json format uses development config",
			level:  "info",
			format: "console",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l, err := New(tc.level, tc.format)
			if tc.wantErr {
				if err == nil {
					t.Fatal("New() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("New(%q, %q) returned error: %v", tc.level, tc.format, err)
			}
			if l == nil {
				t.Fatal("New() returned nil logger")
			}
			// Sync to flush any buffered output. Ignore errors since
			// stdout/stderr sync may fail on some platforms.
			_ = l.Sync()
		})
	}
}

func TestDefault(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
	}{
		{
			name:    "no env vars set",
			envVars: map[string]string{},
		},
		{
			name: "development environment defaults to debug",
			envVars: map[string]string{
				"ARGUS_ENV": "development",
			},
		},
		{
			name: "production environment defaults to info",
			envVars: map[string]string{
				"ARGUS_ENV": "production",
			},
		},
		{
			name: "explicit log level overrides env-based default",
			envVars: map[string]string{
				"ARGUS_ENV":       "development",
				"ARGUS_LOG_LEVEL": "warn",
			},
		},
		{
			name: "text format",
			envVars: map[string]string{
				"ARGUS_LOG_FORMAT": "text",
			},
		},
		{
			name: "all env vars set",
			envVars: map[string]string{
				"ARGUS_ENV":        "staging",
				"ARGUS_LOG_LEVEL":  "error",
				"ARGUS_LOG_FORMAT": "json",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear relevant env vars first
			_ = os.Unsetenv("ARGUS_ENV")
			_ = os.Unsetenv("ARGUS_LOG_LEVEL")
			_ = os.Unsetenv("ARGUS_LOG_FORMAT")

			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			l := Default()
			if l == nil {
				t.Fatal("Default() returned nil logger")
			}
			_ = l.Sync()
		})
	}
}

func TestWithContextAndFromContext(t *testing.T) {
	tests := []struct {
		name        string
		setupLogger bool
		wantFromCtx bool
	}{
		{
			name:        "logger stored and retrieved from context",
			setupLogger: true,
			wantFromCtx: true,
		},
		{
			name:        "empty context returns default logger",
			setupLogger: false,
			wantFromCtx: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			if tc.setupLogger {
				l, err := New("info", "json")
				if err != nil {
					t.Fatalf("New() error: %v", err)
				}

				ctx = WithContext(ctx, l)

				got := FromContext(ctx)
				if got == nil {
					t.Fatal("FromContext() returned nil")
				}
				// Verify it is the same logger we stored
				if got != l {
					t.Error("FromContext() returned a different logger than was stored")
				}
			} else {
				got := FromContext(ctx)
				if got == nil {
					t.Fatal("FromContext() returned nil for empty context; expected default logger")
				}
			}
		})
	}
}

func TestWithContextDoesNotMutateOriginal(t *testing.T) {
	ctx := context.Background()
	l, err := New("debug", "json")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	newCtx := WithContext(ctx, l)

	// The original context should not have the logger
	fromOriginal := ctx.Value(contextKey{})
	if fromOriginal != nil {
		t.Error("WithContext mutated the original context")
	}

	// The new context should have it
	fromNew := FromContext(newCtx)
	if fromNew != l {
		t.Error("FromContext on new context did not return stored logger")
	}
}

func TestFromContextWithDifferentLoggers(t *testing.T) {
	l1, err := New("info", "json")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	l2, err := New("debug", "text")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx1 := WithContext(context.Background(), l1)
	ctx2 := WithContext(context.Background(), l2)

	got1 := FromContext(ctx1)
	got2 := FromContext(ctx2)

	if got1 != l1 {
		t.Error("FromContext(ctx1) did not return l1")
	}
	if got2 != l2 {
		t.Error("FromContext(ctx2) did not return l2")
	}
	if got1 == got2 {
		t.Error("two different loggers retrieved as same pointer")
	}
}

func TestFromContextWithNilValue(t *testing.T) {
	// Store a non-*zap.Logger value in context with the same key type
	// should be impossible from outside, but FromContext should handle gracefully
	// by returning the default logger.
	ctx := context.Background()
	l := FromContext(ctx)
	if l == nil {
		t.Fatal("FromContext() on empty context returned nil; expected default logger")
	}

	// Verify the returned logger is functional by calling a no-op method
	l.Info("test message from FromContext default", zap.String("test", "value"))
	_ = l.Sync()
}
