# API Reference

All REST endpoints follow the convention `GET/POST/PUT/DELETE /api/v1/{resource}`.

## Authentication

All API requests (except `/health` and `/api/v1/auth/token`) require:

| Header | Required | Description |
|---|---|---|
| `X-Tenant-ID` | Yes | Tenant identifier for multi-tenancy isolation |
| `Authorization` | No (dev mode) | `Bearer <JWT>` token. In dev mode, omitting this creates synthetic admin claims |

### Generate a Token (Dev Mode)

```bash
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"subject":"admin","tenant_id":"default","role":"admin"}'
```

Response:
```json
{
  "data": {
    "token": "eyJhbGci...",
    "expires_at": "2026-03-23T12:00:00Z"
  }
}
```

## Response Format

### Success Response
```json
{
  "data": { ... },
  "meta": {
    "tenant_id": "default",
    "request_id": "uuid"
  }
}
```

### Error Response
```json
{
  "error": {
    "code": "AGENT_NOT_FOUND",
    "message": "Agent with ID 'xyz' not found"
  }
}
```

### Pagination
Cursor-based via query parameters:
```
GET /api/v1/agents?after=<cursor>&limit=20
```

## Common Error Codes

| Code | HTTP Status | Description |
|---|---|---|
| `TENANT_REQUIRED` | 400 | Missing `X-Tenant-ID` header |
| `UNAUTHORIZED` | 401 | Invalid or expired JWT token |
| `FORBIDDEN` | 403 | Cross-tenant access attempted |
| `AGENT_NOT_FOUND` | 404 | Agent ID does not exist |
| `TASK_NOT_FOUND` | 404 | Task ID does not exist |
| `VALIDATION_ERROR` | 400 | Invalid request body |
| `METHOD_NOT_ALLOWED` | 405 | HTTP method not supported |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

---

## Gateway (`:8080`)

The gateway is the single entry point for all API traffic. It routes requests to backend services based on path prefix.

### Routing Table

| Path Prefix | Backend Service | Env Var |
|---|---|---|
| `/api/v1/agents` | Orchestrator `:8082` | `ARGUS_BACKEND_ORCHESTRATOR` |
| `/api/v1/tasks` | Orchestrator `:8082` | `ARGUS_BACKEND_ORCHESTRATOR` |
| `/api/v1/telemetry` | Telemetry `:8083` | `ARGUS_BACKEND_TELEMETRY` |
| `/api/v1/identity` | Identity `:8081` | `ARGUS_BACKEND_IDENTITY` |
| `/api/v1/*` (catch-all) | Control Plane `:8084` | `ARGUS_BACKEND_CONTROL_PLANE` |
| `/ws/v1/*` | Control Plane `:8084` | `ARGUS_BACKEND_CONTROL_PLANE` |

### `GET /health`
Returns gateway health status.

---

## Agent Management (Orchestrator)

### `GET /api/v1/agents`
List all agents for the current tenant.

**Response:**
```json
{
  "data": [
    {
      "id": "budget-reconciler",
      "tenant_id": "default",
      "version": "1.2.0",
      "framework": "langchain",
      "capabilities": ["read:budget_db", "write:report_store"],
      "status": "healthy",
      "svid_uri": "spiffe://argus.example.com/tenant/default/agent/budget-reconciler/v1.2.0",
      "last_seen": "2026-03-22T12:00:00Z",
      "node_id": "node-01"
    }
  ],
  "meta": { "tenant_id": "default" }
}
```

### `POST /api/v1/agents`
Register a new agent.

**Request:**
```json
{
  "agent_id": "budget-reconciler",
  "version": "1.2.0",
  "framework": "langchain",
  "capabilities": ["read:budget_db"],
  "node_id": "node-01"
}
```

### `GET /api/v1/agents/{agentId}`
Get details of a specific agent.

### `POST /api/v1/agents/{agentId}/heartbeat`
Send agent heartbeat to maintain "healthy" status.

**Request:**
```json
{
  "status": "healthy"
}
```

### `POST /api/v1/agents/{agentId}/quarantine`
Quarantine an agent (revokes its SVID so it cannot make calls).

**Request:**
```json
{
  "reason": "predicted_failure",
  "probability": 0.92,
  "precursor": "token_escalation"
}
```

---

## Task Management (Orchestrator)

### `GET /api/v1/tasks`
List tasks for the current tenant. Query params: `status`, `agent_id`.

### `POST /api/v1/tasks`
Create a new task.

**Request:**
```json
{
  "agent_id": "budget-reconciler",
  "input_hash": "sha256:abc123..."
}
```

### `GET /api/v1/tasks/{taskId}`
Get task details including status, cost, and token usage.

### `PUT /api/v1/tasks/{taskId}`
Update task status, cost, or token count.

**Request:**
```json
{
  "status": "completed",
  "cost_usd": 0.0042,
  "tokens_used": 1523
}
```

---

## Telemetry Service

### `POST /api/v1/telemetry/spans`
Ingest telemetry spans.

