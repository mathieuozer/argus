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
  { id: 'task-001', tenant_id: 'ministry-finance-tr', agent_id: 'budget-reconciler', status: 'completed', input_hash: 'a1b2c3d4e5f6', started_at: new Date(Date.now() - 3600000).toISOString(), completed_at: new Date(Date.now() - 3000000).toISOString(), cost_usd: 0.42, tokens_used: 12450 },
  { id: 'task-002', tenant_id: 'ministry-finance-tr', agent_id: 'doc-classifier', status: 'running', input_hash: 'b2c3d4e5f6a1', started_at: new Date(Date.now() - 300000).toISOString(), completed_at: null, cost_usd: 0.18, tokens_used: 5200 },
  { id: 'task-003', tenant_id: 'ministry-finance-tr', agent_id: 'fraud-detector', status: 'running', input_hash: 'c3d4e5f6a1b2', started_at: new Date(Date.now() - 180000).toISOString(), completed_at: null, cost_usd: 0.95, tokens_used: 28000 },
  { id: 'task-004', tenant_id: 'ministry-finance-tr', agent_id: 'report-generator', status: 'pending', input_hash: 'd4e5f6a1b2c3', started_at: new Date(Date.now() - 60000).toISOString(), completed_at: null, cost_usd: 0.0, tokens_used: 0 },
  { id: 'task-005', tenant_id: 'ministry-finance-tr', agent_id: 'compliance-checker', status: 'failed', input_hash: 'e5f6a1b2c3d4', started_at: new Date(Date.now() - 7200000).toISOString(), completed_at: new Date(Date.now() - 6800000).toISOString(), cost_usd: 1.23, tokens_used: 35600 },
  { id: 'task-006', tenant_id: 'ministry-finance-tr', agent_id: 'budget-reconciler', status: 'awaiting_approval', input_hash: 'f6a1b2c3d4e5', started_at: new Date(Date.now() - 900000).toISOString(), completed_at: null, cost_usd: 0.67, tokens_used: 19800 },
];

const mockAlerts = [
  { id: 'alert-001', tenant_id: 'ministry-finance-tr', agent_id: 'fraud-detector', probability: 0.87, estimated_ttf: 180, precursor_type: 'latency_spike', evidence: ['p99/p50 ratio: 4.2', 'sustained > 45s', 'memory usage: 89%'], status: 'open', created_at: new Date(Date.now() - 120000).toISOString() },
  { id: 'alert-002', tenant_id: 'ministry-finance-tr', agent_id: 'compliance-checker', probability: 0.92, estimated_ttf: 60, precursor_type: 'token_escalation', evidence: ['context fill: 85%', 'token velocity: 2.4x baseline', 'consecutive loops: 7'], status: 'open', created_at: new Date(Date.now() - 600000).toISOString() },
  { id: 'alert-003', tenant_id: 'ministry-finance-tr', agent_id: 'data-pipeline', probability: 0.78, estimated_ttf: 300, precursor_type: 'retry_storm', evidence: ['retry rate: 42%', 'error rate delta: +18%', 'upstream 503s: 12'], status: 'acknowledged', created_at: new Date(Date.now() - 1800000).toISOString() },
  { id: 'alert-004', tenant_id: 'ministry-finance-tr', agent_id: 'budget-reconciler', probability: 0.65, estimated_ttf: 900, precursor_type: 'cost_runaway', evidence: ['cost acceleration: 2.3x', 'token velocity: 1.8x', 'no useful output in 5min'], status: 'resolved', created_at: new Date(Date.now() - 7200000).toISOString() },
];

const mockMetrics = { total_agents: 6, active_tasks: 3, total_cost: 3.45, alert_count: 3 };

const mockComplianceReports: any[] = [];

// ---- Mock Traces ----
function generateMockTraces() {
  const now = Date.now();
  const agents = ['budget-reconciler', 'doc-classifier', 'fraud-detector', 'report-generator', 'compliance-checker'];
  const ops = ['chain.run', 'llm.completion', 'retrieval.query', 'tool.call', 'embedding.create'];
  return Array.from({ length: 10 }, (_, i) => ({
    trace_id: `trace-${String(i + 1).padStart(4, '0')}-${crypto.randomUUID().slice(0, 8)}`,
    root_operation: ops[i % ops.length],
    agent_id: agents[i % agents.length],
    total_spans: 3 + Math.floor(Math.random() * 8),
    total_duration_ms: 200 + Math.floor(Math.random() * 2000),
    has_errors: i === 2 || i === 7,
    started_at: new Date(now - (10 - i) * 600000).toISOString(),
  }));
}

