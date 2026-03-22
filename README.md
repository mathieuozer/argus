# Argus

**Agentic Orchestration, Monitoring & Predictive Failure Platform**

Argus is a lightweight, distributed, compliance-grade platform for orchestrating, monitoring, and predicting failures in AI agent systems. Built for private enterprises and government entities that cannot use SaaS observability tools due to data residency, air-gap, or compliance requirements.

## Key Features

- **Sidecar-first**: Auto-discover and monitor any AI agent without code changes
- **Predictive failure**: ML-based telemetry analysis predicts agent failures before they happen
- **Compliance-grade**: GDPR, NIS2, KVKK, FedRAMP with immutable audit trails
- **Air-gap ready**: Zero cloud dependency, fully offline deployments supported
- **Multi-tenant**: Physical isolation tiers (shared, dedicated namespace, dedicated deployment)
- **Agent identity**: SPIFFE/SPIRE with short-lived mTLS certificates

## Architecture

```
Dashboard (:5173) → Gateway (:8080) → Orchestrator (:8082)
                                     → Telemetry (:8083) → Predictor (:8090)
                                     → Control Plane (:8084)
                                     → Identity (:8081)

Sidecar (:8085) ← deployed alongside each AI agent
```

| Service | Description | HTTP | gRPC |
|---|---|---|---|
| Gateway | API gateway, mTLS, rate limiting | 8080 | - |
| Identity | Agent PKI, SVID issuance, revocation | 8081 | 9081 |
| Orchestrator | Agent registry, task routing, state machine | 8082 | 9082 |
| Telemetry | Span ingestion, PII scrubbing, prediction | 8083 | 9083 |
| Control Plane | Dashboard API, RBAC, audit, evals, prompts | 8084 | 9084 |
| Sidecar | Agent proxy (one per agent) | 8085 | - |
| Predictor | ONNX failure prediction (Python) | 8090 | - |
| Dashboard | React + TypeScript frontend | 5173 | - |

## Quick Start

### Prerequisites

- Go 1.22+, Node.js 20+, Docker Desktop, buf (`brew install bufbuild/buf/buf`)

### Start Everything

```bash
make dev          # Builds and starts all services + dashboard
```

Open http://localhost:5173 for the dashboard.

### Verify

```bash
curl http://localhost:8080/health
# {"status":"ok"}

curl -H "X-Tenant-ID: default" http://localhost:8080/api/v1/agents
# {"data":[...],"meta":{"tenant_id":"default"}}
```

### Other Commands

```bash
make dev-logs     # Tail all service logs
make dev-down     # Stop everything
make test         # Run all unit tests (58 packages)
make test-int     # Integration tests (auto-starts test DB + NATS)
make proto        # Regenerate protobuf stubs after .proto changes
make build        # Build all Go binaries
```

### Run a Single Service

```bash
make run-orchestrator     # Needs ARGUS_DB_DSN and ARGUS_NATS_URL
make run-control-plane
make run-telemetry
```

### Dashboard Development

```bash
cd dashboard && npm install
npm run dev                              # Mock API mode
ARGUS_API_URL=http://localhost:8080 npm run dev  # Real backend
```

## Documentation

| Document | Description |
|---|---|
| [Getting Started](docs/getting-started.md) | Detailed setup guide |
| [Architecture Overview](docs/architecture/overview.md) | System design, data flows, principles |
| [API Reference](docs/api/README.md) | Complete REST and gRPC API docs |
| [Configuration](docs/configuration.md) | All environment variables |
| [Deployment Guide](docs/deployment.md) | Docker, Kubernetes, air-gap deployment |
| [Dashboard Guide](docs/dashboard.md) | Frontend pages, stores, components |
| [Security Model](docs/security.md) | mTLS, SPIFFE, RBAC, data classification |
| [Troubleshooting](docs/troubleshooting.md) | Common issues and fixes |
| [CLAUDE.md](CLAUDE.md) | Full repository layout and dev conventions |

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22+, gRPC/Protobuf |
| Frontend | TypeScript, React 18, Vite, Zustand |
| Database | PostgreSQL 16 with Row-Level Security |
| Messaging | NATS JetStream |
| Identity | SPIFFE/SPIRE, internal CA |
| Secrets | HashiCorp Vault |
| ML | Python, ONNX Runtime, FastAPI |
| Observability | OpenTelemetry, Prometheus, Grafana |
| Deployment | Docker Compose, Kubernetes, bare-metal |

## Test Coverage

- 58 Go test packages passing
- 31 Python tests passing
- 9 integration tests (PostgreSQL + NATS)
- Dashboard builds clean

## License

Proprietary. All rights reserved.
