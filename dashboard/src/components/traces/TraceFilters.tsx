import { useTranslation } from 'react-i18next';
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
  const { t } = useTranslation();

  return (
    <div className="trace-filters">
      <div className="filter-group">
        <input
          type="text"
          className="filter-input"
          placeholder={t('traces.filterByAgent')}
          value={agentFilter}
          onChange={(e) => onAgentFilterChange(e.target.value)}
        />
        <select
          className="filter-select"
          value={statusFilter}
          onChange={(e) => onStatusFilterChange(e.target.value)}
        >
          <option value="">{t('traces.allTraces')}</option>
          <option value="error">{t('traces.withErrors')}</option>
          <option value="success">{t('traces.successful')}</option>
        </select>
      </div>
      <TimeRangePicker value={timeRange} onChange={onTimeRangeChange} />
    </div>
  );
}

export default TraceFilters;
