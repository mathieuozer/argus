import { Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/layout/Layout';
import LoginPage from './pages/LoginPage';
import ProtectedRoute from './components/auth/ProtectedRoute';
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
import EvalsPage from './pages/EvalsPage';
import PromptsPage from './pages/PromptsPage';
import RAGPage from './pages/RAGPage';
import FeedbackPage from './pages/FeedbackPage';
import PlaygroundPage from './pages/PlaygroundPage';
import CompliancePage from './pages/CompliancePage';

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
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
        <Route path="/evals" element={<EvalsPage />} />
        <Route path="/prompts" element={<PromptsPage />} />
        <Route path="/rag" element={<RAGPage />} />
        <Route path="/feedback" element={<FeedbackPage />} />
        <Route path="/playground" element={<PlaygroundPage />} />
        <Route path="/compliance" element={<CompliancePage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  );
}

export default App;
