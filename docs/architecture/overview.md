# Architecture Overview

Argus is a distributed platform for orchestrating, monitoring, and predicting failures in AI agent systems. It is designed for private enterprises and government entities that cannot use SaaS observability tools.

## System Diagram

```
                                    ┌─────────────────────┐
                                    │   React Dashboard    │
                                    │   (localhost:5173)   │
                                    └──────────┬──────────┘
                                               │
                                    ┌──────────▼──────────┐
                                    │     API Gateway      │
                                    │   (localhost:8080)   │
                                    │  mTLS · Rate Limit   │
                                    │    CORS · Routing    │
                                    └──┬───┬───┬───┬──────┘
                                       │   │   │   │
                ┌──────────────────────┘   │   │   └──────────────────────┐
                │                          │   │                          │
     ┌──────────▼──────────┐   ┌──────────▼──┐│  ┌──────────────────────▼───┐
     │    Orchestrator      │   │  Telemetry  ││  │     Control Plane        │
     │  (localhost:8082)    │   │(:8083/:9083)││  │   (localhost:8084)       │
     │                      │   │             ││  │                          │
     │ Agent Registry       │   │ Span Ingest ││  │ Dashboard API · RBAC     │
     │ Task Router          │   │ PII Scrub   ││  │ Policy Engine · Audit    │
     │ State Machine        │   │ Classifier  ││  │ Alerts · Evals           │
     │ Cost Tracker         │   │ Predictor   ││  │ Feedback · Guardrails    │
     │ Version Manager      │   │ Storage     ││  │ Prompts · RAG · SLO      │
     └──────────┬───────────┘   └──────┬──────┘│  └──────────┬──────────────┘
                │                      │       │             │
                │               ┌──────▼──────┐│             │
                │               │  Identity   │└─────────────┤
                │               │(:8081/:9081)│              │
                │               │             │              │
                │               │ Internal CA │              │
                │               │ SVID Issue  │              │
                │               │ Revocation  │              │
                │               │ Vault Integ │              │
                │               └──────┬──────┘              │
                │                      │                     │
     ┌──────────▼──────────────────────▼─────────────────────▼──┐
     │                   Infrastructure Layer                     │
     │                                                            │
     │  ┌──────────┐  ┌──────────────┐  ┌────────────────────┐  │
     │  │PostgreSQL │  │NATS JetStream│  │  HashiCorp Vault   │  │
     │  │  :5432    │  │    :4222     │  │      :8200         │  │
     │  │  RLS      │  │  Pub/Sub    │  │   Secrets Mgmt     │  │
     │  └──────────┘  └──────────────┘  └────────────────────┘  │
     └──────────────────────────────────────────────────────────┘

     ┌─────────────────┐    ┌─────────────────┐
     │   Agent Host 1   │    │   Agent Host N   │
     │                   │    │                   │
     │ ┌───────┐ ┌─────┐│    │ ┌───────┐ ┌─────┐│
     │ │Sidecar│◄│Agent ││    │ │Sidecar│◄│Agent ││
     │ │ :8085 │ │     ││    │ │       │ │     ││
     │ └───┬───┘ └─────┘│    │ └───┬───┘ └─────┘│
     └─────│─────────────┘    └─────│─────────────┘
           │                        │
           └─── mTLS ── Platform ───┘

     ┌──────────────────┐
     │ Python Predictor  │
     │  (localhost:8090)  │
     │  ONNX Runtime     │
     │  Heuristic FBK    │
     └──────────────────┘
```

## Design Principles

### 1. Sidecar-First Adoption

The sidecar (`/sidecar`) is a transparent proxy deployed alongside any AI agent. It:
- Intercepts agent I/O at the network layer (no code changes required)
- Auto-registers the agent with the orchestrator
- Emits telemetry spans and metrics to NATS
- Manages the agent's mTLS certificate lifecycle

Supported agent frameworks: LangChain, AutoGen, CrewAI, custom Python, legacy RPA, or anything with HTTP I/O.

### 2. Multi-Tenancy at Every Layer

Tenant isolation is enforced at three levels:

| Tier | Isolation | Use Case |
|---|---|---|
| **Tier A** (Shared) | PostgreSQL Row-Level Security (RLS) | SMB, low-sensitivity |
| **Tier B** (Dedicated Namespace) | Separate DB schema + NATS namespace | Regulated enterprise |
| **Tier C** (Dedicated Deployment) | Separate cluster, DB instance, network | Government, defense, classified |

Every database table has a `tenant_id` column with RLS policies. Every API call validates tenant context from the JWT/header before any query. NATS topics are prefixed with `tenant_{id}_`.

### 3. Agent Identity via SPIFFE/SPIRE

