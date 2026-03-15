# ADR 001: Sidecar-First Architecture

## Status
Accepted

## Context
Argus needs to monitor AI agents across diverse frameworks (LangChain, AutoGen, CrewAI, custom) without requiring code changes. Enterprises need a non-invasive adoption path.

## Decision
The sidecar proxy is the primary instrumentation mechanism. It deploys alongside any agent process and intercepts I/O at the network layer, providing auto-discovery, telemetry emission, and identity management without code changes.

Native SDKs (Go, Python, TypeScript) are optional for richer instrumentation (custom spans, business events).

## Consequences
- **Positive:** Zero-code adoption path, framework-agnostic, easy enterprise rollout
- **Positive:** Agent teams don't need to learn Argus internals
- **Negative:** Network-level interception has less visibility than code-level instrumentation
- **Negative:** Sidecar adds a process per agent (resource overhead)
- **Mitigation:** SDKs provide opt-in deeper instrumentation when needed
