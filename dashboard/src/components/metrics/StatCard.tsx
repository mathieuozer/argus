import type { ReactNode } from 'react';

interface StatCardProps {
  icon: ReactNode;
  label: string;
  value: string | number;
  iconBgColor?: string;
  iconColor?: string;
}

function StatCard({ icon, label, value, iconBgColor, iconColor }: StatCardProps) {
  return (
    <div className="stat-card animate-fade-in">
      <div
        className="stat-card-icon"
        style={{
          backgroundColor: iconBgColor || 'var(--color-primary-muted)',
          color: iconColor || 'var(--color-primary)',
        }}
      >
        {icon}
      </div>
      <div className="stat-card-content">
        <div className="stat-card-label">{label}</div>
        <div className="stat-card-value">{value}</div>
      </div>
    </div>
  );
}

export default StatCard;
