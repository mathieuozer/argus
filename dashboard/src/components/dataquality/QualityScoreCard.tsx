import { useTranslation } from 'react-i18next';
import type { DQScore } from '../../types/dataquality';

interface QualityScoreCardProps {
  score: DQScore;
}

function getScoreColor(score: number): string {
  if (score >= 0.9) return 'var(--color-success)';
  if (score >= 0.7) return 'var(--color-warning)';
  return 'var(--color-error)';
}

function QualityScoreCard({ score }: QualityScoreCardProps) {
  const { t } = useTranslation();

  const DIMENSIONS = [
    { key: 'completeness' as const, labelKey: 'dataQuality.completeness', color: 'var(--color-primary)' },
    { key: 'conformance' as const, labelKey: 'dataQuality.conformance', color: 'var(--color-success)' },
    { key: 'consistency' as const, labelKey: 'dataQuality.consistency', color: 'var(--color-warning)' },
    { key: 'freshness' as const, labelKey: 'dataQuality.freshness', color: '#8b5cf6' },
  ];

  return (
    <div className="card quality-score-card">
      <div className="card-header">
        <h4>{score.agent_id}</h4>
        <span className="badge" style={{ backgroundColor: getScoreColor(score.overall), color: '#fff' }}>
          {(score.overall * 100).toFixed(1)}%
        </span>
      </div>
      <div className="quality-dimensions">
        {DIMENSIONS.map(({ key, labelKey, color }) => (
          <div key={key} className="quality-dimension">
            <div className="quality-dimension-header">
              <span>{t(labelKey)}</span>
              <span>{(score[key] * 100).toFixed(1)}%</span>
            </div>
            <div className="quality-bar-bg">
              <div
                className="quality-bar-fill"
                style={{ width: `${score[key] * 100}%`, backgroundColor: color }}
              />
            </div>
          </div>
        ))}
      </div>
      <div className="quality-meta">
        <span>{t('dataQuality.samples', { count: score.sample_size })}</span>
        <span>{t('dataQuality.computed', { date: new Date(score.computed_at).toLocaleString() })}</span>
      </div>
    </div>
  );
}

export default QualityScoreCard;
