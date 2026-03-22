# Deployment Guide

Argus supports three deployment targets: Docker Compose (development), Kubernetes (production), and bare-metal scripts (air-gap).

## Docker Compose (Development)

### Start

```bash
make dev
```

This builds all service images and starts the full stack with hot-reloading for the dashboard.

### Services Started

- **Infrastructure**: PostgreSQL 16, NATS JetStream, HashiCorp Vault (dev mode)
- **Backend**: Gateway, Orchestrator, Telemetry, Control Plane, Identity, Sidecar
- **ML**: Python Predictor (ONNX heuristic fallback)
- **Frontend**: Dashboard (Vite dev server with proxy)

### Stop

```bash
make dev-down    # Stops all containers, removes volumes
```

### Rebuild a Single Service

```bash
docker compose -f deployments/docker/docker-compose.yml up --build --no-deps -d <service>
```

Example:
```bash
docker compose -f deployments/docker/docker-compose.yml up --build --no-deps -d orchestrator
```

### View Logs

```bash
make dev-logs                # All services
docker compose -f deployments/docker/docker-compose.yml logs -f orchestrator  # Single service
```

### Docker Images

All Dockerfiles are in `deployments/docker/`:

| Dockerfile | Service | Base |
|---|---|---|
| `Dockerfile.gateway` | Gateway | `golang:1.26-alpine` -> `alpine:3.23` |
| `Dockerfile.orchestrator` | Orchestrator | `golang:1.26-alpine` -> `alpine:3.23` |
| `Dockerfile.telemetry` | Telemetry | `golang:1.26-alpine` -> `alpine:3.23` |
| `Dockerfile.control-plane` | Control Plane | `golang:1.26-alpine` -> `alpine:3.23` |
| `Dockerfile.identity` | Identity | `golang:1.26-alpine` -> `alpine:3.23` |
| `Dockerfile.sidecar` | Sidecar | `golang:1.26-alpine` -> `alpine:3.23` |
| `services/predictor/Dockerfile` | Predictor | `python:3.11-slim` |

All Go services use multi-stage builds (builder -> minimal Alpine runtime) with:
- `CGO_ENABLED=0` for static binaries
- `-trimpath -ldflags="-s -w"` for smaller binaries
- Non-root `argus` user in runtime stage

### Build All Images

```bash
make docker-build
```

---

## Kubernetes (Production)

Kubernetes manifests use Kustomize with overlays for different environments.

### Directory Structure

```
deployments/k8s/
├── base/              # Base manifests (shared across envs)
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── gateway/
│   ├── orchestrator/
│   ├── telemetry/
│   ├── control-plane/
│   ├── identity/
│   └── predictor/
└── overlays/
    ├── dev/           # Development overlay
    ├── staging/       # Staging overlay
    ├── prod/          # Production overlay
    └── air-gap/       # Air-gap overlay (no image pulls)
```

### Deploy to a Cluster

```bash
# Development
kubectl apply -k deployments/k8s/overlays/dev

# Staging
kubectl apply -k deployments/k8s/overlays/staging

# Production
kubectl apply -k deployments/k8s/overlays/prod
```

### Service Dependencies

Deploy in this order:
1. Infrastructure (PostgreSQL, NATS, Vault)
2. Identity service
3. Orchestrator, Telemetry, Control Plane (parallel)
4. Gateway
5. Sidecar (per agent deployment)
6. Dashboard
7. Predictor

### Environment-Specific Config

Use Kustomize overlays to set:
- Resource limits (CPU/memory)
- Replica counts
- Environment variables
- TLS certificates
- Database connection strings

---

## Air-Gap Deployment

For environments with no internet connectivity.

### Pre-Requisites

1. **Pre-load all Docker images** on the target host:
```bash
# On a connected machine:
make docker-build
docker save argus/gateway argus/orchestrator argus/telemetry \
  argus/control-plane argus/identity argus/sidecar argus/predictor \
  postgres:16-alpine nats:2.10-alpine hashicorp/vault:1.15 \
  node:20-alpine > argus-images.tar

# Transfer to air-gapped host, then:
docker load < argus-images.tar
```

2. **Bundle the ONNX model** into the predictor image (already done in Dockerfile)

3. **Pre-install npm packages** for the dashboard:
```bash
cd dashboard && npm ci
tar czf node_modules.tar.gz node_modules/
# Transfer to air-gapped host
```

### Air-Gap Configuration

```bash
export ARGUS_AIR_GAP=true
```

When `ARGUS_AIR_GAP=true`:
- No outbound HTTP calls are made
- Predictor uses bundled ONNX model or heuristic fallback
- Vault runs in dev mode (single-node, in-memory)
- NATS JetStream stores data locally (no external dependencies)

### Kubernetes Air-Gap Overlay

```bash
kubectl apply -k deployments/k8s/overlays/air-gap
```

This overlay:
- Sets `imagePullPolicy: Never` (all images pre-loaded)
- Configures local storage volumes
- Disables external health check URLs
- Sets `ARGUS_AIR_GAP=true` on all services

---

## Database Initialization

PostgreSQL tables are auto-created by each service on first connection. Row-Level Security (RLS) policies are applied via migration scripts.

### Manual Setup

If you need to initialize the database manually:

```bash
psql -h localhost -U argus -d argus
```

Key tables:
- `agents` - Agent registry (with `tenant_id` RLS)
- `tasks` - Task state machine (with `tenant_id` RLS)
- `telemetry_spans` - Span storage (with `tenant_id` RLS)
- `audit_log` - Immutable audit entries

All tables enforce:
```sql
ALTER TABLE <table> ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON <table>
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### Connection Pooling

The `pkg/database` package manages connection pooling via `pgxpool`. Default: 25 connections. The pool sets `app.tenant_id` via `set_config('app.tenant_id', $1, true)` before each query for RLS enforcement.

---

## Health Checks

Every service exposes `GET /health` returning `{"status":"ok"}`.

Configure readiness/liveness probes in Kubernetes:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8082
  initialDelaySeconds: 5
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /health
    port: 8082
  initialDelaySeconds: 3
  periodSeconds: 5
```

---

## TLS / mTLS Configuration

### Gateway mTLS

Set these environment variables to enable mTLS on the gateway:

```bash
ARGUS_CA_CERT_PATH=/etc/argus/ca.crt
ARGUS_SERVER_CERT_PATH=/etc/argus/server.crt
ARGUS_SERVER_KEY_PATH=/etc/argus/server.key
```

When all three are set, the gateway starts with mTLS (mutual TLS), requiring client certificates signed by the specified CA.

When unset, the gateway starts in plain HTTP mode (development only).

### Agent mTLS via Sidecar

The sidecar manages its own TLS certificate lifecycle:
1. On startup, requests an SVID from the Identity service
2. The SVID is a short-lived X.509 certificate (1-hour TTL)
3. Auto-renewed before expiration
4. Used for mTLS communication with all platform services

---

## Scaling Considerations

| Service | Stateless? | Scaling Strategy |
|---|---|---|
| Gateway | Yes | Horizontal (load balancer) |
| Orchestrator | Mostly* | Horizontal with DB for state |
| Telemetry | Yes | Horizontal (NATS consumer groups) |
| Control Plane | Yes | Horizontal |
| Identity | Yes | Horizontal (CA keys in Vault) |
| Predictor | Yes | Horizontal |
| Sidecar | N/A | One per agent (deployed alongside) |

*Orchestrator uses in-memory registry with DB write-through. Multiple replicas share state via PostgreSQL.
