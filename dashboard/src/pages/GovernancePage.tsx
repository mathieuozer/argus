import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useGovernanceStore } from '../stores/governanceStore';

type Tab = 'overview' | 'classification' | 'retention' | 'pii' | 'compliance' | 'stewards' | 'access';

function GovernancePage() {
  const { t } = useTranslation();
  const {
    summary, classificationPolicies, retentionPolicies, accessLogs,
    piiScans, complianceMappings, stewards, loading, error,
    fetchSummary, fetchClassificationPolicies, fetchRetentionPolicies,
    fetchAccessLogs, fetchPIIScans, fetchComplianceMappings, fetchStewards,
    deleteClassificationPolicy, deleteRetentionPolicy, deleteSteward,
  } = useGovernanceStore();
  const [activeTab, setActiveTab] = useState<Tab>('overview');

  useEffect(() => {
    fetchSummary();
    fetchClassificationPolicies();
    fetchRetentionPolicies();
    fetchAccessLogs();
    fetchPIIScans();
    fetchComplianceMappings();
    fetchStewards();
  }, [fetchSummary, fetchClassificationPolicies, fetchRetentionPolicies, fetchAccessLogs, fetchPIIScans, fetchComplianceMappings, fetchStewards]);

  const refreshAll = () => {
    fetchSummary();
    fetchClassificationPolicies();
    fetchRetentionPolicies();
    fetchAccessLogs();
    fetchPIIScans();
    fetchComplianceMappings();
    fetchStewards();
  };

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('governance.title')}</h2>
          <p>{t('governance.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <button className="btn" onClick={refreshAll} disabled={loading}>{t('common.refresh')}</button>
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}

      <div className="tab-bar">
        {(['overview', 'classification', 'retention', 'pii', 'compliance', 'stewards', 'access'] as Tab[]).map((tab) => (
          <button key={tab} className={`tab ${activeTab === tab ? 'active' : ''}`} onClick={() => setActiveTab(tab)}>
            {t(`governance.tabs.${tab}`)}
          </button>
        ))}
      </div>

      {activeTab === 'overview' && summary && (
        <div>
          <div className="grid grid-auto" style={{ marginBottom: 'var(--spacing-lg)' }}>
            <div className="card">
              <div className="stat-label">{t('governance.classificationPolicies')}</div>
              <div className="stat-value">{summary.active_policies}/{summary.total_policies}</div>
              <div className="stat-sub">{t('governance.activeOfTotal')}</div>
            </div>
            <div className="card">
              <div className="stat-label">{t('governance.retentionPolicies')}</div>
              <div className="stat-value">{summary.retention_policies}</div>
            </div>
            <div className="card">
              <div className="stat-label">{t('governance.piiScans')}</div>
              <div className="stat-value">{summary.pii_scan_results}</div>
              <div className="stat-sub">{summary.high_risk_sources} {t('governance.highRisk')}</div>
            </div>
            <div className="card">
              <div className="stat-label">{t('governance.complianceStatus')}</div>
              <div className="stat-value">{summary.compliant_count}/{summary.compliance_mappings}</div>
              <div className="stat-sub">
                <span className="badge badge-success">{summary.compliant_count} {t('governance.compliant')}</span>{' '}
                <span className="badge badge-error">{summary.non_compliant_count} {t('governance.nonCompliant')}</span>
              </div>
            </div>
            <div className="card">
              <div className="stat-label">{t('governance.dataStewards')}</div>
              <div className="stat-value">{summary.data_stewards}</div>
              <div className="stat-sub">{summary.unowned_sources} {t('governance.unowned')}</div>
            </div>
            <div className="card">
              <div className="stat-label">{t('governance.accessEvents')}</div>
              <div className="stat-value">{summary.recent_access_logs}</div>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'classification' && (
        <div>
          <h3>{t('governance.classificationPolicies')} ({classificationPolicies.length})</h3>
          <table className="table">
            <thead>
              <tr>
                <th>{t('common.name')}</th>
                <th>{t('governance.matchPattern')}</th>
                <th>{t('governance.matchType')}</th>
                <th>{t('governance.classification')}</th>
                <th>{t('governance.autoApply')}</th>
                <th>{t('governance.appliedCount')}</th>
                <th>{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {classificationPolicies.map((p) => (
                <tr key={p.id}>
                  <td><strong>{p.name}</strong><br /><span style={{ opacity: 0.7, fontSize: '0.85em' }}>{p.description}</span></td>
                  <td className="mono" style={{ fontSize: '0.85em' }}>{p.match_pattern}</td>
                  <td><span className="badge">{p.match_type}</span></td>
                  <td><span className={`badge ${p.classification === 'restricted' ? 'badge-error' : p.classification === 'confidential' ? 'badge-warning' : 'badge-info'}`}>{p.classification}</span></td>
                  <td>{p.auto_apply ? <span className="badge badge-success">{t('common.active')}</span> : <span className="badge">{t('common.disabled')}</span>}</td>
                  <td>{p.applied_count}</td>
                  <td><button className="btn btn-sm btn-secondary" onClick={() => deleteClassificationPolicy(p.id)}>{t('common.delete')}</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'retention' && (
        <div>
          <h3>{t('governance.retentionPolicies')} ({retentionPolicies.length})</h3>
          <table className="table">
            <thead>
              <tr>
                <th>{t('common.name')}</th>
                <th>{t('governance.classification')}</th>
                <th>{t('governance.retentionDays')}</th>
                <th>{t('governance.action')}</th>
                <th>{t('common.status')}</th>
                <th>{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {retentionPolicies.map((p) => (
                <tr key={p.id}>
                  <td><strong>{p.name}</strong><br /><span style={{ opacity: 0.7, fontSize: '0.85em' }}>{p.description}</span></td>
                  <td><span className="badge">{p.classification}</span></td>
                  <td>{p.retention_days} days</td>
                  <td><span className="badge">{p.action}</span></td>
                  <td>{p.enabled ? <span className="badge badge-success">{t('common.active')}</span> : <span className="badge">{t('common.disabled')}</span>}</td>
                  <td><button className="btn btn-sm btn-secondary" onClick={() => deleteRetentionPolicy(p.id)}>{t('common.delete')}</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'pii' && (
        <div>
          <h3>{t('governance.piiScanResults')} ({piiScans.length})</h3>
          {piiScans.map((scan) => (
            <div key={scan.id} className="card" style={{ marginBottom: 'var(--spacing-md)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 'var(--spacing-sm)' }}>
                <div>
                  <strong>{scan.source_name}</strong>
                  <span className="mono" style={{ marginLeft: 'var(--spacing-sm)', opacity: 0.7 }}>{scan.source_id}</span>
                </div>
                <div>
                  <span className={`badge ${scan.risk_score > 0.7 ? 'badge-error' : scan.risk_score > 0.3 ? 'badge-warning' : 'badge-info'}`}>
                    Risk: {(scan.risk_score * 100).toFixed(0)}%
                  </span>
                  <span className="badge" style={{ marginLeft: 'var(--spacing-xs)' }}>
                    {scan.pii_field_count}/{scan.total_fields} PII fields
                  </span>
                </div>
              </div>
              {scan.pii_fields && scan.pii_fields.length > 0 && (
                <table className="table">
                  <thead>
                    <tr>
                      <th>{t('governance.fieldName')}</th>
                      <th>{t('governance.piiType')}</th>
                      <th>{t('governance.confidence')}</th>
                      <th>{t('governance.samples')}</th>
                      <th>{t('governance.recommendation')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {scan.pii_fields.map((f, i) => (
                      <tr key={i}>
                        <td className="mono">{f.field_name}</td>
                        <td><span className="badge badge-warning">{f.pii_type}</span></td>
                        <td>{(f.confidence * 100).toFixed(0)}%</td>
                        <td>{f.sample_count.toLocaleString()}</td>
                        <td style={{ fontSize: '0.85em' }}>{f.recommendation}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          ))}
        </div>
      )}

      {activeTab === 'compliance' && (
        <div>
          <h3>{t('governance.complianceMappings')} ({complianceMappings.length})</h3>
          <table className="table">
            <thead>
              <tr>
                <th>{t('governance.source')}</th>
                <th>{t('governance.framework')}</th>
                <th>{t('governance.articleRef')}</th>
                <th>{t('governance.requirement')}</th>
                <th>{t('common.status')}</th>
                <th>{t('governance.evidence')}</th>
              </tr>
            </thead>
            <tbody>
              {complianceMappings.map((cm) => (
                <tr key={cm.id}>
                  <td>{cm.source_name}</td>
                  <td><span className="badge">{cm.framework}</span></td>
                  <td className="mono">{cm.article_ref}</td>
                  <td style={{ fontSize: '0.85em' }}>{cm.requirement}</td>
                  <td>
                    <span className={`badge ${cm.status === 'compliant' ? 'badge-success' : cm.status === 'partial' ? 'badge-warning' : cm.status === 'non_compliant' ? 'badge-error' : ''}`}>
                      {cm.status}
                    </span>
                  </td>
                  <td style={{ fontSize: '0.85em' }}>{cm.evidence && cm.evidence.length > 0 ? cm.evidence.join(', ') : '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'stewards' && (
        <div>
          <h3>{t('governance.dataStewards')} ({stewards.length})</h3>
          <table className="table">
            <thead>
              <tr>
                <th>{t('common.name')}</th>
                <th>{t('governance.email')}</th>
                <th>{t('governance.role')}</th>
                <th>{t('governance.domains')}</th>
                <th>{t('governance.assignedSources')}</th>
                <th>{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {stewards.map((s) => (
                <tr key={s.id}>
                  <td><strong>{s.name}</strong></td>
                  <td>{s.email}</td>
                  <td><span className={`badge ${s.role === 'lead_steward' ? 'badge-info' : ''}`}>{s.role.replace('_', ' ')}</span></td>
                  <td>{s.domains?.join(', ') || '-'}</td>
                  <td>{s.source_ids?.length || 0} sources</td>
                  <td><button className="btn btn-sm btn-secondary" onClick={() => deleteSteward(s.id)}>{t('common.delete')}</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'access' && (
        <div>
          <h3>{t('governance.accessLogs')} ({accessLogs.length})</h3>
          <table className="table">
            <thead>
              <tr>
                <th>{t('governance.source')}</th>
                <th>{t('governance.agent')}</th>
                <th>{t('governance.accessType')}</th>
                <th>{t('governance.user')}</th>
                <th>{t('governance.time')}</th>
              </tr>
            </thead>
            <tbody>
              {accessLogs.map((log) => (
                <tr key={log.id}>
                  <td>{log.source_name}</td>
                  <td className="mono">{log.agent_id || '-'}</td>
                  <td><span className="badge">{log.access_type}</span></td>
                  <td>{log.user_id || '-'}</td>
                  <td>{new Date(log.timestamp).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {loading && !summary && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>{t('common.loading')}</span>
        </div>
      )}
    </div>
  );
}

export default GovernancePage;
