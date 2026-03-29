import { useTranslation } from 'react-i18next';
import type { DQScore } from '../../types/dataquality';

interface QualityScoreCardProps {
  score: DQScore;
}

function getScoreColor(score: number): string {
  if (score >= 90) return 'var(--color-success)';
  if (score >= 70) return 'var(--color-warning)';
  return 'var(--color-error)';
}

function QualityScoreCard({ score }: QualityScoreCardProps) {
  const { t } = useTranslation();

  const DIMENSIONS = [
    { key: 'completeness_score' as const, labelKey: 'dataQuality.completeness', color: 'var(--color-primary)' },
    { key: 'accuracy_score' as const, labelKey: 'dataQuality.conformance', color: 'var(--color-success)' },
    { key: 'consistency_score' as const, labelKey: 'dataQuality.consistency', color: 'var(--color-warning)' },
    { key: 'timeliness_score' as const, labelKey: 'dataQuality.freshness', color: '#8b5cf6' },
  ];

  return (
    <div className="card quality-score-card">
      <div className="card-header">
        <h4>{score.agent_id}</h4>
        <span className="badge" style={{ backgroundColor: getScoreColor(score.overall_score), color: '#fff' }}>
          {score.overall_score.toFixed(1)}%
        </span>
      </div>
      <div className="quality-dimensions">
        {DIMENSIONS.map(({ key, labelKey, color }) => (
          <div key={key} className="quality-dimension">
            <div className="quality-dimension-header">
              <span>{t(labelKey)}</span>
              <span>{score[key].toFixed(1)}%</span>
            </div>
            <div className="quality-bar-bg">
              <div
                className="quality-bar-fill"
                style={{ width: `${score[key]}%`, backgroundColor: color }}
              />
            </div>
          </div>
        ))}
      </div>
      <div className="quality-meta">
        <span>{t('dataQuality.samples', { count: score.total_checks })}</span>
        <span>{score.passed_checks}/{score.total_checks} passed</span>
      </div>
    </div>
  );
}

export default QualityScoreCard;
