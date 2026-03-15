import { useEffect } from 'react';
import { useAgentStore } from '../stores/agentStore';

function AgentsPage() {
  const { agents, loading, error, fetchAgents } = useAgentStore();

  useEffect(() => {
    fetchAgents();
  }, [fetchAgents]);

  return (
    <div>
      <h2 style={{ marginBottom: '16px' }}>Agents</h2>
      {loading && <p style={{ color: 'var(--color-text-muted)' }}>Loading agents...</p>}
      {error && <p style={{ color: 'var(--color-danger)' }}>Error: {error}</p>}
      {!loading && !error && agents.length === 0 && (
        <div
          style={{
            padding: '40px',
            textAlign: 'center',
            color: 'var(--color-text-muted)',
            backgroundColor: 'var(--color-surface)',
            borderRadius: '8px',
            border: '1px solid var(--color-border)',
          }}
        >
          <p>No agents discovered yet.</p>
          <p style={{ fontSize: '14px', marginTop: '8px' }}>
            Deploy a sidecar alongside your agent to get started.
          </p>
        </div>
      )}
      {agents.length > 0 && (
        <div style={{ display: 'grid', gap: '12px' }}>
          {agents.map((agent) => (
            <div
              key={agent.id}
              style={{
                padding: '16px',
                backgroundColor: 'var(--color-surface)',
                borderRadius: '8px',
                border: '1px solid var(--color-border)',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
                <strong>{agent.id}</strong>
                <span
                  style={{
                    fontSize: '12px',
                    padding: '2px 8px',
                    borderRadius: '4px',
                    backgroundColor:
                      agent.status === 'healthy'
                        ? 'rgba(34, 197, 94, 0.2)'
                        : agent.status === 'degraded'
                          ? 'rgba(245, 158, 11, 0.2)'
                          : 'rgba(239, 68, 68, 0.2)',
                    color:
                      agent.status === 'healthy'
                        ? 'var(--color-success)'
                        : agent.status === 'degraded'
                          ? 'var(--color-warning)'
                          : 'var(--color-danger)',
                  }}
                >
                  {agent.status}
                </span>
              </div>
              <div style={{ fontSize: '14px', color: 'var(--color-text-muted)' }}>
                {agent.framework} v{agent.version} | Node: {agent.node_id}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default AgentsPage;
