# ADR 003: SPIFFE/SPIRE for Agent Identity

## Status
Accepted

## Context
AI agents need cryptographically verifiable identities for mTLS communication with the platform. Traditional approaches (static API keys, shared secrets) are problematic for enterprise/government deployments: keys can leak, rotation is manual, and there's no standard for agent identity across heterogeneous frameworks.

## Decision
Every agent receives a SPIFFE Verifiable Identity Document (SVID) - a short-lived X.509 certificate (1-hour TTL) auto-rotated by the sidecar. The SPIFFE ID encodes the agent's organizational hierarchy:

```
spiffe://argus.{domain}/tenant/{tenant_id}/agent/{agent_id}/v{version}
```

All service-to-service and agent-to-platform communication uses mTLS with these certificates. The internal Certificate Authority (CA) issues and manages SVIDs. HashiCorp Vault stores the root CA key material.

Key properties:
- **Short-lived:** 1-hour TTL means compromised certs have minimal blast radius
- **Auto-rotated:** Sidecar handles renewal transparently
- **Hierarchical:** SPIFFE ID encodes tenant and agent identity
- **Revocable:** OCSP responder and CRL generation for immediate revocation

## Consequences
- **Positive:** No static API keys or shared secrets in the system
- **Positive:** mTLS provides mutual authentication and encryption
- **Positive:** SPIFFE is an industry standard (CNCF project), interoperable
- **Positive:** Cert revocation enables agent quarantine (immediate isolation)
- **Negative:** PKI infrastructure complexity (CA, OCSP, CRL management)
- **Negative:** Vault dependency for root key storage (mitigated by Vault dev mode for air-gap)
- **Negative:** Clock skew sensitivity with short-lived certs
- **Mitigation:** NTP requirements documented in deployment runbooks