function generateMockTraceDetail(traceId: string) {
  const now = Date.now();
  const errCode = traceId.includes('0003') || traceId.includes('0008') ? 'TIMEOUT' : null;
  return {
    trace_id: traceId,
    root_span: {
      span_id: 'span-root',
      operation_name: 'chain.run',
      duration_ms: 1200,
      started_at: new Date(now - 60000).toISOString(),
      agent_id: 'budget-reconciler',
      attributes: { framework: 'langchain', task: 'budget_q4' },
      error_code: null,
      children: [
        {
          span_id: 'span-llm-1',
          operation_name: 'llm.completion',
          duration_ms: 800,
          started_at: new Date(now - 59000).toISOString(),
          agent_id: 'budget-reconciler',
          attributes: { model: 'gpt-4', tokens: '1200', prompt_tokens: '400', completion_tokens: '800' },
          error_code: null,
          children: [],
        },
        {
          span_id: 'span-tool-1',
          operation_name: 'tool.call',
          duration_ms: 350,
          started_at: new Date(now - 58200).toISOString(),
          agent_id: 'budget-reconciler',
          attributes: { tool: 'budget_db.query', query_type: 'SELECT' },
          error_code: errCode,
          children: [
            {
              span_id: 'span-db-1',
              operation_name: 'db.query',
              duration_ms: 120,
              started_at: new Date(now - 58100).toISOString(),
              attributes: { 'db.name': 'finance', 'db.statement': 'SELECT * FROM budgets' },
              error_code: null,
              children: [],
            },
          ],
        },
        {
          span_id: 'span-embed-1',
          operation_name: 'embedding.create',
          duration_ms: 50,
          started_at: new Date(now - 57850).toISOString(),
          attributes: { model: 'text-embedding-3-small', dimensions: '1536' },
          error_code: null,
          children: [],
        },
      ],
    },
    total_spans: 5,
    total_duration_ms: 1200,
    has_errors: errCode !== null,
  };
}

// ---- Mock Data Quality ----
const mockDQRules = [
  { id: 'rule-001', agent_id: 'budget-reconciler', name: 'Output must have amount', type: 'schema', target: 'output', schema: { required: ['amount', 'currency', 'date'] }, severity: 'critical', enabled: true, created_at: new Date(Date.now() - 86400000 * 7).toISOString(), updated_at: new Date(Date.now() - 86400000).toISOString() },
  { id: 'rule-002', agent_id: 'budget-reconciler', name: 'Amount in valid range', type: 'range', target: 'output', schema: { field: 'amount', min: 0, max: 1000000 }, severity: 'warning', enabled: true, created_at: new Date(Date.now() - 86400000 * 5).toISOString(), updated_at: new Date(Date.now() - 86400000 * 2).toISOString() },
  { id: 'rule-003', agent_id: 'doc-classifier', name: 'Category must match pattern', type: 'regex', target: 'output', schema: { field: 'category', pattern: '^(finance|legal|hr|ops)$' }, severity: 'warning', enabled: true, created_at: new Date(Date.now() - 86400000 * 3).toISOString(), updated_at: new Date(Date.now() - 86400000 * 3).toISOString() },
];

const mockDQScores = [
  { id: 'score-001', agent_id: 'budget-reconciler', completeness: 0.9800, conformance: 0.9500, consistency: 0.9200, freshness: 0.9900, overall: 0.9575, sample_size: 150, computed_at: new Date(Date.now() - 300000).toISOString() },
  { id: 'score-002', agent_id: 'doc-classifier', completeness: 0.8800, conformance: 0.9200, consistency: 0.8500, freshness: 0.9500, overall: 0.9010, sample_size: 200, computed_at: new Date(Date.now() - 300000).toISOString() },
  { id: 'score-003', agent_id: 'fraud-detector', completeness: 0.9900, conformance: 0.7800, consistency: 0.9100, freshness: 0.8500, overall: 0.8870, sample_size: 80, computed_at: new Date(Date.now() - 300000).toISOString() },
  { id: 'score-004', agent_id: 'report-generator', completeness: 0.7500, conformance: 0.8200, consistency: 0.7000, freshness: 0.9800, overall: 0.8070, sample_size: 50, computed_at: new Date(Date.now() - 300000).toISOString() },
];

const mockDQViolations = [
  { id: 'viol-001', rule_id: 'rule-001', rule_name: 'Output must have amount', agent_id: 'budget-reconciler', severity: 'critical', message: 'Missing required field: currency', occurred_at: new Date(Date.now() - 180000).toISOString() },
  { id: 'viol-002', rule_id: 'rule-002', rule_name: 'Amount in valid range', agent_id: 'budget-reconciler', severity: 'warning', message: 'amount value -50.00 out of range [0, 1000000]', occurred_at: new Date(Date.now() - 360000).toISOString() },
  { id: 'viol-003', rule_id: 'rule-003', rule_name: 'Category must match pattern', agent_id: 'doc-classifier', severity: 'warning', message: 'category "unknown" does not match pattern', occurred_at: new Date(Date.now() - 900000).toISOString() },
];

function generateDriftData() {
  const now = Date.now();
  return Array.from({ length: 12 }, (_, i) => ({
    timestamp: new Date(now - (12 - i) * 3600000).toISOString(),
    consistency: 0.95 - (i > 6 ? (i - 6) * 0.04 : 0) + Math.random() * 0.02,
    baseline: 1.0,
  }));
}

