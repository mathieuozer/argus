import { useMemo } from 'react';
import type { LineageGraph as LineageGraphType } from '../../types/catalog';

interface LineageGraphProps {
  data: LineageGraphType;
}

const TYPE_COLORS: Record<string, string> = {
  database: '#f59e0b',
  api: '#06b6d4',
  file: '#8b5cf6',
  stream: '#10b981',
  model: '#ec4899',
  storage: '#6366f1',
  tool: '#f43f5e',
};

const CLASS_BORDER: Record<string, string> = {
  restricted: 'var(--color-error)',
  confidential: 'var(--color-warning)',
  internal: 'var(--color-info)',
};

const NODE_WIDTH = 180;
const NODE_HEIGHT = 55;
const H_GAP = 60;
const V_GAP = 25;
const PADDING = 40;

function LineageGraph({ data }: LineageGraphProps) {
  const layout = useMemo(() => {
    if (!data.nodes || data.nodes.length === 0) return { positions: {}, totalWidth: 600, totalHeight: 300 };

    // Build adjacency for topological sort
    const adj: Record<string, string[]> = {};
    const inDegree: Record<string, number> = {};
    for (const n of data.nodes) {
      adj[n.id] = [];
      inDegree[n.id] = 0;
    }
    for (const e of data.edges) {
      if (adj[e.source_id]) adj[e.source_id].push(e.target_id);
      if (inDegree[e.target_id] !== undefined) inDegree[e.target_id]++;
    }

    // BFS topological layering
    const layers: string[][] = [];
    const visited = new Set<string>();
    let queue = Object.keys(inDegree).filter(id => inDegree[id] === 0);
    while (queue.length > 0) {
      layers.push(queue);
      queue.forEach(id => visited.add(id));
      const next: string[] = [];
      for (const id of queue) {
        for (const child of (adj[id] || [])) {
          inDegree[child]--;
          if (inDegree[child] === 0 && !visited.has(child)) next.push(child);
        }
      }
      queue = next;
    }
    // Add any remaining nodes not reached
    const remaining = data.nodes.filter(n => !visited.has(n.id)).map(n => n.id);
    if (remaining.length > 0) layers.push(remaining);

    const positions: Record<string, { x: number; y: number }> = {};
    for (let col = 0; col < layers.length; col++) {
      for (let row = 0; row < layers[col].length; row++) {
        positions[layers[col][row]] = {
          x: PADDING + col * (NODE_WIDTH + H_GAP),
          y: PADDING + row * (NODE_HEIGHT + V_GAP),
        };
      }
    }

    const maxCol = layers.length;
    const maxRow = Math.max(...layers.map(l => l.length), 1);
    const totalWidth = PADDING * 2 + maxCol * (NODE_WIDTH + H_GAP);
    const totalHeight = PADDING * 2 + maxRow * (NODE_HEIGHT + V_GAP);

    return { positions, totalWidth, totalHeight };
  }, [data]);

  return (
    <div className="lineage-graph-container">
      <svg
        width={Math.max(layout.totalWidth, 600)}
        height={Math.max(layout.totalHeight, 300)}
        className="lineage-graph-svg"
      >
        <defs>
          <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">
            <polygon points="0 0, 10 3.5, 0 7" fill="var(--color-text-muted)" />
          </marker>
        </defs>

        {data.edges.map((edge, i) => {
          const sourcePos = layout.positions[edge.source_id];
          const targetPos = layout.positions[edge.target_id];
          if (!sourcePos || !targetPos) return null;

          const x1 = sourcePos.x + NODE_WIDTH;
          const y1 = sourcePos.y + NODE_HEIGHT / 2;
          const x2 = targetPos.x;
          const y2 = targetPos.y + NODE_HEIGHT / 2;

          return (
            <g key={i}>
              <line
                x1={x1} y1={y1} x2={x2} y2={y2}
                stroke="var(--color-text-muted)"
                strokeWidth="1.5"
                markerEnd="url(#arrowhead)"
                opacity={0.6}
              />
              <text
                x={(x1 + x2) / 2}
                y={(y1 + y2) / 2 - 6}
                textAnchor="middle"
                fill="var(--color-text-dim)"
                fontSize="9"
              >
                {edge.transform_type}
              </text>
            </g>
          );
        })}

        {data.nodes.map((node) => {
          const pos = layout.positions[node.id];
          if (!pos) return null;
          const fillColor = TYPE_COLORS[node.type] || 'var(--color-text-muted)';
          const borderColor = CLASS_BORDER[node.classification] || fillColor;

          return (
            <g key={node.id}>
              <rect
                x={pos.x} y={pos.y}
                width={NODE_WIDTH} height={NODE_HEIGHT}
                rx="8" ry="8"
                fill="var(--color-surface)"
                stroke={borderColor}
                strokeWidth="2"
              />
              {/* Type color indicator */}
              <rect
                x={pos.x} y={pos.y}
                width="6" height={NODE_HEIGHT}
                rx="8" ry="0"
                fill={fillColor}
                clipPath={`inset(0 ${NODE_WIDTH - 6}px 0 0)`}
              />
              <rect
                x={pos.x} y={pos.y}
                width="6" height={NODE_HEIGHT}
                fill={fillColor}
                style={{ borderRadius: '8px 0 0 8px' }}
              />
              <text
                x={pos.x + 14}
                y={pos.y + 18}
                fill="var(--color-text)"
                fontSize="11"
                fontWeight="600"
              >
                {node.name.length > 20 ? node.name.slice(0, 18) + '..' : node.name}
              </text>
              <text
                x={pos.x + 14}
                y={pos.y + 32}
                fill="var(--color-text-dim)"
                fontSize="9"
              >
                {node.type} | {node.domain || '-'}
              </text>
              {node.quality_score > 0 && (
                <text
                  x={pos.x + 14}
                  y={pos.y + 45}
                  fill={node.quality_score >= 90 ? 'var(--color-success)' : node.quality_score >= 70 ? 'var(--color-warning)' : 'var(--color-error)'}
                  fontSize="9"
                  fontWeight="500"
                >
                  Quality: {node.quality_score.toFixed(0)}%
                </text>
              )}
            </g>
          );
        })}
      </svg>

      {/* Legend */}
      <div style={{ display: 'flex', gap: 'var(--spacing-md)', padding: 'var(--spacing-sm)', flexWrap: 'wrap' }}>
        {Object.entries(TYPE_COLORS).map(([type, color]) => {
          const hasType = data.nodes.some(n => n.type === type);
          if (!hasType) return null;
          return (
            <div key={type} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <div style={{ width: 12, height: 12, borderRadius: 2, backgroundColor: color }} />
              <span style={{ fontSize: '0.8em', opacity: 0.8 }}>{type}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default LineageGraph;
