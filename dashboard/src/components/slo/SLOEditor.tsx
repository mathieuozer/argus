import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { SLO, SLOType } from '../../types/slo';

interface SLOEditorProps {
  onSave: (slo: Omit<SLO, 'id' | 'current' | 'budget_remaining' | 'status' | 'created_at'>) => void;
  onCancel: () => void;
}

function SLOEditor({ onSave, onCancel }: SLOEditorProps) {
  const { t } = useTranslation();
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
        <label>{t('sloEditor.agentId')}</label>
        <input type="text" value={agentId} onChange={(e) => setAgentId(e.target.value)} required />
      </div>
      <div className="form-group">
        <label>{t('sloEditor.sloName')}</label>
        <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder={t('sloEditor.sloNamePlaceholder')} required />
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>{t('sloEditor.type')}</label>
          <select value={sloType} onChange={(e) => setSloType(e.target.value as SLOType)}>
            <option value="availability">{t('sloEditor.typeOptions.availability')}</option>
            <option value="latency_p99">{t('sloEditor.typeOptions.latencyP99')}</option>
            <option value="error_rate">{t('sloEditor.typeOptions.errorRate')}</option>
          </select>
        </div>
        <div className="form-group">
          <label>{t('sloEditor.targetPct')}</label>
          <input type="number" step="0.01" min="0" max="100" value={target} onChange={(e) => setTarget(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>{t('sloEditor.window')}</label>
          <select value={window} onChange={(e) => setWindow(e.target.value as SLO['window'])}>
            <option value="7d">{t('sloEditor.windowOptions.7d')}</option>
            <option value="30d">{t('sloEditor.windowOptions.30d')}</option>
            <option value="90d">{t('sloEditor.windowOptions.90d')}</option>
          </select>
        </div>
      </div>
      <div className="form-actions">
        <button type="button" className="btn btn-secondary" onClick={onCancel}>{t('common.cancel')}</button>
        <button type="submit" className="btn btn-primary">{t('sloEditor.createSlo')}</button>
      </div>
    </form>
  );
}

export default SLOEditor;
