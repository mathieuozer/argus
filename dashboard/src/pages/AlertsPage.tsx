import { useEffect } from 'react';
import { useAlertStore } from '../stores/alertStore';
import AlertRow from '../components/alerts/AlertRow';

function AlertsPage() {
  const { alerts, loading, error, fetchAlerts } = useAlertStore();

  useEffect(() => {
    fetchAlerts();
  }, [fetchAlerts]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>Predictive Alerts</h2>
          <p>ML-powered failure predictions for monitored agents</p>
        </div>
        <div className="page-header-actions">
          <button className="btn" onClick={fetchAlerts} disabled={loading}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="23 4 23 10 17 10" />
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
            </svg>
            Refresh
          </button>
        </div>
      </div>

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
          <span>Loading alerts...</span>
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
          <h3>No active alerts</h3>
          <p>
            Predictive alerts will appear here when the system detects potential agent failures based on telemetry analysis.
          </p>
        </div>
      )}

      {alerts.length > 0 && (
        <div className="table-container animate-fade-in">
          <table className="table">
            <thead>
              <tr>
                <th>Alert ID</th>
                <th>Agent</th>
                <th>Probability</th>
                <th>Precursor</th>
                <th>TTF</th>
                <th>Status</th>
                <th>Created</th>
                <th>Evidence</th>
              </tr>
            </thead>
            <tbody>
              {alerts.map((alert) => (
                <AlertRow key={alert.id} alert={alert} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

export default AlertsPage;
