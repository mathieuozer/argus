export type AlertSeverity = 'info' | 'warning' | 'critical';
export type AlertStatus = 'open' | 'acknowledged' | 'resolved' | 'false_positive';

export interface PredictiveAlert {
  id: string;
  tenant_id: string;
  agent_id: string;
  probability: number;
  estimated_ttf: number;
  precursor_type: string;
  evidence: string[];
  status: AlertStatus;
  created_at: string;
}
