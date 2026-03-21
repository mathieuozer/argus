import { useState } from 'react';
import type { DQRule } from '../../types/dataquality';

interface RuleEditorProps {
  onSave: (rule: Omit<DQRule, 'id' | 'created_at' | 'updated_at'>) => void;
  onCancel: () => void;
}

function RuleEditor({ onSave, onCancel }: RuleEditorProps) {
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
        <label>Agent ID</label>
        <input type="text" value={agentId} onChange={(e) => setAgentId(e.target.value)} required />
      </div>
      <div className="form-group">
        <label>Rule Name</label>
        <input type="text" value={name} onChange={(e) => setName(e.target.value)} required />
      </div>
      <div className="form-row">
        <div className="form-group">
          <label>Type</label>
          <select value={ruleType} onChange={(e) => setRuleType(e.target.value as DQRule['type'])}>
            <option value="schema">Schema</option>
            <option value="range">Range</option>
            <option value="regex">Regex</option>
          </select>
        </div>
        <div className="form-group">
          <label>Target</label>
          <select value={target} onChange={(e) => setTarget(e.target.value as DQRule['target'])}>
            <option value="output">Output</option>
            <option value="input">Input</option>
            <option value="attribute">Attribute</option>
          </select>
        </div>
        <div className="form-group">
          <label>Severity</label>
          <select value={severity} onChange={(e) => setSeverity(e.target.value as DQRule['severity'])}>
            <option value="critical">Critical</option>
            <option value="warning">Warning</option>
            <option value="info">Info</option>
          </select>
        </div>
      </div>
      <div className="form-group">
        <label>Schema / Config (JSON)</label>
        <textarea
          className="code-textarea"
          rows={6}
          value={schemaText}
          onChange={(e) => setSchemaText(e.target.value)}
        />
      </div>
      <div className="form-actions">
        <button type="button" className="btn btn-secondary" onClick={onCancel}>Cancel</button>
        <button type="submit" className="btn btn-primary">Save Rule</button>
      </div>
    </form>
  );
}

export default RuleEditor;
