import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';

function Layout() {
  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      <Sidebar />
      <main style={{ flex: 1, padding: '24px', marginLeft: 'var(--sidebar-width)' }}>
        <Outlet />
      </main>
    </div>
  );
}

export default Layout;
