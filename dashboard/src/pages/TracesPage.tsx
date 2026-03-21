import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTraceStore } from '../stores/traceStore';
import TraceFilters from '../components/traces/TraceFilters';
import type { TimeRange } from '../components/shared/TimeRangePicker';

function TracesPage() {
  const { traces, loading, error, fetchTraces } = useTraceStore();
  const navigate = useNavigate();
  const [agentFilter, setAgentFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [timeRange, setTimeRange] = useState<TimeRange>('24h');

  useEffect(() => {
    fetchTraces();
  }, [fetchTraces]);

  const filteredTraces = useMemo(() => {
    return traces.filter((trace) => {
      const matchesAgent = !agentFilter || trace.agent_id.toLowerCase().includes(agentFilter.toLowerCase());
      const matchesStatus = !statusFilter ||
        (statusFilter === 'error' && trace.has_errors) ||
        (statusFilter === 'success' && !trace.has_errors);
      return matchesAgent && matchesStatus;
    });
  }, [traces, agentFilter, statusFilter]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>Traces</h2>
          <p>Distributed trace explorer for agent operations</p>
        </div>
        <div className="page-header-actions">
          <button className="btn" onClick={() => fetchTraces()} disabled={loading}>Refresh</button>
        </div>
      </div>

      <TraceFilters
        agentFilter={agentFilter}
        onAgentFilterChange={setAgentFilter}
        statusFilter={statusFilter}
        onStatusFilterChange={setStatusFilter}
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
      />

      {error && <div className="error-banner">{error}</div>}

      {loading && traces.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>Loading traces...</span>
        </div>
      )}

      {filteredTraces.length > 0 && (
        <table className="table">
          <thead>
            <tr>
              <th>Trace ID</th>
              <th>Root Operation</th>
              <th>Agent</th>
              <th>Spans</th>
              <th>Duration</th>
              <th>Status</th>
              <th>Started</th>
            </tr>
          </thead>
          <tbody>
            {filteredTraces.map((trace) => (
              <tr key={trace.trace_id} className="clickable" onClick={() => navigate(`/traces/${trace.trace_id}`)}>
                <td className="mono">{trace.trace_id.slice(0, 12)}...</td>
                <td>{trace.root_operation}</td>
                <td className="mono">{trace.agent_id}</td>
                <td>{trace.total_spans}</td>
                <td>{trace.total_duration_ms}ms</td>
                <td>
                  {trace.has_errors
                    ? <span className="badge badge-error">Error</span>
                    : <span className="badge badge-success">OK</span>
                  }
                </td>
                <td className="text-muted">{new Date(trace.started_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {!loading && filteredTraces.length === 0 && (
        <div className="empty-state">
          <h3>No traces found</h3>
          <p>Traces will appear here as agents process requests.</p>
        </div>
      )}
    </div>
  );
}

export default TracesPage;
