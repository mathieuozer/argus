# Incident Response Runbook

## Overview
This runbook covers incident response procedures for the Argus platform.

## Severity Levels

| Level | Description | Response Time | Escalation |
|-------|-------------|---------------|------------|
| P1 - Critical | Platform down, data loss risk | 15 minutes | On-call engineer + team lead |
| P2 - High | Service degraded, partial outage | 1 hour | On-call engineer |
| P3 - Medium | Non-critical feature impacted | 4 hours | Engineering team |
| P4 - Low | Cosmetic, minor issues | Next business day | Sprint backlog |

## Quick Diagnostics

### Service Health Check
```bash
# Check all service health endpoints
for svc in gateway:8080 identity:8081 orchestrator:8082 telemetry:8083 control-plane:8084; do
  name=$(echo $svc | cut -d: -f1)
  port=$(echo $svc | cut -d: -f2)
  status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:$port/health)
  echo "$name ($port): $status"
done
```

### Check Pod Status (Kubernetes)
```bash
kubectl get pods -n argus
kubectl get events -n argus --sort-by=.metadata.creationTimestamp | tail -20
```

### Check Logs
```bash
# Docker Compose
docker compose -f deployments/docker/docker-compose.yml logs --tail=100 <service>

# Kubernetes
kubectl logs -n argus -l app=<service> --tail=100
```

## Procedures

### P1: Platform Down

1. **Identify:** Which services are unhealthy?
```bash
kubectl get pods -n argus | grep -v Running
```

2. **Database connectivity:**
```bash
kubectl exec -n argus -it deploy/control-plane -- /bin/sh -c 'nc -z postgres 5432 && echo OK || echo FAIL'
```

3. **NATS connectivity:**
```bash
kubectl exec -n argus -it deploy/telemetry -- /bin/sh -c 'nc -z nats 4222 && echo OK || echo FAIL'
```

4. **Restart failing services:**
```bash
kubectl rollout restart deployment/<service> -n argus
```

5. **If database is down:**
```bash
kubectl describe pod -n argus -l app=postgres
kubectl logs -n argus -l app=postgres --tail=50
# Check PVC status
kubectl get pvc -n argus
```

### P2: Service Degraded

1. **Check resource usage:**
```bash
kubectl top pods -n argus
```

2. **Check for OOMKilled:**
```bash
kubectl get pods -n argus -o json | jq '.items[] | select(.status.containerStatuses[].lastState.terminated.reason == "OOMKilled") | .metadata.name'
```

3. **Scale up if needed:**
```bash
kubectl scale deployment/<service> -n argus --replicas=<N>
```

### Database Recovery

1. **Check PostgreSQL status:**
```bash
kubectl exec -n argus -it statefulset/postgres -- psql -U argus -c "SELECT pg_is_in_recovery();"
```

2. **Verify RLS is enabled:**
```bash
kubectl exec -n argus -it statefulset/postgres -- psql -U argus -c "
  SELECT tablename, rowsecurity FROM pg_tables WHERE schemaname = 'public';
"
```

3. **Point-in-time recovery (if WAL archiving is configured):**
```bash
# Stop services
kubectl scale deployment --all -n argus --replicas=0

# Restore from backup
pg_restore -h <host> -U argus -d argus < backup.dump

# Restart services
kubectl scale deployment --all -n argus --replicas=2
```

### NATS Recovery

1. **Check JetStream status:**
```bash
kubectl exec -n argus -it deploy/nats -- nats server info
kubectl exec -n argus -it deploy/nats -- nats stream ls
```

2. **Check consumer lag:**
```bash
kubectl exec -n argus -it deploy/nats -- nats consumer info ARGUS_TELEMETRY <consumer>
```

3. **Purge stuck messages (last resort):**
```bash
kubectl exec -n argus -it deploy/nats -- nats stream purge ARGUS_TELEMETRY
```

### Certificate Rotation (Emergency)

1. **Revoke compromised SVID:**
```bash
curl -X POST http://localhost:8081/api/v1/identity/revoke \
  -H "Content-Type: application/json" \
  -d '{"spiffe_id": "spiffe://argus.example.com/tenant/<tid>/agent/<aid>/v1"}'
```

2. **Force certificate renewal for all agents:**
```bash
# Restart all sidecars to trigger SVID re-request
kubectl rollout restart daemonset/sidecar -n argus
```

3. **Rotate root CA (extreme case):**
```bash
# This will invalidate ALL existing certificates
# 1. Generate new CA in Vault
# 2. Update CA cert ConfigMap
# 3. Rolling restart all services
kubectl rollout restart deployment --all -n argus
kubectl rollout restart daemonset --all -n argus
```

### Agent Quarantine

When an agent is detected as compromised or malfunctioning:

1. **Revoke the agent's SVID** (immediately blocks all API calls):
```bash
curl -X POST http://localhost:8081/api/v1/identity/revoke \
  -H "Content-Type: application/json" \
  -d '{"spiffe_id": "<agent_svid>", "reason": "compromised"}'
```

2. **Update agent status to quarantined:**
```bash
curl -X PUT http://localhost:8082/api/v1/agents/<agent_id> \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: <tenant_id>" \
  -d '{"status": "quarantined"}'
```

3. **Review agent's recent activity in audit log:**
```bash
curl http://localhost:8084/api/v1/audit?agent_id=<agent_id> \
  -H "X-Tenant-ID: <tenant_id>"
```

## Post-Incident

1. Write incident report within 48 hours
2. Update this runbook with any new procedures discovered
3. Add monitoring/alerting for the failure mode if not already covered
4. Schedule post-mortem meeting within 1 week