// ---- Mock Catalog ----
const mockCatalogSources = [
  { id: 'src-001', type: 'database', name: 'budget_db', identifier: 'postgres://budget-db:5432/finance', agents: ['budget-reconciler', 'report-generator'], access_types: ['read', 'write'], first_seen: new Date(Date.now() - 86400000 * 30).toISOString(), last_seen: new Date(Date.now() - 60000).toISOString(), span_count: 1243, tier: 2 },
  { id: 'src-002', type: 'api', name: 'openai_api', identifier: 'https://api.openai.com/v1', agents: ['budget-reconciler', 'doc-classifier', 'fraud-detector', 'compliance-checker'], access_types: ['call'], first_seen: new Date(Date.now() - 86400000 * 30).toISOString(), last_seen: new Date(Date.now() - 30000).toISOString(), span_count: 4520, tier: 1 },
  { id: 'src-003', type: 'storage', name: 'report_store', identifier: 's3://argus-reports-eu-west', agents: ['report-generator', 'budget-reconciler'], access_types: ['write', 'read'], first_seen: new Date(Date.now() - 86400000 * 20).toISOString(), last_seen: new Date(Date.now() - 120000).toISOString(), span_count: 567, tier: 2 },
  { id: 'src-004', type: 'database', name: 'document_store', identifier: 'postgres://doc-db:5432/documents', agents: ['doc-classifier'], access_types: ['read'], first_seen: new Date(Date.now() - 86400000 * 15).toISOString(), last_seen: new Date(Date.now() - 300000).toISOString(), span_count: 890, tier: 2 },
  { id: 'src-005', type: 'tool', name: 'pii_scrubber', identifier: 'pii_scrubber', agents: ['fraud-detector', 'compliance-checker'], access_types: ['call'], first_seen: new Date(Date.now() - 86400000 * 10).toISOString(), last_seen: new Date(Date.now() - 600000).toISOString(), span_count: 234, tier: 1 },
  { id: 'src-006', type: 'database', name: 'transactions_db', identifier: 'postgres://txn-db:5432/transactions', agents: ['fraud-detector'], access_types: ['read'], first_seen: new Date(Date.now() - 86400000 * 25).toISOString(), last_seen: new Date(Date.now() - 45000).toISOString(), span_count: 3200, tier: 3 },
];

const mockLineageGraph = {
  nodes: [
    { id: 'budget-reconciler', type: 'agent' },
    { id: 'doc-classifier', type: 'agent' },
    { id: 'fraud-detector', type: 'agent' },
    { id: 'report-generator', type: 'agent' },
    { id: 'compliance-checker', type: 'agent' },
    { id: 'budget_db', type: 'database' },
    { id: 'document_store', type: 'database' },
    { id: 'transactions_db', type: 'database' },
    { id: 'report_store', type: 'storage' },
    { id: 'openai_api', type: 'external_api' },
    { id: 'pii_scrubber', type: 'tool' },
  ],
  edges: [
    { source: 'budget_db', target: 'budget-reconciler', label: 'read', span_count: 800 },
    { source: 'budget-reconciler', target: 'report_store', label: 'write', span_count: 120 },
    { source: 'budget-reconciler', target: 'openai_api', label: 'call', span_count: 950 },
    { source: 'document_store', target: 'doc-classifier', label: 'read', span_count: 890 },
    { source: 'doc-classifier', target: 'openai_api', label: 'call', span_count: 1200 },
    { source: 'transactions_db', target: 'fraud-detector', label: 'read', span_count: 3200 },
    { source: 'fraud-detector', target: 'openai_api', label: 'call', span_count: 1500 },
    { source: 'fraud-detector', target: 'pii_scrubber', label: 'call', span_count: 234 },
    { source: 'budget_db', target: 'report-generator', label: 'read', span_count: 400 },
    { source: 'report-generator', target: 'report_store', label: 'write', span_count: 300 },
    { source: 'compliance-checker', target: 'openai_api', label: 'call', span_count: 870 },
    { source: 'compliance-checker', target: 'pii_scrubber', label: 'call', span_count: 100 },
  ],
};

const mockToolUsage = [
  { tool: 'openai_api', agent_id: 'budget-reconciler', call_count: 950, avg_duration_ms: 420 },
  { tool: 'openai_api', agent_id: 'doc-classifier', call_count: 1200, avg_duration_ms: 380 },
  { tool: 'openai_api', agent_id: 'fraud-detector', call_count: 1500, avg_duration_ms: 510 },
  { tool: 'pii_scrubber', agent_id: 'fraud-detector', call_count: 234, avg_duration_ms: 45 },
  { tool: 'budget_db.query', agent_id: 'budget-reconciler', call_count: 800, avg_duration_ms: 120 },
];

