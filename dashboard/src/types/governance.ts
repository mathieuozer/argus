export interface ClassificationPolicy {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  match_pattern: string;
  match_type: string;
  classification: string;
  auto_apply: boolean;
  applied_count: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface RetentionPolicy {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  classification: string;
  retention_days: number;
  action: string;
  enabled: boolean;
  last_executed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface AccessLog {
  id: string;
  tenant_id: string;
  source_id: string;
  source_name: string;
  agent_id: string;
  access_type: string;
  user_id: string;
  timestamp: string;
}

export interface PIIField {
  field_name: string;
  pii_type: string;
  confidence: number;
  sample_count: number;
  recommendation: string;
}

export interface PIIScanResult {
  id: string;
  tenant_id: string;
  source_id: string;
  source_name: string;
  pii_fields: PIIField[];
  total_fields: number;
  pii_field_count: number;
  risk_score: number;
  scanned_at: string;
}

export interface ComplianceMapping {
  id: string;
  tenant_id: string;
  source_id: string;
  source_name: string;
  framework: string;
  article_ref: string;
  requirement: string;
  status: string;
  evidence: string[];
  created_at: string;
  updated_at: string;
}

export interface DataSteward {
  id: string;
  tenant_id: string;
  user_id: string;
  name: string;
  email: string;
  domains: string[];
  source_ids: string[];
  role: string;
  assigned_at: string;
}

export interface GovernanceSummary {
  total_classification_policies: number;
  active_classification_policies: number;
  total_retention_policies: number;
  total_access_logs: number;
  total_pii_scans: number;
  high_risk_sources: number;
  total_compliance_mappings: number;
  compliant_mappings: number;
  partial_mappings: number;
  non_compliant_mappings: number;
  total_stewards: number;
}
