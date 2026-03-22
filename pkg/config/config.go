package config

import (
	"fmt"
	"os"
	"strconv"
)

// Base holds configuration common to all Argus services.
type Base struct {
	Env               string `env:"ARGUS_ENV" default:"development"`
	LogLevel          string `env:"ARGUS_LOG_LEVEL" default:"info"`
	LogFormat         string `env:"ARGUS_LOG_FORMAT" default:"json"`
	TenantEnforcement string `env:"ARGUS_TENANT_ENFORCEMENT" default:"strict"`
	DBDSN             string `env:"ARGUS_DB_DSN" default:"postgres://argus:argus@localhost:5432/argus?sslmode=disable"`
	DBMaxConns        int    `env:"ARGUS_DB_MAX_CONNS" default:"25"`
	NATSUrl           string `env:"ARGUS_NATS_URL" default:"nats://localhost:4222"`
	NATSStream        string `env:"ARGUS_NATS_STREAM" default:"ARGUS_TELEMETRY"`
	AirGap            bool   `env:"ARGUS_AIR_GAP" default:"false"`
}

// Load populates a Base config from environment variables.
func Load() (*Base, error) {
	cfg := &Base{}

	cfg.Env = getEnv("ARGUS_ENV", "development")
	cfg.LogLevel = getEnv("ARGUS_LOG_LEVEL", "info")
	cfg.LogFormat = getEnv("ARGUS_LOG_FORMAT", "json")
	cfg.TenantEnforcement = getEnv("ARGUS_TENANT_ENFORCEMENT", "strict")
	cfg.DBDSN = getEnv("ARGUS_DB_DSN", "postgres://argus:argus@localhost:5432/argus?sslmode=disable")
	cfg.NATSUrl = getEnv("ARGUS_NATS_URL", "nats://localhost:4222")
	cfg.NATSStream = getEnv("ARGUS_NATS_STREAM", "ARGUS_TELEMETRY")

	maxConns, err := strconv.Atoi(getEnv("ARGUS_DB_MAX_CONNS", "25"))
	if err != nil {
		return nil, fmt.Errorf("invalid ARGUS_DB_MAX_CONNS: %w", err)
	}
	cfg.DBMaxConns = maxConns

	cfg.AirGap = getEnv("ARGUS_AIR_GAP", "false") == "true"

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// Validate checks that the configuration is consistent and safe.
func (c *Base) Validate() error {
	validEnvs := map[string]bool{"development": true, "staging": true, "production": true, "test": true}
	if !validEnvs[c.Env] {
		return fmt.Errorf("ARGUS_ENV must be one of development, staging, production, test; got %q", c.Env)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("ARGUS_LOG_LEVEL must be one of debug, info, warn, error; got %q", c.LogLevel)
	}

	if c.DBMaxConns < 1 || c.DBMaxConns > 200 {
		return fmt.Errorf("ARGUS_DB_MAX_CONNS must be between 1 and 200; got %d", c.DBMaxConns)
	}

	// In production, require a JWT secret
	if c.Env == "production" {
		if os.Getenv("ARGUS_JWT_SECRET") == "" {
			return fmt.Errorf("ARGUS_JWT_SECRET is required in production")
		}
		if c.TenantEnforcement != "strict" {
			return fmt.Errorf("ARGUS_TENANT_ENFORCEMENT must be 'strict' in production")
		}
	}

	return nil
}

// IsProduction returns true if running in production mode.
func (c *Base) IsProduction() bool {
	return c.Env == "production"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
