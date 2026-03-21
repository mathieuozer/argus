export interface CatalogSource {
  id: string;
  type: 'database' | 'api' | 'storage' | 'tool';
  name: string;
  identifier: string;
  agents: string[];
  access_types: string[];
  first_seen: string;
  last_seen: string;
  span_count: number;
  tier: number;
}

export interface LineageNode {
  id: string;
  type: 'agent' | 'database' | 'storage' | 'external_api' | 'tool';
}

export interface LineageEdge {
  source: string;
  target: string;
  label: string;
  span_count: number;
}

export interface LineageGraph {
  nodes: LineageNode[];
  edges: LineageEdge[];
}

export interface ToolUsage {
  tool: string;
  agent_id: string;
  call_count: number;
  avg_duration_ms: number;
}
