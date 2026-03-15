# Argus — Agentic Orchestration, Monitoring & Predictive Failure Platform

## Project Overview

Argus is a **lightweight, distributed, compliance-grade platform** for orchestrating, monitoring, and predicting failures in AI agent systems. It targets **private enterprises and government entities** that cannot use SaaS observability tools due to data residency, air-gap, or compliance requirements.

**Core value proposition:**
- Auto-discover agents without code changes (sidecar-first)
- Predict agent failures before they happen (ML-based telemetry analysis)
- Full audit trails with immutable logs (compliance: GDPR, NIS2, KVKK, FedRAMP)
- Works fully offline / air-gapped — zero cloud dependency required
- Multi-tenant with physical isolation tiers for government clients

---

## Tech Stack

| Layer | Technology | Rationale |
|---|---|---|
| Backend services | Go 1.22+ | Lightweight binaries, excellent concurrency, great for distributed systems, easy to cross-compile for air-gap |
| Frontend dashboard | TypeScript + React 18 + Vite | Fast, type-safe, modern component model |
| State management | Zustand | Lightweight, no boilerplate |
| gRPC / Protobuf | protoc + buf | All inter-service communication, language-agnostic |
| Database | PostgreSQL 16 with Row-Level Security | Multi-tenancy enforced at DB layer |
| Telemetry bus | NATS JetStream | Lightweight alternative to Kafka, works air-gapped, single binary |
| Identity / PKI | SPIFFE/SPIRE + internal CA | Short-lived certs, mTLS, no cloud PKI dependency |
| Secrets | HashiCorp Vault (or Vault in dev mode for air-gap) | Agent credential lifecycle |
| Observability (self) | OpenTelemetry + Prometheus + Grafana | Platform eats its own dog food |
| Deployment | Docker Compose (dev) → Kubernetes (prod) → bare-metal scripts (air-gap) | Flexible deployment targets |
| ML / Predictor | Python (isolated microservice) + ONNX runtime | Predictive failure model, ONNX means no Python in prod for inference |
| CI/CD | GitHub Actions → can be replaced with Gitea for air-gap | |

---

## Repository Layout