**Request:**
```json
{
  "spans": [
    {
      "span_id": "abc123",
      "trace_id": "trace-456",
      "agent_id": "budget-reconciler",
      "task_id": "task-789",
      "operation_name": "llm.completion",
      "started_at": "2026-03-22T12:00:00Z",
      "duration_ms": 350,
      "tier": 1,
      "attributes": { "model": "gpt-4", "tokens": "523" }
    }
  ]
}
```

### `GET /api/v1/telemetry/spans`
Query spans. Query params: `agent_id`, `trace_id`.

### `POST /api/v1/telemetry/metrics`
Ingest metrics (latency, token counts, etc.).

### `POST /api/v1/telemetry/predict`
Get failure prediction for an agent.

**Request:**
```json
{
  "agent_id": "budget-reconciler",
  "features": {
    "latency_p99_ratio": 3.5,
    "token_velocity": 150.0,
    "retry_rate": 0.15,
    "error_rate_delta": 0.08,
    "context_fill_pct": 0.85,
    "tool_call_depth": 4,
    "consecutive_slow": 3,
    "cost_acceleration": 1.2
  }
}
```

**Response:**
```json
{
  "data": {
    "failure_probability": 0.82,
    "ttf_seconds": 180,
    "precursor_type": "token_escalation",
    "recommendation": "Consider quarantining agent"
  }
}
```

### `GET /api/v1/telemetry/catalog`
List discovered data sources.

### `GET /api/v1/telemetry/residency/proof`
Generate a data residency attestation (cryptographically signed proof that data has not left the configured region).

### `GET /api/v1/telemetry/quality`
Get data quality score. Query param: `agent_id`.

---

## Identity Service

### `POST /api/v1/identity/svid`
Issue a new SVID certificate for an agent.

**Request:**
```json
{
  "tenant_id": "default",
  "agent_id": "budget-reconciler",
  "version": "1.2.0"
}
```

**Response:**
```json
{
  "data": {
    "spiffe_id": "spiffe://argus.example.com/tenant/default/agent/budget-reconciler/v1.2.0",
    "certificate": "-----BEGIN CERTIFICATE-----\n...",
    "private_key": "-----BEGIN EC PRIVATE KEY-----\n...",
    "expires_at": "2026-03-22T13:00:00Z"
  }
}
```

### `POST /api/v1/identity/validate`
Validate a SPIFFE ID.

### `POST /api/v1/identity/revoke`
Revoke an agent's SVID (used during quarantine).

### `GET /api/v1/identity/crl`
Get the Certificate Revocation List.

### `GET /api/v1/identity/ca`
Get the CA certificate (public endpoint, no tenant context required).

---

## Control Plane - Core

### Alerts

#### `GET /api/v1/alerts`
List alerts for the current tenant.

#### `GET /api/v1/alerts/{alertId}`
Get alert details.

### Audit Log

#### `GET /api/v1/audit`
Get audit log entries.

#### `GET /api/v1/audit/logs`
Retrieve audit logs with filtering.

#### `GET /api/v1/audit/search`
Full-text search across audit logs.

#### `GET /api/v1/audit/stats`
Get audit log statistics (event counts, top actors).

### Policies

#### `GET /api/v1/policies`
List tenant policies.

#### `POST /api/v1/policies`
Create a new policy rule.

**Request:**
```json
{
  "id": "no-external-api",
  "subject": "agent:*",
  "action": "call:external_api",
  "resource": "*",
  "effect": "deny"
}
```

#### `POST /api/v1/policies/evaluate`
Evaluate a policy decision.

**Request:**
```json
{
  "subject": "agent:budget-reconciler",
  "action": "call:external_api",
  "resource": "https://api.example.com"
}
```

### Metrics

#### `GET /api/v1/metrics`
Get platform-wide metrics (total agents, active tasks, cost, alert count).

---

## Control Plane - Evaluations

### `POST /api/v1/evals/suites`
Create an evaluation suite (benchmark).

### `GET /api/v1/evals/suites`
List evaluation suites.

### `GET /api/v1/evals/suites/{id}`
Get evaluation suite details.

### `POST /api/v1/evals/suites/{id}/run`
Execute an evaluation run against an agent.

### `GET /api/v1/evals/runs`
List evaluation runs.

### `GET /api/v1/evals/runs/{id}`
Get run results with per-case scores.

---

## Control Plane - Human Feedback

### `POST /api/v1/feedback`
Submit feedback on an agent response.

**Request:**
```json
{
  "agent_id": "budget-reconciler",
  "task_id": "task-789",
  "rating": 4,
  "comment": "Good output but slow",
  "tags": ["accuracy", "latency"]
}
```

### `GET /api/v1/feedback`
List feedback entries.

### `GET /api/v1/feedback/summary`
Get feedback summary with average ratings and tag distribution.

---

## Control Plane - Guardrails

### `POST /api/v1/guardrails/rules`
Create a guardrail rule.

### `GET /api/v1/guardrails/rules`
List guardrail rules.

### `GET /api/v1/guardrails/violations`
List guardrail violations.

### `GET /api/v1/guardrails/stats`
Get guardrail statistics (violation counts, top rules triggered).

