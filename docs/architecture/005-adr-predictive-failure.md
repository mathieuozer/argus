# ADR 005: Predictive Failure Analysis Engine

## Status
Accepted

## Context
AI agents fail in characteristic patterns: context window exhaustion, infinite tool call loops, upstream API degradation, and cost runaway. By the time an agent reports failure, damage (cost, time, data) has already occurred. Enterprise clients need advance warning to prevent cascading failures across agent fleets.

## Decision
Implement a predictive failure analysis engine that watches the telemetry stream for known failure precursors and fires pre-failure alerts with probability scores and estimated time-to-failure.

### Architecture
- **Feature extraction:** 8 real-time features computed from sliding windows over telemetry data
- **Dual inference path:**
  - **Local heuristic model:** Sigmoid-based rules running in-process (zero external dependencies, works air-gapped)
  - **ONNX model:** Trained offline on anonymized telemetry, served via Python microservice for higher accuracy
- **Alert pipeline:** Predictions above threshold (p >= 0.5) fire predictive alerts through the tenant's escalation chain

### Feature Set
| Feature | Window | Precursor |
|---|---|---|
| `latency_p99_ratio` | 5 min | Latency spike / OOM / context overflow |
| `token_velocity` | 10 tasks | Token escalation / stuck in loop |
| `retry_rate` | 5 min | Upstream degradation |
| `error_rate_delta` | vs 1h baseline | Model or tool failure |
| `context_fill_pct` | current | Context window exhaustion |
| `tool_call_depth` | current task | Infinite recursion |
| `consecutive_slow` | recent calls | Systematic degradation |
| `cost_acceleration` | per-task rate | Cost runaway |

### Precursor Types
- **latency_spike:** p99/p50 ratio > 3 sustained > 30s. Indicates OOM, context overflow, or upstream degradation.
- **token_escalation:** Context fill > 70% with high token velocity. Agent stuck in loop or approaching context limit.
- **retry_storm:** Retry rate > 30% with rising error rate. Upstream dependency failing.
- **cost_runaway:** Cost acceleration > 2x with high token velocity. Agent consuming resources without useful output.

### Model Retraining
The ONNX model is retrained weekly from anonymized Tier 1 telemetry data. The heuristic model serves as fallback and baseline. Model accuracy is tracked via precision/recall metrics on resolved alerts.

## Consequences
- **Positive:** Prevents agent failures before they cause damage
- **Positive:** Heuristic fallback ensures prediction works in all deployment modes
- **Positive:** ONNX inference is fast (~1ms) and doesn't require Python in production
- **Positive:** Feature set is extensible for new failure patterns
- **Negative:** False positives may cause alert fatigue (mitigated by tunable thresholds)
- **Negative:** Heuristic model has lower accuracy than trained model
- **Negative:** Requires telemetry stream processing with low latency
- **Mitigation:** Alert status includes "false_positive" for feedback loop
