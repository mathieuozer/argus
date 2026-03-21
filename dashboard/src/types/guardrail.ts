export interface GuardrailRule {
  id: string;
  tenantId: string;
  name: string;
  description: string;
  type: 'pii_detection' | 'prompt_injection' | 'toxicity' | 'blocklist' | 'schema_enforcement' | 'custom_regex';
  pattern: string;
  action: 'block' | 'warn' | 'log';
  enabled: boolean;
  agentIds: string[];
  createdAt: string;
}

export interface GuardrailViolation {
  id: string;
  tenantId: string;
  ruleId: string;
  ruleName: string;
  ruleType: string;
  agentId: string;
  spanId: string;
  action: string;
  content: string;
  createdAt: string;
}

export interface GuardrailStats {
  totalChecks: number;
  totalViolations: number;
  passRate: number;
  byRule: Record<string, number>;
  byAgent: Record<string, number>;
}
