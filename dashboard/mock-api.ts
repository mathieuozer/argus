import type { Plugin } from 'vite';

const mockAgents = [
  {
    id: 'budget-reconciler',
    tenant_id: 'ministry-finance-tr',
    version: '2.1.0',
    framework: 'langchain',
    capabilities: ['read:budget_db', 'write:report_store'],
    status: 'healthy',
    svid_uri: 'spiffe://argus.local/tenant/ministry-finance-tr/agent/budget-reconciler/v2',
    last_seen: new Date(Date.now() - 30000).toISOString(),
    node_id: 'node-eu-west-1a',
  },
  {
    id: 'doc-classifier',
    tenant_id: 'ministry-finance-tr',
    version: '1.4.2',
    framework: 'autogen',
    capabilities: ['read:document_store', 'classify:documents'],
    status: 'healthy',
    svid_uri: 'spiffe://argus.local/tenant/ministry-finance-tr/agent/doc-classifier/v1',
    last_seen: new Date(Date.now() - 15000).toISOString(),
    node_id: 'node-eu-west-1b',
  },
  {
    id: 'fraud-detector',
    tenant_id: 'ministry-finance-tr',
    version: '3.0.1',
    framework: 'custom',
    capabilities: ['read:transactions', 'write:alerts', 'read:watchlists'],
    status: 'degraded',
    svid_uri: 'spiffe://argus.local/tenant/ministry-finance-tr/agent/fraud-detector/v3',
    last_seen: new Date(Date.now() - 120000).toISOString(),
    node_id: 'node-eu-west-1a',
  },
  {
    id: 'report-generator',
    tenant_id: 'ministry-finance-tr',
    version: '1.0.0',
    framework: 'crewai',
    capabilities: ['read:budget_db', 'write:report_store', 'send:email'],
    status: 'healthy',
    svid_uri: 'spiffe://argus.local/tenant/ministry-finance-tr/agent/report-generator/v1',
    last_seen: new Date(Date.now() - 5000).toISOString(),
    node_id: 'node-eu-west-1c',
  },
  {
    id: 'compliance-checker',
    tenant_id: 'ministry-finance-tr',
    version: '2.3.0',
    framework: 'langchain',
    capabilities: ['read:regulations', 'audit:transactions'],
    status: 'failed',
    svid_uri: 'spiffe://argus.local/tenant/ministry-finance-tr/agent/compliance-checker/v2',
    last_seen: new Date(Date.now() - 600000).toISOString(),
    node_id: 'node-eu-west-1b',
  },
  {
    id: 'data-pipeline',
    tenant_id: 'ministry-finance-tr',
    version: '1.2.0',
    framework: 'autogen',
    capabilities: ['read:raw_data', 'write:data_lake', 'transform:etl'],
    status: 'quarantined',
    svid_uri: 'spiffe://argus.local/tenant/ministry-finance-tr/agent/data-pipeline/v1',
    last_seen: new Date(Date.now() - 3600000).toISOString(),
    node_id: 'node-eu-west-1a',
  },
];

const mockTasks = [
  {
    id: 'task-001',
    tenant_id: 'ministry-finance-tr',
    agent_id: 'budget-reconciler',
    status: 'completed',
    input_hash: 'a1b2c3d4e5f6',
    started_at: new Date(Date.now() - 3600000).toISOString(),
    completed_at: new Date(Date.now() - 3000000).toISOString(),
    cost_usd: 0.42,
    tokens_used: 12450,
  },
  {
    id: 'task-002',
    tenant_id: 'ministry-finance-tr',
    agent_id: 'doc-classifier',
    status: 'running',
    input_hash: 'b2c3d4e5f6a1',
    started_at: new Date(Date.now() - 300000).toISOString(),
    completed_at: null,
    cost_usd: 0.18,
    tokens_used: 5200,
  },
  {
    id: 'task-003',
    tenant_id: 'ministry-finance-tr',
    agent_id: 'fraud-detector',
    status: 'running',
    input_hash: 'c3d4e5f6a1b2',
    started_at: new Date(Date.now() - 180000).toISOString(),
    completed_at: null,
    cost_usd: 0.95,
    tokens_used: 28000,
  },
  {
    id: 'task-004',
    tenant_id: 'ministry-finance-tr',
    agent_id: 'report-generator',
    status: 'pending',
    input_hash: 'd4e5f6a1b2c3',
    started_at: new Date(Date.now() - 60000).toISOString(),
    completed_at: null,
    cost_usd: 0.0,
    tokens_used: 0,
  },
  {
    id: 'task-005',
    tenant_id: 'ministry-finance-tr',
    agent_id: 'compliance-checker',
    status: 'failed',
    input_hash: 'e5f6a1b2c3d4',
    started_at: new Date(Date.now() - 7200000).toISOString(),
    completed_at: new Date(Date.now() - 6800000).toISOString(),
    cost_usd: 1.23,
    tokens_used: 35600,
  },
  {
    id: 'task-006',
    tenant_id: 'ministry-finance-tr',
    agent_id: 'budget-reconciler',
    status: 'awaiting_approval',
    input_hash: 'f6a1b2c3d4e5',
    started_at: new Date(Date.now() - 900000).toISOString(),
    completed_at: null,
    cost_usd: 0.67,
    tokens_used: 19800,
  },
];

