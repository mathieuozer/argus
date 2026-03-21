import { useState } from 'react';
import { useTranslation } from 'react-i18next';

export type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | 'custom';

interface TimeRangePickerProps {
  value: TimeRange;
  onChange: (range: TimeRange) => void;
}

const OPTIONS: { value: TimeRange }[] = [
  { value: '1h' },
  { value: '6h' },
  { value: '24h' },
  { value: '7d' },
  { value: '30d' },
];

function TimeRangePicker({ value, onChange }: TimeRangePickerProps) {
  const { t } = useTranslation();
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
            {t(`timeRange.${opt.value}`)}
          </button>
        ))}
        <button
          className={`time-range-btn ${value === 'custom' ? 'active' : ''}`}
          onClick={() => setIsOpen(!isOpen)}
        >
          {t('common.custom')}
        </button>
      </div>
      {isOpen && value === 'custom' && (
        <div className="time-range-custom">
          <input type="datetime-local" className="time-range-input" />
          <span className="time-range-separator">{t('common.to')}</span>
          <input type="datetime-local" className="time-range-input" />
        </div>
      )}
    </div>
  );
}

export default TimeRangePicker;
