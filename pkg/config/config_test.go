package config

import (
	"os"
	"testing"
)

func clearArgusEnvVars(t *testing.T) {
	t.Helper()
	envVars := []string{
		"ARGUS_ENV",
		"ARGUS_LOG_LEVEL",
		"ARGUS_LOG_FORMAT",
		"ARGUS_TENANT_ENFORCEMENT",
		"ARGUS_DB_DSN",
		"ARGUS_DB_MAX_CONNS",
		"ARGUS_NATS_URL",
		"ARGUS_NATS_STREAM",
		"ARGUS_AIR_GAP",
	}
	for _, key := range envVars {
		t.Setenv(key, "")
		_ = os.Unsetenv(key)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback string
		envVal   string
		want     string
	}{
		{
			name:     "returns fallback when env not set",
			key:      "ARGUS_TEST_GETENV_NOTSET",
			fallback: "default-value",
			envVal:   "",
			want:     "default-value",
		},
		{
			name:     "returns env value when set",
			key:      "ARGUS_TEST_GETENV_SET",
			fallback: "default-value",
			envVal:   "custom-value",
			want:     "custom-value",
		},
		{
			name:     "returns fallback when env is empty string",
			key:      "ARGUS_TEST_GETENV_EMPTY",
			fallback: "fallback",
			envVal:   "",
			want:     "fallback",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVal != "" {
				t.Setenv(tc.key, tc.envVal)
			} else {
				_ = os.Unsetenv(tc.key)
			}

			got := getEnv(tc.key, tc.fallback)
			if got != tc.want {
				t.Errorf("getEnv(%q, %q) = %q; want %q", tc.key, tc.fallback, got, tc.want)
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	clearArgusEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Env default", cfg.Env, "development"},
		{"LogLevel default", cfg.LogLevel, "info"},
		{"LogFormat default", cfg.LogFormat, "json"},
		{"TenantEnforcement default", cfg.TenantEnforcement, "strict"},
		{"DBDSN default", cfg.DBDSN, "postgres://argus:argus@localhost:5432/argus?sslmode=disable"},
		{"NATSUrl default", cfg.NATSUrl, "nats://localhost:4222"},
		{"NATSStream default", cfg.NATSStream, "ARGUS_TELEMETRY"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %q; want %q", tc.got, tc.want)
			}
		})
	}

	t.Run("DBMaxConns default", func(t *testing.T) {
		if cfg.DBMaxConns != 25 {
			t.Errorf("DBMaxConns = %d; want 25", cfg.DBMaxConns)
		}
	})

	t.Run("AirGap default", func(t *testing.T) {
		if cfg.AirGap != false {
			t.Errorf("AirGap = %v; want false", cfg.AirGap)
		}
	})
}

