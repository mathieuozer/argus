import { useTranslation } from 'react-i18next';
import type { DQScore } from '../../types/dataquality';

interface AgentQualityRowProps {
  score: DQScore;
  onClick: () => void;
}

function AgentQualityRow({ score, onClick }: AgentQualityRowProps) {
  const { t } = useTranslation();

  function getStatusLabel(overall: number): { text: string; className: string } {
    if (overall >= 90) return { text: t('dataQuality.good'), className: 'badge-success' };
    if (overall >= 70) return { text: t('dataQuality.fair'), className: 'badge-warning' };
    return { text: t('dataQuality.poor'), className: 'badge-error' };
  }

  const status = getStatusLabel(score.overall_score);

  return (
    <tr className="agent-quality-row clickable" onClick={onClick}>
      <td className="mono">{score.agent_id}</td>
      <td>{score.overall_score.toFixed(1)}%</td>
      <td>{score.completeness_score.toFixed(1)}%</td>
      <td>{score.accuracy_score.toFixed(1)}%</td>
      <td>{score.consistency_score.toFixed(1)}%</td>
      <td>{score.timeliness_score.toFixed(1)}%</td>
      <td><span className={`badge ${status.className}`}>{status.text}</span></td>
      <td>{score.total_checks}</td>
    </tr>
  );
}

export default AgentQualityRow;
