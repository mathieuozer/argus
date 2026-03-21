import type { CostBudget } from '../../types/cost';

interface BudgetCardProps {
  budget: CostBudget;
}

function BudgetCard({ budget }: BudgetCardProps) {
  const pct = budget.budget_usd > 0 ? (budget.spent_usd / budget.budget_usd) * 100 : 0;
  const isOver = pct >= 100;
  const isWarning = pct >= budget.alert_threshold * 100;

  const barColor = isOver
    ? 'var(--color-error)'
    : isWarning
      ? 'var(--color-warning)'
      : 'var(--color-success)';

  return (
    <div className="card budget-card">
      <div className="card-header">
        <h4>{budget.agent_id || 'Tenant-wide'}</h4>
        <span className="badge">{budget.period}</span>
      </div>
      <div className="budget-gauge">
        <div className="budget-amounts">
          <span className="budget-spent">${budget.spent_usd.toFixed(2)}</span>
          <span className="text-muted">/ ${budget.budget_usd.toFixed(2)}</span>
        </div>
        <div className="budget-bar-bg">
          <div
            className="budget-bar-fill"
            style={{ width: `${Math.min(pct, 100)}%`, backgroundColor: barColor }}
          />
        </div>
        <div className="budget-pct">
          {pct.toFixed(1)}% used
          {isWarning && !isOver && <span className="text-warning"> (alert threshold reached)</span>}
          {isOver && <span className="text-error"> (over budget)</span>}
        </div>
      </div>
    </div>
  );
}

export default BudgetCard;
