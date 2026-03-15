# ADR 004: Three-Tier Telemetry Data Classification

## Status
Accepted

## Context
Government and regulated enterprise clients have strict data residency and classification requirements. Not all telemetry data has the same sensitivity level. Full agent I/O may contain classified information that must never leave a secure boundary, while latency metrics are safe to aggregate centrally.

## Decision
Every telemetry record is classified at collection time (in the sidecar) into one of three tiers:

### Tier 1 - Structural
Metrics and metadata with no business content: `latency_ms`, `token_count`, `error_code`, `agent_id`, timestamps, tool call names (without parameters).

**Handling:** Exits the node freely. Can be stored in shared infrastructure. Safe for cross-tenant aggregation (with tenant isolation).

### Tier 2 - Sensitive
Data that may contain business context: task descriptions, tool call parameters, partial outputs, intermediate reasoning steps.

**Handling:** Stays within tenant boundary. Encrypted at rest (AES-256-GCM). PII-scrubbed before storage using regex patterns (email, IP, credit card, IBAN, phone) and NER models. Queryable from tenant's dashboard only.

### Tier 3 - Restricted
Full agent I/O, user-supplied context, complete conversation histories, document contents.

**Handling:** Never leaves the originating node/site. Stored locally with full disk encryption. Queryable only via on-prem dashboard with additional auth. Zero bytes cross the air-gap boundary. Auto-purged per retention policy.

Classification is configured per-tenant in their compliance profile. Government tenants (gov-tr, fedramp) default to Tier 3 for all data.

## Consequences
- **Positive:** Enables deployment in classified environments
- **Positive:** Granular control matches regulatory requirements (GDPR Art. 25, KVKK, FedRAMP)
- **Positive:** Tier 1 data enables useful analytics without exposing sensitive content
- **Negative:** Classification logic must be maintained as new telemetry types are added
- **Negative:** Tier 3 data cannot be used for platform-wide model training
- **Negative:** PII scrubbing adds processing latency to Tier 2 data
- **Mitigation:** Classification rules are declarative, configured per compliance profile