```
argus/
├── CLAUDE.md                        ← YOU ARE HERE
├── go.work                          ← Go workspace (all services)
├── Makefile                         ← Top-level build, dev, test commands
├── .gitignore
├── README.md
│
├── services/                        ← Backend microservices (Go)
│   ├── control-plane/               ← Dashboard API, RBAC, policy engine, audit log
│   │   ├── cmd/main/main.go
│   │   ├── internal/
│   │   │   ├── auth/                ← JWT validation, RBAC enforcement
│   │   │   ├── policy/              ← OPA-based policy engine
│   │   │   ├── audit/               ← Immutable audit log writer
│   │   │   ├── alerts/              ← Alert routing, escalation chains
│   │   │   └── dashboard/           ← REST/gRPC handlers for frontend
│   │   ├── pkg/                     ← Exported packages for other services
│   │   └── api/                     ← OpenAPI / proto definitions for this service
│   │
│   ├── orchestrator/                ← Task routing, agent registry, state machine
│   │   ├── cmd/main/main.go
│   │   ├── internal/
│   │   │   ├── registry/            ← Agent discovery, registration, health
│   │   │   ├── router/              ← Task-to-agent routing with capability matching
│   │   │   ├── statemachine/        ← Per-task state (pending→running→done/failed)
│   │   │   ├── versioning/          ← Agent version tracking, canary deploy logic
│   │   │   └── costtracker/         ← Token spend, API cost per agent/tenant
│   │   ├── pkg/
│   │   └── api/
│   │
│   ├── telemetry/                   ← Telemetry ingestion, classification, storage
│   │   ├── cmd/main/main.go
│   │   ├── internal/
│   │   │   ├── collector/           ← OpenTelemetry collector, NATS consumer
│   │   │   ├── classifier/          ← Tier 1/2/3 data classification
│   │   │   ├── pii/                 ← PII scrubber (regex + NER)
│   │   │   ├── predictor/           ← Calls ONNX predictor, fires pre-failure alerts
│   │   │   └── storage/             ← Routes to correct storage per classification tier
│   │   ├── pkg/
│   │   └── api/
│   │
│   ├── identity/                    ← Agent PKI, cert issuance, revocation, SPIFFE
│   │   ├── cmd/main/main.go
│   │   ├── internal/
│   │   │   ├── ca/                  ← Internal Certificate Authority
│   │   │   ├── spiffe/              ← SVID generation and validation
│   │   │   ├── vault/               ← HashiCorp Vault integration for root key storage
│   │   │   └── revocation/          ← OCSP responder, CRL generation
│   │   ├── pkg/
│   │   └── api/
│   │
│   └── gateway/                     ← API gateway, mTLS termination, rate limiting
│       ├── cmd/main/main.go
│       ├── internal/
│       │   ├── proxy/               ← Reverse proxy, request routing
│       │   ├── ratelimit/           ← Per-tenant, per-agent rate limits
│       │   └── mtls/                ← mTLS termination, cert validation
│       ├── pkg/
│       └── api/
│
├── sidecar/                         ← Agent sidecar proxy (deploy alongside any agent)
│   ├── cmd/main/main.go
│   ├── internal/
│   │   ├── proxy/                   ← Intercepts agent I/O transparently
│   │   ├── discovery/               ← Auto-registers agent with orchestrator
│   │   ├── telemetry/               ← Emits spans, metrics, structured logs
│   │   └── identity/                ← Manages cert lifecycle, mTLS handshake
│   └── pkg/
│
├── sdk/                             ← Optional native SDKs for richer telemetry
│   ├── go/src/                      ← Go SDK: custom spans, business events
│   ├── python/src/                  ← Python SDK: decorator-based instrumentation
│   └── typescript/src/              ← TS SDK: for Node.js / edge agents
│
├── dashboard/                       ← React frontend (TypeScript + Vite)
│   ├── src/
│   │   ├── components/
│   │   │   ├── agents/              ← Agent list, detail, status cards
│   │   │   ├── metrics/             ← Charts, sparklines, cost panels
│   │   │   ├── alerts/              ← Alert feed, predictive warning cards
│   │   │   ├── settings/            ← Tenant config, policy editor, RBAC
│   │   │   └── layout/              ← Nav, sidebar, shell
│   │   ├── pages/                   ← Route-level page components
│   │   ├── hooks/                   ← Custom React hooks (useAgents, useMetrics...)
│   │   ├── stores/                  ← Zustand stores (agentStore, tenantStore...)
│   │   ├── utils/                   ← Formatters, date helpers, API client
│   │   └── types/                   ← Shared TypeScript interfaces
│   └── public/
│
├── proto/                           ← Protobuf definitions (shared across services)
│   ├── agent/                       ← Agent registration, status, capability messages
│   ├── telemetry/                   ← Span, metric, event messages
│   ├── identity/                    ← SVID, CSR, revocation messages
│   └── orchestration/               ← Task, routing, state machine messages
│
├── pkg/                             ← Shared Go packages (used by all services)
│   ├── tenancy/                     ← Tenant context, namespace enforcement
│   ├── crypto/                      ← Cert utils, HMAC, signing helpers
│   ├── config/                      ← Unified config loader (env + file + vault)
│   ├── logger/                      ← Structured logger (zap-based)
│   ├── middleware/                  ← HTTP/gRPC middleware (tenant, auth, tracing)
│   └── errors/                      ← Typed error system with error codes
│
├── deployments/
│   ├── docker/                      ← Dockerfiles per service + docker-compose.yml
│   ├── k8s/
│   │   ├── base/                    ← Kustomize base manifests
│   │   └── overlays/
│   │       ├── dev/
│   │       ├── staging/
│   │       ├── prod/
│   │       └── air-gap/             ← No image pulls, all images pre-loaded
│   ├── terraform/                   ← IaC for cloud deployments
│   └── scripts/                     ← Air-gap install scripts, cert bootstrapping
│
└── docs/
    ├── architecture/                ← ADRs (Architecture Decision Records)
    ├── api/                         ← Generated API docs
    └── runbooks/                    ← Ops runbooks (incident response, rollback)
```

