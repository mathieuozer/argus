# Troubleshooting

Common issues and their solutions when running Argus.

## Docker Compose Issues

### NATS container is unhealthy

**Symptom:** `dependency failed to start: container docker-nats-1 is unhealthy`

**Cause:** The NATS image may not include `wget` or `curl` for health checks.

**Fix:** Use `nats:2.10-alpine` (which includes `wget`) instead of `nats:latest`:
```yaml
nats:
  image: nats:2.10-alpine
```

### Sidecar fails: "/usr/local/bin/sidecar: is a directory"

**Symptom:** OCI runtime error about sidecar being a directory.

**Cause:** The Dockerfile builds the binary to `/app/sidecar` but the source directory `sidecar/` is already at that path. The `COPY` command copies a directory instead of a binary.

**Fix:** Build to a different path in the Dockerfile:
```dockerfile
RUN go build -o /app/bin/sidecar ./cmd/main
COPY --from=builder /app/bin/sidecar /usr/local/bin/sidecar
```

### Gateway returns 502 Bad Gateway

**Symptom:** All API calls through the gateway return 502.

**Cause:** Gateway proxy routes use `localhost` URLs, but in Docker each service runs in its own container. `localhost` inside the gateway container refers to itself, not the backend services.

**Fix:** Set backend URLs via environment variables in docker-compose.yml:
```yaml
gateway:
  environment:
    ARGUS_BACKEND_ORCHESTRATOR: http://orchestrator:8082
    ARGUS_BACKEND_TELEMETRY: http://telemetry:8083
    ARGUS_BACKEND_IDENTITY: http://identity:8081
    ARGUS_BACKEND_CONTROL_PLANE: http://control-plane:8084
```

### Dashboard shows "Unexpected token '<', <!doctype..."

**Symptom:** Dashboard pages show JSON parse errors.

**Cause:** The dashboard's API client uses relative URLs (`/api/v1/...`). Without a proxy, these requests hit the Vite dev server, which returns HTML (the SPA shell) instead of JSON.

**Fix:** Set `ARGUS_API_URL` to enable the Vite proxy:
```yaml
dashboard:
  environment:
    ARGUS_API_URL: http://gateway:8080
```

### Sidecar can't register with orchestrator

**Symptom:** Sidecar logs show `dial tcp [::1]:8082: connect: connection refused`

**Cause:** Sidecar defaults to `localhost:8082` for the orchestrator address.

**Fix:** Set the orchestrator address in docker-compose.yml:
```yaml
sidecar:
  environment:
    ARGUS_ORCHESTRATOR_ADDR: http://orchestrator:8082
```

---

## Service Issues

### "Tenant not found in context"

**Symptom:** API endpoint returns `{"error":{"code":"UNAUTHORIZED","message":"Tenant not found in context"}}`

**Cause:** The endpoint handler calls `tenancy.FromContext(ctx)` but the `TenantHTTP` middleware was not applied to the route.

**Fix:** Ensure the route is wrapped with `middleware.TenantHTTP()`, either:
- At the individual handler level: `mux.Handle("/path", middleware.TenantHTTP(handler))`
- At the middleware chain level (control-plane uses this approach)

**Client fix:** Ensure `X-Tenant-ID` header is present in the request:
```bash
curl -H "X-Tenant-ID: default" http://localhost:8080/api/v1/agents
```

### "X-Tenant-ID header is required"

**Symptom:** API returns 400 with `TENANT_REQUIRED`.

**Cause:** The `X-Tenant-ID` header is missing from the request.

**Fix:** Include the header:
```bash
curl -H "X-Tenant-ID: my-tenant" http://localhost:8080/api/v1/agents
```

In the dashboard, the API client adds this header automatically (default: `"default"`).

### Services use in-memory stores (no PostgreSQL)

**Symptom:** Data is lost on service restart. Logs show `no ARGUS_DB_DSN set, using in-memory stores`.

**Cause:** `ARGUS_DB_DSN` environment variable is not set.

**Fix:** Set the DSN:
```bash
export ARGUS_DB_DSN=postgres://argus:argus@localhost:5432/argus?sslmode=disable
```

### Predictor uses heuristic fallback

**Symptom:** Predictor logs show `No ONNX model at /app/model.onnx, using heuristic fallback`.

**Cause:** No trained ONNX model file is present. This is expected in development.

**Impact:** Predictions still work using rule-based heuristics instead of the ML model. Accuracy may be lower but functionality is preserved.

**Fix (production):** Train and deploy the ONNX model:
```bash
cd services/predictor
python train.py --output model.onnx
# Copy model.onnx to the predictor container
```

---

## Build Issues

### Missing protobuf stubs

**Symptom:** Compilation errors like `cannot find package "github.com/argus-platform/argus/gen/go/..."`

**Cause:** Generated `*.pb.go` files are gitignored. They must be regenerated after cloning.

**Fix:**
```bash
make proto
```

### Go workspace issues

**Symptom:** Module resolution errors across services.

**Fix:** Ensure `go.work` is present at the project root and includes all modules:
```bash
cat go.work
# Should list all service modules
```

If missing, recreate:
```bash
go work init
go work use ./gen/go ./pkg ./sdk/go ./services/control-plane \
  ./services/orchestrator ./services/telemetry ./services/identity \
  ./services/gateway ./sidecar
```

---

## Testing Issues

### Integration tests fail: "connection refused"

**Symptom:** Integration tests can't connect to PostgreSQL or NATS.

**Cause:** Test infrastructure is not running.

**Fix:**
```bash
make test-infra-up    # Start test PostgreSQL (5433) and NATS (4223)
make test-int         # Run integration tests
make test-infra-down  # Cleanup
```

Or use `docker-compose.test.yml`:
```bash
docker compose -f deployments/docker/docker-compose.test.yml up -d --wait
```

### Tests fail with "GOFLAGS: -mod=readonly"

**Symptom:** `go test` fails because it can't download dependencies.

**Cause:** The Makefile sets `GOFLAGS=-mod=readonly` to prevent unexpected dependency changes.

**Fix:** Download dependencies first:
```bash
go mod download
# Then run tests
make test
```

---

## Performance Issues

### High memory usage in orchestrator

**Possible causes:**
- Large number of registered agents in the in-memory registry
- Unbounded task history

**Fix:** Enable PostgreSQL persistence (`ARGUS_DB_DSN`) so the orchestrator uses DB-backed storage instead of in-memory maps.

### Slow telemetry ingestion

**Possible causes:**
- PII scrubbing is CPU-intensive for large payloads
- Predictor model inference latency

**Fix:**
- Scale telemetry service horizontally (stateless, supports multiple replicas)
- Use NATS consumer groups for parallel processing
- Consider disabling PII scrubbing for Tier 1 data (structural metrics only)

---

## Logs

### View structured logs

All services output JSON logs by default. Use `jq` for readable output:

```bash
docker compose -f deployments/docker/docker-compose.yml logs orchestrator | jq .
```

### Enable debug logging

Set `ARGUS_LOG_LEVEL=debug` in the service's environment:
```yaml
orchestrator:
  environment:
    ARGUS_LOG_LEVEL: debug
```

### Log format

Switch to human-readable text format for development:
```yaml
orchestrator:
  environment:
    ARGUS_LOG_FORMAT: text
```