---

## Control Plane - Prompt Management

### `POST /api/v1/prompts`
Create a new prompt template.

### `GET /api/v1/prompts`
List prompt templates.

### `GET /api/v1/prompts/{id}`
Get prompt details.

### `POST /api/v1/prompts/{id}/versions`
Create a new version of a prompt.

### `GET /api/v1/prompts/{id}/versions`
List prompt versions.

### `PUT /api/v1/prompts/{id}/active`
Set the active version for a prompt.

### `GET /api/v1/prompts/{id}/metrics`
Get usage metrics for a prompt.

---

## Control Plane - RAG Analytics

### `GET /api/v1/rag/retrievals`
List RAG retrieval events.

### `GET /api/v1/rag/sources`
List RAG data sources.

### `GET /api/v1/rag/quality`
Get RAG quality metrics (relevance scores, retrieval latency).

---

## Control Plane - Compliance

### `POST /api/v1/compliance/reports`
Generate a compliance report.

**Request:**
```json
{
  "profile": "eu-gdpr",
  "period_start": "2026-01-01",
  "period_end": "2026-03-22"
}
```

### `GET /api/v1/compliance/reports`
List compliance reports.

### `GET /api/v1/compliance/reports/{id}`
Get report details with findings and attestations.

---

## Control Plane - Observability

### Traces

#### `GET /api/v1/traces`
List distributed traces.

#### `GET /api/v1/traces/{traceId}`
Get trace details with all spans.

### Data Quality

#### `GET /api/v1/dataquality/rules`
List data quality rules.

#### `GET /api/v1/dataquality/scores`
Get quality scores.

#### `GET /api/v1/dataquality/violations`
List quality violations.

### Data Catalog

#### `GET /api/v1/catalog/sources`
List discovered data sources.

#### `GET /api/v1/catalog/sources/{sourceId}`
Get source details.

#### `GET /api/v1/catalog/lineage`
Get data lineage graph.

### Cost Governance

#### `GET /api/v1/costs/breakdown`
Cost breakdown by agent/tenant.

#### `GET /api/v1/costs/trends`
Cost trends over time.

#### `GET /api/v1/costs/agents/{agentId}`
Agent-specific cost history.

#### `GET /api/v1/costs/budgets`
List budgets.

#### `GET /api/v1/costs/anomalies`
Detect cost anomalies.

### SLO Management

#### `GET /api/v1/slos`
List SLO definitions.

#### `GET /api/v1/slos/{sloId}`
Get SLO details.

#### `GET /api/v1/slos/status`
Get current status of all SLOs (error budget remaining, compliance %).

---

## WebSocket Streams

### `WS /ws/v1/agents/stream`
Real-time agent status events (registration, heartbeat, quarantine).

### `WS /ws/v1/telemetry/stream`
Real-time telemetry events (spans, metrics, alerts).

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/v1/agents/stream');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Agent event:', data);
};
```

Headers required: `X-Tenant-ID` (passed as query param or via initial HTTP upgrade headers).

---

## gRPC Services

All gRPC services use the `middleware.TenantUnaryInterceptor()` for tenant context extraction.

### AgentService (`:9082`)

```protobuf
service AgentService {
  rpc RegisterAgent(RegisterAgentRequest) returns (RegisterAgentResponse);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse);
  rpc GetAgent(GetAgentRequest) returns (GetAgentResponse);
}
```

### OrchestrationService (`:9082`)

```protobuf
service OrchestrationService {
  rpc SubmitTask(SubmitTaskRequest) returns (SubmitTaskResponse);
  rpc GetTask(GetTaskRequest) returns (GetTaskResponse);
  rpc UpdateTaskStatus(UpdateTaskStatusRequest) returns (UpdateTaskStatusResponse);
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
}
```

### IdentityService (`:9081`)

```protobuf
service IdentityService {
  rpc CreateSVID(CreateSVIDRequest) returns (CreateSVIDResponse);
  rpc RenewSVID(RenewSVIDRequest) returns (RenewSVIDResponse);
  rpc RevokeSVID(RevokeSVIDRequest) returns (RevokeSVIDResponse);
  rpc ValidateSVID(ValidateSVIDRequest) returns (ValidateSVIDResponse);
}
```

### TelemetryService (`:9083`)

```protobuf
service TelemetryService {
  rpc IngestSpans(IngestSpansRequest) returns (IngestSpansResponse);
  rpc IngestMetrics(IngestMetricsRequest) returns (IngestMetricsResponse);
  rpc IngestEvents(IngestEventsRequest) returns (IngestEventsResponse);
  rpc QuerySpans(QuerySpansRequest) returns (QuerySpansResponse);
}
```

### PredictorService (`:9083`)

```protobuf
service PredictorService {
  rpc Predict(PredictRequest) returns (PredictResponse);
  rpc ListAlerts(ListAlertsRequest) returns (ListAlertsResponse);
  rpc AcknowledgeAlert(AcknowledgeAlertRequest) returns (AcknowledgeAlertResponse);
}
```
