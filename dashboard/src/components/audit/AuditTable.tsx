import { useTranslation } from 'react-i18next';
import type { AuditEntry } from '../../types/audit';

interface AuditTableProps {
  entries: AuditEntry[];
}

function AuditTable({ entries }: AuditTableProps) {
  const { t } = useTranslation();

  if (entries.length === 0) {
    return <p className="text-muted">{t('audit.noEntriesFound')}</p>;
  }

  return (
    <div className="audit-table-container">
      <table className="table">
        <thead>
          <tr>
            <th>{t('audit.timestamp')}</th>
            <th>{t('audit.actor')}</th>
            <th>{t('audit.action')}</th>
            <th>{t('audit.resource')}</th>
            <th>{t('audit.details')}</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr key={entry.id}>
              <td className="text-muted mono">{new Date(entry.timestamp).toLocaleString()}</td>
              <td>{entry.actor}</td>
              <td><span className="badge">{entry.action}</span></td>
              <td className="mono">{entry.resource}</td>
              <td className="text-muted">{entry.details || '-'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default AuditTable;
