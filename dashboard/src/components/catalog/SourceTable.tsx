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
  return (
    <table className="table">
      <thead>
        <tr>
          <th>Type</th>
          <th>Name</th>
          <th>Identifier</th>
          <th>Agents</th>
          <th>Access</th>
          <th>Tier</th>
          <th>Spans</th>
          <th>Last Seen</th>
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
