import { useTranslation } from 'react-i18next';
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
  const { t } = useTranslation();

  if (violations.length === 0) {
    return <p className="text-muted">{t('dataQuality.noViolations')}</p>;
  }

  return (
    <div className="violation-list">
      <table className="table">
        <thead>
          <tr>
            <th>{t('dataQuality.agent')}</th>
            <th>{t('dataQuality.rule')}</th>
            <th>{t('dataQuality.severity')}</th>
            <th>{t('dataQuality.message')}</th>
            <th>{t('dataQuality.time')}</th>
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
