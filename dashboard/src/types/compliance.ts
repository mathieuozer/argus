export interface ComplianceReport {
  id: string;
  tenantId: string;
  profileId: string;
  profileName: string;
  title: string;
  status: 'generating' | 'completed' | 'failed';
  format: string;
  sections: ComplianceSection[];
  generatedAt: string;
  periodStart: string;
  periodEnd: string;
}

export interface ComplianceSection {
  title: string;
  status: 'compliant' | 'non_compliant' | 'partial';
  description: string;
  findings: string[];
  evidence: string[];
}
