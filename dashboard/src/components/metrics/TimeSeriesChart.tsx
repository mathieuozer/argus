import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from 'recharts';
import type { TimeSeriesPoint } from '../../types/telemetry';

interface TimeSeriesChartProps {
  title: string;
  data: TimeSeriesPoint[];
  color: string;
  unit?: string;
  formatValue?: (value: number) => string;
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
}

function TimeSeriesChart({ title, data, color, unit, formatValue }: TimeSeriesChartProps) {
  const displayData = data.map((point) => ({
    time: formatTime(point.timestamp),
    value: +point.value.toFixed(2),
  }));

  const formatter = formatValue || ((v: number) => (unit ? `${v}${unit}` : String(v)));

  return (
    <div className="chart-card">
      <div className="chart-card-header">
        <span className="chart-card-title">{title}</span>
      </div>
      <div className="chart-card-body">
        <ResponsiveContainer width="100%" height={180}>
          <AreaChart data={displayData} margin={{ top: 5, right: 10, left: 0, bottom: 0 }}>
            <defs>
              <linearGradient id={`gradient-${title}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={color} stopOpacity={0.3} />
                <stop offset="95%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
            <XAxis
              dataKey="time"
              tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }}
              axisLine={{ stroke: 'var(--color-border)' }}
              tickLine={false}
              interval="preserveStartEnd"
            />
            <YAxis
              tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }}
              axisLine={false}
              tickLine={false}
              width={45}
              tickFormatter={(v) => formatter(v)}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-surface)',
                border: '1px solid var(--color-border)',
                borderRadius: 'var(--radius-md)',
                fontSize: '12px',
                color: 'var(--color-text)',
              }}
              formatter={(value) => [formatter(Number(value)), title]}
              labelStyle={{ color: 'var(--color-text-muted)' }}
            />
            <Area
              type="monotone"
              dataKey="value"
              stroke={color}
              strokeWidth={2}
              fill={`url(#gradient-${title})`}
              dot={false}
              activeDot={{ r: 3, strokeWidth: 0, fill: color }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export default TimeSeriesChart;
