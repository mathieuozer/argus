export interface ComplianceReport {
  id: string;
  tenant_id: string;
  profile_id: string;
  profile_name: string;
  title: string;
  status: 'generating' | 'completed' | 'failed';
  format: string;
  sections: ComplianceSection[];
  generated_at: string;
  period_start: string;
  period_end: string;
}

export interface ComplianceSection {
  title: string;
  status: 'compliant' | 'non_compliant' | 'partial';
  description: string;
  findings: string[];
  evidence: string[];
}
