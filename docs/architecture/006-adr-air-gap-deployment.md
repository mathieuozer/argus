# ADR 006: Air-Gap Deployment Support

## Status
Accepted

## Context
Government and defense clients operate in environments with no outbound internet connectivity (air-gapped networks). The platform must function fully offline without any cloud dependencies, external package registries, or phone-home behavior.

## Decision
Argus supports three deployment modes, with air-gap as a first-class target:

### Deployment Modes
1. **Cloud/Connected:** Standard deployment with container registries, external dependencies, automatic updates
2. **Hybrid:** Connected to internal network only, no public internet. Uses internal mirrors for dependencies.
3. **Air-Gap:** Zero outbound network. All dependencies pre-loaded. No license validation, telemetry, or update checks.

### Air-Gap Architecture
- **Container images:** Pre-built and loaded via `docker load` or `ctr images import`. No pulls from external registries.
- **Go binaries:** Statically compiled (`CGO_ENABLED=0`), single binary per service. No dynamic linking.
- **ONNX model:** Bundled with the telemetry service image. Local heuristic model as fallback.
- **Vault:** Runs in dev mode with local storage (no external backend).
- **NATS:** Embedded JetStream, single binary, no external dependencies.
- **PostgreSQL:** Local instance with pre-loaded schema.
- **Dashboard:** Pre-built static files served by gateway. No CDN or external asset loading.
- **Time sync:** NTP within the air-gapped network (required for cert validation).

### Configuration
Air-gap mode is activated via `ARGUS_AIR_GAP=true`. When enabled:
- All outbound HTTP clients are disabled
- Package version checks are skipped
- Telemetry self-reporting is local-only
- Model retraining uses only local data

### Kubernetes Overlay
The `air-gap` Kustomize overlay sets `imagePullPolicy: Never` on all containers and uses a local registry prefix. A pre-deployment script (`deployments/scripts/air-gap-load.sh`) loads all images from a tarball.

## Consequences
- **Positive:** Full functionality in classified/disconnected environments
- **Positive:** No cloud vendor lock-in
- **Positive:** Simplified security audit (no outbound traffic to analyze)
- **Negative:** Updates require physical media transfer and redeployment
- **Negative:** ONNX model cannot be retrained on fresh data automatically
- **Negative:** Larger deployment artifact (all dependencies bundled)
- **Mitigation:** Versioned release tarballs with checksums for integrity verification
