import { useTranslation } from 'react-i18next';
import type { CatalogSource } from '../../types/catalog';

interface SourceTableProps {
  sources: CatalogSource[];
  onSelect: (source: CatalogSource) => void;
}

function getTypeIcon(type: string): string {
  switch (type) {
    case 'database': return 'DB';
    case 'api': return 'API';
    case 'storage': return 'S3';
    case 'file': return 'FS';
    case 'stream': return 'ST';
    case 'model': return 'ML';
    case 'tool': return 'TL';
    default: return '?';
  }
}

function SourceTable({ sources, onSelect }: SourceTableProps) {
  const { t } = useTranslation();

  return (
    <table className="table">
      <thead>
        <tr>
          <th>{t('catalog.sourceType')}</th>
          <th>{t('catalog.sourceName')}</th>
          <th>{t('catalog.domain')}</th>
          <th>{t('catalog.classification')}</th>
          <th>Quality</th>
          <th>Owner</th>
          <th>{t('common.status')}</th>
        </tr>
      </thead>
      <tbody>
        {sources.map((source) => (
          <tr key={source.id} className="clickable" onClick={() => onSelect(source)}>
            <td>
              <span className={`source-type-badge source-type-${source.type}`}>
                {getTypeIcon(source.type)}
              </span>
            </td>
            <td>
              <strong>{source.name}</strong>
              <br /><span style={{ opacity: 0.7, fontSize: '0.85em' }}>{source.description?.slice(0, 60)}{source.description?.length > 60 ? '...' : ''}</span>
            </td>
            <td><span className="badge">{source.domain || '-'}</span></td>
            <td>
              <span className={`badge ${source.classification === 'restricted' ? 'badge-error' : source.classification === 'confidential' ? 'badge-warning' : 'badge-info'}`}>
                {source.classification || '-'}
              </span>
            </td>
            <td>
              {source.quality_score > 0 ? (
                <span className={`badge ${source.quality_score >= 90 ? 'badge-success' : source.quality_score >= 70 ? 'badge-warning' : 'badge-error'}`}>
                  {source.quality_score.toFixed(0)}%
                </span>
              ) : '-'}
            </td>
            <td>{source.owner || '-'}</td>
            <td>
              <span className={`badge ${source.status === 'active' ? 'badge-success' : ''}`}>
                {source.status || 'active'}
              </span>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export default SourceTable;
