import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { DQRule } from '../../types/dataquality';

interface RuleEditorProps {
  onSave: (rule: Partial<DQRule>) => void;
  onCancel: () => void;
}

function RuleEditor({ onSave, onCancel }: RuleEditorProps) {
  const { t } = useTranslation();
  const [agentId, setAgentId] = useState('');
  const [name, setName] = useState('');
  const [ruleType, setRuleType] = useState('completeness');
  const [field, setField] = useState('');
  const [operator, setOperator] = useState('lt');
  const [threshold, setThreshold] = useState('');
  const [severity, setSeverity] = useState<DQRule['severity']>('warning');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({
      agent_id: agentId,
      name,
      type: ruleType,
      field,
      operator,
      threshold,
      severity,
      enabled: true,
    });
  };

  return (
    <form className="rule-editor" onSubmit={handleSubmit}>
      <div className="form-group">
        <label>{t('ruleEditor.agentId')}</label>
        <input type="text" value={agentId} onChange={(e) => setAgentId(e.target.value)} />
      </div>
      <div className="form-group">
        <label>{t('ruleEditor.ruleName')}</label>
        <input type="text" value={name} onChange={(e) => setName(e.target.value)} required />
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>{t('ruleEditor.type')}</label>
          <select value={ruleType} onChange={(e) => setRuleType(e.target.value)}>
            <option value="completeness">Completeness</option>
            <option value="accuracy">Accuracy</option>
            <option value="consistency">Consistency</option>
            <option value="timeliness">Timeliness</option>
            <option value="uniqueness">Uniqueness</option>
          </select>
        </div>
        <div className="form-group">
          <label>Field</label>
          <input type="text" value={field} onChange={(e) => setField(e.target.value)} placeholder="e.g. latency_ms" />
        </div>
        <div className="form-group">
          <label>Operator</label>
          <select value={operator} onChange={(e) => setOperator(e.target.value)}>
            <option value="lt">Less than</option>
            <option value="gt">Greater than</option>
            <option value="eq">Equal to</option>
            <option value="not_null">Not null</option>
          </select>
        </div>
        <div className="form-group">
          <label>Threshold</label>
          <input type="text" value={threshold} onChange={(e) => setThreshold(e.target.value)} placeholder="e.g. 5000" />
        </div>
        <div className="form-group">
          <label>{t('ruleEditor.severity')}</label>
          <select value={severity} onChange={(e) => setSeverity(e.target.value as DQRule['severity'])}>
            <option value="critical">{t('ruleEditor.severityOptions.critical')}</option>
            <option value="warning">{t('ruleEditor.severityOptions.warning')}</option>
            <option value="info">{t('ruleEditor.severityOptions.info')}</option>
          </select>
        </div>
      </div>
      <div className="form-actions">
        <button type="button" className="btn btn-secondary" onClick={onCancel}>{t('common.cancel')}</button>
        <button type="submit" className="btn btn-primary">{t('ruleEditor.saveRule')}</button>
      </div>
    </form>
  );
}

export default RuleEditor;
