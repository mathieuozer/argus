import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMetricsStore } from '../stores/metricsStore';
import type { PlatformTimeSeries } from '../types/telemetry';
import apiClient from '../utils/apiClient';
import StatCard from '../components/metrics/StatCard';
import TimeSeriesChart from '../components/metrics/TimeSeriesChart';
import AutoRefreshToggle from '../components/shared/AutoRefreshToggle';

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount);
}

function MetricsPage() {
  const { t } = useTranslation();
  const { metrics, loading, error, fetchMetrics } = useMetricsStore();
  const [timeSeries, setTimeSeries] = useState<PlatformTimeSeries | null>(null);

  useEffect(() => {
    fetchMetrics();
    apiClient.get<PlatformTimeSeries>('/metrics/timeseries').then((res) => {
      setTimeSeries(res.data);
    });
  }, [fetchMetrics]);

  const handleRefresh = () => {
    fetchMetrics();
    apiClient.get<PlatformTimeSeries>('/metrics/timeseries').then((res) => {
      setTimeSeries(res.data);
    });
  };

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('metrics.title')}</h2>
          <p>{t('metrics.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <AutoRefreshToggle onRefresh={handleRefresh} />
          <button className="btn" onClick={handleRefresh} disabled={loading}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="23 4 23 10 17 10" />
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
            </svg>
            {t('common.refresh')}
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
          <span>{t('metrics.loadingMetrics')}</span>
        </div>
      )}

      {metrics && (
        <div className="grid grid-stats mb-6">
          <StatCard
            label={t('metrics.totalAgents')}
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
            label={t('metrics.activeTasks')}
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
            label={t('metrics.totalCost')}
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
            label={t('metrics.activeAlerts')}
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

      {timeSeries && (
        <div className="grid grid-2 animate-fade-in">
          <TimeSeriesChart
            title={t('metrics.tasksOverTime')}
            data={timeSeries.total_tasks}
            color="var(--color-primary)"
          />
          <TimeSeriesChart
            title={t('metrics.activeAgents')}
            data={timeSeries.active_agents}
            color="var(--color-success)"
          />
          <TimeSeriesChart
            title={t('metrics.costOverTime')}
            data={timeSeries.total_cost}
            color="var(--color-warning)"
            formatValue={(v) => `$${v.toFixed(2)}`}
          />
          <TimeSeriesChart
            title={t('metrics.alertCount')}
            data={timeSeries.alert_count}
            color="var(--color-danger)"
          />
          <TimeSeriesChart
            title={t('metrics.errorRate')}
            data={timeSeries.error_rate}
            color="var(--color-danger)"
            formatValue={(v) => `${(v * 100).toFixed(1)}%`}
          />
          <TimeSeriesChart
            title={t('metrics.avgLatency')}
            data={timeSeries.avg_latency}
            color="var(--color-info)"
            unit="ms"
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
          <h3>{t('metrics.noMetrics')}</h3>
          <p>
            {t('metrics.noMetricsDescription')}
          </p>
        </div>
      )}
    </div>
  );
}

export default MetricsPage;
