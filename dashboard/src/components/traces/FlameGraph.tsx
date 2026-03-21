import type { FlameGraphNode } from '../../types/trace';

interface FlameGraphProps {
  data: FlameGraphNode;
}

const COLORS = [
  'var(--color-primary)',
  'var(--color-success)',
  'var(--color-warning)',
  '#8b5cf6',
  '#ec4899',
  '#06b6d4',
];

import type { ReactNode } from 'react';

function renderNode(node: FlameGraphNode, totalValue: number, depth: number = 0, offset: number = 0): ReactNode[] {
  const width = (node.value / totalValue) * 100;
  const color = COLORS[depth % COLORS.length];
  const elements: ReactNode[] = [];

  elements.push(
    <div
      key={`${node.name}-${depth}-${offset}`}
      className="flame-graph-bar"
      style={{
        left: `${offset}%`,
        width: `${Math.max(width, 0.5)}%`,
        top: `${depth * 24}px`,
        backgroundColor: color,
      }}
      title={`${node.name}: ${node.value}ms`}
    >
      {width > 5 && <span className="flame-graph-label">{node.name}</span>}
    </div>
  );

  let childOffset = offset;
  for (const child of node.children) {
    elements.push(...renderNode(child, totalValue, depth + 1, childOffset));
    childOffset += (child.value / totalValue) * 100;
  }

  return elements;
}

function getMaxDepth(node: FlameGraphNode): number {
  if (node.children.length === 0) return 1;
  return 1 + Math.max(...node.children.map(getMaxDepth));
}

function FlameGraph({ data }: FlameGraphProps) {
  const maxDepth = getMaxDepth(data);

  return (
    <div className="flame-graph" style={{ height: `${maxDepth * 24 + 8}px`, position: 'relative' }}>
      {renderNode(data, data.value)}
    </div>
  );
}

export default FlameGraph;
