import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useDataQualityStore } from '../stores/dataQualityStore';
import QualityScoreCard from '../components/dataquality/QualityScoreCard';
import ViolationList from '../components/dataquality/ViolationList';
import DriftChart from '../components/dataquality/DriftChart';
import RuleEditor from '../components/dataquality/RuleEditor';

function DataQualityPage() {
  const { t } = useTranslation();
  const { scores, rules, violations, drift, loading, error, fetchScores, fetchRules, fetchViolations, fetchDrift, createRule, deleteRule } = useDataQualityStore();
  const [showEditor, setShowEditor] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);

  useEffect(() => {
    fetchScores();
    fetchRules();
    fetchViolations();
  }, [fetchScores, fetchRules, fetchViolations]);

  useEffect(() => {
    if (selectedAgent) {
      fetchDrift(selectedAgent);
    }
  }, [selectedAgent, fetchDrift]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('dataQuality.title')}</h2>
          <p>{t('dataQuality.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <button className="btn btn-primary" onClick={() => setShowEditor(true)}>{t('dataQuality.addRule')}</button>
          <button className="btn" onClick={() => { fetchScores(); fetchRules(); fetchViolations(); }} disabled={loading}>{t('common.refresh')}</button>
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {showEditor && (
        <div className="card" style={{ marginBottom: 'var(--spacing-lg)' }}>
          <RuleEditor
            onSave={(rule) => { createRule(rule); setShowEditor(false); }}
            onCancel={() => setShowEditor(false)}
          />
        </div>
      )}

      {scores.length > 0 && (
        <div style={{ marginBottom: 'var(--spacing-lg)' }}>
          <h3>{t('dataQuality.qualityScores')}</h3>
          <div className="grid grid-auto">
            {scores.map((score) => (
              <div key={score.agent_id} onClick={() => setSelectedAgent(score.agent_id)} style={{ cursor: 'pointer' }}>
                <QualityScoreCard score={score} />
              </div>
            ))}
          </div>
        </div>
      )}

      {selectedAgent && drift.length > 0 && (
        <div style={{ marginBottom: 'var(--spacing-lg)' }}>
          <h3>{t('dataQuality.driftAnalysis', { agent: selectedAgent })}</h3>
          <DriftChart data={drift} />
        </div>
      )}

      {rules.length > 0 && (
        <div style={{ marginBottom: 'var(--spacing-lg)' }}>
          <h3>{t('dataQuality.validationRules', { count: rules.length })}</h3>
          <table className="table">
            <thead>
              <tr>
                <th>{t('dataQuality.agent')}</th>
                <th>{t('dataQuality.ruleName')}</th>
                <th>{t('dataQuality.ruleType')}</th>
                <th>{t('dataQuality.target')}</th>
                <th>{t('dataQuality.severity')}</th>
                <th>{t('dataQuality.ruleStatus')}</th>
                <th>{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={rule.id}>
                  <td className="mono">{rule.agent_id}</td>
                  <td>{rule.name}</td>
                  <td><span className="badge">{rule.type}</span></td>
                  <td>{rule.target}</td>
                  <td><span className={`badge ${rule.severity === 'critical' ? 'badge-error' : rule.severity === 'warning' ? 'badge-warning' : 'badge-info'}`}>{rule.severity}</span></td>
                  <td>{rule.enabled ? <span className="badge badge-success">{t('dataQuality.ruleActive')}</span> : <span className="badge">{t('dataQuality.ruleDisabled')}</span>}</td>
                  <td>
                    <button className="btn btn-sm btn-secondary" onClick={() => deleteRule(rule.id)}>{t('common.delete')}</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div>
        <h3>{t('dataQuality.recentViolations')}</h3>
        <ViolationList violations={violations} />
      </div>

      {loading && scores.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>{t('dataQuality.loadingMetrics')}</span>
        </div>
      )}
    </div>
  );
}

export default DataQualityPage;
