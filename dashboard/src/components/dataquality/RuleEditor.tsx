import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { DQRule } from '../../types/dataquality';

interface RuleEditorProps {
  onSave: (rule: Omit<DQRule, 'id' | 'created_at' | 'updated_at'>) => void;
  onCancel: () => void;
}

function RuleEditor({ onSave, onCancel }: RuleEditorProps) {
  const { t } = useTranslation();
  const [agentId, setAgentId] = useState('');
  const [name, setName] = useState('');
  const [ruleType, setRuleType] = useState<DQRule['type']>('schema');
  const [target, setTarget] = useState<DQRule['target']>('output');
  const [severity, setSeverity] = useState<DQRule['severity']>('warning');
  const [schemaText, setSchemaText] = useState('{\n  "required": ["amount", "currency"]\n}');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const schema = JSON.parse(schemaText);
      onSave({
        agent_id: agentId,
        name,
        type: ruleType,
        target,
        schema,
        severity,
        enabled: true,
      });
    } catch {
      // Invalid JSON
    }
  };

  return (
    <form className="rule-editor" onSubmit={handleSubmit}>
      <div className="form-group">
        <label>{t('ruleEditor.agentId')}</label>
        <input type="text" value={agentId} onChange={(e) => setAgentId(e.target.value)} required />
      </div>
      <div className="form-group">
        <label>{t('ruleEditor.ruleName')}</label>
        <input type="text" value={name} onChange={(e) => setName(e.target.value)} required />
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>{t('ruleEditor.type')}</label>
          <select value={ruleType} onChange={(e) => setRuleType(e.target.value as DQRule['type'])}>
            <option value="schema">{t('ruleEditor.typeOptions.schema')}</option>
            <option value="range">{t('ruleEditor.typeOptions.range')}</option>
            <option value="regex">{t('ruleEditor.typeOptions.regex')}</option>
          </select>
        </div>
        <div className="form-group">
          <label>{t('ruleEditor.target')}</label>
          <select value={target} onChange={(e) => setTarget(e.target.value as DQRule['target'])}>
            <option value="output">{t('ruleEditor.targetOptions.output')}</option>
            <option value="input">{t('ruleEditor.targetOptions.input')}</option>
            <option value="attribute">{t('ruleEditor.targetOptions.attribute')}</option>
          </select>
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
      <div className="form-group">
        <label>{t('ruleEditor.schemaConfig')}</label>
        <textarea
          className="code-textarea"
          rows={6}
          value={schemaText}
          onChange={(e) => setSchemaText(e.target.value)}
        />
      </div>
      <div className="form-actions">
        <button type="button" className="btn btn-secondary" onClick={onCancel}>{t('common.cancel')}</button>
        <button type="submit" className="btn btn-primary">{t('ruleEditor.saveRule')}</button>
      </div>
    </form>
  );
}

export default RuleEditor;
