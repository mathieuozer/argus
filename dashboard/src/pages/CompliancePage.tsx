import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useComplianceStore } from '../stores/complianceStore';

export default function CompliancePage() {
  const { t } = useTranslation();
  const { reports, selectedReport, isLoading, fetchReports, generateReport, fetchReport } = useComplianceStore();
  const [profileId, setProfileId] = useState('gcc-sa');

  useEffect(() => { fetchReports(); }, [fetchReports]);

  const handleGenerate = () => {
    const now = new Date();
    const monthAgo = new Date(now.getFullYear(), now.getMonth() - 1, now.getDate());
    generateReport(profileId, monthAgo.toISOString(), now.toISOString());
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1>{t('complianceReports.title')}</h1>
          <p className="page-subtitle">{t('complianceReports.subtitle')}</p>
        </div>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <select className="input" value={profileId} onChange={(e) => setProfileId(e.target.value)}>
            <option value="gcc-sa">{t('complianceReports.profileOptions.gccSa')}</option>
            <option value="gcc-ae">{t('complianceReports.profileOptions.gccAe')}</option>
            <option value="gcc-qa">{t('complianceReports.profileOptions.gccQa')}</option>
            <option value="gov-tr">{t('complianceReports.profileOptions.govTr')}</option>
            <option value="eu-gdpr">{t('complianceReports.profileOptions.euGdpr')}</option>
            <option value="fedramp-moderate">{t('complianceReports.profileOptions.fedRamp')}</option>
          </select>
          <button className="btn btn-primary" onClick={handleGenerate} disabled={isLoading}>
            {isLoading ? t('complianceReports.generating') : t('complianceReports.generateReport')}
          </button>
        </div>
      </div>

      <div className="grid grid-2">
        <div className="card">
          <div className="card-header"><h3>{t('complianceReports.reports', { count: reports.length })}</h3></div>
          <div className="card-body">
            {reports.length === 0 ? <p className="text-muted">{t('complianceReports.noReports')}</p> : (
              <div className="table-container">
                <table>
                  <thead><tr><th>{t('complianceReports.title_col')}</th><th>{t('complianceReports.profile')}</th><th>{t('common.status')}</th><th>{t('complianceReports.generated')}</th><th></th></tr></thead>
                  <tbody>
                    {reports.map((r) => (
                      <tr key={r.id}>
                        <td>{r.title}</td>
                        <td><code>{r.profile_id}</code></td>
                        <td><span className={`badge badge-${r.status === 'completed' ? 'success' : 'warning'}`}>{r.status}</span></td>
                        <td className="text-muted">{new Date(r.generated_at).toLocaleDateString()}</td>
                        <td><button className="btn btn-sm" onClick={() => fetchReport(r.id)}>{t('complianceReports.view')}</button></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>

        <div className="card">
          <div className="card-header"><h3>{t('complianceReports.reportDetails')}</h3></div>
          <div className="card-body">
            {!selectedReport ? <p className="text-muted">{t('complianceReports.selectReport')}</p> : (
              <div>
                <h4>{selectedReport.title}</h4>
                <p className="text-muted" style={{ marginBottom: '1rem' }}>
                  {t('complianceReports.period')} {new Date(selectedReport.period_start).toLocaleDateString()} - {new Date(selectedReport.period_end).toLocaleDateString()}
                </p>
                {selectedReport.sections.map((section, i) => (
                  <div key={i} style={{ marginBottom: '1rem', padding: '0.75rem', background: 'var(--color-bg)', borderRadius: '6px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
                      <strong>{section.title}</strong>
                      <span className={`badge badge-${section.status === 'compliant' ? 'success' : section.status === 'non_compliant' ? 'danger' : 'warning'}`}>
                        {section.status}
                      </span>
                    </div>
                    <p className="text-muted" style={{ fontSize: '0.8125rem' }}>{section.description}</p>
                    {section.findings.length > 0 && (
                      <ul style={{ margin: '0.5rem 0 0 1rem', fontSize: '0.8125rem' }}>
                        {section.findings.map((f, j) => <li key={j}>{f}</li>)}
                      </ul>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
