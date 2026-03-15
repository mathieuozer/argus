export type TaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'awaiting_approval';

export interface Task {
  id: string;
  tenant_id: string;
  agent_id: string;
  status: TaskStatus;
  input_hash: string;
  started_at: string;
  completed_at: string | null;
  cost_usd: number;
  tokens_used: number;
}
