import type { SLOStatus } from '../../types/slo';

interface SLOStatusBadgeProps {
  status: SLOStatus;
}

const STATUS_CONFIG: Record<SLOStatus, { label: string; className: string }> = {
  met: { label: 'Met', className: 'badge-success' },
  at_risk: { label: 'At Risk', className: 'badge-warning' },
  breached: { label: 'Breached', className: 'badge-error' },
};

function SLOStatusBadge({ status }: SLOStatusBadgeProps) {
  const config = STATUS_CONFIG[status];
  return <span className={`badge ${config.className}`}>{config.label}</span>;
}

export default SLOStatusBadge;
