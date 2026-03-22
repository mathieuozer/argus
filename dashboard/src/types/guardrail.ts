export interface GuardrailRule {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type: 'pii_detection' | 'prompt_injection' | 'toxicity' | 'blocklist' | 'schema_enforcement' | 'custom_regex';
  pattern: string;
  action: 'block' | 'warn' | 'log';
  enabled: boolean;
  agent_ids: string[];
  created_at: string;
}

export interface GuardrailViolation {
  id: string;
  tenant_id: string;
  rule_id: string;
  rule_name: string;
  rule_type: string;
  agent_id: string;
  span_id: string;
  action: string;
  content: string;
  created_at: string;
}

export interface GuardrailStats {
  total_checks: number;
  total_violations: number;
  pass_rate: number;
  by_rule: Record<string, number>;
  by_agent: Record<string, number>;
}