---

## Core Architectural Principles

### 1. Sidecar-first, no code changes required
The sidecar (`/sidecar`) is the primary adoption path. Drop it alongside any agent process — LangChain, AutoGen, CrewAI, custom Python, or legacy RPA. It intercepts I/O at the network layer, auto-registers the agent, and emits telemetry. The SDK is optional for richer instrumentation.

### 2. Multi-tenancy is foundational, not an afterthought
Every DB table has `tenant_id` with PostgreSQL Row-Level Security. Every API call validates the tenant claim from the JWT before any query. Every NATS topic is prefixed `tenant_{id}_`. Never query across tenant boundaries. See `/pkg/tenancy` for the enforcement helpers.

**Isolation tiers:**
- **Tier A** (shared): Logical isolation via RLS. For SMB / low-sensitivity.
- **Tier B** (dedicated namespace): Dedicated DB schema, dedicated NATS namespace. For regulated enterprise.
- **Tier C** (dedicated deployment): Separate cluster/bare-metal, separate DB instance. For government, defense, classified. Can be fully air-gapped.

### 3. Agent identity via SPIFFE/SPIRE
Every agent gets a short-lived X.509 SVID (1-hour TTL, auto-rotated by sidecar). The SPIFFE ID encodes: `spiffe://argus.{domain}/tenant/{tenant_id}/agent/{agent_id}/v{version}`. All service-to-service and agent-to-platform communication uses mTLS with these certs. No static API keys, no shared secrets.

### 4. Telemetry data classification (three tiers)
At collection time in the sidecar, every telemetry record is classified:
- **Tier 1 (structural):** `latency_ms`, `token_count`, `error_code`, `agent_id`, timestamps — exits the node freely.
- **Tier 2 (sensitive):** Task descriptions, tool call params, partial outputs — stays within tenant boundary, encrypted at rest, PII-scrubbed before storage.
- **Tier 3 (restricted):** Full I/O content, user-supplied context — never leaves the node/site. Only queryable via on-prem dashboard. Zero bytes cross the air-gap boundary.

Configured per-tenant in their policy file. Government tenants default to Tier 3 for all data.

### 5. Predictive failure is the core differentiator
The `/services/telemetry/internal/predictor` service watches the telemetry stream for known failure precursors:
- Latency spikes (p99 > 3x p50 for >30s) → likely OOM or context overflow
- Token escalation pattern → agent stuck in loop
- Retry clustering → upstream dependency degrading
- Error rate inflection → model or tool failure imminent

The predictor calls an ONNX model (trained offline, served via the Python predictor microservice) and fires a pre-failure alert with a probability score and estimated time-to-failure. The control plane routes this alert through the tenant's escalation chain.

---

## Data Models (key entities)

