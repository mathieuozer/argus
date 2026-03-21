-- Observability tables for Argus Phase 3+4 features
-- All tables include tenant_id with RLS policies

-- Data Quality Rules
CREATE TABLE IF NOT EXISTS dq_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL, -- completeness, accuracy, consistency, timeliness
    expression TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning', -- info, warning, critical
    agent_id VARCHAR(255),
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Data Quality Scores
CREATE TABLE IF NOT EXISTS dq_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id VARCHAR(255) NOT NULL,
    rule_id UUID REFERENCES dq_rules(id),
    score DECIMAL(5,4) NOT NULL, -- 0.0000 to 1.0000
    dimension VARCHAR(50) NOT NULL,
    sample_size INTEGER NOT NULL DEFAULT 0,
    measured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Data Catalog Entries
CREATE TABLE IF NOT EXISTS catalog_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    source_type VARCHAR(50) NOT NULL, -- api, database, file, model
    agent_id VARCHAR(255),
    schema_json JSONB,
    tags TEXT[],
    access_count INTEGER NOT NULL DEFAULT 0,
    last_accessed_at TIMESTAMPTZ,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cost Records
CREATE TABLE IF NOT EXISTS cost_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(255),
    cost_usd DECIMAL(12,6) NOT NULL,
    tokens_input INTEGER NOT NULL DEFAULT 0,
    tokens_output INTEGER NOT NULL DEFAULT 0,
    model VARCHAR(100),
    provider VARCHAR(50),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cost Budgets
CREATE TABLE IF NOT EXISTS cost_budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    budget_usd DECIMAL(12,2) NOT NULL,
    spent_usd DECIMAL(12,2) NOT NULL DEFAULT 0,
    period VARCHAR(20) NOT NULL DEFAULT 'monthly', -- daily, weekly, monthly
    alert_threshold DECIMAL(5,2) NOT NULL DEFAULT 80.00, -- percentage
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- SLO Definitions
CREATE TABLE IF NOT EXISTS slos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    agent_id VARCHAR(255),
    metric VARCHAR(100) NOT NULL, -- latency_p99, error_rate, availability
    target_value DECIMAL(10,4) NOT NULL,
    comparator VARCHAR(5) NOT NULL DEFAULT '<=', -- <=, >=, <, >
    window_days INTEGER NOT NULL DEFAULT 30,
    error_budget DECIMAL(10,6) NOT NULL DEFAULT 0.001, -- 0.1%
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- SLO Measurements
CREATE TABLE IF NOT EXISTS slo_measurements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    slo_id UUID NOT NULL REFERENCES slos(id),
    measured_value DECIMAL(10,4) NOT NULL,
    target_value DECIMAL(10,4) NOT NULL,
    budget_remaining DECIMAL(10,6) NOT NULL,
    compliant BOOLEAN NOT NULL,
    measured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Evaluation Test Suites
CREATE TABLE IF NOT EXISTS eval_suites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    agent_id VARCHAR(255) NOT NULL,
    test_cases JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Evaluation Runs
CREATE TABLE IF NOT EXISTS eval_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    suite_id UUID NOT NULL REFERENCES eval_suites(id),
    agent_id VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    score DECIMAL(5,4),
    total_cases INTEGER NOT NULL DEFAULT 0,
    passed_cases INTEGER NOT NULL DEFAULT 0,
    failed_cases INTEGER NOT NULL DEFAULT 0,
    results JSONB,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Guardrail Rules
CREATE TABLE IF NOT EXISTS guardrail_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL,
    pattern TEXT,
    action VARCHAR(20) NOT NULL DEFAULT 'warn',
    enabled BOOLEAN NOT NULL DEFAULT true,
    agent_ids TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Guardrail Violations
CREATE TABLE IF NOT EXISTS guardrail_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    rule_id UUID NOT NULL REFERENCES guardrail_rules(id),
    agent_id VARCHAR(255) NOT NULL,
    span_id VARCHAR(255),
    action VARCHAR(20) NOT NULL,
    content_excerpt TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prompt Templates
CREATE TABLE IF NOT EXISTS prompt_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    agent_id VARCHAR(255),
    active_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prompt Versions
CREATE TABLE IF NOT EXISTS prompt_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    prompt_id UUID NOT NULL REFERENCES prompt_templates(id),
    version INTEGER NOT NULL,
    content TEXT NOT NULL,
    change_log TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(prompt_id, version)
);

