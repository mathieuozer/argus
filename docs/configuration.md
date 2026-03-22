# Configuration Reference

All Argus services are configured via environment variables. This document lists every variable, its default, and which services use it.

## Core Configuration

These variables apply to all Go services.

| Variable | Default | Description |
|---|---|---|
| `ARGUS_ENV` | `development` | Environment name: `development`, `staging`, `production` |
| `ARGUS_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `ARGUS_LOG_FORMAT` | `json` | Log output format: `json` (structured) or `text` (human-readable) |
| `ARGUS_TENANT_ENFORCEMENT` | `strict` | Tenant isolation mode. **Never disable in production.** |
| `ARGUS_AIR_GAP` | `false` | Enable air-gap mode (disables all outbound network calls) |

## Database

| Variable | Default | Services | Description |
|---|---|---|---|
| `ARGUS_DB_DSN` | *(none)* | orchestrator, telemetry, control-plane, gateway | PostgreSQL connection string. When unset, services use in-memory stores. Example: `postgres://argus:argus@localhost:5432/argus?sslmode=disable` |
| `ARGUS_DB_MAX_CONNS` | `25` | All with DB | Maximum database connection pool size |
| `ARGUS_TEST_DB_DSN` | *(none)* | Tests only | Test database connection string (port 5433) |

## NATS / Messaging

| Variable | Default | Services | Description |
|---|---|---|---|
| `ARGUS_NATS_URL` | `nats://localhost:4222` | orchestrator, telemetry, control-plane, sidecar | NATS server connection URL |
| `ARGUS_NATS_STREAM` | `ARGUS_TELEMETRY` | orchestrator, telemetry, sidecar | JetStream stream name for telemetry events |
| `ARGUS_TEST_NATS_URL` | *(none)* | Tests only | Test NATS URL (port 4223) |

## Authentication & Identity

| Variable | Default | Services | Description |
|---|---|---|---|
| `ARGUS_JWT_SECRET` | *(empty = dev mode)* | control-plane | Secret for JWT signing. Empty or `"dev"` enables dev mode (no auth required, synthetic admin claims) |
| `ARGUS_SPIRE_SOCKET` | `/tmp/spire-agent/public/api.sock` | identity, sidecar | SPIRE agent socket path |
| `ARGUS_CA_CERT_PATH` | *(none)* | gateway, identity | Path to CA certificate for mTLS |
| `ARGUS_SERVER_CERT_PATH` | *(none)* | gateway | Path to server TLS certificate |
| `ARGUS_SERVER_KEY_PATH` | *(none)* | gateway | Path to server TLS private key |

## HashiCorp Vault

| Variable | Default | Services | Description |
|---|---|---|---|
| `ARGUS_VAULT_ADDR` | `https://vault.internal:8200` | identity | Vault server address |
| `ARGUS_VAULT_ROLE` | `argus-service` | identity | Vault auth role |
| `ARGUS_VAULT_MOUNT` | `secret/argus` | identity | Vault secrets engine mount |

## Gateway - Backend Routing

These control where the gateway proxies requests. Default to `localhost` for local development.

| Variable | Default | Description |
|---|---|---|
| `ARGUS_BACKEND_ORCHESTRATOR` | `http://localhost:8082` | Orchestrator service URL |
| `ARGUS_BACKEND_TELEMETRY` | `http://localhost:8083` | Telemetry service URL |
| `ARGUS_BACKEND_IDENTITY` | `http://localhost:8081` | Identity service URL |
| `ARGUS_BACKEND_CONTROL_PLANE` | `http://localhost:8084` | Control Plane service URL |

In Docker Compose, these are set to Docker service names:
```yaml
ARGUS_BACKEND_ORCHESTRATOR: http://orchestrator:8082
ARGUS_BACKEND_TELEMETRY: http://telemetry:8083
ARGUS_BACKEND_IDENTITY: http://identity:8081
ARGUS_BACKEND_CONTROL_PLANE: http://control-plane:8084
```

In Kubernetes, use DNS names:
```yaml
ARGUS_BACKEND_ORCHESTRATOR: http://orchestrator.argus.svc.cluster.local:8082
```

## Sidecar

| Variable | Default | Description |
|---|---|---|
| `ARGUS_ORCHESTRATOR_ADDR` | `http://localhost:8082` | Orchestrator address for registration |
| `ARGUS_IDENTITY_ADDR` | `http://localhost:8081` | Identity service address for SVID requests |
| `ARGUS_AGENT_ID` | `unknown-agent` | Agent identifier (should be unique per agent) |
| `ARGUS_AGENT_VERSION` | `0.0.1` | Agent version (semver) |
| `ARGUS_AGENT_FRAMEWORK` | `custom` | Agent framework: `langchain`, `autogen`, `crewai`, `custom` |
| `ARGUS_TENANT_ID` | `default` | Tenant the agent belongs to |
| `ARGUS_UPSTREAM_ADDR` | *(none)* | Upstream address to proxy agent traffic to |

## Telemetry Service

| Variable | Default | Description |
|---|---|---|
| `ARGUS_NODE_ID` | *(hostname)* | Node identifier for data residency tracking |
| `ARGUS_REGION` | *(none)* | Region for data residency proofs |
| `ARGUS_RESIDENCY_SIGNING_KEY` | *(none)* | HMAC key for signing residency attestations |

## Dashboard

| Variable | Default | Description |
|---|---|---|
| `ARGUS_API_URL` | *(none)* | Backend API URL. When set, enables Vite proxy and disables mock API. Example: `http://localhost:8080` |
| `ARGUS_WS_URL` | `ws://localhost:8080` | WebSocket URL for real-time streams |

When `ARGUS_API_URL` is not set, the dashboard uses the built-in mock API for standalone development.

## Port Reference

| Service | HTTP Port | gRPC Port |
|---|---|---|
| Gateway | 8080 | - |
| Identity | 8081 | 9081 |
| Orchestrator | 8082 | 9082 |
| Telemetry | 8083 | 9083 |
| Control Plane | 8084 | 9084 |
| Sidecar | 8085 | - |
| Predictor (Python) | 8090 | - |
| Dashboard (Vite) | 5173 | - |
| PostgreSQL | 5432 | - |
| NATS | 4222 (client), 8222 (monitoring) | - |
| Vault | 8200 | - |

## Example: Minimal Local Development

```bash
export ARGUS_ENV=development
export ARGUS_LOG_LEVEL=debug
export ARGUS_LOG_FORMAT=text
export ARGUS_DB_DSN=postgres://argus:argus@localhost:5432/argus?sslmode=disable
export ARGUS_NATS_URL=nats://localhost:4222
```

## Example: Docker Compose (set automatically)

See `deployments/docker/docker-compose.yml` for the complete environment configuration per service.

## Example: Air-Gap Production

```bash
export ARGUS_ENV=production
export ARGUS_AIR_GAP=true
export ARGUS_TENANT_ENFORCEMENT=strict
export ARGUS_LOG_FORMAT=json
export ARGUS_DB_DSN=postgres://argus:$DB_PASS@db.internal:5432/argus?sslmode=require
export ARGUS_NATS_URL=nats://nats.internal:4222
export ARGUS_CA_CERT_PATH=/etc/argus/ca.crt
export ARGUS_SERVER_CERT_PATH=/etc/argus/server.crt
export ARGUS_SERVER_KEY_PATH=/etc/argus/server.key
export ARGUS_VAULT_ADDR=https://vault.internal:8200
```
