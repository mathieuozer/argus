import { useTranslation } from 'react-i18next';
import type { AuditEntry } from '../../types/audit';

interface AuditExportButtonProps {
  entries: AuditEntry[];
}

function exportCSV(entries: AuditEntry[]) {
  const headers = ['ID', 'Timestamp', 'Actor', 'Action', 'Resource', 'Details'];
  const rows = entries.map((e) => [
    e.id,
    e.timestamp,
    e.actor,
    e.action,
    e.resource,
    e.details,
  ]);

  const csv = [headers.join(','), ...rows.map((r) => r.map((v) => `"${v}"`).join(','))].join('\n');
  const blob = new Blob([csv], { type: 'text/csv' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `audit-log-${new Date().toISOString().slice(0, 10)}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

function AuditExportButton({ entries }: AuditExportButtonProps) {
  const { t } = useTranslation();

  return (
    <button className="btn btn-secondary" onClick={() => exportCSV(entries)} disabled={entries.length === 0}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
        <polyline points="7 10 12 15 17 10" />
        <line x1="12" y1="15" x2="12" y2="3" />
      </svg>
      {t('audit.exportCsv')}
    </button>
  );
}

export default AuditExportButton;
