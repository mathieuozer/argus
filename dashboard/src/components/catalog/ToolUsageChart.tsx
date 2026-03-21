import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from 'recharts';
import type { ToolUsage } from '../../types/catalog';

interface ToolUsageChartProps {
  data: ToolUsage[];
}

function ToolUsageChart({ data }: ToolUsageChartProps) {
  const chartData = data.map((t) => ({
    name: t.tool.length > 20 ? t.tool.slice(0, 18) + '..' : t.tool,
    calls: t.call_count,
    avgMs: t.avg_duration_ms,
  }));

  return (
    <div className="chart-card">
      <div className="chart-card-header">
        <span className="chart-card-title">Tool / API Usage</span>
      </div>
      <div className="chart-card-body">
        <ResponsiveContainer width="100%" height={200}>
          <BarChart data={chartData} margin={{ top: 5, right: 10, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
            <XAxis dataKey="name" tick={{ fontSize: 10, fill: 'var(--color-text-dim)' }} tickLine={false} />
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
            <Bar dataKey="calls" fill="var(--color-primary)" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export default ToolUsageChart;
