import { useEffect, useState } from 'react';
import { useCostStore } from '../stores/costStore';
import CostBreakdownChart from '../components/costs/CostBreakdownChart';
import CostTrendChart from '../components/costs/CostTrendChart';
import BudgetCard from '../components/costs/BudgetCard';
import BudgetEditor from '../components/costs/BudgetEditor';
import CostAnomalyRow from '../components/costs/CostAnomalyRow';

function CostPage() {
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
          <h2>Cost Governance</h2>
          <p>Track spending, set budgets, detect cost anomalies</p>
        </div>
        <div className="page-header-actions">
          <button className="btn btn-primary" onClick={() => setShowBudgetEditor(true)}>Set Budget</button>
          <button className="btn" onClick={() => { fetchBreakdown(groupBy); fetchTrends(); fetchBudgets(); fetchAnomalies(); }} disabled={loading}>Refresh</button>
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}

      <div className="stat-row">
        <div className="stat-card">
          <span className="stat-label">Total Spend</span>
          <span className="stat-value">${totalCost.toFixed(2)}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Active Budgets</span>
          <span className="stat-value">{budgets.filter((b) => b.enabled).length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-label">Cost Anomalies</span>
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
            <span>Breakdown by</span>
            <select className="filter-select" value={groupBy} onChange={(e) => setGroupBy(e.target.value)}>
              <option value="agent">Agent</option>
              <option value="operation">Operation</option>
              <option value="day">Day</option>
            </select>
          </div>
          <CostBreakdownChart data={breakdown} />
        </div>
        <CostTrendChart data={trends} />
      </div>

      {budgets.length > 0 && (
        <div style={{ marginTop: 'var(--spacing-lg)' }}>
          <h3>Budgets</h3>
          <div className="grid grid-auto">
            {budgets.map((budget) => (
              <BudgetCard key={budget.id} budget={budget} />
            ))}
          </div>
        </div>
      )}

      {anomalies.length > 0 && (
        <div style={{ marginTop: 'var(--spacing-lg)' }}>
          <h3>Cost Anomalies</h3>
          <table className="table">
            <thead>
              <tr>
                <th>Agent</th>
                <th>Expected</th>
                <th>Actual</th>
                <th>Ratio</th>
                <th>Detected</th>
                <th>Status</th>
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
          <span>Loading cost data...</span>
        </div>
      )}
    </div>
  );
}

export default CostPage;
