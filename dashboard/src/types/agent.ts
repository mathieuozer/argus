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