const mockAlerts = [
  {
    id: 'alert-001',
    tenant_id: 'ministry-finance-tr',
    agent_id: 'fraud-detector',
    probability: 0.87,
    estimated_ttf: 180,
    precursor_type: 'latency_spike',
    evidence: ['p99/p50 ratio: 4.2', 'sustained > 45s', 'memory usage: 89%'],
    status: 'open',
    created_at: new Date(Date.now() - 120000).toISOString(),
  },
  {
    id: 'alert-002',
    tenant_id: 'ministry-finance-tr',
    agent_id: 'compliance-checker',
    probability: 0.92,
    estimated_ttf: 60,
    precursor_type: 'token_escalation',
    evidence: ['context fill: 85%', 'token velocity: 2.4x baseline', 'consecutive loops: 7'],
    status: 'open',
    created_at: new Date(Date.now() - 600000).toISOString(),
  },
  {
    id: 'alert-003',
    tenant_id: 'ministry-finance-tr',
    agent_id: 'data-pipeline',
    probability: 0.78,
    estimated_ttf: 300,
    precursor_type: 'retry_storm',
    evidence: ['retry rate: 42%', 'error rate delta: +18%', 'upstream 503s: 12'],
    status: 'acknowledged',
    created_at: new Date(Date.now() - 1800000).toISOString(),
  },
  {
    id: 'alert-004',
    tenant_id: 'ministry-finance-tr',
    agent_id: 'budget-reconciler',
    probability: 0.65,
    estimated_ttf: 900,
    precursor_type: 'cost_runaway',
    evidence: ['cost acceleration: 2.3x', 'token velocity: 1.8x', 'no useful output in 5min'],
    status: 'resolved',
    created_at: new Date(Date.now() - 7200000).toISOString(),
  },
];

const mockMetrics = {
  total_agents: 6,
  active_tasks: 3,
  total_cost: 3.45,
  alert_count: 3,
};

function generateTimeSeries(points: number, base: number, variance: number, trend: number = 0) {
  const now = Date.now();
  const interval = 5 * 60 * 1000; // 5 min intervals
  return Array.from({ length: points }, (_, i) => ({
    timestamp: new Date(now - (points - 1 - i) * interval).toISOString(),
    value: Math.max(0, base + trend * i + (Math.random() - 0.5) * variance),
  }));
}

function generatePlatformTimeSeries() {
  return {
    total_tasks: generateTimeSeries(24, 12, 6, 0.3),
    active_agents: generateTimeSeries(24, 4, 2),
    total_cost: generateTimeSeries(24, 1.2, 0.8, 0.1),
    alert_count: generateTimeSeries(24, 2, 3),
    error_rate: generateTimeSeries(24, 0.05, 0.04),
    avg_latency: generateTimeSeries(24, 250, 150, 2),
  };
}

function generateAgentTimeSeries() {
  return {
    latency_p50: generateTimeSeries(24, 120, 40),
    latency_p99: generateTimeSeries(24, 450, 200, 5),
    token_rate: generateTimeSeries(24, 80, 30),
    error_rate: generateTimeSeries(24, 0.03, 0.02),
    cost: generateTimeSeries(24, 0.08, 0.05, 0.005),
  };
}

