# ADR 002: Multi-Tenancy with Row-Level Security

## Status
Accepted

## Context
Argus serves private enterprises and government entities with varying isolation requirements. We need to support shared infrastructure for SMBs while providing dedicated isolation for government clients.

## Decision
Implement three isolation tiers:
- **Tier A (shared):** Logical isolation via PostgreSQL Row-Level Security. All tenants share the same database and NATS instance.
- **Tier B (dedicated namespace):** Dedicated DB schema and NATS namespace per tenant. For regulated enterprise.
- **Tier C (dedicated deployment):** Separate cluster/bare-metal per tenant. For government, defense, classified environments.

Every database table includes `tenant_id UUID NOT NULL` with RLS policies. Every API call validates the tenant claim from the JWT before any query. Every NATS topic is prefixed with `tenant_{id}_`.

## Consequences
- **Positive:** Single codebase supports all isolation levels
- **Positive:** RLS enforces isolation at the database layer (defense in depth)
- **Positive:** Government clients get full physical isolation with air-gap support
- **Negative:** RLS adds overhead to every query
- **Negative:** Tier C requires per-tenant deployment automation
