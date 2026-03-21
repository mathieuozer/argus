import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import type { AgentDetail } from '../types/agent';
import type { AgentTimeSeries, TelemetrySpan } from '../types/telemetry';
import type { Task } from '../types/task';
import type { PredictiveAlert } from '../types/alert';
import apiClient from '../utils/apiClient';
import StatusBadge from '../components/agents/StatusBadge';
import TimeSeriesChart from '../components/metrics/TimeSeriesChart';

function formatUptime(pct: number): string {
  return `${pct.toFixed(1)}%`;
}

function formatCost(cost: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
  }).format(cost);
}

function formatTokens(tokens: number): string {
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(1)}M`;
  if (tokens >= 1_000) return `${(tokens / 1_000).toFixed(1)}K`;
  return tokens.toString();
}

function formatDuration(ms: number): string {
  if (ms < 1) return '<1ms';
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function formatTimestamp(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function AgentDetailPage() {
  const { agentId } = useParams<{ agentId: string }>();
  const navigate = useNavigate();
  const [agent, setAgent] = useState<AgentDetail | null>(null);
  const [timeSeries, setTimeSeries] = useState<AgentTimeSeries | null>(null);
  const [spans, setSpans] = useState<TelemetrySpan[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [alerts, setAlerts] = useState<PredictiveAlert[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'spans' | 'tasks' | 'alerts'>('overview');

  useEffect(() => {
    if (!agentId) return;

    async function load() {
      setLoading(true);
      setError(null);
      try {
        const [agentRes, tsRes, spansRes, tasksRes, alertsRes] = await Promise.all([
          apiClient.get<AgentDetail>(`/agents/${agentId}`),
          apiClient.get<AgentTimeSeries>(`/agents/${agentId}/timeseries`),
          apiClient.get<TelemetrySpan[]>(`/agents/${agentId}/spans`),
          apiClient.get<Task[]>('/tasks'),
          apiClient.get<PredictiveAlert[]>('/alerts'),
        ]);
        setAgent(agentRes.data);
        setTimeSeries(tsRes.data);
        setSpans(spansRes.data);
        setTasks(tasksRes.data.filter((t) => t.agent_id === agentId));
        setAlerts(alertsRes.data.filter((a) => a.agent_id === agentId));
      } catch (err) {
        setError((err as Error).message);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [agentId]);

  if (loading) {
    return (
      <div className="loading-container">
        <div className="loading-spinner" />
        <span>Loading agent details...</span>
      </div>
    );
  }

  if (error || !agent) {
    return (
      <div>
        <div className="error-banner">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
          {error || 'Agent not found'}
        </div>
        <button className="btn" onClick={() => navigate('/agents')}>Back to Agents</button>
      </div>
    );
  }

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <div className="flex items-center gap-3">
            <button className="btn btn-icon" onClick={() => navigate('/agents')} title="Back">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="19" y1="12" x2="5" y2="12" />
                <polyline points="12 19 5 12 12 5" />
              </svg>
            </button>
            <div>
              <div className="flex items-center gap-3">
                <h2>{agent.id}</h2>
                <StatusBadge status={agent.status} />
              </div>
              <p>
                <span className="badge badge-info">{agent.framework}</span>
                <span className="text-sm text-muted" style={{ marginLeft: '8px' }}>v{agent.version}</span>
                <span className="text-sm text-dim" style={{ marginLeft: '12px' }}>{agent.node_id}</span>
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Stats Row */}
      <div className="grid grid-stats mb-6">
        <div className="stat-card animate-fade-in">
          <div className="stat-card-content">
            <div className="stat-card-label">Tasks Completed</div>
            <div className="stat-card-value">{agent.tasks_completed}</div>
          </div>
        </div>
        <div className="stat-card animate-fade-in">
          <div className="stat-card-content">
            <div className="stat-card-label">Tasks Failed</div>
            <div className="stat-card-value" style={{ color: agent.tasks_failed > 0 ? 'var(--color-danger)' : undefined }}>{agent.tasks_failed}</div>
          </div>
        </div>
        <div className="stat-card animate-fade-in">
          <div className="stat-card-content">
            <div className="stat-card-label">Total Cost</div>
            <div className="stat-card-value">{formatCost(agent.total_cost_usd)}</div>
          </div>
        </div>
        <div className="stat-card animate-fade-in">
          <div className="stat-card-content">
            <div className="stat-card-label">Total Tokens</div>
            <div className="stat-card-value">{formatTokens(agent.total_tokens)}</div>
          </div>
        </div>
        <div className="stat-card animate-fade-in">
          <div className="stat-card-content">
            <div className="stat-card-label">Avg Latency</div>
            <div className="stat-card-value">{formatDuration(agent.avg_latency_ms)}</div>
          </div>
        </div>
        <div className="stat-card animate-fade-in">
          <div className="stat-card-content">
            <div className="stat-card-label">Uptime</div>
            <div className="stat-card-value" style={{ color: agent.uptime_pct >= 99 ? 'var(--color-success)' : agent.uptime_pct >= 95 ? 'var(--color-warning)' : 'var(--color-danger)' }}>{formatUptime(agent.uptime_pct)}</div>
          </div>
        </div>
      </div>

      {/* Tab navigation */}
      <div className="tab-bar mb-4">
        {(['overview', 'spans', 'tasks', 'alerts'] as const).map((tab) => (
          <button
            key={tab}
            className={`tab-item ${activeTab === tab ? 'tab-item-active' : ''}`}
            onClick={() => setActiveTab(tab)}
          >
            {tab.charAt(0).toUpperCase() + tab.slice(1)}
            {tab === 'alerts' && alerts.length > 0 && (
              <span className="badge badge-danger" style={{ marginLeft: '6px', fontSize: '10px', padding: '1px 5px' }}>
                {alerts.filter((a) => a.status === 'open').length}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {activeTab === 'overview' && timeSeries && (
        <div className="grid grid-2 animate-fade-in">
          <TimeSeriesChart title="Latency P50" data={timeSeries.latency_p50} color="var(--color-primary)" unit="ms" />
          <TimeSeriesChart title="Latency P99" data={timeSeries.latency_p99} color="var(--color-warning)" unit="ms" />
          <TimeSeriesChart title="Token Rate" data={timeSeries.token_rate} color="var(--color-info)" unit="/s" />
          <TimeSeriesChart title="Error Rate" data={timeSeries.error_rate} color="var(--color-danger)" formatValue={(v) => `${(v * 100).toFixed(1)}%`} />
          <TimeSeriesChart title="Cost per Task" data={timeSeries.cost} color="var(--color-success)" formatValue={(v) => `$${v.toFixed(3)}`} />
          <div className="card">
            <div className="card-header">
              <span className="card-title">Agent Identity</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">SPIFFE ID</span>
              <span className="detail-value text-mono text-sm truncate" style={{ maxWidth: '300px' }}>{agent.svid_uri}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Node</span>
              <span className="detail-value text-mono text-sm">{agent.node_id}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Framework</span>
              <span className="detail-value">{agent.framework}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Version</span>
              <span className="detail-value">v{agent.version}</span>
            </div>
            <div style={{ marginTop: 'var(--space-3)' }}>
              <span className="detail-label" style={{ marginBottom: 'var(--space-2)', display: 'block' }}>Capabilities</span>
              <div className="tag-list">
                {agent.capabilities.map((cap) => (
                  <span key={cap} className="tag">{cap}</span>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'spans' && (
        <div className="table-container animate-fade-in">
          <table className="table">
            <thead>
              <tr>
                <th>Span ID</th>
                <th>Operation</th>
                <th>Task</th>
                <th>Duration</th>
                <th>Tier</th>
                <th>Started</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {spans.map((span) => (
                <tr key={span.span_id}>
                  <td><span className="text-mono text-sm">{span.span_id.slice(-8)}</span></td>
                  <td><span className="badge badge-info">{span.operation_name}</span></td>
                  <td><span className="text-mono text-sm">{span.task_id}</span></td>
                  <td><span className="text-mono text-sm">{formatDuration(span.duration_ms)}</span></td>
                  <td><span className={`badge ${span.tier === 1 ? 'badge-success' : span.tier === 2 ? 'badge-warning' : 'badge-danger'}`}>Tier {span.tier}</span></td>
                  <td><span className="text-sm text-muted">{formatTimestamp(span.started_at)}</span></td>
                  <td>
                    {span.error_code ? (
                      <span className="badge badge-danger">{span.error_code}</span>
                    ) : (
                      <span className="badge badge-success">OK</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'tasks' && (
        <div className="table-container animate-fade-in">
          {tasks.length === 0 ? (
            <div className="empty-state" style={{ border: 'none' }}>
              <h3>No tasks for this agent</h3>
            </div>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>Task ID</th>
                  <th>Status</th>
                  <th>Cost</th>
                  <th>Tokens</th>
                  <th>Started</th>
                </tr>
              </thead>
              <tbody>
                {tasks.map((task) => (
                  <tr key={task.id}>
                    <td><span className="text-mono text-sm">{task.id}</span></td>
                    <td><StatusBadge status={task.status} /></td>
                    <td><span className="text-mono text-sm">{formatCost(task.cost_usd)}</span></td>
                    <td><span className="text-mono text-sm">{formatTokens(task.tokens_used)}</span></td>
                    <td><span className="text-sm text-muted">{formatTimestamp(task.started_at)}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {activeTab === 'alerts' && (
        <div className="table-container animate-fade-in">
          {alerts.length === 0 ? (
            <div className="empty-state" style={{ border: 'none' }}>
              <h3>No alerts for this agent</h3>
              <p>This agent has no predictive failure alerts.</p>
            </div>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>Alert ID</th>
                  <th>Probability</th>
                  <th>Precursor</th>
                  <th>TTF</th>
                  <th>Status</th>
                  <th>Created</th>
                </tr>
              </thead>
              <tbody>
                {alerts.map((alert) => (
                  <tr key={alert.id}>
                    <td><span className="text-mono text-sm">{alert.id}</span></td>
                    <td>
                      <span style={{ color: alert.probability >= 0.8 ? 'var(--color-danger)' : alert.probability >= 0.5 ? 'var(--color-warning)' : 'var(--color-success)', fontWeight: 600 }}>
                        {(alert.probability * 100).toFixed(0)}%
                      </span>
                    </td>
                    <td><span className="badge badge-warning">{alert.precursor_type.replace(/_/g, ' ')}</span></td>
                    <td><span className="text-mono">{alert.estimated_ttf}s</span></td>
                    <td><StatusBadge status={alert.status} /></td>
                    <td><span className="text-sm text-muted">{formatTimestamp(alert.created_at)}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}

export default AgentDetailPage;
