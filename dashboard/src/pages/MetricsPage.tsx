import { useEffect } from 'react';
import { useMetricsStore } from '../stores/metricsStore';
import StatCard from '../components/metrics/StatCard';

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount);
}

function MetricsPage() {
  const { metrics, loading, error, fetchMetrics } = useMetricsStore();

  useEffect(() => {
    fetchMetrics();
  }, [fetchMetrics]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>Metrics</h2>
          <p>Platform-wide statistics and resource usage</p>
        </div>
        <div className="page-header-actions">
          <button className="btn" onClick={fetchMetrics} disabled={loading}>
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

      {loading && !metrics && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>Loading metrics...</span>
        </div>
      )}

      {metrics && (
        <div className="grid grid-stats">
          <StatCard
            label="Total Agents"
            value={metrics.total_agents}
            icon={
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                <path d="M16 3.13a4 4 0 0 1 0 7.75" />
              </svg>
            }
            iconBgColor="var(--color-primary-muted)"
            iconColor="var(--color-primary)"
          />
          <StatCard
            label="Active Tasks"
            value={metrics.active_tasks}
            icon={
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
              </svg>
            }
            iconBgColor="var(--color-success-muted)"
            iconColor="var(--color-success)"
          />
          <StatCard
            label="Total Cost"
            value={formatCurrency(metrics.total_cost)}
            icon={
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="12" y1="1" x2="12" y2="23" />
                <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
              </svg>
            }
            iconBgColor="var(--color-warning-muted)"
            iconColor="var(--color-warning)"
          />
          <StatCard
            label="Active Alerts"
            value={metrics.alert_count}
            icon={
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
                <path d="M13.73 21a2 2 0 0 1-3.46 0" />
              </svg>
            }
            iconBgColor={metrics.alert_count > 0 ? 'var(--color-danger-muted)' : 'rgba(139, 141, 152, 0.15)'}
            iconColor={metrics.alert_count > 0 ? 'var(--color-danger)' : 'var(--color-text-muted)'}
          />
        </div>
      )}

      {!loading && !error && !metrics && (
        <div className="empty-state">
          <div className="empty-state-icon">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="20" x2="18" y2="10" />
              <line x1="12" y1="20" x2="12" y2="4" />
              <line x1="6" y1="20" x2="6" y2="14" />
            </svg>
          </div>
          <h3>No metrics available</h3>
          <p>
            Metrics will populate once agents begin reporting telemetry data.
          </p>
        </div>
      )}
    </div>
  );
}

export default MetricsPage;
