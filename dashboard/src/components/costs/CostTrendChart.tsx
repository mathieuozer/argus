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
import type { CostTrend } from '../../types/cost';

interface CostTrendChartProps {
  data: CostTrend[];
}

function CostTrendChart({ data }: CostTrendChartProps) {
  const { t } = useTranslation();

  const chartData = data.map((point) => ({
    time: new Date(point.timestamp).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
    cost: +point.cost_usd.toFixed(2),
  }));

  return (
    <div className="chart-card">
      <div className="chart-card-header">
        <span className="chart-card-title">{t('costs.spendingTrend')}</span>
      </div>
      <div className="chart-card-body">
        <ResponsiveContainer width="100%" height={200}>
          <AreaChart data={chartData} margin={{ top: 5, right: 10, left: 0, bottom: 0 }}>
            <defs>
              <linearGradient id="cost-gradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--color-success)" stopOpacity={0.3} />
                <stop offset="95%" stopColor="var(--color-success)" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
            <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }} tickLine={false} />
            <YAxis tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }} tickLine={false} width={50} tickFormatter={(v) => `$${v}`} />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-surface)',
                border: '1px solid var(--color-border)',
                borderRadius: 'var(--radius-md)',
                fontSize: '12px',
                color: 'var(--color-text)',
              }}
              formatter={(value) => [`$${value}`, t('agentDetail.cost')]}
            />
            <Area type="monotone" dataKey="cost" stroke="var(--color-success)" fill="url(#cost-gradient)" strokeWidth={2} dot={false} />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export default CostTrendChart;