// ---- Mock Costs ----
const mockCostBreakdown = [
  { group: 'budget-reconciler', cost_usd: 4.82, tokens_used: 142000, task_count: 45 },
  { group: 'doc-classifier', cost_usd: 3.15, tokens_used: 98000, task_count: 62 },
  { group: 'fraud-detector', cost_usd: 8.94, tokens_used: 265000, task_count: 38 },
  { group: 'report-generator', cost_usd: 2.10, tokens_used: 62000, task_count: 15 },
  { group: 'compliance-checker', cost_usd: 5.60, tokens_used: 180000, task_count: 28 },
  { group: 'data-pipeline', cost_usd: 1.45, tokens_used: 42000, task_count: 20 },
];

function generateCostTrends() {
  const now = Date.now();
  return Array.from({ length: 14 }, (_, i) => ({
    timestamp: new Date(now - (14 - i) * 86400000).toISOString(),
    cost_usd: +(2 + Math.random() * 5 + i * 0.2).toFixed(2),
    tokens_used: Math.floor(50000 + Math.random() * 80000 + i * 5000),
  }));
}

const mockBudgets = [
  { id: 'budget-001', agent_id: null, budget_usd: 100.00, spent_usd: 26.06, period: 'monthly', alert_threshold: 0.80, enabled: true, created_at: new Date(Date.now() - 86400000 * 15).toISOString() },
  { id: 'budget-002', agent_id: 'fraud-detector', budget_usd: 30.00, spent_usd: 8.94, period: 'weekly', alert_threshold: 0.80, enabled: true, created_at: new Date(Date.now() - 86400000 * 10).toISOString() },
  { id: 'budget-003', agent_id: 'budget-reconciler', budget_usd: 5.00, spent_usd: 4.82, period: 'daily', alert_threshold: 0.80, enabled: true, created_at: new Date(Date.now() - 86400000 * 5).toISOString() },
];

const mockCostAnomalies = [
  { id: 'anom-001', agent_id: 'fraud-detector', expected_usd: 3.50, actual_usd: 8.94, ratio: 2.55, detected_at: new Date(Date.now() - 3600000).toISOString(), status: 'open' },
  { id: 'anom-002', agent_id: 'compliance-checker', expected_usd: 2.00, actual_usd: 5.60, ratio: 2.80, detected_at: new Date(Date.now() - 7200000).toISOString(), status: 'acknowledged' },
];

// ---- Mock Audit ----
const mockAuditEntries = [
  { id: 'audit-001', tenant_id: 'ministry-finance-tr', actor: 'admin@ministry.gov.tr', action: 'agent.register', resource: 'agent/budget-reconciler', details: 'Registered v2.1.0', timestamp: new Date(Date.now() - 86400000).toISOString() },
  { id: 'audit-002', tenant_id: 'ministry-finance-tr', actor: 'system', action: 'cert.rotate', resource: 'identity/budget-reconciler', details: 'Auto-rotated SVID', timestamp: new Date(Date.now() - 82800000).toISOString() },
  { id: 'audit-003', tenant_id: 'ministry-finance-tr', actor: 'admin@ministry.gov.tr', action: 'policy.create', resource: 'policy/budget-access', details: 'Created budget access policy', timestamp: new Date(Date.now() - 72000000).toISOString() },
  { id: 'audit-004', tenant_id: 'ministry-finance-tr', actor: 'system', action: 'agent.quarantine', resource: 'agent/data-pipeline', details: 'Quarantined due to retry storm', timestamp: new Date(Date.now() - 3600000).toISOString() },
  { id: 'audit-005', tenant_id: 'ministry-finance-tr', actor: 'ops@ministry.gov.tr', action: 'alert.acknowledge', resource: 'alert/alert-003', details: 'Acknowledged retry_storm alert', timestamp: new Date(Date.now() - 1800000).toISOString() },
  { id: 'audit-006', tenant_id: 'ministry-finance-tr', actor: 'system', action: 'cert.issue', resource: 'identity/fraud-detector', details: 'Issued SVID for v3.0.1', timestamp: new Date(Date.now() - 600000).toISOString() },
  { id: 'audit-007', tenant_id: 'ministry-finance-tr', actor: 'admin@ministry.gov.tr', action: 'slo.create', resource: 'slo/slo-001', details: 'Created Availability SLO for budget-reconciler', timestamp: new Date(Date.now() - 43200000).toISOString() },
  { id: 'audit-008', tenant_id: 'ministry-finance-tr', actor: 'system', action: 'budget.alert', resource: 'cost/budget-003', details: 'budget-reconciler daily budget at 96%', timestamp: new Date(Date.now() - 7200000).toISOString() },
];

