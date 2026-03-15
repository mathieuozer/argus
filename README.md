# Argus

Agentic Orchestration, Monitoring & Predictive Failure Platform.

Argus is a lightweight, distributed, compliance-grade platform for orchestrating, monitoring, and predicting failures in AI agent systems. It targets private enterprises and government entities that cannot use SaaS observability tools due to data residency, air-gap, or compliance requirements.

## Architecture

| Service | Description | Port |
|---|---|---|
| Gateway | API gateway, mTLS termination, rate limiting | 8080 |
| Identity | Agent PKI, cert issuance, SPIFFE IDs | 8081 |
| Orchestrator | Task routing, agent registry, state machine | 8082 |
| Telemetry | Telemetry ingestion, classification, storage | 8083 |
| Control Plane | Dashboard API, RBAC, audit log | 8084 |
| Sidecar | Agent sidecar proxy | 8085 |
| Dashboard | React frontend | 5173 |

## Prerequisites

- Go 1.22+
- Node.js 20+
- Docker + Docker Compose
- buf (`brew install bufbuild/buf/buf`)

## Quickstart

```bash
# Start the full stack
make dev

# View logs
make dev-logs

# Run tests
make test

# Regenerate protobuf code
make proto
```

## Development

```bash
# Run a single service
make run-gateway
make run-identity
make run-orchestrator
make run-telemetry
make run-control-plane

# Dashboard dev server
cd dashboard && npm install && npm run dev
```

## Project Structure

See [CLAUDE.md](CLAUDE.md) for the full repository layout, architectural principles, and development conventions.
