import { useTranslation } from 'react-i18next';
import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from 'recharts';
import type { CostBreakdown } from '../../types/cost';

interface CostBreakdownChartProps {
  data: CostBreakdown[];
}

function CostBreakdownChart({ data }: CostBreakdownChartProps) {
  const { t } = useTranslation();

  const chartData = data.map((item) => ({
    name: item.group,
    cost: +item.cost_usd.toFixed(2),
    tokens: item.tokens_used,
  }));

  return (
    <div className="chart-card">
      <div className="chart-card-header">
        <span className="chart-card-title">{t('catalog.costBreakdown')}</span>
      </div>
      <div className="chart-card-body">
        <ResponsiveContainer width="100%" height={220}>
          <BarChart data={chartData} margin={{ top: 5, right: 10, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
            <XAxis dataKey="name" tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }} tickLine={false} />
            <YAxis tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }} tickLine={false} width={50} tickFormatter={(v) => `$${v}`} />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-surface)',
                border: '1px solid var(--color-border)',
                borderRadius: 'var(--radius-md)',
                fontSize: '12px',
                color: 'var(--color-text)',
              }}
              formatter={(value, name) => [name === 'cost' ? `$${value}` : Number(value).toLocaleString(), name === 'cost' ? t('agentDetail.cost') : t('agentDetail.tokens')]}
            />
            <Bar dataKey="cost" fill="var(--color-primary)" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export default CostBreakdownChart;
