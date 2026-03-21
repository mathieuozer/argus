import type { DQScore } from '../../types/dataquality';

interface AgentQualityRowProps {
  score: DQScore;
  onClick: () => void;
}

function getStatusLabel(overall: number): { text: string; className: string } {
  if (overall >= 0.9) return { text: 'Good', className: 'badge-success' };
  if (overall >= 0.7) return { text: 'Fair', className: 'badge-warning' };
  return { text: 'Poor', className: 'badge-error' };
}

function AgentQualityRow({ score, onClick }: AgentQualityRowProps) {
  const status = getStatusLabel(score.overall);

  return (
    <tr className="agent-quality-row clickable" onClick={onClick}>
      <td className="mono">{score.agent_id}</td>
      <td>{(score.overall * 100).toFixed(1)}%</td>
      <td>{(score.completeness * 100).toFixed(1)}%</td>
      <td>{(score.conformance * 100).toFixed(1)}%</td>
      <td>{(score.consistency * 100).toFixed(1)}%</td>
      <td>{(score.freshness * 100).toFixed(1)}%</td>
      <td><span className={`badge ${status.className}`}>{status.text}</span></td>
      <td>{score.sample_size}</td>
    </tr>
  );
}

export default AgentQualityRow;
