import { useState } from 'react';
import type { CostBudget } from '../../types/cost';

interface BudgetEditorProps {
  onSave: (budget: Omit<CostBudget, 'id' | 'spent_usd' | 'created_at'>) => void;
  onCancel: () => void;
}

function BudgetEditor({ onSave, onCancel }: BudgetEditorProps) {
  const [agentId, setAgentId] = useState('');
  const [budgetUsd, setBudgetUsd] = useState('100');
  const [period, setPeriod] = useState<CostBudget['period']>('monthly');
  const [threshold, setThreshold] = useState('0.80');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({
      agent_id: agentId || null,
      budget_usd: parseFloat(budgetUsd),
      period,
      alert_threshold: parseFloat(threshold),
      enabled: true,
    });
  };

  return (
    <form className="budget-editor" onSubmit={handleSubmit}>
      <div className="form-group">
        <label>Agent ID (leave empty for tenant-wide)</label>
        <input type="text" value={agentId} onChange={(e) => setAgentId(e.target.value)} placeholder="e.g. budget-reconciler" />
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>Budget (USD)</label>
          <input type="number" step="0.01" min="0" value={budgetUsd} onChange={(e) => setBudgetUsd(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Period</label>
          <select value={period} onChange={(e) => setPeriod(e.target.value as CostBudget['period'])}>
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
          </select>
        </div>
        <div className="form-group">
          <label>Alert Threshold</label>
          <select value={threshold} onChange={(e) => setThreshold(e.target.value)}>
            <option value="0.50">50%</option>
            <option value="0.70">70%</option>
            <option value="0.80">80%</option>
            <option value="0.90">90%</option>
          </select>
        </div>
      </div>
      <div className="form-actions">
        <button type="button" className="btn btn-secondary" onClick={onCancel}>Cancel</button>
        <button type="submit" className="btn btn-primary">Set Budget</button>
      </div>
    </form>
  );
}

export default BudgetEditor;
