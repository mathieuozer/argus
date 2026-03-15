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

function formatStatusLabel(status: BadgeStatus): string {
  return status.replace(/_/g, ' ');
}

function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span className={`badge ${getStatusClass(status)}`}>
      {formatStatusLabel(status)}
    </span>
  );
}

export default StatusBadge;
