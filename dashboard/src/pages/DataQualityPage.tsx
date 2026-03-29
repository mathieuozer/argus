import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useDataQualityStore } from '../stores/dataQualityStore';
import QualityScoreCard from '../components/dataquality/QualityScoreCard';
import ViolationList from '../components/dataquality/ViolationList';
import DriftChart from '../components/dataquality/DriftChart';
import RuleEditor from '../components/dataquality/RuleEditor';

type Tab = 'overview' | 'scores' | 'contracts' | 'incidents' | 'anomalies' | 'rules' | 'violations';

function DataQualityPage() {
  const { t } = useTranslation();
  const {
    scores, rules, violations, drift, contracts, incidents, anomalies, summary,
    loading, error,
    fetchScores, fetchRules, fetchViolations, fetchDrift,
    fetchContracts, fetchIncidents, fetchAnomalies, fetchSummary,
    createRule, deleteRule, deleteContract,
  } = useDataQualityStore();
  const [showEditor, setShowEditor] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('overview');

  useEffect(() => {
    fetchSummary();
    fetchScores();
    fetchRules();
    fetchViolations();
    fetchContracts();
    fetchIncidents();
    fetchAnomalies();
  }, [fetchSummary, fetchScores, fetchRules, fetchViolations, fetchContracts, fetchIncidents, fetchAnomalies]);

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
          <button className="btn" onClick={() => { fetchSummary(); fetchScores(); fetchRules(); fetchViolations(); fetchContracts(); fetchIncidents(); fetchAnomalies(); }} disabled={loading}>{t('common.refresh')}</button>
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

      {/* Summary cards */}
      {summary && (
        <div className="grid grid-auto" style={{ marginBottom: 'var(--spacing-lg)' }}>
          <div className="card">
            <div className="stat-label">{t('dataQuality.avgScore')}</div>
            <div className="stat-value">{summary.avg_quality_score.toFixed(1)}%</div>
          </div>
          <div className="card">
            <div className="stat-label">{t('dataQuality.activeRules')}</div>
            <div className="stat-value">{summary.active_rules}/{summary.total_rules}</div>
          </div>
          <div className="card">
            <div className="stat-label">{t('dataQuality.violations')}</div>
            <div className="stat-value">{summary.total_violations}</div>
            <div className="stat-sub">{summary.critical_violations} critical</div>
          </div>
          <div className="card">
            <div className="stat-label">{t('dataQuality.contracts')}</div>
            <div className="stat-value">{summary.active_contracts}/{summary.total_contracts}</div>
            <div className="stat-sub">{summary.violated_contracts} violated</div>
          </div>
          <div className="card">
            <div className="stat-label">{t('dataQuality.incidents')}</div>
            <div className="stat-value">{summary.open_incidents}/{summary.total_incidents}</div>
            <div className="stat-sub">{t('dataQuality.openIncidents')}</div>
          </div>
          <div className="card">
            <div className="stat-label">{t('dataQuality.anomalies')}</div>
            <div className="stat-value">{summary.total_anomalies}</div>
          </div>
        </div>
      )}

      <div className="tab-bar">
        {(['overview', 'contracts', 'incidents', 'anomalies', 'rules', 'violations'] as Tab[]).map((tab) => (
          <button key={tab} className={`tab ${activeTab === tab ? 'active' : ''}`} onClick={() => setActiveTab(tab)}>
            {t(`dataQuality.tabs.${tab}`)}
          </button>
        ))}
      </div>

      {activeTab === 'overview' && (
        <div>
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
        </div>
      )}

      {activeTab === 'contracts' && (
        <div>
          <h3>{t('dataQuality.dataContracts')} ({contracts.length})</h3>
          {contracts.length === 0 ? (
            <p style={{ opacity: 0.7 }}>{t('dataQuality.noContracts')}</p>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>{t('common.name')}</th>
                  <th>{t('dataQuality.producer')}</th>
                  <th>{t('dataQuality.consumers')}</th>
                  <th>{t('dataQuality.freshness')}</th>
                  <th>{t('dataQuality.qualityReq')}</th>
                  <th>{t('common.status')}</th>
                  <th>{t('common.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {contracts.map((c) => (
                  <tr key={c.id}>
                    <td><strong>{c.name}</strong><br /><span style={{ opacity: 0.7, fontSize: '0.85em' }}>{c.description}</span></td>
                    <td className="mono">{c.producer_agent}</td>
                    <td>{c.consumer_agents?.join(', ') || '-'}</td>
                    <td>{c.freshness_spec ? `${c.freshness_spec.max_staleness_seconds}s max` : '-'}</td>
                    <td>{c.quality_spec ? `${c.quality_spec.min_completeness}% complete` : '-'}</td>
                    <td><span className={`badge ${c.status === 'active' ? 'badge-success' : c.status === 'violated' ? 'badge-error' : 'badge-warning'}`}>{c.status}</span></td>
                    <td><button className="btn btn-sm btn-secondary" onClick={() => deleteContract(c.id)}>{t('common.delete')}</button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {activeTab === 'incidents' && (
        <div>
          <h3>{t('dataQuality.qualityIncidents')} ({incidents.length})</h3>
          {incidents.length === 0 ? (
            <p style={{ opacity: 0.7 }}>{t('dataQuality.noIncidents')}</p>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>{t('dataQuality.incident')}</th>
                  <th>{t('dataQuality.agent')}</th>
                  <th>{t('dataQuality.severity')}</th>
                  <th>{t('common.status')}</th>
                  <th>{t('dataQuality.created')}</th>
                </tr>
              </thead>
              <tbody>
                {incidents.map((inc) => (
                  <tr key={inc.id}>
                    <td><strong>{inc.title}</strong><br /><span style={{ opacity: 0.7, fontSize: '0.85em' }}>{inc.description}</span></td>
                    <td className="mono">{inc.agent_id}</td>
                    <td><span className={`badge ${inc.severity === 'critical' ? 'badge-error' : inc.severity === 'warning' ? 'badge-warning' : 'badge-info'}`}>{inc.severity}</span></td>
                    <td><span className={`badge ${inc.status === 'open' ? 'badge-error' : inc.status === 'resolved' ? 'badge-success' : ''}`}>{inc.status}</span></td>
                    <td>{new Date(inc.created_at).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {activeTab === 'anomalies' && (
        <div>
          <h3>{t('dataQuality.detectedAnomalies')} ({anomalies.length})</h3>
          {anomalies.length === 0 ? (
            <p style={{ opacity: 0.7 }}>{t('dataQuality.noAnomalies')}</p>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>{t('dataQuality.agent')}</th>
                  <th>{t('dataQuality.metric')}</th>
                  <th>{t('dataQuality.expected')}</th>
                  <th>{t('dataQuality.actual')}</th>
                  <th>{t('dataQuality.deviation')}</th>
                  <th>{t('dataQuality.severity')}</th>
                  <th>{t('dataQuality.detected')}</th>
                </tr>
              </thead>
              <tbody>
                {anomalies.map((a) => (
                  <tr key={a.id}>
                    <td className="mono">{a.agent_id}</td>
                    <td>{a.metric_name}</td>
                    <td>{a.expected.toFixed(2)}</td>
                    <td>{a.actual.toFixed(2)}</td>
                    <td><span className={`badge ${a.deviation > 50 ? 'badge-error' : a.deviation > 20 ? 'badge-warning' : 'badge-info'}`}>{a.deviation.toFixed(1)}%</span></td>
                    <td><span className={`badge ${a.severity === 'critical' ? 'badge-error' : a.severity === 'warning' ? 'badge-warning' : 'badge-info'}`}>{a.severity}</span></td>
                    <td>{new Date(a.detected_at).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {activeTab === 'rules' && (
        <div>
          {rules.length > 0 && (
            <div style={{ marginBottom: 'var(--spacing-lg)' }}>
              <h3>{t('dataQuality.validationRules', { count: rules.length })}</h3>
              <table className="table">
                <thead>
                  <tr>
                    <th>{t('dataQuality.agent')}</th>
                    <th>{t('dataQuality.ruleName')}</th>
                    <th>{t('dataQuality.ruleType')}</th>
                    <th>{t('dataQuality.field')}</th>
                    <th>{t('dataQuality.severity')}</th>
                    <th>{t('dataQuality.ruleStatus')}</th>
                    <th>{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {rules.map((rule) => (
                    <tr key={rule.id}>
                      <td className="mono">{rule.agent_id || t('common.all')}</td>
                      <td>{rule.name}</td>
                      <td><span className="badge">{rule.type}</span></td>
                      <td className="mono">{rule.field}</td>
                      <td><span className={`badge ${rule.severity === 'critical' ? 'badge-error' : rule.severity === 'warning' ? 'badge-warning' : 'badge-info'}`}>{rule.severity}</span></td>
                      <td>{rule.enabled ? <span className="badge badge-success">{t('dataQuality.ruleActive')}</span> : <span className="badge">{t('dataQuality.ruleDisabled')}</span>}</td>
                      <td><button className="btn btn-sm btn-secondary" onClick={() => deleteRule(rule.id)}>{t('common.delete')}</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {activeTab === 'violations' && (
        <div>
          <h3>{t('dataQuality.recentViolations')}</h3>
          <ViolationList violations={violations} />
        </div>
      )}

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
