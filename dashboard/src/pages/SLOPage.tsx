import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSLOStore } from '../stores/sloStore';
import SLOCard from '../components/slo/SLOCard';
import ErrorBudgetChart from '../components/slo/ErrorBudgetChart';
import SLOEditor from '../components/slo/SLOEditor';

function SLOPage() {
  const { t } = useTranslation();
  const { slos, errorBudget, loading, error, fetchSLOs, fetchErrorBudget, createSLO } = useSLOStore();
  const [showEditor, setShowEditor] = useState(false);
  const [selectedSLO, setSelectedSLO] = useState<string | null>(null);

  useEffect(() => {
    fetchSLOs();
  }, [fetchSLOs]);

  useEffect(() => {
    if (selectedSLO) {
      fetchErrorBudget(selectedSLO);
    }
  }, [selectedSLO, fetchErrorBudget]);

  const metCount = slos.filter((s) => s.status === 'met').length;
  const atRiskCount = slos.filter((s) => s.status === 'at_risk').length;
  const breachedCount = slos.filter((s) => s.status === 'breached').length;

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('slos.title')}</h2>
          <p>{t('slos.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <button className="btn btn-primary" onClick={() => setShowEditor(true)}>{t('slos.createSlo')}</button>
          <button className="btn" onClick={() => fetchSLOs()} disabled={loading}>{t('common.refresh')}</button>
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {slos.length > 0 && (
        <div className="stat-row">
          <div className="stat-card">
            <span className="stat-label">{t('slos.totalSlos')}</span>
            <span className="stat-value">{slos.length}</span>
          </div>
          <div className="stat-card">
            <span className="stat-label">{t('slos.met')}</span>
            <span className="stat-value" style={{ color: 'var(--color-success)' }}>{metCount}</span>
          </div>
          <div className="stat-card">
            <span className="stat-label">{t('slos.atRisk')}</span>
            <span className="stat-value" style={{ color: 'var(--color-warning)' }}>{atRiskCount}</span>
          </div>
          <div className="stat-card">
            <span className="stat-label">{t('slos.breached')}</span>
            <span className="stat-value" style={{ color: 'var(--color-error)' }}>{breachedCount}</span>
          </div>
        </div>
      )}

      {showEditor && (
        <div className="card" style={{ marginBottom: 'var(--spacing-lg)' }}>
          <SLOEditor
            onSave={(slo) => { createSLO(slo); setShowEditor(false); }}
            onCancel={() => setShowEditor(false)}
          />
        </div>
      )}

      {selectedSLO && errorBudget.length > 0 && (
        <div style={{ marginBottom: 'var(--spacing-lg)' }}>
          <ErrorBudgetChart data={errorBudget} />
        </div>
      )}

      {loading && slos.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>{t('slos.loadingSlos')}</span>
        </div>
      )}

      {slos.length > 0 && (
        <div className="grid grid-auto">
          {slos.map((slo) => (
            <SLOCard
              key={slo.id}
              slo={slo}
              onClick={() => setSelectedSLO(slo.id === selectedSLO ? null : slo.id)}
            />
          ))}
        </div>
      )}

      {!loading && slos.length === 0 && !showEditor && (
        <div className="empty-state">
          <h3>{t('slos.noSlos')}</h3>
          <p>{t('slos.noSlosDescription')}</p>
          <button className="btn btn-primary" onClick={() => setShowEditor(true)}>{t('slos.createFirstSlo')}</button>
        </div>
      )}
    </div>
  );
}

export default SLOPage;
