import { useTranslation } from 'react-i18next';
import type { AgentStatus } from '../../types/agent';
import type { TaskStatus } from '../../types/task';
import type { AlertStatus } from '../../types/alert';

type BadgeStatus = AgentStatus | TaskStatus | AlertStatus | string;

interface StatusBadgeProps {
  status: BadgeStatus;
}

function getStatusClass(status: BadgeStatus): string {
  switch (status) {
    case 'healthy':
    case 'completed':
    case 'resolved':
      return 'badge-success';
    case 'degraded':
    case 'warning':
    case 'awaiting_approval':
    case 'acknowledged':
      return 'badge-warning';
    case 'failed':
    case 'quarantined':
    case 'open':
    case 'false_positive':
      return 'badge-danger';
    case 'discovered':
    case 'info':
      return 'badge-info';
    case 'running':
      return 'badge-primary';
    case 'pending':
      return 'badge-muted';
    default:
      return 'badge-muted';
  }
}

const STATUS_KEY_MAP: Record<string, string> = {
  healthy: 'statuses.healthy',
  degraded: 'statuses.degraded',
  failed: 'statuses.failed',
  quarantined: 'statuses.quarantined',
  discovered: 'statuses.discovered',
  pending: 'statuses.pending',
  running: 'statuses.running',
  completed: 'statuses.completed',
  awaiting_approval: 'statuses.awaitingApproval',
  open: 'statuses.open',
  acknowledged: 'statuses.acknowledged',
  resolved: 'statuses.resolved',
  false_positive: 'statuses.falsePositive',
};

function StatusBadge({ status }: StatusBadgeProps) {
  const { t } = useTranslation();
  const labelKey = STATUS_KEY_MAP[status];
  const label = labelKey ? t(labelKey) : status.replace(/_/g, ' ');

  return (
    <span className={`badge ${getStatusClass(status)}`}>
      {label}
    </span>
  );
}

export default StatusBadge;
