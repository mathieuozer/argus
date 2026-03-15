# Deployment Runbook

## Prerequisites

- Docker 24+ and Docker Compose v2
- kubectl 1.28+ (for Kubernetes deployments)
- Go 1.24+ (for building from source)
- Node.js 20+ (for dashboard)
- Access to container registry (ghcr.io for production)

## Local Development

### Quick Start
```bash
make dev            # Start all services via docker-compose
make dev-logs       # Tail logs
make dashboard-dev  # Start dashboard dev server (http://localhost:5173)
```

### Run Tests
```bash
make test           # All unit tests
make test-cover     # With coverage report
```

### Build Binaries
```bash
make build          # Build all services to bin/
```

## Kubernetes Deployment

### Development
```bash
# Apply dev overlay (single replica, debug logging)
kubectl apply -k deployments/k8s/overlays/dev/

# Verify
kubectl get pods -n argus
kubectl get svc -n argus
```

### Production
```bash
# 1. Update image tags in overlays/prod/kustomization.yaml
# 2. Apply
kubectl apply -k deployments/k8s/overlays/prod/

# 3. Verify rollout
kubectl rollout status deployment --all -n argus

# 4. Run smoke tests
curl -s https://argus.example.com/health | jq .
```

### Air-Gap Deployment

1. **On a connected machine, build and export images:**
```bash
make docker-build
docker save argus/control-plane argus/orchestrator argus/telemetry \
  argus/identity argus/gateway argus/sidecar | gzip > argus-images.tar.gz
```

2. **Transfer to air-gapped environment** (USB, secure file transfer, etc.)

3. **On the air-gapped machine:**
```bash
# Load images
docker load < argus-images.tar.gz

# For Kubernetes (containerd)
ctr -n k8s.io images import argus-images.tar.gz

# Apply air-gap overlay
kubectl apply -k deployments/k8s/overlays/air-gap/
```

## Database Setup

### Initialize Schema
```bash
# Using the init script
export ARGUS_DB_DSN="postgres://argus:argus@localhost:5432/argus?sslmode=disable"
bash deployments/scripts/db/init.sh

# Or manually
psql "$ARGUS_DB_DSN" -f deployments/scripts/db/001_schema.sql
psql "$ARGUS_DB_DSN" -f deployments/scripts/db/002_rls.sql
psql "$ARGUS_DB_DSN" -f deployments/scripts/db/003_seed.sql  # dev only
```

### Verify RLS
```bash
psql "$ARGUS_DB_DSN" -c "SELECT tablename, rowsecurity FROM pg_tables WHERE schemaname='public';"
```

## Rolling Updates

### Zero-Downtime Update
```bash
# 1. Build new images
make docker-build DOCKER_TAG=v1.2.0

# 2. Push to registry
docker push ghcr.io/mathieuozer/argus/control-plane:v1.2.0
# ... repeat for all services

# 3. Update image tags
kubectl set image deployment/control-plane control-plane=ghcr.io/mathieuozer/argus/control-plane:v1.2.0 -n argus

# 4. Monitor rollout
kubectl rollout status deployment/control-plane -n argus
```

### Rollback
```bash
kubectl rollout undo deployment/<service> -n argus
kubectl rollout status deployment/<service> -n argus
```

## Health Checks

All services expose `/health` on their HTTP port:

| Service | HTTP Port | gRPC Port |
|---------|-----------|-----------|
| Gateway | 8080 | - |
| Identity | 8081 | 9081 |
| Orchestrator | 8082 | 9082 |
| Telemetry | 8083 | 9083 |
| Control Plane | 8084 | 9084 |
| Sidecar | 8090 | - |

```bash
curl http://localhost:8080/health  # {"status":"ok"}
```

## Environment Variables

See CLAUDE.md for the full list. Key variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| ARGUS_ENV | Yes | development | Environment name |
| ARGUS_DB_DSN | Yes | - | PostgreSQL connection string |
| ARGUS_NATS_URL | Yes | nats://localhost:4222 | NATS server URL |
| ARGUS_JWT_SECRET | Yes (prod) | dev | JWT signing secret |
| ARGUS_AIR_GAP | No | false | Enable air-gap mode |
| ARGUS_TENANT_ENFORCEMENT | No | strict | Tenant isolation enforcement |
