export interface DQRule {
  id: string;
  agent_id: string;
  name: string;
  type: 'schema' | 'range' | 'regex' | 'custom';
  target: 'input' | 'output' | 'attribute';
  schema: Record<string, unknown>;
  severity: 'critical' | 'warning' | 'info';
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface DQScore {
  id: string;
  agent_id: string;
  completeness: number;
  conformance: number;
  consistency: number;
  freshness: number;
  overall: number;
  sample_size: number;
  computed_at: string;
}

export interface DQViolation {
  id: string;
  rule_id: string;
  rule_name: string;
  agent_id: string;
  severity: 'critical' | 'warning' | 'info';
  message: string;
  occurred_at: string;
}

export interface DriftPoint {
  timestamp: string;
  consistency: number;
  baseline: number;
}
