import { NavLink } from 'react-router-dom';

const navItems = [
  { path: '/agents', label: 'Agents', icon: '[]' },
  { path: '/metrics', label: 'Metrics', icon: '#' },
  { path: '/alerts', label: 'Alerts', icon: '!' },
  { path: '/settings', label: 'Settings', icon: '*' },
];

function Sidebar() {
  return (
    <aside
      style={{
        width: 'var(--sidebar-width)',
        backgroundColor: 'var(--color-surface)',
        borderRight: '1px solid var(--color-border)',
        position: 'fixed',
        top: 0,
        left: 0,
        bottom: 0,
        padding: '20px 0',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <div style={{ padding: '0 20px', marginBottom: '32px' }}>
        <h1 style={{ fontSize: '20px', fontWeight: 700 }}>Argus</h1>
        <p style={{ fontSize: '12px', color: 'var(--color-text-muted)' }}>
          Agent Orchestration Platform
        </p>
      </div>
      <nav style={{ flex: 1 }}>
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            style={({ isActive }) => ({
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              padding: '10px 20px',
              color: isActive ? 'var(--color-primary)' : 'var(--color-text-muted)',
              backgroundColor: isActive ? 'rgba(99, 102, 241, 0.1)' : 'transparent',
              borderLeft: isActive ? '3px solid var(--color-primary)' : '3px solid transparent',
              fontSize: '14px',
              transition: 'all 0.15s',
            })}
          >
            <span style={{ fontFamily: 'monospace', fontSize: '16px' }}>{item.icon}</span>
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}

export default Sidebar;
