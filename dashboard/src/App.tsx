import { Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/layout/Layout';
import AgentsPage from './pages/AgentsPage';
import AgentDetailPage from './pages/AgentDetailPage';
import MetricsPage from './pages/MetricsPage';
import TasksPage from './pages/TasksPage';
import AlertsPage from './pages/AlertsPage';
import SettingsPage from './pages/SettingsPage';
import TracesPage from './pages/TracesPage';
import TraceDetailPage from './pages/TraceDetailPage';
import DataQualityPage from './pages/DataQualityPage';
import CatalogPage from './pages/CatalogPage';
import CostPage from './pages/CostPage';
import AuditPage from './pages/AuditPage';
import SLOPage from './pages/SLOPage';

function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Navigate to="/agents" replace />} />
        <Route path="/agents" element={<AgentsPage />} />
        <Route path="/agents/:agentId" element={<AgentDetailPage />} />
        <Route path="/metrics" element={<MetricsPage />} />
        <Route path="/tasks" element={<TasksPage />} />
        <Route path="/alerts" element={<AlertsPage />} />
        <Route path="/traces" element={<TracesPage />} />
        <Route path="/traces/:traceId" element={<TraceDetailPage />} />
        <Route path="/data-quality" element={<DataQualityPage />} />
        <Route path="/catalog" element={<CatalogPage />} />
        <Route path="/costs" element={<CostPage />} />
        <Route path="/audit" element={<AuditPage />} />
        <Route path="/slos" element={<SLOPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  );
}

export default App;
