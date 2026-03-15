import { useEffect } from 'react';
import { useAlertStore } from '../stores/alertStore';

function AlertsPage() {
  const { alerts, loading, error, fetchAlerts } = useAlertStore();

  useEffect(() => {
    fetchAlerts();
  }, [fetchAlerts]);

  return (
    <div>
      <h2 style={{ marginBottom: '16px' }}>Alerts</h2>
      {loading && <p style={{ color: 'var(--color-text-muted)' }}>Loading alerts...</p>}
      {error && <p style={{ color: 'var(--color-danger)' }}>Error: {error}</p>}
      {!loading && !error && alerts.length === 0 && (
        <div
          style={{
            padding: '40px',
            textAlign: 'center',
            color: 'var(--color-text-muted)',
            backgroundColor: 'var(--color-surface)',
            borderRadius: '8px',
            border: '1px solid var(--color-border)',
          }}
        >
          <p>No active alerts.</p>
          <p style={{ fontSize: '14px', marginTop: '8px' }}>
            Predictive alerts will appear here when the system detects potential failures.
          </p>
        </div>
      )}
      {alerts.length > 0 && (
        <div style={{ display: 'grid', gap: '12px' }}>
          {alerts.map((alert) => (
            <div
              key={alert.id}
              style={{
                padding: '16px',
                backgroundColor: 'var(--color-surface)',
                borderRadius: '8px',
                border: '1px solid var(--color-border)',
                borderLeftWidth: '3px',
                borderLeftColor: 'var(--color-warning)',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
                <strong>Agent: {alert.agent_id}</strong>
                <span style={{ color: 'var(--color-warning)' }}>
                  {(alert.probability * 100).toFixed(0)}% failure probability
                </span>
              </div>
              <div style={{ fontSize: '14px', color: 'var(--color-text-muted)' }}>
                Precursor: {alert.precursor_type} | TTF: {alert.estimated_ttf}s
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default AlertsPage;
