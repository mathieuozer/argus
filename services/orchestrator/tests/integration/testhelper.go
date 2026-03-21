package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/argus-platform/argus/pkg/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultTestDSN = "postgres://argus_test:argus_test@localhost:5433/argus_test?sslmode=disable"
	defaultNATSURL = "nats://localhost:4223"
)

// TestDSN returns the PostgreSQL DSN for integration tests.
func TestDSN() string {
	if dsn := os.Getenv("ARGUS_TEST_DB_DSN"); dsn != "" {
		return dsn
	}
	return defaultTestDSN
}

// TestNATSURL returns the NATS URL for integration tests.
func TestNATSURL() string {
	if url := os.Getenv("ARGUS_TEST_NATS_URL"); url != "" {
		return url
	}
	return defaultNATSURL
}

// SetupTestDB creates a connection pool and runs migrations.
// Returns the pool and a cleanup function.
func SetupTestDB(t *testing.T) (*database.Pool, func()) {
	t.Helper()

	dsn := TestDSN()
	ctx := context.Background()

	pool, err := database.NewPool(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test: cannot connect to test database: %v", err)
	}

	// Run migrations
	if err := runMigrations(ctx, pool.Pool); err != nil {
		pool.Close()
		t.Fatalf("failed to run migrations: %v", err)
	}

	cleanup := func() {
		cleanupDB(ctx, pool.Pool)
		pool.Close()
	}

	return pool, cleanup
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		`CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			display_name TEXT NOT NULL,
			isolation_tier TEXT NOT NULL CHECK (isolation_tier IN ('A', 'B', 'C')),
			storage_regions TEXT[] NOT NULL DEFAULT '{}',
			pii_scrub BOOLEAN NOT NULL DEFAULT true,
			compliance_profile TEXT NOT NULL DEFAULT 'eu-gdpr',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS agents (
			id TEXT NOT NULL,
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			version TEXT NOT NULL,
			framework TEXT NOT NULL DEFAULT 'unknown',
			capabilities TEXT[] NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'discovered' CHECK (status IN ('discovered', 'healthy', 'degraded', 'failed', 'quarantined')),
			svid_uri TEXT,
			last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			node_id TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (tenant_id, id)
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			agent_id TEXT,
			status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'awaiting_approval')),
			input_hash TEXT NOT NULL,
			started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMPTZ,
			cost_usd NUMERIC(12, 6) NOT NULL DEFAULT 0,
			tokens_used BIGINT NOT NULL DEFAULT 0,
			approval_id TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS telemetry_spans (
			span_id TEXT NOT NULL,
			trace_id TEXT NOT NULL,
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			agent_id TEXT NOT NULL,
			task_id TEXT,
			operation_name TEXT NOT NULL,
			started_at TIMESTAMPTZ NOT NULL,
			duration_ms BIGINT NOT NULL,
			tier INTEGER NOT NULL DEFAULT 1 CHECK (tier IN (1, 2, 3)),
			attributes JSONB NOT NULL DEFAULT '{}',
			error_code TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (tenant_id, span_id)
		)`,
		`CREATE TABLE IF NOT EXISTS predictive_alerts (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			agent_id TEXT NOT NULL,
			probability NUMERIC(5, 4) NOT NULL,
			estimated_ttf_seconds INTEGER NOT NULL,
			precursor_type TEXT NOT NULL,
			evidence TEXT[] NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'acknowledged', 'resolved', 'false_positive')),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			resolved_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			actor TEXT NOT NULL,
			action TEXT NOT NULL,
			resource TEXT NOT NULL,
			details JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}

	for _, m := range migrations {
		if _, err := pool.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func cleanupDB(ctx context.Context, pool *pgxpool.Pool) {
	tables := []string{"audit_logs", "predictive_alerts", "telemetry_spans", "tasks", "agents", "tenants"}
	for _, table := range tables {
		_, _ = pool.Exec(ctx, "DELETE FROM "+table)
	}
}

// CreateTestTenant inserts a test tenant and returns its UUID.
func CreateTestTenant(t *testing.T, pool *database.Pool, name string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO tenants (display_name, isolation_tier, storage_regions, compliance_profile)
		 VALUES ($1, 'A', ARRAY['us-east-1'], 'eu-gdpr') RETURNING id`,
		name).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}
	return id
}
