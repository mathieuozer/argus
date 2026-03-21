import { useEffect, useState, useMemo } from 'react';
import { useAuditStore } from '../stores/auditStore';
import AuditTable from '../components/audit/AuditTable';
import AuditFilters from '../components/audit/AuditFilters';
import AuditExportButton from '../components/audit/AuditExportButton';
import type { TimeRange } from '../components/shared/TimeRangePicker';

function AuditPage() {
  const { entries, loading, error, fetchEntries } = useAuditStore();
  const [actorFilter, setActorFilter] = useState('');
  const [actionFilter, setActionFilter] = useState('');
  const [timeRange, setTimeRange] = useState<TimeRange>('24h');

  useEffect(() => {
    fetchEntries();
  }, [fetchEntries]);

  const filteredEntries = useMemo(() => {
    return entries.filter((entry) => {
      const matchesActor = !actorFilter || entry.actor.toLowerCase().includes(actorFilter.toLowerCase());
      const matchesAction = !actionFilter || entry.action.toLowerCase().includes(actionFilter.toLowerCase());
      return matchesActor && matchesAction;
    });
  }, [entries, actorFilter, actionFilter]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>Audit Log</h2>
          <p>Immutable audit trail of all platform actions</p>
        </div>
        <div className="page-header-actions">
          <AuditExportButton entries={filteredEntries} />
          <button className="btn" onClick={() => fetchEntries()} disabled={loading}>Refresh</button>
        </div>
      </div>

      <AuditFilters
        actorFilter={actorFilter}
        onActorFilterChange={setActorFilter}
        actionFilter={actionFilter}
        onActionFilterChange={setActionFilter}
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
      />

      {error && <div className="error-banner">{error}</div>}

      {loading && entries.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>Loading audit log...</span>
        </div>
      )}

      <AuditTable entries={filteredEntries} />

      {!loading && entries.length === 0 && (
        <div className="empty-state">
          <h3>No audit entries</h3>
          <p>Audit entries will appear here as platform actions are performed.</p>
        </div>
      )}
    </div>
  );
}

export default AuditPage;
