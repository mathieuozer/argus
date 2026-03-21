import { useEffect, useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useAlertStore } from '../stores/alertStore';
import AlertRow from '../components/alerts/AlertRow';
import SearchFilter from '../components/shared/SearchFilter';
import AutoRefreshToggle from '../components/shared/AutoRefreshToggle';

function AlertsPage() {
  const { t } = useTranslation();
  const { alerts, loading, error, fetchAlerts, updateAlertStatus } = useAlertStore();

  const STATUS_OPTIONS = [
    { label: t('alerts.statusOptions.open'), value: 'open' },
    { label: t('alerts.statusOptions.acknowledged'), value: 'acknowledged' },
    { label: t('alerts.statusOptions.resolved'), value: 'resolved' },
    { label: t('alerts.statusOptions.falsePositive'), value: 'false_positive' },
  ];

  const PRECURSOR_OPTIONS = [
    { label: t('alerts.precursorOptions.latencySpike'), value: 'latency_spike' },
    { label: t('alerts.precursorOptions.tokenEscalation'), value: 'token_escalation' },
    { label: t('alerts.precursorOptions.retryStorm'), value: 'retry_storm' },
    { label: t('alerts.precursorOptions.costRunaway'), value: 'cost_runaway' },
  ];
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [precursorFilter, setPrecursorFilter] = useState('');

  useEffect(() => {
    fetchAlerts();
  }, [fetchAlerts]);

  const filteredAlerts = useMemo(() => {
    return alerts.filter((alert) => {
      const matchesSearch = !search ||
        alert.id.toLowerCase().includes(search.toLowerCase()) ||
        alert.agent_id.toLowerCase().includes(search.toLowerCase());
      const matchesStatus = !statusFilter || alert.status === statusFilter;
      const matchesPrecursor = !precursorFilter || alert.precursor_type === precursorFilter;
      return matchesSearch && matchesStatus && matchesPrecursor;
    });
  }, [alerts, search, statusFilter, precursorFilter]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('alerts.title')}</h2>
          <p>{t('alerts.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <AutoRefreshToggle onRefresh={fetchAlerts} />
          <button className="btn" onClick={fetchAlerts} disabled={loading}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="23 4 23 10 17 10" />
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
            </svg>
            {t('common.refresh')}
          </button>
        </div>
      </div>

      {alerts.length > 0 && (
        <SearchFilter
          searchValue={search}
          onSearchChange={setSearch}
          searchPlaceholder={t('alerts.searchPlaceholder')}
          filters={[
            { label: t('alerts.allStatuses'), value: statusFilter, options: STATUS_OPTIONS, onChange: setStatusFilter },
            { label: t('alerts.allPrecursors'), value: precursorFilter, options: PRECURSOR_OPTIONS, onChange: setPrecursorFilter },
          ]}
        />
      )}

      {error && (
        <div className="error-banner">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
          {error}
        </div>
      )}

      {loading && alerts.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>{t('alerts.loadingAlerts')}</span>
        </div>
      )}

      {!loading && !error && alerts.length === 0 && (
        <div className="empty-state">
          <div className="empty-state-icon">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
              <path d="M13.73 21a2 2 0 0 1-3.46 0" />
            </svg>
          </div>
          <h3>{t('alerts.noAlerts')}</h3>
          <p>
            {t('alerts.noAlertsDescription')}
          </p>
        </div>
      )}

      {filteredAlerts.length > 0 && (
        <div className="table-container animate-fade-in">
          <table className="table">
            <thead>
              <tr>
                <th>{t('alerts.alertId')}</th>
                <th>{t('alerts.agent')}</th>
                <th>{t('alerts.probability')}</th>
                <th>{t('alerts.precursor')}</th>
                <th>{t('alerts.ttf')}</th>
                <th>{t('alerts.status')}</th>
                <th>{t('alerts.created')}</th>
                <th>{t('alerts.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {filteredAlerts.map((alert) => (
                <AlertRow key={alert.id} alert={alert} onUpdateStatus={updateAlertStatus} />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {alerts.length > 0 && filteredAlerts.length === 0 && (
        <div className="empty-state">
          <h3>{t('alerts.noMatching')}</h3>
          <p>{t('alerts.noMatchingDescription')}</p>
        </div>
      )}
    </div>
  );
}

export default AlertsPage;
