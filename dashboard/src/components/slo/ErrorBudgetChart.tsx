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
import type { ErrorBudgetPoint } from '../../types/slo';

interface ErrorBudgetChartProps {
  data: ErrorBudgetPoint[];
}

function ErrorBudgetChart({ data }: ErrorBudgetChartProps) {
  const chartData = data.map((point) => ({
    time: new Date(point.timestamp).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
    remaining: +(point.remaining * 100).toFixed(1),
  }));

  return (
    <div className="chart-card">
      <div className="chart-card-header">
        <span className="chart-card-title">Error Budget Burn-down</span>
      </div>
      <div className="chart-card-body">
        <ResponsiveContainer width="100%" height={200}>
          <AreaChart data={chartData} margin={{ top: 5, right: 10, left: 0, bottom: 0 }}>
            <defs>
              <linearGradient id="budget-gradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--color-primary)" stopOpacity={0.3} />
                <stop offset="95%" stopColor="var(--color-primary)" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
            <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }} tickLine={false} />
            <YAxis domain={[0, 100]} tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }} tickLine={false} width={40} tickFormatter={(v) => `${v}%`} />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-surface)',
                border: '1px solid var(--color-border)',
                borderRadius: 'var(--radius-md)',
                fontSize: '12px',
                color: 'var(--color-text)',
              }}
              formatter={(value) => [`${value}%`, 'Budget Remaining']}
            />
            <ReferenceLine y={20} stroke="var(--color-error)" strokeDasharray="5 5" />
            <Area type="monotone" dataKey="remaining" stroke="var(--color-primary)" fill="url(#budget-gradient)" strokeWidth={2} dot={false} />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export default ErrorBudgetChart;
