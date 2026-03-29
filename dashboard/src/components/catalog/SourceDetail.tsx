import { useTranslation } from 'react-i18next';
import type { CatalogSource } from '../../types/catalog';

interface SourceDetailProps {
  source: CatalogSource;
  onClose: () => void;
}

function SourceDetail({ source, onClose }: SourceDetailProps) {
  const { t } = useTranslation();

  return (
    <div className="source-detail-panel">
      <div className="source-detail-header">
        <h4>{source.name}</h4>
        <button className="btn-icon" onClick={onClose} aria-label={t('common.close')}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>
      <div className="source-detail-body">
        <p style={{ opacity: 0.8, marginBottom: 'var(--spacing-md)' }}>{source.description}</p>

        <div className="detail-row">
          <span className="detail-label">{t('catalog.sourceType')}</span>
          <span className="detail-value">{source.type}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">{t('catalog.domain')}</span>
          <span className="detail-value">{source.domain || '-'}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">{t('catalog.classification')}</span>
          <span className={`badge ${source.classification === 'restricted' ? 'badge-error' : source.classification === 'confidential' ? 'badge-warning' : 'badge-info'}`}>
            {source.classification || '-'}
          </span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Quality Score</span>
          <span className="detail-value">{source.quality_score ? `${source.quality_score.toFixed(1)}%` : '-'}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Owner</span>
          <span className="detail-value">{source.owner || '-'}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Steward</span>
          <span className="detail-value">{source.steward || '-'}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Agent</span>
          <span className="detail-value mono">{source.agent_id || '-'}</span>
        </div>

        {source.tags && source.tags.length > 0 && (
          <div className="detail-row">
            <span className="detail-label">Tags</span>
            <div className="detail-tags">
              {source.tags.map((tag) => (
                <span key={tag} className="badge">{tag}</span>
              ))}
            </div>
          </div>
        )}

        {source.freshness && (
          <div style={{ marginTop: 'var(--spacing-md)' }}>
            <h5>Freshness</h5>
            <div className="detail-row">
              <span className="detail-label">Last Refreshed</span>
              <span className="detail-value">{new Date(source.freshness.last_refreshed).toLocaleString()}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Frequency</span>
              <span className="detail-value">{source.freshness.refresh_frequency}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Status</span>
              <span className={`badge ${source.freshness.is_stale ? 'badge-error' : 'badge-success'}`}>
                {source.freshness.is_stale ? 'Stale' : 'Fresh'}
              </span>
            </div>
          </div>
        )}

        {source.profile && (
          <div style={{ marginTop: 'var(--spacing-md)' }}>
            <h5>Profile</h5>
            <div className="detail-row">
              <span className="detail-label">Rows</span>
              <span className="detail-value">{source.profile.row_count.toLocaleString()}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Columns</span>
              <span className="detail-value">{source.profile.column_count}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Size</span>
              <span className="detail-value">{(source.profile.size_bytes / 1e9).toFixed(1)} GB</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Completeness</span>
              <span className="detail-value">{source.profile.completeness.toFixed(1)}%</span>
            </div>
          </div>
        )}

        {source.columns && source.columns.length > 0 && (
          <div style={{ marginTop: 'var(--spacing-md)' }}>
            <h5>Columns ({source.columns.length})</h5>
            <table className="table" style={{ fontSize: '0.85em' }}>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Type</th>
                  <th>Class</th>
                  <th>PII</th>
                </tr>
              </thead>
              <tbody>
                {source.columns.map((col) => (
                  <tr key={col.name}>
                    <td className="mono">{col.name}</td>
                    <td>{col.type}</td>
                    <td><span className={`badge ${col.classification === 'restricted' ? 'badge-error' : col.classification === 'confidential' ? 'badge-warning' : 'badge-info'}`}>{col.classification}</span></td>
                    <td>{col.is_pii ? <span className="badge badge-error">PII</span> : '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {source.popularity && (
          <div style={{ marginTop: 'var(--spacing-md)' }}>
            <h5>Popularity</h5>
            <div className="detail-row">
              <span className="detail-label">Views</span>
              <span className="detail-value">{source.popularity.view_count}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Queries</span>
              <span className="detail-value">{source.popularity.query_count}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Trend</span>
              <span className="detail-value">{source.popularity.trend_direction}</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default SourceDetail;
