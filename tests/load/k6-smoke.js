// k6 Smoke Test for Argus Platform
// Validates basic functionality under minimal load.
//
// Usage:
//   k6 run tests/load/k6-smoke.js
//   k6 run --env BASE_URL=http://staging:8080 tests/load/k6-smoke.js

import http from "k6/http";
import { check, group, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

// Custom metrics
const errorRate = new Rate("errors");
const latencyTrend = new Trend("request_latency", true);

// Configuration
const BASE_URL = __ENV.BASE_URL || "http://localhost:8084";
const TENANT_ID = __ENV.TENANT_ID || "tenant-acme";

export const options = {
  stages: [
    { duration: "30s", target: 5 }, // Ramp up to 5 users
    { duration: "1m", target: 5 }, // Hold at 5 users
    { duration: "10s", target: 0 }, // Ramp down
  ],
  thresholds: {
    http_req_duration: ["p(95)<500", "p(99)<1000"], // p95 < 500ms, p99 < 1s
    errors: ["rate<0.05"], // Error rate < 5%
    http_req_failed: ["rate<0.05"],
  },
};

const headers = {
  "Content-Type": "application/json",
  "X-Tenant-ID": TENANT_ID,
};

export default function () {
  group("Health Checks", function () {
    const liveRes = http.get(`${BASE_URL}/health/live`);
    check(liveRes, {
      "liveness returns 200": (r) => r.status === 200,
    });
    errorRate.add(liveRes.status !== 200);
    latencyTrend.add(liveRes.timings.duration);

    const readyRes = http.get(`${BASE_URL}/health/ready`);
    check(readyRes, {
      "readiness returns 200": (r) => r.status === 200,
    });
    errorRate.add(readyRes.status !== 200);
  });

  group("Metrics Endpoint", function () {
    const metricsRes = http.get(`${BASE_URL}/metrics`);
    check(metricsRes, {
      "metrics returns 200": (r) => r.status === 200,
      "metrics contains counters": (r) =>
        r.body.includes("http_requests_total"),
    });
    errorRate.add(metricsRes.status !== 200);
  });

  group("Dashboard API", function () {
    const alertsRes = http.get(`${BASE_URL}/api/v1/dashboard/alerts`, {
      headers,
    });
    check(alertsRes, {
      "alerts returns 200": (r) => r.status === 200,
      "alerts returns JSON": (r) =>
        r.headers["Content-Type"].includes("application/json"),
    });
    errorRate.add(alertsRes.status !== 200);
    latencyTrend.add(alertsRes.timings.duration);
  });

  group("Trace API", function () {
    const traceRes = http.get(`${BASE_URL}/api/v1/traces`, { headers });
    check(traceRes, {
      "traces returns 200": (r) => r.status === 200,
    });
    errorRate.add(traceRes.status !== 200);
    latencyTrend.add(traceRes.timings.duration);
  });

  group("SLO API", function () {
    const sloRes = http.get(`${BASE_URL}/api/v1/slos`, { headers });
    check(sloRes, {
      "SLOs returns 200": (r) => r.status === 200,
    });
    errorRate.add(sloRes.status !== 200);
    latencyTrend.add(sloRes.timings.duration);
  });

  sleep(1);
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: " ", enableColors: true }),
  };
}

function textSummary(data, opts) {
  const lines = [];
  lines.push("\n=== Argus Smoke Test Results ===\n");

  if (data.metrics.http_req_duration) {
    const d = data.metrics.http_req_duration.values;
    lines.push(`  HTTP Duration (avg): ${d.avg.toFixed(2)}ms`);
    lines.push(`  HTTP Duration (p95): ${d["p(95)"].toFixed(2)}ms`);
    lines.push(`  HTTP Duration (p99): ${d["p(99)"].toFixed(2)}ms`);
  }

  if (data.metrics.http_reqs) {
    lines.push(
      `  Total Requests: ${data.metrics.http_reqs.values.count}`
    );
    lines.push(
      `  Request Rate: ${data.metrics.http_reqs.values.rate.toFixed(2)}/s`
    );
  }

  if (data.metrics.errors) {
    lines.push(
      `  Error Rate: ${(data.metrics.errors.values.rate * 100).toFixed(2)}%`
    );
  }

  lines.push("");
  return lines.join("\n");
}
