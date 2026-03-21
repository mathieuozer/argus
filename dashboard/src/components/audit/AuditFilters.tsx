import TimeRangePicker, { type TimeRange } from '../shared/TimeRangePicker';

interface AuditFiltersProps {
  actorFilter: string;
  onActorFilterChange: (value: string) => void;
  actionFilter: string;
  onActionFilterChange: (value: string) => void;
  timeRange: TimeRange;
  onTimeRangeChange: (range: TimeRange) => void;
}

function AuditFilters({
  actorFilter,
  onActorFilterChange,
  actionFilter,
  onActionFilterChange,
  timeRange,
  onTimeRangeChange,
}: AuditFiltersProps) {
  return (
    <div className="audit-filters">
      <div className="filter-group">
        <input
          type="text"
          className="filter-input"
          placeholder="Filter by actor..."
          value={actorFilter}
          onChange={(e) => onActorFilterChange(e.target.value)}
        />
        <input
          type="text"
          className="filter-input"
          placeholder="Filter by action..."
          value={actionFilter}
          onChange={(e) => onActionFilterChange(e.target.value)}
        />
      </div>
      <TimeRangePicker value={timeRange} onChange={onTimeRangeChange} />
    </div>
  );
}

export default AuditFilters;
