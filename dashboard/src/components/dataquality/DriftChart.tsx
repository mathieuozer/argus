import { useTranslation } from 'react-i18next';
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from 'recharts';
import type { DriftPoint } from '../../types/dataquality';

interface DriftChartProps {
  data: DriftPoint[];
}

function DriftChart({ data }: DriftChartProps) {
  const { t } = useTranslation();

  const displayData = data.map((point) => ({
    time: new Date(point.timestamp).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }),
    value: +point.value.toFixed(2),
    baseline: +point.baseline.toFixed(2),
    isAnomaly: point.is_anomaly,
  }));

  return (
    <div className="chart-card">
      <div className="chart-card-header">
        <span className="chart-card-title">{t('dataQuality.dataDrift')}</span>
      </div>
      <div className="chart-card-body">
        <ResponsiveContainer width="100%" height={200}>
          <AreaChart data={displayData} margin={{ top: 5, right: 10, left: 0, bottom: 0 }}>
            <defs>
              <linearGradient id="drift-gradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--color-warning)" stopOpacity={0.3} />
                <stop offset="95%" stopColor="var(--color-warning)" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
            <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }} tickLine={false} />
            <YAxis tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }} tickLine={false} width={40} />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-surface)',
                border: '1px solid var(--color-border)',
                borderRadius: 'var(--radius-md)',
                fontSize: '12px',
                color: 'var(--color-text)',
              }}
            />
            <Area
              type="monotone"
              dataKey="baseline"
              stroke="var(--color-text-muted)"
              fill="none"
              strokeWidth={1}
              strokeDasharray="5 3"
              dot={false}
            />
            <Area
              type="monotone"
              dataKey="value"
              stroke="var(--color-warning)"
              fill="url(#drift-gradient)"
              strokeWidth={2}
              dot={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export default DriftChart;
