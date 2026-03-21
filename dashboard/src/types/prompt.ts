export interface Prompt {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  agent_id: string;
  active_version: number;
  created_at: string;
  updated_at: string;
}

export interface PromptVersion {
  id: string;
  prompt_id: string;
  version: number;
  content: string;
  change_log: string;
  metrics?: VersionMetrics;
  created_at: string;
}

export interface VersionMetrics {
  avg_latency_ms: number;
  success_rate: number;
  tokens_used: number;
  invocations: number;
}
