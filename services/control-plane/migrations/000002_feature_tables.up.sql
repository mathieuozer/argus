-- Feature tables migration: adds all tables required by control-plane PG repositories.
-- Every table includes tenant_id with Row-Level Security policies.

-- Fix: add parent_span_id to telemetry_spans (needed by trace PGService)
ALTER TABLE telemetry_spans ADD COLUMN IF NOT EXISTS parent_span_id TEXT NOT NULL DEFAULT '';

-- ============================================================
-- Catalog: data sources and lineage
-- ============================================================
CREATE TABLE IF NOT EXISTS catalog_sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'unknown',
    owner TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    schema JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_catalog_sources_tenant ON catalog_sources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_catalog_sources_type ON catalog_sources(tenant_id, type);

CREATE TABLE IF NOT EXISTS catalog_lineage_edges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    transform_type TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lineage_edges_tenant ON catalog_lineage_edges(tenant_id);
CREATE INDEX IF NOT EXISTS idx_lineage_edges_source ON catalog_lineage_edges(tenant_id, source_id);
CREATE INDEX IF NOT EXISTS idx_lineage_edges_target ON catalog_lineage_edges(tenant_id, target_id);

-- ============================================================
-- Cost governance: entries and budgets
-- ============================================================
CREATE TABLE IF NOT EXISTS cost_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id TEXT NOT NULL DEFAULT '',
    task_id TEXT NOT NULL DEFAULT '',
    cost_usd NUMERIC(12, 6) NOT NULL DEFAULT 0,
    tokens_used BIGINT NOT NULL DEFAULT 0,
    model TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cost_entries_tenant ON cost_entries(tenant_id);
CREATE INDEX IF NOT EXISTS idx_cost_entries_agent ON cost_entries(tenant_id, agent_id);
CREATE INDEX IF NOT EXISTS idx_cost_entries_time ON cost_entries(tenant_id, timestamp);

CREATE TABLE IF NOT EXISTS cost_budgets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    limit_usd NUMERIC(12, 6) NOT NULL DEFAULT 0,
    period_type TEXT NOT NULL DEFAULT 'monthly',
    current_spend NUMERIC(12, 6) NOT NULL DEFAULT 0,
    alert_threshold NUMERIC(5, 4) NOT NULL DEFAULT 0.8,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cost_budgets_tenant ON cost_budgets(tenant_id);

-- ============================================================
-- Data quality: rules, scores, violations
-- ============================================================
CREATE TABLE IF NOT EXISTS dq_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    field TEXT NOT NULL DEFAULT '',
    operator TEXT NOT NULL DEFAULT '',
    threshold TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL DEFAULT 'warning',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dq_rules_tenant ON dq_rules(tenant_id);

