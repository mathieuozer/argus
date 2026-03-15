# Incident Response Runbook

## Overview
This runbook covers incident response procedures for the Argus platform.

## Severity Levels

| Level | Description | Response Time |
|-------|-------------|---------------|
| P1 - Critical | Platform down, data loss risk | 15 minutes |
| P2 - High | Service degraded, partial outage | 1 hour |
| P3 - Medium | Non-critical feature impacted | 4 hours |
| P4 - Low | Cosmetic, minor issues | Next business day |

## Procedures

### Service Health Check
```bash
# Check all service health endpoints
for port in 8080 8081 8082 8083 8084; do
  curl -s http://localhost:$port/health | jq .
done
```

### Database Recovery
_TODO: Document database recovery procedures._

### NATS Recovery
_TODO: Document NATS JetStream recovery procedures._

### Certificate Rotation
_TODO: Document emergency certificate rotation procedures._
