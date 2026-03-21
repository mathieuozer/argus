export interface AuditEntry {
  id: string;
  tenant_id: string;
  actor: string;
  action: string;
  resource: string;
  details: string;
  timestamp: string;
}

export interface AuditFilters {
  actor?: string;
  action?: string;
  from?: string;
  to?: string;
  limit?: number;
}
