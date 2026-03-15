package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey struct{}

// New creates a new structured logger.
func New(level, format string) (*zap.Logger, error) {
	var cfg zap.Config
	if format == "json" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		lvl = zapcore.InfoLevel
	}
	cfg.Level.SetLevel(lvl)

	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	return cfg.Build()
}

// Default creates a default logger based on environment.
func Default() *zap.Logger {
	env := os.Getenv("ARGUS_ENV")
	level := os.Getenv("ARGUS_LOG_LEVEL")
	format := os.Getenv("ARGUS_LOG_FORMAT")

	if level == "" {
		if env == "development" {
			level = "debug"
		} else {
			level = "info"
		}
	}
	if format == "" {
		format = "json"
	}

	l, err := New(level, format)
	if err != nil {
		l, _ = zap.NewProduction()
	}
	return l
}

// WithContext returns a new context with the logger attached.
func WithContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext retrieves the logger from the context, or returns the default.
func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(contextKey{}).(*zap.Logger); ok {
		return l
	}
	return Default()
}