```go
// Tenant — top-level isolation boundary
type Tenant struct {
    ID              string          // e.g. "ministry-finance-tr"
    DisplayName     string
    IsolationTier   IsolationTier   // A, B, or C
    StorageRegions  []string        // allowed regions for telemetry data
    PIIScrub        bool
    ComplianceProfile string        // "gov-tr", "eu-gdpr", "fedramp-moderate"
    CreatedAt       time.Time
}

// Agent — a registered agent instance
type Agent struct {
    ID          string    // stable, human-readable: "budget-reconciler"
    TenantID    string
    Version     string    // semver
    Framework   string    // "langchain", "autogen", "custom"
    Capabilities []string // ["read:budget_db", "write:report_store"]
    Status      AgentStatus // discovered | healthy | degraded | failed | quarantined
    SVIDURI     string    // SPIFFE ID
    LastSeen    time.Time
    NodeID      string    // which host is running it
}

// Task — a unit of work routed through the orchestrator
type Task struct {
    ID          string
    TenantID    string
    AgentID     string
    Status      TaskStatus  // pending | running | completed | failed | awaiting_approval
    InputHash   string      // SHA256 of input (not the input itself — classification)
    StartedAt   time.Time
    CompletedAt *time.Time
    CostUSD     float64
    TokensUsed  int64
    ApprovalID  *string     // set when human approval required
}

// TelemetrySpan — a single traced operation
type TelemetrySpan struct {
    SpanID      string
    TraceID     string
    TenantID    string
    AgentID     string
    TaskID      string
    OperationName string
    StartedAt   time.Time
    DurationMs  int64
    Tier        DataTier    // 1, 2, or 3
    Attributes  map[string]string // tier-classified, PII-scrubbed
    ErrorCode   *string
}

// PredictiveAlert — fired before an agent fails
type PredictiveAlert struct {
    ID              string
    TenantID        string
    AgentID         string
    Probability     float64   // 0.0 - 1.0
    EstimatedTTF    Duration  // estimated time to failure
    PrecursorType   string    // "latency_spike", "token_escalation", "retry_storm"
    Evidence        []string  // telemetry points that triggered this
    Status          AlertStatus // open | acknowledged | resolved | false_positive
    CreatedAt       time.Time
}
```

---

## API Conventions

- All REST endpoints: `GET/POST/PUT/DELETE /api/v1/{resource}`
- All responses: `{ "data": {...}, "meta": { "tenant_id": "...", "request_id": "..." } }`
- All errors: `{ "error": { "code": "AGENT_NOT_FOUND", "message": "...", "details": {} } }`
- Pagination: cursor-based via `?after=<cursor>&limit=<n>` (never offset pagination)
- Authentication: Bearer JWT from control-plane, tenant claim mandatory
- gRPC: all inter-service calls, defined in `/proto`
- WebSocket: `/ws/v1/agents/stream` and `/ws/v1/telemetry/stream` for live dashboard feeds

---

## Environment Variables (universal across services)

```bash
# Required for all services
ARGUS_ENV=development|staging|production
ARGUS_TENANT_ENFORCEMENT=strict        # never disable in prod
ARGUS_LOG_LEVEL=info|debug|warn|error
ARGUS_LOG_FORMAT=json|text

# Database
ARGUS_DB_DSN=postgres://user:pass@host:5432/argus?sslmode=require
ARGUS_DB_MAX_CONNS=25

# NATS
ARGUS_NATS_URL=nats://localhost:4222
ARGUS_NATS_STREAM=ARGUS_TELEMETRY

# Identity
ARGUS_SPIRE_SOCKET=/tmp/spire-agent/public/api.sock
ARGUS_CA_CERT_PATH=/etc/argus/ca.crt

# Vault
ARGUS_VAULT_ADDR=https://vault.internal:8200
ARGUS_VAULT_ROLE=argus-service
ARGUS_VAULT_MOUNT=secret/argus

# Air-gap mode (disables all outbound network calls)
ARGUS_AIR_GAP=false
```

---

## Development Workflow

### Prerequisites
- Go 1.22+
- Node.js 20+
- Docker + Docker Compose
- buf (protobuf toolchain): `brew install bufbuild/buf/buf`
- IntelliJ IDEA (recommended: Go plugin + Node.js plugin)

### Start the full stack locally
```bash
make dev          # starts all services + dashboard via docker-compose
make dev-logs     # tail all service logs
make proto        # regenerate protobuf code after .proto changes
```

### Run a single service
```bash
make run-orchestrator
make run-telemetry
make run-control-plane
```

### Run tests
```bash
make test          # all unit tests
make test-int      # integration tests (requires running DB + NATS)
make test-e2e      # end-to-end tests against local stack
make test-cover    # with coverage report
```

### Dashboard dev server
```bash
cd dashboard && npm install && npm run dev   # http://localhost:5173
```

---

## Key Conventions for AI-Assisted Development

When writing code in this project, always follow these rules:

