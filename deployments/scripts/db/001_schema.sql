-- ============================================================================
-- Argus Platform — Core Database Schema
-- Migration: 001_schema.sql
-- Description: Creates all core tables, constraints, and indexes.
-- ============================================================================

BEGIN;

-- --------------------------------------------------------------------------
-- Extensions
-- --------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- gen_random_uuid()

-- --------------------------------------------------------------------------
-- Tenants — top-level isolation boundary
-- --------------------------------------------------------------------------
CREATE TABLE tenants (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name        TEXT        NOT NULL,
    isolation_tier      TEXT        NOT NULL CHECK (isolation_tier IN ('A', 'B', 'C')),
    storage_regions     TEXT[]      NOT NULL DEFAULT '{}',
    pii_scrub           BOOLEAN     NOT NULL DEFAULT false,
    compliance_profile  TEXT        NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_isolation_tier ON tenants (isolation_tier);
CREATE INDEX idx_tenants_compliance_profile ON tenants (compliance_profile);

-- --------------------------------------------------------------------------
-- Agents — registered agent instances, scoped to a tenant
-- --------------------------------------------------------------------------
CREATE TABLE agents (
    id              TEXT        NOT NULL,
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    version         TEXT        NOT NULL DEFAULT '',
    framework       TEXT        NOT NULL DEFAULT '',
    capabilities    TEXT[]      NOT NULL DEFAULT '{}',
    status          TEXT        NOT NULL DEFAULT 'discovered'
                        CHECK (status IN ('discovered', 'healthy', 'degraded', 'failed', 'quarantined')),
    svid_uri        TEXT        NOT NULL DEFAULT '',
    last_seen       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    node_id         TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX idx_agents_tenant_id ON agents (tenant_id);
CREATE INDEX idx_agents_status ON agents (tenant_id, status);
CREATE INDEX idx_agents_framework ON agents (tenant_id, framework);
CREATE INDEX idx_agents_last_seen ON agents (tenant_id, last_seen DESC);
CREATE INDEX idx_agents_node_id ON agents (tenant_id, node_id);

-- --------------------------------------------------------------------------
-- Tasks — units of work routed through the orchestrator
-- --------------------------------------------------------------------------
CREATE TABLE tasks (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agent_id        TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'running', 'completed', 'failed', 'awaiting_approval')),
    input_hash      TEXT        NOT NULL DEFAULT '',
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    cost_usd        DECIMAL(12,6) NOT NULL DEFAULT 0,
    tokens_used     BIGINT      NOT NULL DEFAULT 0,
    approval_id     UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_tenant_id ON tasks (tenant_id);
CREATE INDEX idx_tasks_agent_id ON tasks (tenant_id, agent_id);
CREATE INDEX idx_tasks_status ON tasks (tenant_id, status);
CREATE INDEX idx_tasks_started_at ON tasks (tenant_id, started_at DESC);
CREATE INDEX idx_tasks_created_at ON tasks (tenant_id, created_at DESC);
CREATE INDEX idx_tasks_approval_id ON tasks (approval_id) WHERE approval_id IS NOT NULL;

-- Foreign key to agents (composite key)
ALTER TABLE tasks
    ADD CONSTRAINT fk_tasks_agent
    FOREIGN KEY (tenant_id, agent_id) REFERENCES agents(tenant_id, id);

-- --------------------------------------------------------------------------
-- Telemetry Spans — traced operations from agents
-- --------------------------------------------------------------------------
CREATE TABLE telemetry_spans (
    span_id         TEXT        NOT NULL,
    trace_id        TEXT        NOT NULL,
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agent_id        TEXT        NOT NULL,
    task_id         UUID,
    operation_name  TEXT        NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL,
    duration_ms     BIGINT      NOT NULL DEFAULT 0,
    tier            INT         NOT NULL DEFAULT 1
                        CHECK (tier IN (1, 2, 3)),
    attributes      JSONB       NOT NULL DEFAULT '{}',
    error_code      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, span_id)
);

CREATE INDEX idx_telemetry_spans_tenant_id ON telemetry_spans (tenant_id);
CREATE INDEX idx_telemetry_spans_trace_id ON telemetry_spans (tenant_id, trace_id);
CREATE INDEX idx_telemetry_spans_agent_id ON telemetry_spans (tenant_id, agent_id);
CREATE INDEX idx_telemetry_spans_task_id ON telemetry_spans (tenant_id, task_id) WHERE task_id IS NOT NULL;
CREATE INDEX idx_telemetry_spans_operation ON telemetry_spans (tenant_id, operation_name);
CREATE INDEX idx_telemetry_spans_started_at ON telemetry_spans (tenant_id, started_at DESC);
CREATE INDEX idx_telemetry_spans_tier ON telemetry_spans (tenant_id, tier);
CREATE INDEX idx_telemetry_spans_error_code ON telemetry_spans (tenant_id, error_code) WHERE error_code IS NOT NULL;
CREATE INDEX idx_telemetry_spans_attributes ON telemetry_spans USING GIN (attributes);

-- --------------------------------------------------------------------------
-- Predictive Alerts — fired before an agent fails
-- --------------------------------------------------------------------------
CREATE TABLE predictive_alerts (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agent_id                TEXT        NOT NULL,
    probability             DECIMAL(5,4) NOT NULL,
    estimated_ttf_seconds   INT         NOT NULL DEFAULT 0,
    precursor_type          TEXT        NOT NULL,
    evidence                TEXT[]      NOT NULL DEFAULT '{}',
    status                  TEXT        NOT NULL DEFAULT 'open'
                                CHECK (status IN ('open', 'acknowledged', 'resolved', 'false_positive')),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_predictive_alerts_tenant_id ON predictive_alerts (tenant_id);
CREATE INDEX idx_predictive_alerts_agent_id ON predictive_alerts (tenant_id, agent_id);
CREATE INDEX idx_predictive_alerts_status ON predictive_alerts (tenant_id, status);
CREATE INDEX idx_predictive_alerts_precursor ON predictive_alerts (tenant_id, precursor_type);
CREATE INDEX idx_predictive_alerts_created_at ON predictive_alerts (tenant_id, created_at DESC);
CREATE INDEX idx_predictive_alerts_probability ON predictive_alerts (tenant_id, probability DESC);

-- --------------------------------------------------------------------------
-- Audit Log — immutable record of all platform actions
-- No UPDATE or DELETE should ever be performed on this table.
-- --------------------------------------------------------------------------
CREATE TABLE audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    subject     TEXT        NOT NULL,
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    details     TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_tenant_id ON audit_log (tenant_id);
CREATE INDEX idx_audit_log_subject ON audit_log (tenant_id, subject);
CREATE INDEX idx_audit_log_action ON audit_log (tenant_id, action);
CREATE INDEX idx_audit_log_resource ON audit_log (tenant_id, resource);
CREATE INDEX idx_audit_log_created_at ON audit_log (tenant_id, created_at DESC);

-- --------------------------------------------------------------------------
-- Schema migration tracking
-- --------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     TEXT        PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO schema_migrations (version) VALUES ('001_schema');

COMMIT;
