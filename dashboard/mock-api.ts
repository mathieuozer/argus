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

        if (req.url === '/agents') {
          res.end(jsonResponse(mockAgents, tenantId));
        } else if (req.url === '/tasks') {
          res.end(jsonResponse(mockTasks, tenantId));
        } else if (req.url === '/alerts') {
          res.end(jsonResponse(mockAlerts, tenantId));
        } else if (req.url === '/metrics') {
          res.end(jsonResponse(mockMetrics, tenantId));
        } else {
          next();
        }
      });
    },
  };
}