Every agent gets a short-lived X.509 SVID (1-hour TTL, auto-rotated). The SPIFFE ID format:

```
spiffe://argus.{domain}/tenant/{tenant_id}/agent/{agent_id}/v{version}
```

All service-to-service communication uses mTLS. No static API keys or shared secrets.

### 4. Telemetry Data Classification

Three tiers of data sensitivity, classified at collection time:

| Tier | Examples | Handling |
|---|---|---|
| **Tier 1** (Structural) | Latency, token count, error codes, timestamps | Exits the node freely |
| **Tier 2** (Sensitive) | Task descriptions, tool params, partial outputs | Stays within tenant boundary, encrypted, PII-scrubbed |
| **Tier 3** (Restricted) | Full I/O content, user context | Never leaves the node. Zero bytes cross air-gap |

### 5. Predictive Failure Detection

The predictor watches the telemetry stream for failure precursors:

| Signal | Meaning |
|---|---|
| `latency_p99_ratio` > 3x for >30s | Likely OOM or context overflow |
| Token velocity acceleration | Agent stuck in loop |
| Retry clustering | Upstream dependency degrading |
| Error rate inflection | Model or tool failure imminent |

Output: `{ "failure_probability": 0.82, "ttf_seconds": 180, "precursor_type": "token_escalation" }`

### 6. Air-Gap Ready

The entire platform can run without internet connectivity:
- All Docker images pre-loaded
- NATS JetStream (single binary, no Kafka/Zookeeper)
- Vault in dev mode for secrets
- ONNX model bundled (no cloud ML inference)
- Gitea replaces GitHub Actions for CI/CD

## Service Responsibilities

### Gateway (`:8080`)
- Reverse proxy routing to backend services
- mTLS termination (when configured)
- Per-tenant rate limiting (100 req/min default)
- CORS handling
- WebSocket proxying for real-time streams

### Orchestrator (`:8082` / `:9082` gRPC)
- Agent registration and discovery
- Task-to-agent routing with capability matching
- Per-task state machine (pending -> running -> completed/failed)
- Agent heartbeat monitoring
- Auto-quarantine pipeline (revokes agent cert on predicted failure)
- Cost tracking per agent/tenant
- Agent version management and canary deploy logic

### Telemetry (`:8083` / `:9083` gRPC)
- OpenTelemetry span ingestion
- Metric ingestion and aggregation
- PII scrubbing (regex + NER patterns)
- Data classification (Tier 1/2/3)
- Predictive failure model invocation
- Data residency proof generation
- Data quality scoring

### Control Plane (`:8084` / `:9084` gRPC)
- REST API for the dashboard
- JWT authentication and RBAC
- OPA-based policy engine
- Immutable audit log
- Alert routing and escalation chains
- Evaluation framework (agent benchmarks)
- Human feedback collection
- Guardrail rule management
- Prompt versioning and management
- RAG retrieval analytics
- Compliance report generation
- SLO definition and status tracking
- Cost governance and budgets
- Data quality dashboards
- WebSocket streams for real-time events

### Identity (`:8081` / `:9081` gRPC)
- Internal Certificate Authority
- SVID issuance and renewal (1-hour TTL)
- Certificate revocation (OCSP + CRL)
- SPIFFE ID validation
- HashiCorp Vault integration for root key storage

### Sidecar (`:8085`)
- Transparent network proxy for agent I/O
- Auto-registration with orchestrator
- Telemetry emission (spans, metrics, logs)
- Certificate lifecycle management
- Health and status endpoints

### Predictor (`:8090`)
- Python FastAPI microservice
- ONNX model inference for failure prediction
- Heuristic fallback when no model is loaded
- Health endpoint for readiness probes

## Inter-Service Communication

- **REST/HTTP**: Dashboard -> Gateway -> Backend services
- **gRPC**: Service-to-service (orchestrator <-> telemetry, etc.)
- **NATS JetStream**: Telemetry event streaming, async messaging
- **WebSocket**: Real-time dashboard feeds (agent events, telemetry)
- **PostgreSQL**: Persistent state (all services share the same database with RLS isolation)

## Architecture Decision Records

See the ADR files in this directory:

| ADR | Decision |
|---|---|
| [001](001-adr-sidecar-first.md) | Sidecar-first agent instrumentation |
| [002](002-adr-multi-tenancy.md) | RLS-based multi-tenancy |
| [003](003-adr-spiffe-identity.md) | SPIFFE/SPIRE for agent identity |
| [004](004-adr-telemetry-tiers.md) | Three-tier telemetry classification |
| [005](005-adr-predictive-failure.md) | ONNX-based predictive failure |
| [006](006-adr-air-gap-deployment.md) | Air-gap deployment strategy |
