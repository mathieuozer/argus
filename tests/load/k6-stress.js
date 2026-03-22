// k6 Stress Test for Argus Platform
// Tests system behavior under heavy load to find breaking points.
//
// Usage:
//   k6 run tests/load/k6-stress.js
//   k6 run --env BASE_URL=http://staging:8080 tests/load/k6-stress.js

import http from "k6/http";
import { check, group, sleep } from "k6";
import { Rate, Trend, Counter } from "k6/metrics";

const errorRate = new Rate("errors");
const latencyTrend = new Trend("request_latency", true);
const requestsCounter = new Counter("total_requests");

const BASE_URL = __ENV.BASE_URL || "http://localhost:8084";
const TENANT_ID = __ENV.TENANT_ID || "tenant-acme";

export const options = {
  stages: [
    { duration: "1m", target: 20 }, // Ramp to 20 users
    { duration: "2m", target: 50 }, // Ramp to 50 users
    { duration: "3m", target: 100 }, // Ramp to 100 users
    { duration: "2m", target: 100 }, // Hold at 100 users
    { duration: "1m", target: 200 }, // Push to 200 users
    { duration: "2m", target: 200 }, // Hold at 200
    { duration: "1m", target: 0 }, // Ramp down
  ],
  thresholds: {
    http_req_duration: ["p(95)<2000", "p(99)<5000"], // Relaxed for stress
    errors: ["rate<0.15"], // Allow up to 15% errors under stress
    http_req_failed: ["rate<0.15"],
  },
};

const headers = {
  "Content-Type": "application/json",
  "X-Tenant-ID": TENANT_ID,
};

const TENANTS = [
  "tenant-acme",
  "tenant-globex",
  "ministry-finance-tr",
  "tenant-defense-uk",
];

export default function () {
  const tenant = TENANTS[Math.floor(Math.random() * TENANTS.length)];
  const reqHeaders = { ...headers, "X-Tenant-ID": tenant };

  // Mix of read-heavy and write operations
  const scenario = Math.random();

  if (scenario < 0.4) {
    // 40%: Read dashboard data
    group("Dashboard Reads", function () {
      const res = http.get(`${BASE_URL}/api/v1/dashboard/alerts`, {
        headers: reqHeaders,
      });
      check(res, { "dashboard 200": (r) => r.status === 200 });
      errorRate.add(res.status !== 200);
      latencyTrend.add(res.timings.duration);
      requestsCounter.add(1);
    });
  } else if (scenario < 0.7) {
    // 30%: Read traces
    group("Trace Reads", function () {
      const res = http.get(`${BASE_URL}/api/v1/traces`, {
        headers: reqHeaders,
      });
      check(res, { "traces 200": (r) => r.status === 200 });
      errorRate.add(res.status !== 200);
      latencyTrend.add(res.timings.duration);
      requestsCounter.add(1);
    });
  } else if (scenario < 0.85) {
    // 15%: Read SLOs + evals
    group("SLO + Eval Reads", function () {
      const responses = http.batch([
        ["GET", `${BASE_URL}/api/v1/slos`, null, { headers: reqHeaders }],
        [
          "GET",
          `${BASE_URL}/api/v1/evals/suites`,
          null,
          { headers: reqHeaders },
        ],
      ]);
      responses.forEach((res) => {
        errorRate.add(res.status !== 200);
        latencyTrend.add(res.timings.duration);
        requestsCounter.add(1);
      });
    });
  } else if (scenario < 0.95) {
    // 10%: Health checks (should always be fast)
    group("Health Checks", function () {
      const responses = http.batch([
        ["GET", `${BASE_URL}/health/live`, null, {}],
        ["GET", `${BASE_URL}/health/ready`, null, {}],
      ]);
      responses.forEach((res) => {
        check(res, { "health 200": (r) => r.status === 200 });
        errorRate.add(res.status !== 200);
        requestsCounter.add(1);
      });
    });
  } else {
    // 5%: Write feedback
    group("Write Feedback", function () {
      const payload = JSON.stringify({
        agent_id: `agent-${Math.floor(Math.random() * 100)}`,
        task_id: `task-${Date.now()}`,
        rating: Math.floor(Math.random() * 5) + 1,
        comment: "Load test feedback",
        source: "k6-stress",
      });
      const res = http.post(`${BASE_URL}/api/v1/feedback`, payload, {
        headers: reqHeaders,
      });
      check(res, {
        "feedback created": (r) => r.status === 200 || r.status === 201,
      });
      errorRate.add(res.status >= 400);
      latencyTrend.add(res.timings.duration);
      requestsCounter.add(1);
    });
  }

  sleep(0.1 + Math.random() * 0.4); // 100-500ms between requests
}
