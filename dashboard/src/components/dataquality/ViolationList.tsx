import type { DQViolation } from '../../types/dataquality';

interface ViolationListProps {
  violations: DQViolation[];
}

function getSeverityClass(severity: string): string {
  switch (severity) {
    case 'critical': return 'badge-error';
    case 'warning': return 'badge-warning';
    default: return 'badge-info';
  }
}

function ViolationList({ violations }: ViolationListProps) {
  if (violations.length === 0) {
    return <p className="text-muted">No recent violations.</p>;
  }

  return (
    <div className="violation-list">
      <table className="table">
        <thead>
          <tr>
            <th>Agent</th>
            <th>Rule</th>
            <th>Severity</th>
            <th>Message</th>
            <th>Time</th>
          </tr>
        </thead>
        <tbody>
          {violations.map((v) => (
            <tr key={v.id}>
              <td className="mono">{v.agent_id}</td>
              <td>{v.rule_name}</td>
              <td><span className={`badge ${getSeverityClass(v.severity)}`}>{v.severity}</span></td>
              <td>{v.message}</td>
              <td className="text-muted">{new Date(v.occurred_at).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default ViolationList;
