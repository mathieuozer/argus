import { useEffect, useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useTaskStore } from '../stores/taskStore';
import type { TaskStatus } from '../types/task';
import StatusBadge from '../components/agents/StatusBadge';
import SearchFilter from '../components/shared/SearchFilter';
import AutoRefreshToggle from '../components/shared/AutoRefreshToggle';

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
  const { t } = useTranslation();
  const { tasks, loading, error, fetchTasks } = useTaskStore();

  const STATUS_OPTIONS = [
    { label: t('tasks.statusOptions.pending'), value: 'pending' },
    { label: t('tasks.statusOptions.running'), value: 'running' },
    { label: t('tasks.statusOptions.completed'), value: 'completed' },
    { label: t('tasks.statusOptions.failed'), value: 'failed' },
    { label: t('tasks.statusOptions.awaitingApproval'), value: 'awaiting_approval' },
  ];
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  const filteredTasks = useMemo(() => {
    return tasks.filter((task) => {
      const matchesSearch = !search ||
        task.id.toLowerCase().includes(search.toLowerCase()) ||
        task.agent_id.toLowerCase().includes(search.toLowerCase());
      const matchesStatus = !statusFilter || task.status === statusFilter;
      return matchesSearch && matchesStatus;
    });
  }, [tasks, search, statusFilter]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('tasks.title')}</h2>
          <p>{t('tasks.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <AutoRefreshToggle onRefresh={fetchTasks} />
          <button className="btn" onClick={fetchTasks} disabled={loading}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="23 4 23 10 17 10" />
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
            </svg>
            {t('common.refresh')}
          </button>
        </div>
      </div>

      {tasks.length > 0 && (
        <SearchFilter
          searchValue={search}
          onSearchChange={setSearch}
          searchPlaceholder={t('tasks.searchPlaceholder')}
          filters={[
            { label: t('tasks.allStatuses'), value: statusFilter, options: STATUS_OPTIONS, onChange: setStatusFilter },
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

      {loading && tasks.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>{t('tasks.loadingTasks')}</span>
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
          <h3>{t('tasks.noTasks')}</h3>
          <p>
            {t('tasks.noTasksDescription')}
          </p>
        </div>
      )}

      {filteredTasks.length > 0 && (
        <div className="table-container animate-fade-in">
          <table className="table">
            <thead>
              <tr>
                <th>{t('tasks.taskId')}</th>
                <th>{t('tasks.agent')}</th>
                <th>{t('tasks.status')}</th>
                <th>{t('tasks.cost')}</th>
                <th>{t('tasks.tokens')}</th>
                <th>{t('tasks.started')}</th>
                <th>{t('tasks.duration')}</th>
              </tr>
            </thead>
            <tbody>
              {filteredTasks.map((task) => (
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

      {tasks.length > 0 && filteredTasks.length === 0 && (
        <div className="empty-state">
          <h3>{t('tasks.noMatching')}</h3>
          <p>{t('tasks.noMatchingDescription')}</p>
        </div>
      )}
    </div>
  );
}

export default TasksPage;
