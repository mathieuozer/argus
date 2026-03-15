import { useState } from 'react';
import type { PredictiveAlert } from '../../types/alert';
import StatusBadge from '../agents/StatusBadge';

interface AlertRowProps {
  alert: PredictiveAlert;
}

function formatTTF(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`;
  }
  const hours = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
}

function getProbabilityColor(probability: number): string {
  if (probability >= 0.8) return 'var(--color-danger)';
  if (probability >= 0.5) return 'var(--color-warning)';
  return 'var(--color-success)';
}

function getPrecursorBadgeClass(type: string): string {
  switch (type) {
    case 'latency_spike':
      return 'badge-warning';
    case 'token_escalation':
      return 'badge-danger';
    case 'retry_storm':
      return 'badge-danger';
    case 'error_rate_inflection':
      return 'badge-warning';
    default:
      return 'badge-muted';
  }
}

function formatPrecursorLabel(type: string): string {
  return type.replace(/_/g, ' ');
}

function formatTimestamp(isoString: string): string {
  const date = new Date(isoString);
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function AlertRow({ alert }: AlertRowProps) {
  const [showEvidence, setShowEvidence] = useState(false);
  const probabilityPct = (alert.probability * 100).toFixed(0);
  const probabilityColor = getProbabilityColor(alert.probability);

  return (
    <>
      <tr>
        <td>
          <span className="text-mono text-sm">{alert.id.slice(0, 8)}</span>
        </td>
        <td>
          <span className="font-medium">{alert.agent_id}</span>
        </td>
        <td>
          <span style={{ color: probabilityColor, fontWeight: 600 }}>
            {probabilityPct}%
          </span>
        </td>
        <td>
          <span className={`badge ${getPrecursorBadgeClass(alert.precursor_type)}`}>
            {formatPrecursorLabel(alert.precursor_type)}
          </span>
        </td>
        <td>
          <span className="text-mono">{formatTTF(alert.estimated_ttf)}</span>
        </td>
        <td>
          <StatusBadge status={alert.status} />
        </td>
        <td>
          <span className="text-sm text-muted">{formatTimestamp(alert.created_at)}</span>
        </td>
        <td>
          {alert.evidence.length > 0 && (
            <button
              className="btn btn-sm"
              onClick={() => setShowEvidence(!showEvidence)}
            >
              {showEvidence ? 'Hide' : 'Show'} ({alert.evidence.length})
            </button>
          )}
        </td>
      </tr>
      {showEvidence && alert.evidence.length > 0 && (
        <tr>
          <td colSpan={8} style={{ padding: 0 }}>
            <div
              className="animate-fade-in"
              style={{
                padding: 'var(--space-3) var(--space-4)',
                backgroundColor: 'var(--color-surface)',
                borderBottom: '1px solid var(--color-border)',
              }}
            >
              <span className="text-sm font-medium text-muted" style={{ display: 'block', marginBottom: 'var(--space-2)' }}>
                Evidence Points
              </span>
              <ul style={{ listStyle: 'none', display: 'flex', flexDirection: 'column', gap: 'var(--space-1)' }}>
                {alert.evidence.map((item, index) => (
                  <li key={index} className="text-sm text-mono" style={{ color: 'var(--color-text-muted)' }}>
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

export default AlertRow;
