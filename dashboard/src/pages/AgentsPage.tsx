import { useEffect } from 'react';
import { useAgentStore } from '../stores/agentStore';
import AgentCard from '../components/agents/AgentCard';

function AgentsPage() {
  const { agents, loading, error, fetchAgents } = useAgentStore();

  useEffect(() => {
    fetchAgents();
  }, [fetchAgents]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>Agents</h2>
          <p>Monitor and manage discovered AI agents</p>
        </div>
        <div className="page-header-actions">
          <button className="btn" onClick={fetchAgents} disabled={loading}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="23 4 23 10 17 10" />
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
            </svg>
            Refresh
          </button>
        </div>
      </div>

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

      {agents.length > 0 && (
        <div className="grid grid-auto">
          {agents.map((agent) => (
            <AgentCard key={agent.id} agent={agent} />
          ))}
        </div>
      )}
    </div>
  );
}

export default AgentsPage;