// ---- Mock SLOs ----
const mockSLOs = [
  { id: 'slo-001', agent_id: 'budget-reconciler', name: 'Availability SLO', type: 'availability', target: 0.9950, window: '30d', current: 0.9980, budget_remaining: 0.72, status: 'met', enabled: true, created_at: new Date(Date.now() - 86400000 * 20).toISOString() },
  { id: 'slo-002', agent_id: 'fraud-detector', name: 'Latency P99 SLO', type: 'latency_p99', target: 0.9900, window: '7d', current: 0.9750, budget_remaining: 0.25, status: 'at_risk', enabled: true, created_at: new Date(Date.now() - 86400000 * 15).toISOString() },
  { id: 'slo-003', agent_id: 'doc-classifier', name: 'Error Rate SLO', type: 'error_rate', target: 0.9900, window: '30d', current: 0.9920, budget_remaining: 0.60, status: 'met', enabled: true, created_at: new Date(Date.now() - 86400000 * 10).toISOString() },
  { id: 'slo-004', agent_id: 'compliance-checker', name: 'Availability SLO', type: 'availability', target: 0.9990, window: '30d', current: 0.9800, budget_remaining: -0.10, status: 'breached', enabled: true, created_at: new Date(Date.now() - 86400000 * 25).toISOString() },
];

function generateErrorBudget() {
  const now = Date.now();
  return Array.from({ length: 30 }, (_, i) => ({
    timestamp: new Date(now - (30 - i) * 86400000).toISOString(),
    remaining: Math.max(0, 1.0 - i * 0.012 - Math.random() * 0.02),
  }));
}

// ---- Mock Evals ----
const mockEvalSuites: any[] = [
  {
    id: 'suite-001',
    tenant_id: 'ministry-finance-tr',
    name: 'Budget Accuracy Tests',
    description: 'Validates budget reconciliation accuracy',
    agent_id: 'budget-reconciler',
    test_cases: [
      { id: 'tc-001', name: 'Simple addition', input: 'Calculate sum of 100 and 200', expected_output: '300', criteria: {}, max_latency_ms: 5000 },
      { id: 'tc-002', name: 'Currency conversion', input: 'Convert 100 USD to EUR', expected_output: '~92 EUR', criteria: { accuracy: '0.95' }, max_latency_ms: 3000 },
    ],
    created_at: new Date(Date.now() - 86400000 * 5).toISOString(),
    updated_at: new Date(Date.now() - 86400000).toISOString(),
  },
  {
    id: 'suite-002',
    tenant_id: 'ministry-finance-tr',
    name: 'Classification Quality',
    description: 'Ensures document classifier returns valid categories',
    agent_id: 'doc-classifier',
    test_cases: [
      { id: 'tc-003', name: 'Legal doc', input: 'Contract amendment for vendor ABC', expected_output: 'legal', criteria: {}, max_latency_ms: 2000 },
      { id: 'tc-004', name: 'Finance doc', input: 'Q4 budget report', expected_output: 'finance', criteria: {}, max_latency_ms: 2000 },
      { id: 'tc-005', name: 'HR doc', input: 'Employee onboarding form', expected_output: 'hr', criteria: {}, max_latency_ms: 2000 },
    ],
    created_at: new Date(Date.now() - 86400000 * 3).toISOString(),
    updated_at: new Date(Date.now() - 86400000 * 2).toISOString(),
  },
];

const mockEvalRuns: any[] = [
  {
    id: 'run-001',
    tenant_id: 'ministry-finance-tr',
    suite_id: 'suite-001',
    suite_name: 'Budget Accuracy Tests',
    agent_id: 'budget-reconciler',
    status: 'completed',
    score: 0.85,
    total_cases: 2,
    passed_cases: 2,
    failed_cases: 0,
    results: [
      { test_case_id: 'tc-001', test_case_name: 'Simple addition', status: 'passed', actual_output: '300', latency_ms: 180, score: 0.9, reason: '' },
      { test_case_id: 'tc-002', test_case_name: 'Currency conversion', status: 'passed', actual_output: '92.15 EUR', latency_ms: 420, score: 0.8, reason: '' },
    ],
    started_at: new Date(Date.now() - 3600000).toISOString(),
    completed_at: new Date(Date.now() - 3500000).toISOString(),
  },
  {
    id: 'run-002',
    tenant_id: 'ministry-finance-tr',
    suite_id: 'suite-002',
    suite_name: 'Classification Quality',
    agent_id: 'doc-classifier',
    status: 'completed',
    score: 0.67,
    total_cases: 3,
    passed_cases: 2,
    failed_cases: 1,
    results: [
      { test_case_id: 'tc-003', test_case_name: 'Legal doc', status: 'passed', actual_output: 'legal', latency_ms: 150, score: 1.0, reason: '' },
      { test_case_id: 'tc-004', test_case_name: 'Finance doc', status: 'passed', actual_output: 'finance', latency_ms: 130, score: 1.0, reason: '' },
      { test_case_id: 'tc-005', test_case_name: 'HR doc', status: 'failed', actual_output: 'ops', latency_ms: 140, score: 0.0, reason: 'Expected hr but got ops' },
    ],
    started_at: new Date(Date.now() - 7200000).toISOString(),
    completed_at: new Date(Date.now() - 7100000).toISOString(),
  },
];

