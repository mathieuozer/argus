export interface DQRule {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type: string;
  agent_id: string;
  field: string;
  operator: string;
  threshold: string;
  severity: 'critical' | 'warning' | 'info';
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface DQScore {
  id: string;
  tenant_id: string;
  agent_id: string;
  overall_score: number;
  completeness_score: number;
  accuracy_score: number;
  consistency_score: number;
  timeliness_score: number;
  total_checks: number;
  passed_checks: number;
  failed_checks: number;
  evaluated_at: string;
}

export interface DQViolation {
  id: string;
  tenant_id: string;
  rule_id: string;
  rule_name: string;
  agent_id: string;
  field: string;
  value: string;
  expected: string;
  severity: 'critical' | 'warning' | 'info';
  message: string;
  occurred_at: string;
}

export interface DriftPoint {
  timestamp: string;
  metric: string;
  value: number;
  baseline: number;
  is_anomaly: boolean;
}

export interface ColumnProfile {
  name: string;
  type: string;
  null_rate: number;
  unique_rate: number;
  min_value?: string;
  max_value?: string;
  mean_value?: string;
  top_values?: string[];
}

export interface DataProfile {
  id: string;
  tenant_id: string;
  agent_id: string;
  source_id: string;
  row_count: number;
  column_count: number;
  null_rate: number;
  duplicate_rate: number;
  completeness: number;
  column_profiles: ColumnProfile[];
  profiled_at: string;
}

export interface DataContract {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  producer_agent: string;
  consumer_agents: string[];
  source_id: string;
  schema_spec: Record<string, string>;
  freshness_spec?: {
    max_staleness_seconds: number;
    refresh_schedule: string;
  };
  quality_spec?: {
    min_completeness: number;
    min_accuracy: number;
    max_null_rate: number;
  };
  status: string;
  last_checked_at: string;
  created_at: string;
  updated_at: string;
}

export interface QualityTrend {
  agent_id: string;
  points: TrendPoint[];
}

export interface TrendPoint {
  timestamp: string;
  overall: number;
  completeness: number;
  accuracy: number;
  consistency: number;
  timeliness: number;
}

export interface QualityIncident {
  id: string;
  tenant_id: string;
  agent_id: string;
  contract_id: string;
  title: string;
  description: string;
  severity: 'critical' | 'warning' | 'info';
  status: string;
  violation_ids: string[];
  created_at: string;
  resolved_at?: string;
}

export interface Anomaly {
  id: string;
  tenant_id: string;
  agent_id: string;
  metric_name: string;
  expected: number;
  actual: number;
  deviation: number;
  severity: 'critical' | 'warning' | 'info';
  detected_at: string;
}

export interface DQSummary {
  total_rules: number;
  active_rules: number;
  total_violations: number;
  critical_violations: number;
  avg_quality_score: number;
  total_contracts: number;
  active_contracts: number;
  violated_contracts: number;
  total_incidents: number;
  open_incidents: number;
  total_anomalies: number;
  agent_count: number;
}
