import { useState } from 'react';

export type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | 'custom';

interface TimeRangePickerProps {
  value: TimeRange;
  onChange: (range: TimeRange) => void;
}

const OPTIONS: { label: string; value: TimeRange }[] = [
  { label: '1h', value: '1h' },
  { label: '6h', value: '6h' },
  { label: '24h', value: '24h' },
  { label: '7d', value: '7d' },
  { label: '30d', value: '30d' },
];

function TimeRangePicker({ value, onChange }: TimeRangePickerProps) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="time-range-picker">
      <div className="time-range-buttons">
        {OPTIONS.map((opt) => (
          <button
            key={opt.value}
            className={`time-range-btn ${value === opt.value ? 'active' : ''}`}
            onClick={() => onChange(opt.value)}
          >
            {opt.label}
          </button>
        ))}
        <button
          className={`time-range-btn ${value === 'custom' ? 'active' : ''}`}
          onClick={() => setIsOpen(!isOpen)}
        >
          Custom
        </button>
      </div>
      {isOpen && value === 'custom' && (
        <div className="time-range-custom">
          <input type="datetime-local" className="time-range-input" />
          <span className="time-range-separator">to</span>
          <input type="datetime-local" className="time-range-input" />
        </div>
      )}
    </div>
  );
}

export default TimeRangePicker;
