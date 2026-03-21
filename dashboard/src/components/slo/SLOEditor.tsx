import { useState } from 'react';
import type { SLO, SLOType } from '../../types/slo';

interface SLOEditorProps {
  onSave: (slo: Omit<SLO, 'id' | 'current' | 'budget_remaining' | 'status' | 'created_at'>) => void;
  onCancel: () => void;
}

function SLOEditor({ onSave, onCancel }: SLOEditorProps) {
  const [agentId, setAgentId] = useState('');
  const [name, setName] = useState('');
  const [sloType, setSloType] = useState<SLOType>('availability');
  const [target, setTarget] = useState('99.50');
  const [window, setWindow] = useState<SLO['window']>('30d');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({
      agent_id: agentId,
      name,
      type: sloType,
      target: parseFloat(target) / 100,
      window,
      enabled: true,
    });
  };

  return (
    <form className="slo-editor" onSubmit={handleSubmit}>
      <div className="form-group">
        <label>Agent ID</label>
        <input type="text" value={agentId} onChange={(e) => setAgentId(e.target.value)} required />
      </div>
      <div className="form-group">
        <label>SLO Name</label>
        <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Availability SLO" required />
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>Type</label>
          <select value={sloType} onChange={(e) => setSloType(e.target.value as SLOType)}>
            <option value="availability">Availability</option>
            <option value="latency_p99">Latency (p99)</option>
            <option value="error_rate">Error Rate</option>
          </select>
        </div>
        <div className="form-group">
          <label>Target (%)</label>
          <input type="number" step="0.01" min="0" max="100" value={target} onChange={(e) => setTarget(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Window</label>
          <select value={window} onChange={(e) => setWindow(e.target.value as SLO['window'])}>
            <option value="7d">7 days</option>
            <option value="30d">30 days</option>
            <option value="90d">90 days</option>
          </select>
        </div>
      </div>
      <div className="form-actions">
        <button type="button" className="btn btn-secondary" onClick={onCancel}>Cancel</button>
        <button type="submit" className="btn btn-primary">Create SLO</button>
      </div>
    </form>
  );
}

export default SLOEditor;
