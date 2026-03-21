export type AgentStatus = 'discovered' | 'healthy' | 'degraded' | 'failed' | 'quarantined';

export interface Agent {
  id: string;
  tenant_id: string;
  version: string;
  framework: string;
  capabilities: string[];
  status: AgentStatus;
  svid_uri: string;
  last_seen: string;
  node_id: string;
}

export interface AgentDetail extends Agent {
  tasks_completed: number;
  tasks_failed: number;
  total_cost_usd: number;
  total_tokens: number;
  avg_latency_ms: number;
  uptime_pct: number;
}
