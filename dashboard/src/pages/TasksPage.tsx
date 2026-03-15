import { useEffect } from 'react';
import { useTaskStore } from '../stores/taskStore';
import type { TaskStatus } from '../types/task';
import StatusBadge from '../components/agents/StatusBadge';

function formatDuration(startedAt: string, completedAt: string | null): string {
  const start = new Date(startedAt).getTime();
  const end = completedAt ? new Date(completedAt).getTime() : Date.now();
  const diffMs = end - start;

  if (diffMs < 1000) return `${diffMs}ms`;
  const seconds = Math.floor(diffMs / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m`;
}

function formatTimestamp(isoString: string): string {
  const date = new Date(isoString);
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatCost(cost: number): string {
  return `$${cost.toFixed(4)}`;
}

function formatTokens(tokens: number): string {
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(1)}M`;
  if (tokens >= 1_000) return `${(tokens / 1_000).toFixed(1)}K`;
  return tokens.toString();
}

function getStatusIndicator(status: TaskStatus): string {
  switch (status) {
    case 'running':
      return 'animate-pulse';
    default:
      return '';
  }
}

function TasksPage() {
  const { tasks, loading, error, fetchTasks } = useTaskStore();

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>Tasks</h2>
          <p>View and manage orchestrated agent tasks</p>
        </div>
        <div className="page-header-actions">
          <button className="btn" onClick={fetchTasks} disabled={loading}>
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

      {loading && tasks.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>Loading tasks...</span>
        </div>
      )}

      {!loading && !error && tasks.length === 0 && (
        <div className="empty-state">
          <div className="empty-state-icon">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="8" y1="6" x2="21" y2="6" />
              <line x1="8" y1="12" x2="21" y2="12" />
              <line x1="8" y1="18" x2="21" y2="18" />
              <line x1="3" y1="6" x2="3.01" y2="6" />
              <line x1="3" y1="12" x2="3.01" y2="12" />
              <line x1="3" y1="18" x2="3.01" y2="18" />
            </svg>
          </div>
          <h3>No tasks yet</h3>
          <p>
            Tasks will appear here when agents begin processing work through the orchestrator.
          </p>
        </div>
      )}

      {tasks.length > 0 && (
        <div className="table-container animate-fade-in">
          <table className="table">
            <thead>
              <tr>
                <th>Task ID</th>
                <th>Agent</th>
                <th>Status</th>
                <th>Cost</th>
                <th>Tokens</th>
                <th>Started</th>
                <th>Duration</th>
              </tr>
            </thead>
            <tbody>
              {tasks.map((task) => (
                <tr key={task.id}>
                  <td>
                    <span className="text-mono text-sm">{task.id.slice(0, 8)}</span>
                  </td>
                  <td>
                    <span className="font-medium">{task.agent_id}</span>
                  </td>
                  <td>
                    <span className={getStatusIndicator(task.status)}>
                      <StatusBadge status={task.status} />
                    </span>
                  </td>
                  <td>
                    <span className="text-mono text-sm">{formatCost(task.cost_usd)}</span>
                  </td>
                  <td>
                    <span className="text-mono text-sm">{formatTokens(task.tokens_used)}</span>
                  </td>
                  <td>
                    <span className="text-sm text-muted">
                      {formatTimestamp(task.started_at)}
                    </span>
                  </td>
                  <td>
                    <span className="text-mono text-sm">
                      {formatDuration(task.started_at, task.completed_at)}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

export default TasksPage;
