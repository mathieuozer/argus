import { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAgentStore } from '../stores/agentStore';
import AgentCard from '../components/agents/AgentCard';
import SearchFilter from '../components/shared/SearchFilter';
import AutoRefreshToggle from '../components/shared/AutoRefreshToggle';

const STATUS_OPTIONS = [
  { label: 'Healthy', value: 'healthy' },
  { label: 'Degraded', value: 'degraded' },
  { label: 'Failed', value: 'failed' },
  { label: 'Quarantined', value: 'quarantined' },
  { label: 'Discovered', value: 'discovered' },
];

const FRAMEWORK_OPTIONS = [
  { label: 'LangChain', value: 'langchain' },
  { label: 'AutoGen', value: 'autogen' },
  { label: 'CrewAI', value: 'crewai' },
  { label: 'Custom', value: 'custom' },
];

function AgentsPage() {
  const { agents, loading, error, fetchAgents } = useAgentStore();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [frameworkFilter, setFrameworkFilter] = useState('');

  useEffect(() => {
    fetchAgents();
  }, [fetchAgents]);

  const filteredAgents = useMemo(() => {
    return agents.filter((agent) => {
      const matchesSearch = !search ||
        agent.id.toLowerCase().includes(search.toLowerCase()) ||
        agent.node_id.toLowerCase().includes(search.toLowerCase()) ||
        agent.capabilities.some((c) => c.toLowerCase().includes(search.toLowerCase()));
      const matchesStatus = !statusFilter || agent.status === statusFilter;
      const matchesFramework = !frameworkFilter || agent.framework === frameworkFilter;
      return matchesSearch && matchesStatus && matchesFramework;
    });
  }, [agents, search, statusFilter, frameworkFilter]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>Agents</h2>
          <p>Monitor and manage discovered AI agents</p>
        </div>
        <div className="page-header-actions">
          <AutoRefreshToggle onRefresh={fetchAgents} />
          <button className="btn" onClick={fetchAgents} disabled={loading}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="23 4 23 10 17 10" />
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
            </svg>
            Refresh
          </button>
        </div>
      </div>

      {agents.length > 0 && (
        <SearchFilter
          searchValue={search}
          onSearchChange={setSearch}
          searchPlaceholder="Search agents by name, node, or capability..."
          filters={[
            { label: 'All Statuses', value: statusFilter, options: STATUS_OPTIONS, onChange: setStatusFilter },
            { label: 'All Frameworks', value: frameworkFilter, options: FRAMEWORK_OPTIONS, onChange: setFrameworkFilter },
          ]}
        />
      )}

      {error && (
        <div className="error-banner">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
          {error}
        </div>
      )}

      {loading && agents.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>Loading agents...</span>
        </div>
      )}

      {!loading && !error && agents.length === 0 && (
        <div className="empty-state">
          <div className="empty-state-icon">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
              <circle cx="9" cy="7" r="4" />
              <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
              <path d="M16 3.13a4 4 0 0 1 0 7.75" />
            </svg>
          </div>
          <h3>No agents discovered yet</h3>
          <p>
            Deploy a sidecar alongside your agent to automatically discover and register it with Argus.
          </p>
        </div>
      )}

      {filteredAgents.length > 0 && (
        <div className="grid grid-auto">
          {filteredAgents.map((agent) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              onNavigate={() => navigate(`/agents/${agent.id}`)}
            />
          ))}
        </div>
      )}

      {agents.length > 0 && filteredAgents.length === 0 && (
        <div className="empty-state">
          <h3>No matching agents</h3>
          <p>Try adjusting your search or filters.</p>
        </div>
      )}
    </div>
  );
}

export default AgentsPage;
