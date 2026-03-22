// k6 Soak Test for Argus Platform
// Tests system stability under sustained moderate load for extended periods.
// Detects memory leaks, connection pool exhaustion, and resource degradation.
//
// Usage:
//   k6 run tests/load/k6-soak.js
//   k6 run --env BASE_URL=http://staging:8080 --env DURATION=30m tests/load/k6-soak.js

import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const errorRate = new Rate("errors");
const latencyTrend = new Trend("request_latency", true);

const BASE_URL = __ENV.BASE_URL || "http://localhost:8084";
const TENANT_ID = __ENV.TENANT_ID || "tenant-acme";
const DURATION = __ENV.DURATION || "10m";

export const options = {
  stages: [
    { duration: "2m", target: 30 }, // Ramp up
    { duration: DURATION, target: 30 }, // Sustained load
    { duration: "1m", target: 0 }, // Ramp down
  ],
  thresholds: {
    http_req_duration: ["p(95)<1000", "p(99)<2000"],
    errors: ["rate<0.02"], // Stricter for soak: < 2% errors
    http_req_failed: ["rate<0.02"],
  },
};

const headers = {
  "Content-Type": "application/json",
  "X-Tenant-ID": TENANT_ID,
};

const ENDPOINTS = [
  "/health/live",
  "/health/ready",
  "/metrics",
  "/api/v1/dashboard/alerts",
  "/api/v1/traces",
  "/api/v1/slos",
  "/api/v1/evals/suites",
  "/api/v1/guardrails/rules",
  "/api/v1/prompts",
  "/api/v1/rag/sources",
  "/api/v1/compliance/reports",
];

export default function () {
  const endpoint = ENDPOINTS[Math.floor(Math.random() * ENDPOINTS.length)];
  const isHealthOrMetrics =
    endpoint.startsWith("/health") || endpoint === "/metrics";

  const res = http.get(`${BASE_URL}${endpoint}`, {
    headers: isHealthOrMetrics ? {} : headers,
  });

  check(res, {
    "status is 200": (r) => r.status === 200,
    "response time < 1s": (r) => r.timings.duration < 1000,
  });

  errorRate.add(res.status !== 200);
  latencyTrend.add(res.timings.duration);

  sleep(0.5 + Math.random() * 1.0); // 500ms-1.5s between requests
}
