import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useGuardrailStore } from '../stores/guardrailStore';
import type { GuardrailRule } from '../types/guardrail';

type RuleType = GuardrailRule['type'];
type RuleAction = GuardrailRule['action'];

export default function GuardrailsPage() {
  const { t } = useTranslation();
  const { rules, violations, stats, isLoading, fetchRules, fetchViolations, fetchStats, createRule } = useGuardrailStore();
  const [showCreate, setShowCreate] = useState(false);
  const [newRule, setNewRule] = useState<{ name: string; type: RuleType; action: RuleAction; pattern: string; enabled: boolean }>({ name: '', type: 'prompt_injection', action: 'block', pattern: '', enabled: true });

  useEffect(() => {
    fetchRules();
    fetchViolations();
    fetchStats();
  }, [fetchRules, fetchViolations, fetchStats]);

  const handleCreate = async () => {
    await createRule(newRule);
    setShowCreate(false);
    setNewRule({ name: '', type: 'prompt_injection', action: 'block', pattern: '', enabled: true });
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1>{t('guardrails.title')}</h1>
          <p className="page-subtitle">{t('guardrails.subtitle')}</p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowCreate(!showCreate)}>
          {t('guardrails.addRule')}
        </button>
      </div>

      {stats && (
        <div className="grid grid-3" style={{ marginBottom: '1.5rem' }}>
          <div className="stat-card">
            <div className="stat-label">{t('guardrails.totalChecks')}</div>
            <div className="stat-value">{stats.total_checks.toLocaleString()}</div>
          </div>
          <div className="stat-card">
            <div className="stat-label">{t('guardrails.violations')}</div>
            <div className="stat-value text-danger">{stats.total_violations}</div>
          </div>
          <div className="stat-card">
            <div className="stat-label">{t('guardrails.passRate')}</div>
            <div className="stat-value text-success">{(stats.pass_rate * 100).toFixed(1)}%</div>
          </div>
        </div>
      )}

      {showCreate && (
        <div className="card" style={{ marginBottom: '1.5rem' }}>
          <div className="card-header"><h3>{t('guardrails.newRule')}</h3></div>
          <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            <input className="input" placeholder={t('guardrails.ruleName')} value={newRule.name} onChange={(e) => setNewRule({ ...newRule, name: e.target.value })} />
            <select className="input" value={newRule.type} onChange={(e) => setNewRule({ ...newRule, type: e.target.value as RuleType })}>
              <option value="prompt_injection">{t('guardrails.typeOptions.promptInjection')}</option>
              <option value="pii_detection">{t('guardrails.typeOptions.piiDetection')}</option>
              <option value="toxicity">{t('guardrails.typeOptions.toxicity')}</option>
              <option value="blocklist">{t('guardrails.typeOptions.blocklist')}</option>
              <option value="custom_regex">{t('guardrails.typeOptions.customRegex')}</option>
            </select>
            <select className="input" value={newRule.action} onChange={(e) => setNewRule({ ...newRule, action: e.target.value as RuleAction })}>
              <option value="block">{t('guardrails.actionOptions.block')}</option>
              <option value="warn">{t('guardrails.actionOptions.warn')}</option>
              <option value="log">{t('guardrails.actionOptions.log')}</option>
            </select>
            <input className="input" placeholder={t('guardrails.pattern')} value={newRule.pattern} onChange={(e) => setNewRule({ ...newRule, pattern: e.target.value })} />
            <button className="btn btn-primary" onClick={handleCreate}>{t('guardrails.createRule')}</button>
          </div>
        </div>
      )}

      <div className="grid grid-2">
        <div className="card">
          <div className="card-header"><h3>{t('guardrails.rules', { count: rules.length })}</h3></div>
          <div className="card-body">
            {isLoading ? (
              <p className="text-muted">{t('guardrails.loading')}</p>
            ) : rules.length === 0 ? (
              <p className="text-muted">{t('guardrails.noRules')}</p>
            ) : (
              <div className="table-container">
                <table>
                  <thead>
                    <tr>
                      <th>{t('common.name')}</th>
                      <th>{t('common.type')}</th>
                      <th>{t('common.actions')}</th>
                      <th>{t('common.status')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rules.map((rule) => (
                      <tr key={rule.id}>
                        <td>{rule.name}</td>
                        <td><code>{rule.type}</code></td>
                        <td><span className={`badge badge-${rule.action === 'block' ? 'danger' : rule.action === 'warn' ? 'warning' : 'info'}`}>{rule.action}</span></td>
                        <td><span className={`badge badge-${rule.enabled ? 'success' : 'default'}`}>{rule.enabled ? t('common.enabled') : t('common.disabled')}</span></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>

        <div className="card">
          <div className="card-header"><h3>{t('guardrails.recentViolations', { count: violations.length })}</h3></div>
          <div className="card-body">
            {violations.length === 0 ? (
              <p className="text-muted">{t('guardrails.noViolations')}</p>
            ) : (
              <div className="table-container">
                <table>
                  <thead>
                    <tr>
                      <th>{t('dataQuality.rule')}</th>
                      <th>{t('dataQuality.agent')}</th>
                      <th>{t('common.actions')}</th>
                      <th>{t('dataQuality.time')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {violations.slice(0, 20).map((v) => (
                      <tr key={v.id}>
                        <td>{v.rule_name}</td>
                        <td><code>{v.agent_id}</code></td>
                        <td><span className={`badge badge-${v.action === 'block' ? 'danger' : 'warning'}`}>{v.action}</span></td>
                        <td className="text-muted">{new Date(v.created_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
