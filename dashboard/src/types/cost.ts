export interface CostBreakdown {
  group: string;
  cost_usd: number;
  tokens_used: number;
  task_count: number;
}

export interface CostTrend {
  timestamp: string;
  cost_usd: number;
  tokens_used: number;
}

export interface CostBudget {
  id: string;
  agent_id: string | null;
  budget_usd: number;
  spent_usd: number;
  period: 'daily' | 'weekly' | 'monthly';
  alert_threshold: number;
  enabled: boolean;
  created_at: string;
}

export interface CostAnomaly {
  id: string;
  agent_id: string;
  expected_usd: number;
  actual_usd: number;
  ratio: number;
  detected_at: string;
  status: 'open' | 'acknowledged' | 'resolved';
}
