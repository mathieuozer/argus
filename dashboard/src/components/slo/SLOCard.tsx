import { useTranslation } from 'react-i18next';
import type { SLO } from '../../types/slo';
import SLOStatusBadge from './SLOStatusBadge';

interface SLOCardProps {
  slo: SLO;
  onClick: () => void;
}

function SLOCard({ slo, onClick }: SLOCardProps) {
  const { t } = useTranslation();
  const budgetPct = slo.budget_remaining * 100;
  const budgetColor = budgetPct > 50
    ? 'var(--color-success)'
    : budgetPct > 20
      ? 'var(--color-warning)'
      : 'var(--color-error)';

  return (
    <div className="card slo-card clickable" onClick={onClick}>
      <div className="card-header">
        <div>
          <h4>{slo.name}</h4>
          <span className="text-muted mono">{slo.agent_id}</span>
        </div>
        <SLOStatusBadge status={slo.status} />
      </div>
      <div className="slo-metrics">
        <div className="slo-metric">
          <span className="slo-metric-label">{t('slos.target')}</span>
          <span className="slo-metric-value">{(slo.target * 100).toFixed(2)}%</span>
        </div>
        <div className="slo-metric">
          <span className="slo-metric-label">{t('slos.current')}</span>
          <span className="slo-metric-value" style={{ color: slo.current >= slo.target ? 'var(--color-success)' : 'var(--color-error)' }}>
            {(slo.current * 100).toFixed(2)}%
          </span>
        </div>
        <div className="slo-metric">
          <span className="slo-metric-label">{t('slos.window')}</span>
          <span className="slo-metric-value">{slo.window}</span>
        </div>
      </div>
      <div className="slo-budget">
        <div className="slo-budget-header">
          <span>{t('slos.errorBudget')}</span>
          <span style={{ color: budgetColor }}>{t('slos.remaining', { pct: budgetPct.toFixed(1) })}</span>
        </div>
        <div className="budget-bar-bg">
          <div
            className="budget-bar-fill"
            style={{ width: `${Math.max(budgetPct, 0)}%`, backgroundColor: budgetColor }}
          />
        </div>
      </div>
    </div>
  );
}

export default SLOCard;
