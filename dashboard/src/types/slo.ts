export type SLOType = 'availability' | 'latency_p99' | 'error_rate';
export type SLOStatus = 'met' | 'at_risk' | 'breached';

export interface SLO {
  id: string;
  agent_id: string;
  name: string;
  type: SLOType;
  target: number;
  window: '7d' | '30d' | '90d';
  current: number;
  budget_remaining: number;
  status: SLOStatus;
  enabled: boolean;
  created_at: string;
}

export interface ErrorBudgetPoint {
  timestamp: string;
  remaining: number;
}