CREATE TABLE IF NOT EXISTS dq_scores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id TEXT NOT NULL DEFAULT '',
    overall_score NUMERIC(5, 4) NOT NULL DEFAULT 0,
    completeness_score NUMERIC(5, 4) NOT NULL DEFAULT 0,
    accuracy_score NUMERIC(5, 4) NOT NULL DEFAULT 0,
    consistency_score NUMERIC(5, 4) NOT NULL DEFAULT 0,
    timeliness_score NUMERIC(5, 4) NOT NULL DEFAULT 0,
    total_checks INTEGER NOT NULL DEFAULT 0,
    passed_checks INTEGER NOT NULL DEFAULT 0,
    failed_checks INTEGER NOT NULL DEFAULT 0,
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dq_scores_tenant ON dq_scores(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dq_scores_agent ON dq_scores(tenant_id, agent_id);

CREATE TABLE IF NOT EXISTS dq_violations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    rule_id TEXT NOT NULL DEFAULT '',
    rule_name TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    field TEXT NOT NULL DEFAULT '',
    value TEXT NOT NULL DEFAULT '',
    expected TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL DEFAULT 'warning',
    message TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dq_violations_tenant ON dq_violations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dq_violations_agent ON dq_violations(tenant_id, agent_id);

-- ============================================================
-- SLOs: objectives and measurements
-- ============================================================
CREATE TABLE IF NOT EXISTS slos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '',
    target NUMERIC(7, 4) NOT NULL DEFAULT 0,
    window TEXT NOT NULL DEFAULT '30d',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_slos_tenant ON slos(tenant_id);

CREATE TABLE IF NOT EXISTS slo_measurements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    slo_id TEXT NOT NULL,
    agent_id TEXT NOT NULL DEFAULT '',
    value NUMERIC(7, 4) NOT NULL DEFAULT 0,
    good BIGINT NOT NULL DEFAULT 0,
    total BIGINT NOT NULL DEFAULT 0,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_slo_measurements_tenant ON slo_measurements(tenant_id);
CREATE INDEX IF NOT EXISTS idx_slo_measurements_slo ON slo_measurements(tenant_id, slo_id);

-- ============================================================
-- Trace: extended span storage (uses existing telemetry_spans with parent_span_id)
-- Also create a dedicated trace_spans table for the control-plane trace service
-- ============================================================
CREATE TABLE IF NOT EXISTS trace_spans (
    span_id TEXT NOT NULL,
    trace_id TEXT NOT NULL,
    parent_span_id TEXT NOT NULL DEFAULT '',
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id TEXT NOT NULL DEFAULT '',
    task_id TEXT NOT NULL DEFAULT '',
    operation_name TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    attributes JSONB NOT NULL DEFAULT '{}',
    error_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, span_id)
);

CREATE INDEX IF NOT EXISTS idx_trace_spans_trace ON trace_spans(tenant_id, trace_id);
CREATE INDEX IF NOT EXISTS idx_trace_spans_agent ON trace_spans(tenant_id, agent_id);
CREATE INDEX IF NOT EXISTS idx_trace_spans_time ON trace_spans(tenant_id, started_at);