// ---- Helpers ----
function generateTimeSeries(points: number, base: number, variance: number, trend: number = 0) {
  const now = Date.now();
  const interval = 5 * 60 * 1000;
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
    attributes: { model: i % 5 === 0 ? 'gpt-4' : 'gpt-3.5-turbo', tokens: String(Math.floor(100 + Math.random() * 2000)) },
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
      // Auth endpoints — mounted before the /api/v1 middleware
      server.middlewares.use('/api/v1/auth', (req, res, next) => {
        res.setHeader('Content-Type', 'application/json');
        const url = req.url || '';

        if (url.startsWith('/login') && req.method === 'POST') {
          let body = '';
          req.on('data', (chunk: Buffer) => { body += chunk.toString(); });
          req.on('end', () => {
            const { username, tenantId } = JSON.parse(body);
            const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTAwMSIsInRlbmFudCI6Im1vY2siLCJyb2xlIjoiYWRtaW4iLCJleHAiOjk5OTk5OTk5OTl9.mock';
            const mockRefreshToken = 'rt_mock_' + crypto.randomUUID();
            res.end(JSON.stringify({
              data: {
                token: mockToken,
                refreshToken: mockRefreshToken,
                user: {
                  id: 'user-001',
                  username: username || 'admin',
                  email: `${username || 'admin'}@argus.local`,
                  tenantId: tenantId || 'ministry-finance-tr',
                  tenantName: tenantId ? tenantId.replace(/-/g, ' ').replace(/\b\w/g, (c: string) => c.toUpperCase()) : 'Ministry Finance TR',
                  role: 'admin',
                },
                expiresAt: new Date(Date.now() + 3600000).toISOString(),
              },
              meta: { tenant_id: tenantId || 'ministry-finance-tr', request_id: crypto.randomUUID() },
            }));
          });
          return;
        }

        if (url.startsWith('/refresh') && req.method === 'POST') {
          let body = '';
          req.on('data', (chunk: Buffer) => { body += chunk.toString(); });
          req.on('end', () => {
            const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTAwMSIsInRlbmFudCI6Im1vY2siLCJyb2xlIjoiYWRtaW4iLCJleHAiOjk5OTk5OTk5OTl9.refreshed';
            const mockRefreshToken = 'rt_mock_' + crypto.randomUUID();
            res.end(JSON.stringify({
              data: {
                token: mockToken,
                refreshToken: mockRefreshToken,
                user: {
                  id: 'user-001',
                  username: 'admin',
                  email: 'admin@argus.local',
                  tenantId: 'ministry-finance-tr',
                  tenantName: 'Ministry Finance TR',
                  role: 'admin',
                },
                expiresAt: new Date(Date.now() + 3600000).toISOString(),
              },
              meta: { tenant_id: 'ministry-finance-tr', request_id: crypto.randomUUID() },
            }));
          });
          return;
        }

        next();
      });

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
          res.end(jsonResponse(generateAgentSpans(agentSpansMatch[1]), tenantId));
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

        // ---- Traces ----
        const traceDetailMatch = url.match(/^\/traces\/([^/?]+)$/);
        if (traceDetailMatch && req.method === 'GET') {
          res.end(jsonResponse(generateMockTraceDetail(traceDetailMatch[1]), tenantId));
          return;
        }
        if (url.startsWith('/traces') && req.method === 'GET') {
          res.end(jsonResponse(generateMockTraces(), tenantId));
          return;
        }

        // ---- Data Quality ----
        if (url.startsWith('/data-quality/drift/') && req.method === 'GET') {
          res.end(jsonResponse(generateDriftData(), tenantId));
          return;
        }
        if (url.startsWith('/data-quality/scores') && req.method === 'GET') {
          res.end(jsonResponse(mockDQScores, tenantId));
          return;
        }
        if (url.startsWith('/data-quality/violations') && req.method === 'GET') {
          res.end(jsonResponse(mockDQViolations, tenantId));
          return;
        }
        if (url.startsWith('/data-quality/rules') && req.method === 'GET') {
          res.end(jsonResponse(mockDQRules, tenantId));
          return;
        }
        if (url.startsWith('/data-quality/rules') && req.method === 'POST') {
          let body = '';
          req.on('data', (chunk: Buffer) => { body += chunk.toString(); });
          req.on('end', () => {
            const rule = JSON.parse(body);
            rule.id = `rule-${String(mockDQRules.length + 1).padStart(3, '0')}`;
            rule.created_at = new Date().toISOString();
            rule.updated_at = new Date().toISOString();
            mockDQRules.push(rule);
            res.end(jsonResponse(rule, tenantId));
          });
          return;
        }
        const dqRuleDeleteMatch = url.match(/^\/data-quality\/rules\/([^/]+)$/);
        if (dqRuleDeleteMatch && req.method === 'DELETE') {
          const idx = mockDQRules.findIndex((r) => r.id === dqRuleDeleteMatch[1]);
          if (idx >= 0) mockDQRules.splice(idx, 1);
          res.end(jsonResponse({ deleted: true }, tenantId));
          return;
        }

        // ---- Catalog ----
        if (url.startsWith('/catalog/lineage/graph') && req.method === 'GET') {
          res.end(jsonResponse(mockLineageGraph, tenantId));
          return;
        }
        if (url.startsWith('/catalog/tools') && req.method === 'GET') {
          res.end(jsonResponse(mockToolUsage, tenantId));
          return;
        }
        const catalogAgentMatch = url.match(/^\/catalog\/sources\/([^/]+)$/);
        if (catalogAgentMatch && req.method === 'GET') {
          const agentSources = mockCatalogSources.filter((s) => s.agents.includes(catalogAgentMatch[1]));
          res.end(jsonResponse(agentSources, tenantId));
          return;
        }
        if (url.startsWith('/catalog/sources') && req.method === 'GET') {
          res.end(jsonResponse(mockCatalogSources, tenantId));
          return;
        }

        // ---- Costs ----
        if (url.startsWith('/costs/breakdown') && req.method === 'GET') {
          res.end(jsonResponse(mockCostBreakdown, tenantId));
          return;
        }
        if (url.startsWith('/costs/trends') && req.method === 'GET') {
          res.end(jsonResponse(generateCostTrends(), tenantId));
          return;
        }
        if (url.startsWith('/costs/budgets') && req.method === 'POST') {
          let body = '';
          req.on('data', (chunk: Buffer) => { body += chunk.toString(); });
          req.on('end', () => {
            const budget = JSON.parse(body);
            budget.id = `budget-${String(mockBudgets.length + 1).padStart(3, '0')}`;
            budget.spent_usd = 0;
            budget.created_at = new Date().toISOString();
            mockBudgets.push(budget);
            res.end(jsonResponse(budget, tenantId));
          });
          return;
        }
        if (url.startsWith('/costs/budgets') && req.method === 'GET') {
          res.end(jsonResponse(mockBudgets, tenantId));
          return;
        }
        if (url.startsWith('/costs/anomalies') && req.method === 'GET') {
          res.end(jsonResponse(mockCostAnomalies, tenantId));
          return;
        }

        // ---- Audit ----
        if (url.startsWith('/audit') && req.method === 'GET') {
          res.end(jsonResponse(mockAuditEntries, tenantId));
          return;
        }

        // ---- SLOs ----
        const sloBudgetMatch = url.match(/^\/slos\/([^/]+)\/budget$/);
        if (sloBudgetMatch && req.method === 'GET') {
          res.end(jsonResponse(generateErrorBudget(), tenantId));
          return;
        }
        const sloDetailMatch = url.match(/^\/slos\/([^/]+)$/);
        if (sloDetailMatch && req.method === 'GET') {
          const slo = mockSLOs.find((s) => s.id === sloDetailMatch[1]);
          res.end(jsonResponse(slo || null, tenantId));
          return;
        }
        if (sloDetailMatch && req.method === 'DELETE') {
          const idx = mockSLOs.findIndex((s) => s.id === sloDetailMatch[1]);
          if (idx >= 0) mockSLOs.splice(idx, 1);
          res.end(jsonResponse({ deleted: true }, tenantId));
          return;
        }
        if (url.startsWith('/slos') && req.method === 'POST') {
          let body = '';
          req.on('data', (chunk: Buffer) => { body += chunk.toString(); });
          req.on('end', () => {
            const slo = JSON.parse(body);
            slo.id = `slo-${String(mockSLOs.length + 1).padStart(3, '0')}`;
            slo.current = slo.target + (Math.random() - 0.3) * 0.01;
            slo.budget_remaining = 0.5 + Math.random() * 0.5;
            slo.status = slo.current >= slo.target ? 'met' : 'at_risk';
            slo.created_at = new Date().toISOString();
            mockSLOs.push(slo);
            res.end(jsonResponse(slo, tenantId));
          });
          return;
        }
        if (url.startsWith('/slos') && req.method === 'GET') {
          res.end(jsonResponse(mockSLOs, tenantId));
          return;
        }

        // ---- Evals ----
        const evalRunDetailMatch = url.match(/^\/evals\/runs\/([^/]+)$/);
        if (evalRunDetailMatch && req.method === 'GET') {
          const run = mockEvalRuns.find((r: any) => r.id === evalRunDetailMatch[1]);
          if (run) {
            res.end(jsonResponse(run, tenantId));
          } else {
            res.statusCode = 404;
            res.end(JSON.stringify({ error: { code: 'NOT_FOUND', message: 'Eval run not found' } }));
          }
          return;
        }
        if (url.startsWith('/evals/runs') && req.method === 'GET') {
          res.end(jsonResponse(mockEvalRuns, tenantId));
          return;
        }
        const evalSuiteRunMatch = url.match(/^\/evals\/suites\/([^/]+)\/run$/);
        if (evalSuiteRunMatch && req.method === 'POST') {
          const suite = mockEvalSuites.find((s: any) => s.id === evalSuiteRunMatch[1]);
          if (suite) {
            const run = {
              id: `run-${String(mockEvalRuns.length + 1).padStart(3, '0')}`,
              tenant_id: tenantId,
              suite_id: suite.id,
              suite_name: suite.name,
              agent_id: suite.agent_id,
              status: 'completed',
              score: 0.85,
              total_cases: suite.test_cases.length,
              passed_cases: suite.test_cases.length,
              failed_cases: 0,
              results: suite.test_cases.map((tc: any) => ({
                test_case_id: tc.id,
                test_case_name: tc.name,
                status: 'passed',
                actual_output: 'Simulated: ' + tc.expected_output,
                latency_ms: 150 + Math.floor(Math.random() * 300),
                score: 0.85,
                reason: '',
              })),
              started_at: new Date().toISOString(),
              completed_at: new Date().toISOString(),
            };
            mockEvalRuns.unshift(run);
            res.end(jsonResponse(run, tenantId));
          } else {
            res.statusCode = 404;
            res.end(JSON.stringify({ error: { code: 'NOT_FOUND', message: 'Test suite not found' } }));
          }
          return;
        }
        const evalSuiteDetailMatch = url.match(/^\/evals\/suites\/([^/]+)$/);
        if (evalSuiteDetailMatch && req.method === 'GET') {
          const suite = mockEvalSuites.find((s: any) => s.id === evalSuiteDetailMatch[1]);
          if (suite) {
            res.end(jsonResponse(suite, tenantId));
          } else {
            res.statusCode = 404;
            res.end(JSON.stringify({ error: { code: 'NOT_FOUND', message: 'Test suite not found' } }));
          }
          return;
        }
        if (url.startsWith('/evals/suites') && req.method === 'POST') {
          let body = '';
          req.on('data', (chunk: Buffer) => { body += chunk.toString(); });
          req.on('end', () => {
            const suite = JSON.parse(body);
            suite.id = `suite-${String(mockEvalSuites.length + 1).padStart(3, '0')}`;
            suite.tenant_id = tenantId;
            suite.created_at = new Date().toISOString();
            suite.updated_at = new Date().toISOString();
            mockEvalSuites.push(suite);
            res.end(jsonResponse(suite, tenantId));
          });
          return;
        }
        if (url.startsWith('/evals/suites') && req.method === 'GET') {
          res.end(jsonResponse(mockEvalSuites, tenantId));
          return;
        }

        // ---- Compliance Reports ----
        const complianceReportDetailMatch = url.match(/^\/compliance\/reports\/([^/]+)$/);
        if (complianceReportDetailMatch && req.method === 'GET') {
          const report = mockComplianceReports.find((r) => r.id === complianceReportDetailMatch[1]);
          if (report) {
            res.end(jsonResponse(report, tenantId));
          } else {
            res.statusCode = 404;
            res.end(JSON.stringify({ error: { code: 'NOT_FOUND', message: 'Report not found' } }));
          }
          return;
        }
        if (url.startsWith('/compliance/reports') && req.method === 'POST') {
          let body = '';
          req.on('data', (chunk: Buffer) => { body += chunk.toString(); });
          req.on('end', () => {
            const { profile_id, period_start, period_end } = JSON.parse(body);
            const profiles: Record<string, string> = {
              'gcc-sa': 'Saudi Arabia (NDMO)',
              'gcc-ae': 'UAE (NESA)',
              'gcc-qa': 'Qatar (NIA)',
              'gov-tr': 'Turkey (KVKK)',
              'eu-gdpr': 'EU GDPR',
              'fedramp-moderate': 'FedRAMP Moderate',
            };
            const profileName = profiles[profile_id] || 'Unknown';
            const report = {
              id: `rpt-${String(mockComplianceReports.length + 1).padStart(3, '0')}`,
              tenant_id: tenantId,
              profile_id,
              profile_name: profileName,
              title: `${profileName} Compliance Report`,
              status: 'completed',
              format: 'json',
              generated_at: new Date().toISOString(),
              period_start: period_start || new Date(Date.now() - 30 * 86400000).toISOString(),
              period_end: period_end || new Date().toISOString(),
              sections: [
                { title: 'Data Residency', status: 'compliant', description: 'All data stored within authorized regions', findings: ['No cross-region transfers detected'], evidence: ['Storage audit logs'] },
                { title: 'PII Protection', status: 'compliant', description: 'PII scrubbing active', findings: ['PII scrubbing enabled'], evidence: ['PII scrubber config'] },
                { title: 'Access Control', status: 'compliant', description: 'RBAC and tenant isolation enforced', findings: ['mTLS enforced'], evidence: ['RBAC policy config'] },
                { title: 'Audit Trail', status: 'compliant', description: 'Immutable audit logs retained', findings: ['All actions logged'], evidence: ['Hash chain verification'] },
              ],
            };
            mockComplianceReports.push(report);
            res.statusCode = 201;
            res.end(jsonResponse(report, tenantId));
          });
          return;
        }
        if (url.startsWith('/compliance/reports') && req.method === 'GET') {
          res.end(jsonResponse(mockComplianceReports, tenantId));
          return;
        }

        // ---- Existing ----
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
