import { useState } from 'react';
import type { Agent } from '../../types/agent';
import StatusBadge from './StatusBadge';

interface AgentCardProps {
  agent: Agent;
}

function formatLastSeen(lastSeen: string): string {
  const date = new Date(lastSeen);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffSeconds / 60);
  const diffHours = Math.floor(diffMinutes / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSeconds < 60) return `${diffSeconds}s ago`;
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${diffDays}d ago`;
}

function AgentCard({ agent }: AgentCardProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div
      className="card card-clickable animate-fade-in"
      onClick={() => setExpanded(!expanded)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          setExpanded(!expanded);
        }
      }}
    >
      <div className="card-header">
        <div className="flex items-center gap-3">
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            style={{ color: 'var(--color-text-muted)' }}
          >
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
            <path d="M7 11V7a5 5 0 0 1 10 0v4" />
          </svg>
          <div>
            <span className="font-semibold text-md">{agent.id}</span>
            <div className="flex items-center gap-2 mt-1">
              <span className="badge badge-info">{agent.framework}</span>
              <span className="text-sm text-muted">v{agent.version}</span>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted">{formatLastSeen(agent.last_seen)}</span>
          <StatusBadge status={agent.status} />
        </div>
      </div>

      {expanded && (
        <div className="animate-fade-in" style={{ borderTop: '1px solid var(--color-border)', paddingTop: 'var(--space-4)' }}>
          <div className="detail-row">
            <span className="detail-label">Node ID</span>
            <span className="detail-value text-mono text-sm">{agent.node_id}</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">SPIFFE ID</span>
            <span className="detail-value text-mono text-sm truncate" style={{ maxWidth: '400px' }}>
              {agent.svid_uri}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Framework</span>
            <span className="detail-value">{agent.framework}</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Version</span>
            <span className="detail-value">v{agent.version}</span>
          </div>
          <div style={{ marginTop: 'var(--space-3)' }}>
            <span className="detail-label" style={{ marginBottom: 'var(--space-2)', display: 'block' }}>
              Capabilities
            </span>
            {agent.capabilities.length > 0 ? (
              <div className="tag-list">
                {agent.capabilities.map((cap) => (
                  <span key={cap} className="tag">{cap}</span>
                ))}
              </div>
            ) : (
              <span className="text-sm text-dim">No capabilities declared</span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export default AgentCard;
