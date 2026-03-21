import { useMemo } from 'react';
import type { LineageGraph as LineageGraphType } from '../../types/catalog';

interface LineageGraphProps {
  data: LineageGraphType;
}

const NODE_COLORS: Record<string, string> = {
  agent: 'var(--color-primary)',
  database: '#f59e0b',
  storage: '#8b5cf6',
  external_api: '#06b6d4',
  tool: '#ec4899',
};

const NODE_WIDTH = 140;
const NODE_HEIGHT = 40;
const PADDING = 60;

function LineageGraph({ data }: LineageGraphProps) {
  const layout = useMemo(() => {
    const nodesByType: Record<string, typeof data.nodes> = {};
    for (const node of data.nodes) {
      const type = node.type;
      if (!nodesByType[type]) nodesByType[type] = [];
      nodesByType[type].push(node);
    }

    const columns = ['database', 'storage', 'agent', 'external_api', 'tool'];
    const positions: Record<string, { x: number; y: number }> = {};
    let colIdx = 0;

    for (const type of columns) {
      const nodes = nodesByType[type] || [];
      nodes.forEach((node, rowIdx) => {
        positions[node.id] = {
          x: PADDING + colIdx * (NODE_WIDTH + 80),
          y: PADDING + rowIdx * (NODE_HEIGHT + 30),
        };
      });
      if (nodes.length > 0) colIdx++;
    }

    const totalWidth = PADDING * 2 + colIdx * (NODE_WIDTH + 80);
    const maxRows = Math.max(...Object.values(nodesByType).map((n) => n.length), 1);
    const totalHeight = PADDING * 2 + maxRows * (NODE_HEIGHT + 30);

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
          const sourcePos = layout.positions[edge.source];
          const targetPos = layout.positions[edge.target];
          if (!sourcePos || !targetPos) return null;

          const x1 = sourcePos.x + NODE_WIDTH / 2;
          const y1 = sourcePos.y + NODE_HEIGHT / 2;
          const x2 = targetPos.x + NODE_WIDTH / 2;
          const y2 = targetPos.y + NODE_HEIGHT / 2;

          return (
            <g key={i}>
              <line
                x1={x1} y1={y1} x2={x2} y2={y2}
                stroke="var(--color-text-muted)"
                strokeWidth="1.5"
                strokeDasharray={edge.label === 'read' ? '5,3' : undefined}
                markerEnd="url(#arrowhead)"
                opacity={0.6}
              />
              <text
                x={(x1 + x2) / 2}
                y={(y1 + y2) / 2 - 6}
                textAnchor="middle"
                fill="var(--color-text-dim)"
                fontSize="10"
              >
                {edge.label} ({edge.span_count})
              </text>
            </g>
          );
        })}

        {data.nodes.map((node) => {
          const pos = layout.positions[node.id];
          if (!pos) return null;
          const color = NODE_COLORS[node.type] || 'var(--color-text-muted)';

          return (
            <g key={node.id}>
              <rect
                x={pos.x} y={pos.y}
                width={NODE_WIDTH} height={NODE_HEIGHT}
                rx="6" ry="6"
                fill="var(--color-surface)"
                stroke={color}
                strokeWidth="2"
              />
              <text
                x={pos.x + NODE_WIDTH / 2}
                y={pos.y + NODE_HEIGHT / 2 - 4}
                textAnchor="middle"
                fill="var(--color-text)"
                fontSize="11"
                fontWeight="500"
              >
                {node.id.length > 16 ? node.id.slice(0, 14) + '..' : node.id}
              </text>
              <text
                x={pos.x + NODE_WIDTH / 2}
                y={pos.y + NODE_HEIGHT / 2 + 10}
                textAnchor="middle"
                fill="var(--color-text-dim)"
                fontSize="9"
              >
                {node.type}
              </text>
            </g>
          );
        })}
      </svg>
    </div>
  );
}

export default LineageGraph;