-- RAG Retrieval Events
CREATE TABLE IF NOT EXISTS rag_retrievals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id VARCHAR(255) NOT NULL,
    span_id VARCHAR(255),
    query TEXT NOT NULL,
    num_chunks INTEGER NOT NULL DEFAULT 0,
    avg_relevance DECIMAL(5,4),
    latency_ms INTEGER,
    source_ids TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RAG Sources
CREATE TABLE IF NOT EXISTS rag_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    source_type VARCHAR(50) NOT NULL,
    total_chunks INTEGER NOT NULL DEFAULT 0,
    avg_relevance DECIMAL(5,4),
    usage_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Human Feedback
CREATE TABLE IF NOT EXISTS feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    agent_id VARCHAR(255) NOT NULL,
    span_id VARCHAR(255),
    task_id VARCHAR(255),
    rating INTEGER NOT NULL, -- -1 (thumbs down) or 1 (thumbs up), or 1-5
    comment TEXT,
    user_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Compliance Reports
CREATE TABLE IF NOT EXISTS compliance_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    profile_id VARCHAR(50) NOT NULL,
    profile_name VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'generating',
    format VARCHAR(10) NOT NULL DEFAULT 'json',
    sections JSONB,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Data Residency Attestations
CREATE TABLE IF NOT EXISTS residency_attestations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    node_id VARCHAR(255) NOT NULL,
    region VARCHAR(50) NOT NULL,
    data_hash VARCHAR(64) NOT NULL,
    signature VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_dq_rules_tenant ON dq_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dq_scores_tenant_agent ON dq_scores(tenant_id, agent_id);
CREATE INDEX IF NOT EXISTS idx_catalog_entries_tenant ON catalog_entries(tenant_id);
CREATE INDEX IF NOT EXISTS idx_cost_records_tenant_agent ON cost_records(tenant_id, agent_id);
CREATE INDEX IF NOT EXISTS idx_cost_records_recorded ON cost_records(recorded_at);
CREATE INDEX IF NOT EXISTS idx_cost_budgets_tenant ON cost_budgets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_slos_tenant ON slos(tenant_id);
CREATE INDEX IF NOT EXISTS idx_slo_measurements_slo ON slo_measurements(slo_id);
CREATE INDEX IF NOT EXISTS idx_eval_suites_tenant ON eval_suites(tenant_id);
CREATE INDEX IF NOT EXISTS idx_eval_runs_suite ON eval_runs(suite_id);
CREATE INDEX IF NOT EXISTS idx_guardrail_rules_tenant ON guardrail_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_guardrail_violations_tenant ON guardrail_violations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_tenant ON prompt_templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_prompt_versions_prompt ON prompt_versions(prompt_id);
CREATE INDEX IF NOT EXISTS idx_rag_retrievals_tenant ON rag_retrievals(tenant_id, agent_id);
CREATE INDEX IF NOT EXISTS idx_rag_sources_tenant ON rag_sources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_feedback_tenant_agent ON feedback(tenant_id, agent_id);
CREATE INDEX IF NOT EXISTS idx_compliance_reports_tenant ON compliance_reports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_residency_attestations_tenant ON residency_attestations(tenant_id);

-- RLS Policies for all new tables
ALTER TABLE dq_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE dq_scores ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE cost_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE cost_budgets ENABLE ROW LEVEL SECURITY;
ALTER TABLE slos ENABLE ROW LEVEL SECURITY;
ALTER TABLE slo_measurements ENABLE ROW LEVEL SECURITY;
ALTER TABLE eval_suites ENABLE ROW LEVEL SECURITY;
ALTER TABLE eval_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE guardrail_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE guardrail_violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE prompt_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE prompt_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE rag_retrievals ENABLE ROW LEVEL SECURITY;
ALTER TABLE rag_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE feedback ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE residency_attestations ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_dq_rules ON dq_rules USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_dq_scores ON dq_scores USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_catalog ON catalog_entries USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_cost_records ON cost_records USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_cost_budgets ON cost_budgets USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_slos ON slos USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_slo_measurements ON slo_measurements USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_eval_suites ON eval_suites USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_eval_runs ON eval_runs USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_guardrail_rules ON guardrail_rules USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_guardrail_violations ON guardrail_violations USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_prompt_templates ON prompt_templates USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_prompt_versions ON prompt_versions USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_rag_retrievals ON rag_retrievals USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_rag_sources ON rag_sources USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_feedback ON feedback USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_compliance_reports ON compliance_reports USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_residency_attestations ON residency_attestations USING (tenant_id::text = current_setting('app.tenant_id', true));
