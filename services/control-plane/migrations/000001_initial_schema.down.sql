-- Rollback initial schema
DROP POLICY IF EXISTS tenant_isolation_audit ON audit_logs;
DROP POLICY IF EXISTS tenant_isolation_alerts ON predictive_alerts;
DROP POLICY IF EXISTS tenant_isolation_spans ON telemetry_spans;
DROP POLICY IF EXISTS tenant_isolation_tasks ON tasks;
DROP POLICY IF EXISTS tenant_isolation_agents ON agents;

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS predictive_alerts;
DROP TABLE IF EXISTS telemetry_spans;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS tenants;

DROP EXTENSION IF EXISTS "uuid-ossp";
