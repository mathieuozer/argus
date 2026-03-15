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

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