function generateAgentSpans(agentId: string) {
  const operations = ['llm.completion', 'tool.call', 'retrieval.query', 'embedding.create', 'chain.run'];
  const now = Date.now();
  return Array.from({ length: 15 }, (_, i) => ({
    span_id: `span-${agentId}-${String(i).padStart(3, '0')}`,
    trace_id: `trace-${agentId}-${String(Math.floor(i / 3)).padStart(3, '0')}`,
    tenant_id: 'ministry-finance-tr',
    agent_id: agentId,
    task_id: `task-${String(Math.floor(i / 3) + 1).padStart(3, '0')}`,
    operation_name: operations[i % operations.length],
    started_at: new Date(now - (15 - i) * 30000).toISOString(),
    duration_ms: Math.floor(50 + Math.random() * 500),
    tier: (i % 3 === 0 ? 1 : i % 3 === 1 ? 2 : 1) as 1 | 2 | 3,
    attributes: {
      model: i % 5 === 0 ? 'gpt-4' : 'gpt-3.5-turbo',
      tokens: String(Math.floor(100 + Math.random() * 2000)),
    },
    error_code: i === 7 ? 'TIMEOUT' : i === 12 ? 'RATE_LIMITED' : null,
  }));
}

function getAgentDetail(agentId: string) {
  const agent = mockAgents.find((a) => a.id === agentId);
  if (!agent) return null;
  return {
    ...agent,
    tasks_completed: Math.floor(20 + Math.random() * 80),
    tasks_failed: Math.floor(Math.random() * 10),
    total_cost_usd: +(Math.random() * 15 + 2).toFixed(2),
    total_tokens: Math.floor(50000 + Math.random() * 200000),
    avg_latency_ms: Math.floor(80 + Math.random() * 300),
    uptime_pct: +(95 + Math.random() * 5).toFixed(1),
  };
}

function jsonResponse(data: unknown, tenantId: string) {
  return JSON.stringify({
    data,
    meta: { tenant_id: tenantId, request_id: crypto.randomUUID() },
  });
}

export function mockApiPlugin(): Plugin {
  return {
    name: 'mock-api',
    configureServer(server) {
      server.middlewares.use('/api/v1', (req, res, next) => {
        const tenantId = (req.headers['x-tenant-id'] as string) || 'default';
        res.setHeader('Content-Type', 'application/json');

        const url = req.url || '';

        // Agent detail: /agents/:id
        const agentDetailMatch = url.match(/^\/agents\/([^/]+)$/);
        if (agentDetailMatch && req.method === 'GET') {
          const detail = getAgentDetail(agentDetailMatch[1]);
          if (detail) {
            res.end(jsonResponse(detail, tenantId));
          } else {
            res.statusCode = 404;
            res.end(JSON.stringify({ error: { code: 'AGENT_NOT_FOUND', message: 'Agent not found' } }));
          }
          return;
        }

        // Agent spans: /agents/:id/spans
        const agentSpansMatch = url.match(/^\/agents\/([^/]+)\/spans$/);
        if (agentSpansMatch && req.method === 'GET') {
          const spans = generateAgentSpans(agentSpansMatch[1]);
          res.end(jsonResponse(spans, tenantId));
          return;
        }

        // Agent time series: /agents/:id/timeseries
        const agentTsMatch = url.match(/^\/agents\/([^/]+)\/timeseries$/);
        if (agentTsMatch && req.method === 'GET') {
          res.end(jsonResponse(generateAgentTimeSeries(), tenantId));
          return;
        }

        // Alert actions: PATCH /alerts/:id
        const alertPatchMatch = url.match(/^\/alerts\/([^/]+)$/);
        if (alertPatchMatch && req.method === 'PATCH') {
          let body = '';
          req.on('data', (chunk: Buffer) => { body += chunk.toString(); });
          req.on('end', () => {
            const update = JSON.parse(body);
            const alert = mockAlerts.find((a) => a.id === alertPatchMatch[1]);
            if (alert) {
              alert.status = update.status || alert.status;
              res.end(jsonResponse(alert, tenantId));
            } else {
              res.statusCode = 404;
              res.end(JSON.stringify({ error: { code: 'ALERT_NOT_FOUND', message: 'Alert not found' } }));
            }
          });
          return;
        }

        if (url === '/agents') {
          res.end(jsonResponse(mockAgents, tenantId));
        } else if (url === '/tasks') {
          res.end(jsonResponse(mockTasks, tenantId));
        } else if (url === '/alerts') {
          res.end(jsonResponse(mockAlerts, tenantId));
        } else if (url === '/metrics') {
          res.end(jsonResponse(mockMetrics, tenantId));
        } else if (url === '/metrics/timeseries') {
          res.end(jsonResponse(generatePlatformTimeSeries(), tenantId));
        } else {
          next();
        }
      });
    },
  };
}
