import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
  ReferenceLine,
} from 'recharts';
import type { DriftPoint } from '../../types/dataquality';

interface DriftChartProps {
  data: DriftPoint[];
}

function DriftChart({ data }: DriftChartProps) {
  const displayData = data.map((point) => ({
    time: new Date(point.timestamp).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }),
    consistency: +(point.consistency * 100).toFixed(1),
    baseline: +(point.baseline * 100).toFixed(1),
  }));

  return (
    <div className="chart-card">
      <div className="chart-card-header">
        <span className="chart-card-title">Data Drift</span>
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
            <YAxis domain={[0, 100]} tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }} tickLine={false} width={40} />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-surface)',
                border: '1px solid var(--color-border)',
                borderRadius: 'var(--radius-md)',
                fontSize: '12px',
                color: 'var(--color-text)',
              }}
              formatter={(value) => [`${value}%`]}
            />
            <ReferenceLine y={70} stroke="var(--color-error)" strokeDasharray="5 5" label="Threshold" />
            <Area
              type="monotone"
              dataKey="consistency"
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