-- ============================================================
-- Evaluations: test suites and runs
-- ============================================================
CREATE TABLE IF NOT EXISTS eval_suites (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    test_cases JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_eval_suites_tenant ON eval_suites(tenant_id);

CREATE TABLE IF NOT EXISTS eval_runs (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    suite_id TEXT NOT NULL DEFAULT '',
    suite_name TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    score NUMERIC(5, 4) NOT NULL DEFAULT 0,
    total_cases INTEGER NOT NULL DEFAULT 0,
    passed_cases INTEGER NOT NULL DEFAULT 0,
    failed_cases INTEGER NOT NULL DEFAULT 0,
    results JSONB NOT NULL DEFAULT '[]',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_eval_runs_tenant ON eval_runs(tenant_id);

-- ============================================================
-- Feedback: human ratings on agent outputs
-- ============================================================
CREATE TABLE IF NOT EXISTS feedback (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id TEXT NOT NULL DEFAULT '',
    span_id TEXT NOT NULL DEFAULT '',
    task_id TEXT NOT NULL DEFAULT '',
    rating INTEGER NOT NULL DEFAULT 0,
    comment TEXT NOT NULL DEFAULT '',
    user_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_feedback_tenant ON feedback(tenant_id);
CREATE INDEX IF NOT EXISTS idx_feedback_agent ON feedback(tenant_id, agent_id);

-- ============================================================
-- Guardrails: rules and violations
-- ============================================================
CREATE TABLE IF NOT EXISTS guardrail_rules (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '',
    pattern TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL DEFAULT 'warn',
    enabled BOOLEAN NOT NULL DEFAULT true,
    agent_ids TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_guardrail_rules_tenant ON guardrail_rules(tenant_id);

CREATE TABLE IF NOT EXISTS guardrail_violations (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    rule_id TEXT NOT NULL DEFAULT '',
    rule_name TEXT NOT NULL DEFAULT '',
    rule_type TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    span_id TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_guardrail_violations_tenant ON guardrail_violations(tenant_id);

-- ============================================================
-- Prompts: versioned prompt management
-- ============================================================
CREATE TABLE IF NOT EXISTS prompts (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    active_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_prompts_tenant ON prompts(tenant_id);

CREATE TABLE IF NOT EXISTS prompt_versions (
    id TEXT NOT NULL,
    prompt_id TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    version INTEGER NOT NULL DEFAULT 1,
    content TEXT NOT NULL DEFAULT '',
    change_log TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_prompt_versions_tenant ON prompt_versions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_prompt_versions_prompt ON prompt_versions(tenant_id, prompt_id);

-- ============================================================
-- RAG: retrieval monitoring and sources
-- ============================================================
CREATE TABLE IF NOT EXISTS rag_retrievals (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id TEXT NOT NULL DEFAULT '',
    span_id TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL DEFAULT '',
    num_chunks INTEGER NOT NULL DEFAULT 0,
    avg_relevance NUMERIC(5, 4) NOT NULL DEFAULT 0,
    latency_ms BIGINT NOT NULL DEFAULT 0,
    source_ids TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_rag_retrievals_tenant ON rag_retrievals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rag_retrievals_agent ON rag_retrievals(tenant_id, agent_id);

CREATE TABLE IF NOT EXISTS rag_sources (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '',
    total_chunks INTEGER NOT NULL DEFAULT 0,
    avg_relevance NUMERIC(5, 4) NOT NULL DEFAULT 0,
    usage_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_rag_sources_tenant ON rag_sources(tenant_id);

-- ============================================================
-- Compliance: report storage
-- ============================================================
CREATE TABLE IF NOT EXISTS compliance_reports (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    profile_id TEXT NOT NULL DEFAULT '',
    profile_name TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    format TEXT NOT NULL DEFAULT 'json',
    sections JSONB NOT NULL DEFAULT '[]',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period_end TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_compliance_reports_tenant ON compliance_reports(tenant_id);

-- ============================================================
-- Row-Level Security for all new tables
-- ============================================================
ALTER TABLE catalog_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_lineage_edges ENABLE ROW LEVEL SECURITY;
ALTER TABLE cost_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE cost_budgets ENABLE ROW LEVEL SECURITY;
ALTER TABLE dq_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE dq_scores ENABLE ROW LEVEL SECURITY;
ALTER TABLE dq_violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE slos ENABLE ROW LEVEL SECURITY;
ALTER TABLE slo_measurements ENABLE ROW LEVEL SECURITY;
ALTER TABLE trace_spans ENABLE ROW LEVEL SECURITY;
ALTER TABLE eval_suites ENABLE ROW LEVEL SECURITY;
ALTER TABLE eval_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE feedback ENABLE ROW LEVEL SECURITY;
ALTER TABLE guardrail_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE guardrail_violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE prompts ENABLE ROW LEVEL SECURITY;
ALTER TABLE prompt_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE rag_retrievals ENABLE ROW LEVEL SECURITY;
ALTER TABLE rag_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_reports ENABLE ROW LEVEL SECURITY;

-- RLS policies: enforce tenant isolation
CREATE POLICY tenant_isolation_catalog_sources ON catalog_sources
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_catalog_lineage ON catalog_lineage_edges
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_cost_entries ON cost_entries
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_cost_budgets ON cost_budgets
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_dq_rules ON dq_rules
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_dq_scores ON dq_scores
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_dq_violations ON dq_violations
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_slos ON slos
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_slo_measurements ON slo_measurements
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_trace_spans ON trace_spans
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_eval_suites ON eval_suites
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_eval_runs ON eval_runs
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_feedback ON feedback
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_guardrail_rules ON guardrail_rules
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_guardrail_violations ON guardrail_violations
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_prompts ON prompts
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_prompt_versions ON prompt_versions
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_rag_retrievals ON rag_retrievals
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_rag_sources ON rag_sources
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation_compliance_reports ON compliance_reports
    USING (tenant_id = current_setting('app.tenant_id')::UUID);
