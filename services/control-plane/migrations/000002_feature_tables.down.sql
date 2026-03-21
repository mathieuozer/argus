-- Reverse migration: drop all feature tables added in 000002

DROP TABLE IF EXISTS compliance_reports CASCADE;
DROP TABLE IF EXISTS rag_sources CASCADE;
DROP TABLE IF EXISTS rag_retrievals CASCADE;
DROP TABLE IF EXISTS prompt_versions CASCADE;
DROP TABLE IF EXISTS prompts CASCADE;
DROP TABLE IF EXISTS guardrail_violations CASCADE;
DROP TABLE IF EXISTS guardrail_rules CASCADE;
DROP TABLE IF EXISTS feedback CASCADE;
DROP TABLE IF EXISTS eval_runs CASCADE;
DROP TABLE IF EXISTS eval_suites CASCADE;
DROP TABLE IF EXISTS trace_spans CASCADE;
DROP TABLE IF EXISTS slo_measurements CASCADE;
DROP TABLE IF EXISTS slos CASCADE;
DROP TABLE IF EXISTS dq_violations CASCADE;
DROP TABLE IF EXISTS dq_scores CASCADE;
DROP TABLE IF EXISTS dq_rules CASCADE;
DROP TABLE IF EXISTS cost_budgets CASCADE;
DROP TABLE IF EXISTS cost_entries CASCADE;
DROP TABLE IF EXISTS catalog_lineage_edges CASCADE;
DROP TABLE IF EXISTS catalog_sources CASCADE;

-- Remove parent_span_id from telemetry_spans
ALTER TABLE telemetry_spans DROP COLUMN IF EXISTS parent_span_id;