### Go conventions
- Every function that touches the database must accept `ctx context.Context` as first arg
- Always extract `tenantID` from context using `tenancy.FromContext(ctx)` — never trust user-supplied tenant IDs
- Use `pkg/errors` typed errors, never raw `fmt.Errorf` in service code
- Use `pkg/logger` structured logger (zap), never `log.Printf`
- gRPC handlers live in `api/`, business logic lives in `internal/` — never put business logic in handlers
- All DB queries must go through repository pattern in `internal/*/repository.go`
- Write table-driven tests — `TestXxx(t *testing.T)` with `t.Run(tc.name, ...)`

### TypeScript / React conventions
- All API calls go through `/dashboard/src/utils/apiClient.ts` — never raw fetch in components
- Use Zustand stores for global state, local useState for component-local state
- All types live in `/dashboard/src/types/` — no inline type definitions in components
- Components are functional only — no class components
- Every component file: one default export, name matches filename

### Security rules (non-negotiable)
- Never log raw agent inputs or outputs — they may be Tier 3 data
- Never store secrets in env files committed to the repo — use `.env.local` (gitignored)
- Never bypass tenant isolation even in tests — use fixture tenants
- Every new DB table must have `tenant_id UUID NOT NULL` and a corresponding RLS policy
- Every new API endpoint must have a test asserting that cross-tenant access returns 403

### When adding a new service
1. Add it under `/services/{name}/`
2. Add `go.work` entry
3. Define its proto in `/proto/{name}/`
4. Add a Dockerfile in `/deployments/docker/`
5. Add a Kustomize base manifest in `/deployments/k8s/base/`
6. Add env vars to the universal env section in this file
7. Update the README service table

---

## Compliance Profiles

### gov-tr (Turkey government)
- Storage: `tr-east-1` or `tr-west-1` only
- PII scrub: enabled, KVKK patterns (TC kimlik, IBAN, ad-soyad)
- Data classification default: Tier 3
- Audit retention: 5 years
- Air-gap capable: yes
- Required certs: TS ISO/IEC 27001

### eu-gdpr (EU private sector)
- Storage: EU regions only
- PII scrub: enabled, GDPR patterns (email, IP, name, national ID)
- Audit retention: configurable, hard delete on request (Art. 17)
- Right-to-erasure: cryptographic key destruction

### fedramp-moderate (US Federal)
- Storage: US Gov regions only
- FIPS 140-2 crypto modules required
- Continuous monitoring reports
- POA&M tracking for findings

---

## Predictive Failure Model — Feature Reference

The ONNX model is retrained weekly from anonymized telemetry. Input features:

| Feature | Description |
|---|---|
| `latency_p99_ratio` | p99/p50 latency ratio over 5-min window |
| `token_velocity` | Tokens/sec, rate of change over last 10 tasks |
| `retry_rate` | Retries / total calls over last 5 min |
| `error_rate_delta` | Error rate change vs 1h baseline |
| `context_fill_pct` | Estimated context window utilization |
| `tool_call_depth` | Average tool call nesting depth |
| `consecutive_slow` | Count of consecutive >2s tool calls |
| `cost_acceleration` | Cost/task rate of change |

Output: `{ "failure_probability": 0.82, "ttf_seconds": 180, "precursor_type": "token_escalation" }`

---

## Glossary

| Term | Meaning |
|---|---|
| Agent | Any AI agent process (LangChain, AutoGen, custom, etc.) being monitored |
| Sidecar | The Argus proxy process deployed alongside an agent — handles discovery, telemetry, identity |
| SVID | SPIFFE Verifiable Identity Document — the short-lived X.509 cert identifying an agent |
| TTF | Time-to-Failure — estimated from the predictive model |
| Tenant | An isolated organizational unit (a ministry, a company, a department) |
| Isolation Tier | A/B/C — the level of infrastructure isolation for a tenant |
| Precursor | A telemetry pattern that historically precedes agent failure |
| Quarantine | Revoking an agent's cert so it can't initiate calls but can still be inspected |
| Air-gap | A deployment with no outbound internet connectivity — all deps must be pre-loaded |
