import TimeRangePicker, { type TimeRange } from '../shared/TimeRangePicker';

interface TraceFiltersProps {
  agentFilter: string;
  onAgentFilterChange: (value: string) => void;
  statusFilter: string;
  onStatusFilterChange: (value: string) => void;
  timeRange: TimeRange;
  onTimeRangeChange: (range: TimeRange) => void;
}

function TraceFilters({
  agentFilter,
  onAgentFilterChange,
  statusFilter,
  onStatusFilterChange,
  timeRange,
  onTimeRangeChange,
}: TraceFiltersProps) {
  return (
    <div className="trace-filters">
      <div className="filter-group">
        <input
          type="text"
          className="filter-input"
          placeholder="Filter by agent ID..."
          value={agentFilter}
          onChange={(e) => onAgentFilterChange(e.target.value)}
        />
        <select
          className="filter-select"
          value={statusFilter}
          onChange={(e) => onStatusFilterChange(e.target.value)}
        >
          <option value="">All Traces</option>
          <option value="error">With Errors</option>
          <option value="success">Successful</option>
        </select>
      </div>
      <TimeRangePicker value={timeRange} onChange={onTimeRangeChange} />
    </div>
  );
}

export default TraceFilters;
