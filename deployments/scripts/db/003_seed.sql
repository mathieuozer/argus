-- ============================================================================
-- Argus Platform — Development Seed Data
-- Migration: 003_seed.sql
-- Description: Populates the database with realistic dev/test data.
--              This file should only be run in development environments.
-- ============================================================================

BEGIN;

-- --------------------------------------------------------------------------
-- Tenants
-- --------------------------------------------------------------------------
-- We use fixed UUIDs so that foreign keys in the rest of the seed are stable.

INSERT INTO tenants (id, display_name, isolation_tier, storage_regions, pii_scrub, compliance_profile)
VALUES
    ('a0000000-0000-0000-0000-000000000001', 'Acme Corp', 'A',
     ARRAY['eu-west-1', 'eu-central-1'], true, 'eu-gdpr'),
    ('b0000000-0000-0000-0000-000000000002', 'Ministry of Finance', 'C',
     ARRAY['tr-east-1', 'tr-west-1'], true, 'gov-tr');

-- --------------------------------------------------------------------------
-- Agents — 3 per tenant (6 total)
-- --------------------------------------------------------------------------

-- Acme Corp agents
INSERT INTO agents (id, tenant_id, version, framework, capabilities, status, svid_uri, last_seen, node_id)
VALUES
    ('invoice-processor', 'a0000000-0000-0000-0000-000000000001',
     '1.2.0', 'langchain',
     ARRAY['read:invoices', 'write:payments'],
     'healthy',
     'spiffe://argus.acme.com/tenant/a0000000-0000-0000-0000-000000000001/agent/invoice-processor/v1.2.0',
     NOW() - INTERVAL '30 seconds', 'node-eu-west-1a'),

    ('support-chatbot', 'a0000000-0000-0000-0000-000000000001',
     '2.0.1', 'autogen',
     ARRAY['read:knowledge_base', 'write:tickets'],
     'degraded',
     'spiffe://argus.acme.com/tenant/a0000000-0000-0000-0000-000000000001/agent/support-chatbot/v2.0.1',
     NOW() - INTERVAL '5 minutes', 'node-eu-west-1b'),

    ('report-generator', 'a0000000-0000-0000-0000-000000000001',
     '0.9.3', 'crewai',
     ARRAY['read:analytics', 'write:report_store'],
     'discovered',
     'spiffe://argus.acme.com/tenant/a0000000-0000-0000-0000-000000000001/agent/report-generator/v0.9.3',
     NOW() - INTERVAL '1 hour', 'node-eu-central-1a');

-- Ministry of Finance agents
INSERT INTO agents (id, tenant_id, version, framework, capabilities, status, svid_uri, last_seen, node_id)
VALUES
    ('budget-reconciler', 'b0000000-0000-0000-0000-000000000002',
     '3.1.0', 'langchain',
     ARRAY['read:budget_db', 'write:report_store'],
     'healthy',
     'spiffe://argus.gov.tr/tenant/b0000000-0000-0000-0000-000000000002/agent/budget-reconciler/v3.1.0',
     NOW() - INTERVAL '10 seconds', 'node-tr-east-1a'),

    ('compliance-auditor', 'b0000000-0000-0000-0000-000000000002',
     '1.0.0', 'custom',
     ARRAY['read:regulations', 'read:transactions', 'write:audit_findings'],
     'healthy',
     'spiffe://argus.gov.tr/tenant/b0000000-0000-0000-0000-000000000002/agent/compliance-auditor/v1.0.0',
     NOW() - INTERVAL '1 minute', 'node-tr-east-1b'),

    ('procurement-assistant', 'b0000000-0000-0000-0000-000000000002',
     '0.5.0', 'autogen',
     ARRAY['read:vendor_db', 'write:purchase_orders'],
     'quarantined',
     'spiffe://argus.gov.tr/tenant/b0000000-0000-0000-0000-000000000002/agent/procurement-assistant/v0.5.0',
     NOW() - INTERVAL '2 hours', 'node-tr-west-1a');

-- --------------------------------------------------------------------------
-- Tasks — 5 total (mix of tenants and statuses)
-- --------------------------------------------------------------------------

