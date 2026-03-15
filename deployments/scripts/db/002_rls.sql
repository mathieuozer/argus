-- ============================================================================
-- Argus Platform — Row-Level Security Policies
-- Migration: 002_rls.sql
-- Description: Enforces multi-tenant isolation at the database layer.
--              Every query must set `app.tenant_id` via:
--                SET LOCAL app.tenant_id = '<tenant-uuid>';
-- ============================================================================

BEGIN;

-- --------------------------------------------------------------------------
-- Helper function: returns the current tenant ID from session settings.
-- Every RLS policy references this function.
-- --------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION current_tenant_id()
RETURNS UUID
LANGUAGE sql
STABLE
AS $$
    SELECT NULLIF(current_setting('app.tenant_id', true), '')::UUID;
$$;

-- ============================================================================
-- Enable RLS on ALL tables
-- ============================================================================
ALTER TABLE tenants           ENABLE ROW LEVEL SECURITY;
ALTER TABLE agents            ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks             ENABLE ROW LEVEL SECURITY;
ALTER TABLE telemetry_spans   ENABLE ROW LEVEL SECURITY;
ALTER TABLE predictive_alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log         ENABLE ROW LEVEL SECURITY;

-- Force RLS even for table owners (prevents accidental bypasses in app code).
ALTER TABLE tenants           FORCE ROW LEVEL SECURITY;
ALTER TABLE agents            FORCE ROW LEVEL SECURITY;
ALTER TABLE tasks             FORCE ROW LEVEL SECURITY;
ALTER TABLE telemetry_spans   FORCE ROW LEVEL SECURITY;
ALTER TABLE predictive_alerts FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_log         FORCE ROW LEVEL SECURITY;

-- ============================================================================
-- Tenants — a tenant can only see its own row
-- ============================================================================
CREATE POLICY tenants_select ON tenants
    FOR SELECT
    USING (id = current_tenant_id());

CREATE POLICY tenants_insert ON tenants
    FOR INSERT
    WITH CHECK (id = current_tenant_id());

CREATE POLICY tenants_update ON tenants
    FOR UPDATE
    USING (id = current_tenant_id())
    WITH CHECK (id = current_tenant_id());

CREATE POLICY tenants_delete ON tenants
    FOR DELETE
    USING (id = current_tenant_id());

-- ============================================================================
-- Agents — scoped to tenant_id
-- ============================================================================
CREATE POLICY agents_select ON agents
    FOR SELECT
    USING (tenant_id = current_tenant_id());

CREATE POLICY agents_insert ON agents
    FOR INSERT
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY agents_update ON agents
    FOR UPDATE
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY agents_delete ON agents
    FOR DELETE
    USING (tenant_id = current_tenant_id());

-- ============================================================================
-- Tasks — scoped to tenant_id
-- ============================================================================
CREATE POLICY tasks_select ON tasks
    FOR SELECT
    USING (tenant_id = current_tenant_id());

CREATE POLICY tasks_insert ON tasks
    FOR INSERT
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY tasks_update ON tasks
    FOR UPDATE
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY tasks_delete ON tasks
    FOR DELETE
    USING (tenant_id = current_tenant_id());

-- ============================================================================
-- Telemetry Spans — scoped to tenant_id
-- ============================================================================
CREATE POLICY telemetry_spans_select ON telemetry_spans
    FOR SELECT
    USING (tenant_id = current_tenant_id());

CREATE POLICY telemetry_spans_insert ON telemetry_spans
    FOR INSERT
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY telemetry_spans_update ON telemetry_spans
    FOR UPDATE
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY telemetry_spans_delete ON telemetry_spans
    FOR DELETE
    USING (tenant_id = current_tenant_id());

-- ============================================================================
-- Predictive Alerts — scoped to tenant_id
-- ============================================================================
CREATE POLICY predictive_alerts_select ON predictive_alerts
    FOR SELECT
    USING (tenant_id = current_tenant_id());

CREATE POLICY predictive_alerts_insert ON predictive_alerts
    FOR INSERT
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY predictive_alerts_update ON predictive_alerts
    FOR UPDATE
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY predictive_alerts_delete ON predictive_alerts
    FOR DELETE
    USING (tenant_id = current_tenant_id());

-- ============================================================================
-- Audit Log — IMMUTABLE: SELECT and INSERT only, UPDATE and DELETE denied.
-- ============================================================================
CREATE POLICY audit_log_select ON audit_log
    FOR SELECT
    USING (tenant_id = current_tenant_id());

CREATE POLICY audit_log_insert ON audit_log
    FOR INSERT
    WITH CHECK (tenant_id = current_tenant_id());

-- Explicitly deny UPDATE by creating a policy that always evaluates to false.
CREATE POLICY audit_log_deny_update ON audit_log
    FOR UPDATE
    USING (false);

-- Explicitly deny DELETE by creating a policy that always evaluates to false.
CREATE POLICY audit_log_deny_delete ON audit_log
    FOR DELETE
    USING (false);

-- --------------------------------------------------------------------------
-- Create an application role that respects RLS (not a superuser).
-- Services should connect as this role, not as the DB owner.
-- --------------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'argus_app') THEN
        CREATE ROLE argus_app LOGIN PASSWORD 'argus_app_dev';
    END IF;
END $$;

GRANT USAGE ON SCHEMA public TO argus_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO argus_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO argus_app;

-- Ensure future tables also grant to argus_app
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO argus_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO argus_app;

-- --------------------------------------------------------------------------
-- Migration tracking
-- --------------------------------------------------------------------------
INSERT INTO schema_migrations (version) VALUES ('002_rls');

COMMIT;
