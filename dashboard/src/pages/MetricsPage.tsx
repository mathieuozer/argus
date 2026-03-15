import { useEffect } from 'react';
import { useMetricsStore } from '../stores/metricsStore';

function MetricsPage() {
  const { metrics, loading, error, fetchMetrics } = useMetricsStore();

  useEffect(() => {
    fetchMetrics();
  }, [fetchMetrics]);

  return (
    <div>
      <h2 style={{ marginBottom: '16px' }}>Metrics</h2>
      {loading && <p style={{ color: 'var(--color-text-muted)' }}>Loading metrics...</p>}
      {error && <p style={{ color: 'var(--color-danger)' }}>Error: {error}</p>}
      {metrics && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
          {[
            { label: 'Total Agents', value: metrics.total_agents },
            { label: 'Active Tasks', value: metrics.active_tasks },
            { label: 'Total Cost', value: `$${metrics.total_cost.toFixed(2)}` },
            { label: 'Active Alerts', value: metrics.alert_count },
          ].map((card) => (
            <div
              key={card.label}
              style={{
                padding: '20px',
                backgroundColor: 'var(--color-surface)',
                borderRadius: '8px',
                border: '1px solid var(--color-border)',
              }}
            >
              <div style={{ fontSize: '14px', color: 'var(--color-text-muted)', marginBottom: '4px' }}>
                {card.label}
              </div>
              <div style={{ fontSize: '28px', fontWeight: 700 }}>{card.value}</div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default MetricsPage;