func TestLoadWithEnvVars(t *testing.T) {
	tests := []struct {
		name   string
		envs   map[string]string
		check  func(t *testing.T, cfg *Base)
	}{
		{
			name: "custom environment",
			envs: map[string]string{
				"ARGUS_ENV":        "production",
				"ARGUS_JWT_SECRET": "test-secret-for-production-validation",
			},
			check: func(t *testing.T, cfg *Base) {
				if cfg.Env != "production" {
					t.Errorf("Env = %q; want %q", cfg.Env, "production")
				}
			},
		},
		{
			name: "custom log level and format",
			envs: map[string]string{
				"ARGUS_LOG_LEVEL":  "debug",
				"ARGUS_LOG_FORMAT": "text",
			},
			check: func(t *testing.T, cfg *Base) {
				if cfg.LogLevel != "debug" {
					t.Errorf("LogLevel = %q; want %q", cfg.LogLevel, "debug")
				}
				if cfg.LogFormat != "text" {
					t.Errorf("LogFormat = %q; want %q", cfg.LogFormat, "text")
				}
			},
		},
		{
			name: "custom DB DSN and max conns",
			envs: map[string]string{
				"ARGUS_DB_DSN":       "postgres://user:pass@db.prod:5432/argus?sslmode=require",
				"ARGUS_DB_MAX_CONNS": "50",
			},
			check: func(t *testing.T, cfg *Base) {
				if cfg.DBDSN != "postgres://user:pass@db.prod:5432/argus?sslmode=require" {
					t.Errorf("DBDSN = %q; want custom DSN", cfg.DBDSN)
				}
				if cfg.DBMaxConns != 50 {
					t.Errorf("DBMaxConns = %d; want 50", cfg.DBMaxConns)
				}
			},
		},
		{
			name: "custom NATS config",
			envs: map[string]string{
				"ARGUS_NATS_URL":    "nats://nats.prod:4222",
				"ARGUS_NATS_STREAM": "CUSTOM_STREAM",
			},
			check: func(t *testing.T, cfg *Base) {
				if cfg.NATSUrl != "nats://nats.prod:4222" {
					t.Errorf("NATSUrl = %q; want custom URL", cfg.NATSUrl)
				}
				if cfg.NATSStream != "CUSTOM_STREAM" {
					t.Errorf("NATSStream = %q; want %q", cfg.NATSStream, "CUSTOM_STREAM")
				}
			},
		},
		{
			name: "air gap enabled",
			envs: map[string]string{
				"ARGUS_AIR_GAP": "true",
			},
			check: func(t *testing.T, cfg *Base) {
				if cfg.AirGap != true {
					t.Errorf("AirGap = %v; want true", cfg.AirGap)
				}
			},
		},
		{
			name: "air gap explicitly false",
			envs: map[string]string{
				"ARGUS_AIR_GAP": "false",
			},
			check: func(t *testing.T, cfg *Base) {
				if cfg.AirGap != false {
					t.Errorf("AirGap = %v; want false", cfg.AirGap)
				}
			},
		},
		{
			name: "air gap non-true value treated as false",
			envs: map[string]string{
				"ARGUS_AIR_GAP": "yes",
			},
			check: func(t *testing.T, cfg *Base) {
				if cfg.AirGap != false {
					t.Errorf("AirGap = %v; want false (only 'true' should enable)", cfg.AirGap)
				}
			},
		},
		{
			name: "tenant enforcement override",
			envs: map[string]string{
				"ARGUS_TENANT_ENFORCEMENT": "permissive",
			},
			check: func(t *testing.T, cfg *Base) {
				if cfg.TenantEnforcement != "permissive" {
					t.Errorf("TenantEnforcement = %q; want %q", cfg.TenantEnforcement, "permissive")
				}
			},
		},
		{
			name: "all values overridden",
			envs: map[string]string{
				"ARGUS_ENV":                "staging",
				"ARGUS_LOG_LEVEL":          "warn",
				"ARGUS_LOG_FORMAT":         "text",
				"ARGUS_TENANT_ENFORCEMENT": "relaxed",
				"ARGUS_DB_DSN":             "postgres://staging:staging@db:5432/argus_stg",
				"ARGUS_DB_MAX_CONNS":       "10",
				"ARGUS_NATS_URL":           "nats://nats-stg:4222",
				"ARGUS_NATS_STREAM":        "STG_TELEMETRY",
				"ARGUS_AIR_GAP":            "true",
			},
			check: func(t *testing.T, cfg *Base) {
				if cfg.Env != "staging" {
					t.Errorf("Env = %q; want %q", cfg.Env, "staging")
				}
				if cfg.LogLevel != "warn" {
					t.Errorf("LogLevel = %q; want %q", cfg.LogLevel, "warn")
				}
				if cfg.LogFormat != "text" {
					t.Errorf("LogFormat = %q; want %q", cfg.LogFormat, "text")
				}
				if cfg.TenantEnforcement != "relaxed" {
					t.Errorf("TenantEnforcement = %q; want %q", cfg.TenantEnforcement, "relaxed")
				}
				if cfg.DBDSN != "postgres://staging:staging@db:5432/argus_stg" {
					t.Errorf("DBDSN = %q; want staging DSN", cfg.DBDSN)
				}
				if cfg.DBMaxConns != 10 {
					t.Errorf("DBMaxConns = %d; want 10", cfg.DBMaxConns)
				}
				if cfg.NATSUrl != "nats://nats-stg:4222" {
					t.Errorf("NATSUrl = %q; want staging URL", cfg.NATSUrl)
				}
				if cfg.NATSStream != "STG_TELEMETRY" {
					t.Errorf("NATSStream = %q; want %q", cfg.NATSStream, "STG_TELEMETRY")
				}
				if cfg.AirGap != true {
					t.Errorf("AirGap = %v; want true", cfg.AirGap)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearArgusEnvVars(t)
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}
			tc.check(t, cfg)
		})
	}
}

func TestLoadInvalidDBMaxConns(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"non-numeric string", "abc"},
		{"floating point number", "25.5"},
		{"empty-looking whitespace", " "},
		{"special characters", "!@#"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearArgusEnvVars(t)
			t.Setenv("ARGUS_DB_MAX_CONNS", tc.value)

			cfg, err := Load()
			if err == nil {
				t.Fatalf("Load() returned nil error for ARGUS_DB_MAX_CONNS=%q; got cfg.DBMaxConns=%d", tc.value, cfg.DBMaxConns)
			}
		})
	}
}
