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
          <th>{t('catalog.identifier')}</th>
          <th>{t('catalog.agents')}</th>
          <th>{t('catalog.access')}</th>
          <th>{t('catalog.tier')}</th>
          <th>{t('catalog.spans')}</th>
          <th>{t('catalog.lastSeen')}</th>
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
            <td className="mono">{source.name}</td>
            <td className="text-muted mono" title={source.identifier}>
              {source.identifier.length > 40 ? source.identifier.slice(0, 40) + '...' : source.identifier}
            </td>
            <td>{source.agents.join(', ')}</td>
            <td>{source.access_types.join(', ')}</td>
            <td><span className="badge">T{source.tier}</span></td>
            <td>{source.span_count.toLocaleString()}</td>
            <td className="text-muted">{new Date(source.last_seen).toLocaleDateString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export default SourceTable;
