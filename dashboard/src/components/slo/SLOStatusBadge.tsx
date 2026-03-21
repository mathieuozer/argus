import { useTranslation } from 'react-i18next';
import type { SLOStatus } from '../../types/slo';

interface SLOStatusBadgeProps {
  status: SLOStatus;
}

const STATUS_CLASS: Record<SLOStatus, string> = {
  met: 'badge-success',
  at_risk: 'badge-warning',
  breached: 'badge-error',
};

const STATUS_KEY: Record<SLOStatus, string> = {
  met: 'slos.met',
  at_risk: 'slos.atRisk',
  breached: 'slos.breached',
};

function SLOStatusBadge({ status }: SLOStatusBadgeProps) {
  const { t } = useTranslation();
  return <span className={`badge ${STATUS_CLASS[status]}`}>{t(STATUS_KEY[status])}</span>;
}

export default SLOStatusBadge;
