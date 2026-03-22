# Getting Started

This guide walks through setting up Argus for local development.

## Prerequisites

| Tool | Version | Install |
|---|---|---|
| Go | 1.22+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 20+ | [nodejs.org](https://nodejs.org/) |
| Docker Desktop | Latest | [docker.com](https://www.docker.com/products/docker-desktop/) |
| buf | Latest | `brew install bufbuild/buf/buf` |
| protoc | Latest | `brew install protobuf` |

## Quick Start (Docker Compose)

Start the entire platform with one command:

```bash
make dev
```

This launches all services, infrastructure, and the dashboard:

| Component | URL | Description |
|---|---|---|
| Dashboard | http://localhost:5173 | React frontend |
| Gateway | http://localhost:8080 | API gateway (entry point for all API calls) |
| Identity | http://localhost:8081 | Agent PKI and SVID management |
| Orchestrator | http://localhost:8082 | Agent registry and task routing |
| Telemetry | http://localhost:8083 | Span ingestion and predictive failure |
| Control Plane | http://localhost:8084 | Dashboard API, policies, audit |
| Sidecar | http://localhost:8085 | Agent proxy (demo instance) |
| Predictor | http://localhost:8090 | Python ONNX failure prediction |
| PostgreSQL | localhost:5432 | Database (user: `argus`, pass: `argus`) |
| NATS | localhost:4222 | Message bus (monitoring: http://localhost:8222) |
| Vault | http://localhost:8200 | Secrets (dev token: `argus-dev-token`) |

## Verifying the Stack

Once all containers are running, verify:

```bash
# Gateway health
curl http://localhost:8080/health
# Expected: {"status":"ok"}

# List agents (through gateway)
curl -H "X-Tenant-ID: default" http://localhost:8080/api/v1/agents
# Expected: {"data":[...],"meta":{"tenant_id":"default"}}

# Predictor health
curl http://localhost:8090/health
# Expected: {"status":"healthy","model":"heuristic",...}
```

## Running a Single Service

For faster iteration on a specific service:

```bash
make run-orchestrator     # Port 8082
make run-telemetry        # Port 8083
make run-control-plane    # Port 8084
make run-identity         # Port 8081
make run-gateway          # Port 8080
make run-sidecar          # Port 8085
```

Note: services run locally with `localhost` defaults. You need PostgreSQL and NATS running locally or via Docker:

```bash
# Start just infrastructure
docker compose -f deployments/docker/docker-compose.yml up -d postgres nats vault
```

## Running the Dashboard

```bash
cd dashboard
npm install
npm run dev    # http://localhost:5173
```

The dashboard uses a mock API by default. To connect to real backend services, set:

```bash
ARGUS_API_URL=http://localhost:8080 npm run dev
```

## Running Tests

```bash
make test          # All unit tests (58 packages)
make test-int      # Integration tests (requires Docker)
make test-cover    # With HTML coverage report
make test-e2e      # End-to-end tests against local stack
```

Integration tests automatically start a test PostgreSQL and NATS via `docker-compose.test.yml`:
- Test PostgreSQL: port 5433
- Test NATS: port 4223

## Protobuf Generation

After modifying `.proto` files in `/proto/`:

```bash
make proto
```

This runs `buf lint` and `buf generate`, outputting Go stubs to `gen/go/`.

Note: Generated `*.pb.go` files are gitignored. You must run `make proto` after cloning.

## Project Structure

```
argus/
├── services/           # Go microservices
│   ├── control-plane/  # Dashboard API, RBAC, policy, audit
│   ├── orchestrator/   # Agent registry, task routing
│   ├── telemetry/      # Span ingestion, prediction
│   ├── identity/       # PKI, SPIFFE/SPIRE, Vault
│   └── gateway/        # API gateway, mTLS, rate limiting
├── sidecar/            # Agent proxy (deploy alongside agents)
├── sdk/                # Optional SDKs (Go, Python, TypeScript)
├── dashboard/          # React + TypeScript frontend
├── proto/              # Protobuf definitions
├── pkg/                # Shared Go libraries
├── deployments/        # Docker, Kubernetes, Terraform configs
└── docs/               # Documentation (you are here)
```

## Stopping the Stack

```bash
make dev-down    # Stops all containers and removes volumes
```
