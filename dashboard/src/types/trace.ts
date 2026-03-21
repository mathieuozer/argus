export interface TraceSpan {
  span_id: string;
  operation_name: string;
  duration_ms: number;
  started_at: string;
  agent_id?: string;
  attributes: Record<string, string>;
  error_code: string | null;
  children: TraceSpan[];
}

export interface TraceSummary {
  trace_id: string;
  root_operation: string;
  agent_id: string;
  total_spans: number;
  total_duration_ms: number;
  has_errors: boolean;
  started_at: string;
}

export interface TraceDetail {
  trace_id: string;
  root_span: TraceSpan;
  total_spans: number;
  total_duration_ms: number;
  has_errors: boolean;
}

export interface FlameGraphNode {
  name: string;
  value: number;
  children: FlameGraphNode[];
}
