import type { AuditEntry } from '../../types/audit';

interface AuditTableProps {
  entries: AuditEntry[];
}

function AuditTable({ entries }: AuditTableProps) {
  if (entries.length === 0) {
    return <p className="text-muted">No audit entries found.</p>;
  }

  return (
    <div className="audit-table-container">
      <table className="table">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>Actor</th>
            <th>Action</th>
            <th>Resource</th>
            <th>Details</th>
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
