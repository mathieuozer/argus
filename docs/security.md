# Security Model

Argus is designed for compliance-grade environments (GDPR, NIS2, KVKK, FedRAMP). Security is not an add-on; it is woven into every layer.

## Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Security Layers                       │
│                                                          │
│  1. Network: mTLS (all service-to-service)              │
│  2. Identity: SPIFFE/SPIRE (short-lived X.509 certs)   │
│  3. AuthN: JWT tokens (control-plane issued)            │
│  4. AuthZ: OPA-based policy engine + RBAC               │
│  5. Data: 3-tier classification + PII scrubbing         │
│  6. Audit: Immutable audit log (all mutations)          │
│  7. Isolation: Multi-tenant RLS at DB layer             │
│  8. Secrets: HashiCorp Vault (no static keys)           │
└─────────────────────────────────────────────────────────┘
```

## Agent Identity (SPIFFE/SPIRE)

Every agent receives a SPIFFE Verifiable Identity Document (SVID):

```
spiffe://argus.{domain}/tenant/{tenant_id}/agent/{agent_id}/v{version}
```

### Lifecycle

1. **Sidecar starts** alongside the agent
2. Sidecar requests an SVID from the Identity service
3. Identity service generates a short-lived X.509 cert (1-hour TTL)
4. Sidecar uses the SVID for mTLS communication with all platform services
5. SVID is auto-renewed before expiration
6. On quarantine, the SVID is revoked (agent can no longer make calls)

### Key Properties

- **Short-lived**: 1-hour TTL eliminates stale credential risk
- **Auto-rotated**: Sidecar handles renewal transparently
- **Revocable**: OCSP responder and CRL for immediate revocation
- **No static secrets**: No API keys, no shared passwords
- **Cryptographic identity**: SPIFFE ID encodes tenant, agent, and version

## Authentication

### JWT Tokens

The control-plane issues JWT tokens for dashboard users and API consumers.

**Token structure:**
```json
{
  "sub": "admin@example.com",
  "tenant_id": "ministry-finance-tr",
  "role": "admin",
  "iat": 1679500000,
  "exp": 1679586400
}
```

**Roles:**
| Role | Permissions |
|---|---|
| `admin` | Full access (CRUD all resources, manage policies, manage users) |
| `operator` | Read/write agents and tasks, read policies |
| `viewer` | Read-only access to all resources |
| `agent` | Limited to agent-specific operations (heartbeat, telemetry) |

### Development Mode

When `ARGUS_JWT_SECRET` is empty or `"dev"`, the auth middleware creates synthetic admin claims from the `X-Tenant-ID` header. This allows API testing without JWT tokens.

**Never use dev mode in production.**

## Authorization (Policy Engine)

The OPA-based policy engine evaluates authorization decisions:

```json
{
  "subject": "agent:budget-reconciler",
  "action": "call:external_api",
  "resource": "https://api.example.com"
}
```

Policies are per-tenant and support:
- Allow/deny rules with pattern matching
- Role-based access control
- Resource-level permissions
- Audit logging of all policy evaluations

## Multi-Tenant Isolation

### Database Layer (RLS)

Every table has:
```sql
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE agents FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON agents
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

Before each query, the connection sets:
```sql
SELECT set_config('app.tenant_id', $1, true);
```

This ensures that even with SQL injection, a query cannot access another tenant's data.

### API Layer

The `TenantHTTP` middleware:
1. Extracts `X-Tenant-ID` from the request header
2. Validates the tenant claim in the JWT matches
3. Injects the tenant ID into the request context
4. All downstream handlers use `tenancy.FromContext(ctx)` to get the tenant

### Messaging Layer

NATS topics are prefixed: `tenant_{id}_telemetry`, `tenant_{id}_alerts`. Subscriptions are scoped to the authenticated tenant.

### Isolation Tiers

| Tier | DB | Network | Deployment |
|---|---|---|---|
| **A** (Shared) | Row-Level Security | Shared NATS | Shared cluster |
| **B** (Namespace) | Separate schema | Separate NATS namespace | Shared cluster, dedicated namespace |
| **C** (Dedicated) | Separate DB instance | Separate NATS cluster | Separate cluster or bare-metal |

Government tenants default to Tier C.

## Data Classification

### Three Tiers

| Tier | Classification | Examples | Handling |
|---|---|---|---|
| **1** (Structural) | Non-sensitive metrics | Latency, token count, error codes, timestamps | Can leave the node |
| **2** (Sensitive) | Business context | Task descriptions, tool call params, partial outputs | Encrypted at rest, PII-scrubbed, stays in tenant boundary |
| **3** (Restricted) | Full content | Raw I/O, user prompts, full responses | Never leaves the node/site, queryable only via on-prem dashboard |

### PII Scrubbing

The telemetry service runs PII detection before storage:

- **Regex patterns**: Email, IP address, phone numbers, credit cards
- **Locale-specific**: Turkish (TC kimlik, IBAN), EU (national IDs), US (SSN)
- **NER-based**: Named entity recognition for person names and addresses

Scrubbed fields are replaced with `[REDACTED:pii_type]`.

### Data Residency

The telemetry service generates cryptographically signed attestations proving data has not left the configured region:

```json
{
  "attestation": {
    "tenant_id": "ministry-finance-tr",
    "region": "tr-east-1",
    "node_id": "node-01",
    "timestamp": "2026-03-22T12:00:00Z",
    "data_hash": "sha256:abc123...",
    "signature": "hmac-sha256:..."
  }
}
```

## Audit Log

All mutations are recorded in an immutable audit log:

```json
{
  "tenant_id": "default",
  "actor": "admin@example.com",
  "action": "quarantine_agent",
  "resource": "agent/budget-reconciler",
  "details": "predicted_failure: probability=0.92",
  "timestamp": "2026-03-22T12:00:00Z"
}
```

Properties:
- **Immutable**: Append-only, no updates or deletes
- **Tenant-scoped**: Each tenant sees only their own audit entries
- **Searchable**: Full-text search across all fields
- **Retained**: Configurable retention per compliance profile (e.g., 5 years for gov-tr)

## Secrets Management

HashiCorp Vault stores:
- CA private keys (Identity service)
- Database credentials
- JWT signing secrets
- Encryption keys for Tier 2/3 data

In development: Vault runs in dev mode (in-memory, root token: `argus-dev-token`).

In production: Vault uses auto-unseal with a cloud KMS or HSM.

## Compliance Profiles

### gov-tr (Turkey Government)
- Storage: `tr-east-1` or `tr-west-1` only
- PII: KVKK patterns (TC kimlik, IBAN, ad-soyad)
- Default classification: Tier 3
- Audit retention: 5 years
- Certs: TS ISO/IEC 27001

### eu-gdpr (EU Private Sector)
- Storage: EU regions only
- PII: GDPR patterns (email, IP, name, national ID)
- Right-to-erasure: Cryptographic key destruction (Art. 17)
- Configurable audit retention

### fedramp-moderate (US Federal)
- Storage: US Gov regions only
- FIPS 140-2 crypto required
- Continuous monitoring reports
- POA&M tracking

## Security Rules for Contributors

1. **Never log raw agent I/O** - It may be Tier 3 data
2. **Never store secrets in committed files** - Use `.env.local` (gitignored)
3. **Never bypass tenant isolation** - Even in tests, use fixture tenants
4. **Every new DB table must have `tenant_id`** and an RLS policy
5. **Every new API endpoint must have a cross-tenant access test** (assert 403)
6. **Never use static API keys** - Use SPIFFE SVIDs for service-to-service auth
