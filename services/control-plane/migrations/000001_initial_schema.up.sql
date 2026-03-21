-- Argus initial schema
-- All tables have tenant_id with Row-Level Security policies

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    display_name TEXT NOT NULL,
    isolation_tier TEXT NOT NULL CHECK (isolation_tier IN ('A', 'B', 'C')),
    storage_regions TEXT[] NOT NULL DEFAULT '{}',
    pii_scrub BOOLEAN NOT NULL DEFAULT true,
    compliance_profile TEXT NOT NULL DEFAULT 'eu-gdpr',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Agents table
CREATE TABLE agents (
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
);

-- Tasks table
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
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
);

CREATE INDEX idx_tasks_tenant_id ON tasks(tenant_id);
CREATE INDEX idx_tasks_agent_id ON tasks(tenant_id, agent_id);
CREATE INDEX idx_tasks_status ON tasks(tenant_id, status);

-- Telemetry spans table
CREATE TABLE telemetry_spans (
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
);

CREATE INDEX idx_spans_trace ON telemetry_spans(tenant_id, trace_id);
CREATE INDEX idx_spans_agent ON telemetry_spans(tenant_id, agent_id);
CREATE INDEX idx_spans_time ON telemetry_spans(tenant_id, started_at);

-- Predictive alerts table
CREATE TABLE predictive_alerts (
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
);

CREATE INDEX idx_alerts_tenant ON predictive_alerts(tenant_id);
CREATE INDEX idx_alerts_status ON predictive_alerts(tenant_id, status);

-- Audit logs table (append-only)
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_time ON audit_logs(tenant_id, created_at);

-- Row-Level Security policies
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE telemetry_spans ENABLE ROW LEVEL SECURITY;
ALTER TABLE predictive_alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- RLS policies: enforce tenant isolation
CREATE POLICY tenant_isolation_agents ON agents
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_tasks ON tasks
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_spans ON telemetry_spans
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_alerts ON predictive_alerts
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_audit ON audit_logs
    USING (tenant_id = current_setting('app.tenant_id')::UUID);