INSERT INTO tasks (id, tenant_id, agent_id, status, input_hash, started_at, completed_at, cost_usd, tokens_used, approval_id)
VALUES
    ('c0000000-0000-0000-0000-000000000001',
     'a0000000-0000-0000-0000-000000000001', 'invoice-processor',
     'completed',
     'sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2',
     NOW() - INTERVAL '10 minutes', NOW() - INTERVAL '8 minutes',
     0.042500, 3200, NULL),

    ('c0000000-0000-0000-0000-000000000002',
     'a0000000-0000-0000-0000-000000000001', 'support-chatbot',
     'running',
     'sha256:b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3',
     NOW() - INTERVAL '2 minutes', NULL,
     0.018000, 1450, NULL),

    ('c0000000-0000-0000-0000-000000000003',
     'a0000000-0000-0000-0000-000000000001', 'invoice-processor',
     'failed',
     'sha256:c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4',
     NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '28 minutes',
     0.005100, 410, NULL),

    ('c0000000-0000-0000-0000-000000000004',
     'b0000000-0000-0000-0000-000000000002', 'budget-reconciler',
     'completed',
     'sha256:d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5',
     NOW() - INTERVAL '1 hour', NOW() - INTERVAL '55 minutes',
     0.125000, 9800, NULL),

    ('c0000000-0000-0000-0000-000000000005',
     'b0000000-0000-0000-0000-000000000002', 'compliance-auditor',
     'awaiting_approval',
     'sha256:e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6',
     NOW() - INTERVAL '15 minutes', NULL,
     0.078000, 6100,
     'f0000000-0000-0000-0000-000000000001');

-- --------------------------------------------------------------------------
-- Telemetry Spans — 10 total
-- --------------------------------------------------------------------------

-- Spans for Acme Corp / invoice-processor / task 1
INSERT INTO telemetry_spans (span_id, trace_id, tenant_id, agent_id, task_id, operation_name, started_at, duration_ms, tier, attributes, error_code)
VALUES
    ('span-001', 'trace-aaa-001',
     'a0000000-0000-0000-0000-000000000001', 'invoice-processor',
     'c0000000-0000-0000-0000-000000000001',
     'llm.chat_completion', NOW() - INTERVAL '10 minutes', 1200, 1,
     '{"model": "gpt-4", "token_count": 1800, "temperature": 0.2}', NULL),

    ('span-002', 'trace-aaa-001',
     'a0000000-0000-0000-0000-000000000001', 'invoice-processor',
     'c0000000-0000-0000-0000-000000000001',
     'tool.read_database', NOW() - INTERVAL '9 minutes', 340, 2,
     '{"table": "invoices", "row_count": 42}', NULL),

    ('span-003', 'trace-aaa-001',
     'a0000000-0000-0000-0000-000000000001', 'invoice-processor',
     'c0000000-0000-0000-0000-000000000001',
     'tool.write_payment', NOW() - INTERVAL '8 minutes 30 seconds', 520, 2,
     '{"payment_id": "PAY-9921", "amount_eur": 4250.00}', NULL),

-- Spans for Acme Corp / support-chatbot / task 2
    ('span-004', 'trace-aaa-002',
     'a0000000-0000-0000-0000-000000000001', 'support-chatbot',
     'c0000000-0000-0000-0000-000000000002',
     'llm.chat_completion', NOW() - INTERVAL '2 minutes', 2800, 1,
     '{"model": "gpt-4", "token_count": 950, "temperature": 0.7}', NULL),

    ('span-005', 'trace-aaa-002',
     'a0000000-0000-0000-0000-000000000001', 'support-chatbot',
     'c0000000-0000-0000-0000-000000000002',
     'tool.search_knowledge_base', NOW() - INTERVAL '1 minute 30 seconds', 450, 2,
     '{"query_terms": 3, "results_found": 7}', NULL),

-- Spans for Acme Corp / invoice-processor / task 3 (failed)
    ('span-006', 'trace-aaa-003',
     'a0000000-0000-0000-0000-000000000001', 'invoice-processor',
     'c0000000-0000-0000-0000-000000000003',
     'llm.chat_completion', NOW() - INTERVAL '30 minutes', 8500, 1,
     '{"model": "gpt-4", "token_count": 410, "temperature": 0.2}', 'CONTEXT_OVERFLOW'),

