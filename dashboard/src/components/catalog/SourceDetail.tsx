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
        <div className="detail-row">
          <span className="detail-label">{t('catalog.sourceType')}</span>
          <span className="detail-value">{source.type}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">{t('catalog.identifier')}</span>
          <span className="detail-value mono">{source.identifier}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">{t('catalog.dataTier')}</span>
          <span className="detail-value">{t('catalog.tierLabel', { tier: source.tier })}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">{t('catalog.agents')}</span>
          <div className="detail-tags">
            {source.agents.map((agent) => (
              <span key={agent} className="badge">{agent}</span>
            ))}
          </div>
        </div>
        <div className="detail-row">
          <span className="detail-label">{t('catalog.accessTypes')}</span>
          <div className="detail-tags">
            {source.access_types.map((access) => (
              <span key={access} className="badge">{access}</span>
            ))}
          </div>
        </div>
        <div className="detail-row">
          <span className="detail-label">{t('catalog.spanCount')}</span>
          <span className="detail-value">{source.span_count.toLocaleString()}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">{t('catalog.firstSeen')}</span>
          <span className="detail-value">{new Date(source.first_seen).toLocaleString()}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">{t('catalog.lastSeen')}</span>
          <span className="detail-value">{new Date(source.last_seen).toLocaleString()}</span>
        </div>
      </div>
    </div>
  );
}

export default SourceDetail;
