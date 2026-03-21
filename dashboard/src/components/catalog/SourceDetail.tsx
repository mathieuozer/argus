import type { CatalogSource } from '../../types/catalog';

interface SourceDetailProps {
  source: CatalogSource;
  onClose: () => void;
}

function SourceDetail({ source, onClose }: SourceDetailProps) {
  return (
    <div className="source-detail-panel">
      <div className="source-detail-header">
        <h4>{source.name}</h4>
        <button className="btn-icon" onClick={onClose} aria-label="Close">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>
      <div className="source-detail-body">
        <div className="detail-row">
          <span className="detail-label">Type</span>
          <span className="detail-value">{source.type}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Identifier</span>
          <span className="detail-value mono">{source.identifier}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Data Tier</span>
          <span className="detail-value">Tier {source.tier}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Agents</span>
          <div className="detail-tags">
            {source.agents.map((agent) => (
              <span key={agent} className="badge">{agent}</span>
            ))}
          </div>
        </div>
        <div className="detail-row">
          <span className="detail-label">Access Types</span>
          <div className="detail-tags">
            {source.access_types.map((access) => (
              <span key={access} className="badge">{access}</span>
            ))}
          </div>
        </div>
        <div className="detail-row">
          <span className="detail-label">Span Count</span>
          <span className="detail-value">{source.span_count.toLocaleString()}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">First Seen</span>
          <span className="detail-value">{new Date(source.first_seen).toLocaleString()}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Last Seen</span>
          <span className="detail-value">{new Date(source.last_seen).toLocaleString()}</span>
        </div>
      </div>
    </div>
  );
}

export default SourceDetail;