-- Spans for Ministry of Finance / budget-reconciler / task 4
    ('span-007', 'trace-bbb-001',
     'b0000000-0000-0000-0000-000000000002', 'budget-reconciler',
     'c0000000-0000-0000-0000-000000000004',
     'llm.chat_completion', NOW() - INTERVAL '1 hour', 980, 1,
     '{"model": "gpt-4-turbo", "token_count": 4200, "temperature": 0.1}', NULL),

    ('span-008', 'trace-bbb-001',
     'b0000000-0000-0000-0000-000000000002', 'budget-reconciler',
     'c0000000-0000-0000-0000-000000000004',
     'tool.read_budget_db', NOW() - INTERVAL '58 minutes', 1650, 3,
     '{"query_type": "fiscal_year_summary", "row_count": 1240}', NULL),

    ('span-009', 'trace-bbb-001',
     'b0000000-0000-0000-0000-000000000002', 'budget-reconciler',
     'c0000000-0000-0000-0000-000000000004',
     'tool.write_report', NOW() - INTERVAL '56 minutes', 890, 3,
     '{"report_type": "quarterly_reconciliation", "pages": 24}', NULL),

-- Span for Ministry of Finance / compliance-auditor / task 5
    ('span-010', 'trace-bbb-002',
     'b0000000-0000-0000-0000-000000000002', 'compliance-auditor',
     'c0000000-0000-0000-0000-000000000005',
     'llm.chat_completion', NOW() - INTERVAL '15 minutes', 3200, 1,
     '{"model": "gpt-4-turbo", "token_count": 6100, "temperature": 0.0}', NULL);

-- --------------------------------------------------------------------------
-- Predictive Alerts — 2 total
-- --------------------------------------------------------------------------

INSERT INTO predictive_alerts (id, tenant_id, agent_id, probability, estimated_ttf_seconds, precursor_type, evidence, status)
VALUES
    ('d0000000-0000-0000-0000-000000000001',
     'a0000000-0000-0000-0000-000000000001', 'support-chatbot',
     0.8200, 180, 'token_escalation',
     ARRAY[
         'token_velocity increased 3.2x over last 10 tasks',
         'context_fill_pct at 87% and rising',
         'consecutive_slow count = 4'
     ],
     'open'),

    ('d0000000-0000-0000-0000-000000000002',
     'b0000000-0000-0000-0000-000000000002', 'procurement-assistant',
     0.9100, 60, 'retry_storm',
     ARRAY[
         'retry_rate jumped from 0.02 to 0.45 in 5 minutes',
         'error_rate_delta = +0.38 vs 1h baseline',
         'latency_p99_ratio = 5.2 (threshold: 3.0)'
     ],
     'acknowledged');

-- --------------------------------------------------------------------------
-- Audit Log — 5 entries
-- --------------------------------------------------------------------------

INSERT INTO audit_log (tenant_id, subject, action, resource, details)
VALUES
    ('a0000000-0000-0000-0000-000000000001',
     'admin@acme.com', 'tenant.created', 'tenant/a0000000-0000-0000-0000-000000000001',
     'Initial tenant provisioning for Acme Corp with eu-gdpr compliance profile'),

    ('a0000000-0000-0000-0000-000000000001',
     'system:sidecar', 'agent.registered', 'agent/invoice-processor',
     'Auto-discovered agent invoice-processor v1.2.0 on node-eu-west-1a via sidecar'),

    ('a0000000-0000-0000-0000-000000000001',
     'system:predictor', 'alert.created', 'alert/d0000000-0000-0000-0000-000000000001',
     'Predictive alert fired for support-chatbot: token_escalation (p=0.82, ttf=180s)'),

    ('b0000000-0000-0000-0000-000000000002',
     'admin@maliye.gov.tr', 'tenant.created', 'tenant/b0000000-0000-0000-0000-000000000002',
     'Initial tenant provisioning for Ministry of Finance with gov-tr compliance profile'),

    ('b0000000-0000-0000-0000-000000000002',
     'security@maliye.gov.tr', 'agent.quarantined', 'agent/procurement-assistant',
     'Agent procurement-assistant quarantined due to retry storm detection; SVID revoked');

-- --------------------------------------------------------------------------
-- Migration tracking
-- --------------------------------------------------------------------------
INSERT INTO schema_migrations (version) VALUES ('003_seed');

COMMIT;
