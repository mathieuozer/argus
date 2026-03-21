import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useCostStore } from '../stores/costStore';
import CostBreakdownChart from '../components/costs/CostBreakdownChart';
import CostTrendChart from '../components/costs/CostTrendChart';
import BudgetCard from '../components/costs/BudgetCard';
import BudgetEditor from '../components/costs/BudgetEditor';
import CostAnomalyRow from '../components/costs/CostAnomalyRow';

function CostPage() {
  const { t } = useTranslation();
  const { breakdown, trends, budgets, anomalies, loading, error, fetchBreakdown, fetchTrends, fetchBudgets, fetchAnomalies, createBudget } = useCostStore();
  const [showBudgetEditor, setShowBudgetEditor] = useState(false);
  const [groupBy, setGroupBy] = useState('agent');

  useEffect(() => {
    fetchBreakdown(groupBy);
    fetchTrends();
    fetchBudgets();
    fetchAnomalies();
  }, [fetchBreakdown, fetchTrends, fetchBudgets, fetchAnomalies, groupBy]);

  const totalCost = breakdown.reduce((sum, b) => sum + b.cost_usd, 0);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('costs.title')}</h2>
          <p>{t('costs.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <button className="btn btn-primary" onClick={() => setShowBudgetEditor(true)}>{t('costs.setBudget')}</button>
          <button className="btn" onClick={() => { fetchBreakdown(groupBy); fetchTrends(); fetchBudgets(); fetchAnomalies(); }} disabled={loading}>{t('common.refresh')}</button>
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}

      <div className="stat-row">
        <div className="stat-card">
          <span className="stat-label">{t('costs.totalSpend')}</span>
          <span className="stat-value">${totalCost.toFixed(2)}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">{t('costs.activeBudgets')}</span>
          <span className="stat-value">{budgets.filter((b) => b.enabled).length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">{t('costs.costAnomalies')}</span>
          <span className="stat-value">{anomalies.filter((a) => a.status === 'open').length}</span>
        </div>
      </div>

      {showBudgetEditor && (
        <div className="card" style={{ marginBottom: 'var(--spacing-lg)' }}>
          <BudgetEditor
            onSave={(budget) => { createBudget(budget); setShowBudgetEditor(false); }}
            onCancel={() => setShowBudgetEditor(false)}
          />
        </div>
      )}

      <div className="grid grid-2">
        <div>
          <div className="chart-group-header">
            <span>{t('costs.breakdownBy')}</span>
            <select className="filter-select" value={groupBy} onChange={(e) => setGroupBy(e.target.value)}>
              <option value="agent">{t('costs.groupByAgent')}</option>
              <option value="operation">{t('costs.groupByOperation')}</option>
              <option value="day">{t('costs.groupByDay')}</option>
            </select>
          </div>
          <CostBreakdownChart data={breakdown} />
        </div>
        <CostTrendChart data={trends} />
      </div>

      {budgets.length > 0 && (
        <div style={{ marginTop: 'var(--spacing-lg)' }}>
          <h3>{t('costs.budgets')}</h3>
          <div className="grid grid-auto">
            {budgets.map((budget) => (
              <BudgetCard key={budget.id} budget={budget} />
            ))}
          </div>
        </div>
      )}

      {anomalies.length > 0 && (
        <div style={{ marginTop: 'var(--spacing-lg)' }}>
          <h3>{t('costs.anomaliesTitle')}</h3>
          <table className="table">
            <thead>
              <tr>
                <th>{t('costs.anomalyAgent')}</th>
                <th>{t('costs.expected')}</th>
                <th>{t('costs.actual')}</th>
                <th>{t('costs.ratio')}</th>
                <th>{t('costs.detected')}</th>
                <th>{t('common.status')}</th>
              </tr>
            </thead>
            <tbody>
              {anomalies.map((anomaly) => (
                <CostAnomalyRow key={anomaly.id} anomaly={anomaly} />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {loading && breakdown.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>{t('costs.loadingCosts')}</span>
        </div>
      )}
    </div>
  );
}

export default CostPage;
