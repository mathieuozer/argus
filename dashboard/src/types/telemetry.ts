export interface TelemetrySpan {
  span_id: string;
  trace_id: string;
  tenant_id: string;
  agent_id: string;
  task_id: string;
  operation_name: string;
  started_at: string;
  duration_ms: number;
  tier: 1 | 2 | 3;
  attributes: Record<string, string>;
  error_code: string | null;
}

export interface TimeSeriesPoint {
  timestamp: string;
  value: number;
}

export interface AgentTimeSeries {
  latency_p50: TimeSeriesPoint[];
  latency_p99: TimeSeriesPoint[];
  token_rate: TimeSeriesPoint[];
  error_rate: TimeSeriesPoint[];
  cost: TimeSeriesPoint[];
}

export interface PlatformTimeSeries {
  total_tasks: TimeSeriesPoint[];
  active_agents: TimeSeriesPoint[];
  total_cost: TimeSeriesPoint[];
  alert_count: TimeSeriesPoint[];
  error_rate: TimeSeriesPoint[];
  avg_latency: TimeSeriesPoint[];
}
