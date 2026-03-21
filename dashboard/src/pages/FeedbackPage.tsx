import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useFeedbackStore } from '../stores/feedbackStore';

function FeedbackPage() {
  const { t } = useTranslation();
  const { feedbacks, summaries, isLoading, error, fetchFeedbacks, submitFeedback, fetchSummary } = useFeedbackStore();
  const [showSubmit, setShowSubmit] = useState(false);
  const [newFeedback, setNewFeedback] = useState({ agent_id: '', span_id: '', rating: 1, comment: '' });

  useEffect(() => {
    fetchFeedbacks();
    fetchSummary();
  }, [fetchFeedbacks, fetchSummary]);

  const handleSubmit = async () => {
    await submitFeedback(newFeedback);
    setShowSubmit(false);
    setNewFeedback({ agent_id: '', span_id: '', rating: 1, comment: '' });
    fetchFeedbacks();
    fetchSummary();
  };

  return (
    <div className="page">
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('feedback.title')}</h2>
          <p>{t('feedback.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <button className="btn btn-primary" onClick={() => setShowSubmit(!showSubmit)}>
            {t('feedback.submitFeedback')}
          </button>
        </div>
      </div>

      {error && (
        <div className="error-banner">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
          {error}
        </div>
      )}

      {summaries.length > 0 && (
        <div className="grid grid-3" style={{ marginBottom: '1.5rem' }}>
          <div className="stat-card">
            <div className="stat-label">{t('feedback.totalFeedback')}</div>
            <div className="stat-value">
              {summaries.reduce((sum, s) => sum + s.total_feedback, 0)}
            </div>
          </div>
          <div className="stat-card">
            <div className="stat-label">{t('feedback.positiveRate')}</div>
            <div className="stat-value text-success">
              {(() => {
                const total = summaries.reduce((sum, s) => sum + s.total_feedback, 0);
                const positive = summaries.reduce((sum, s) => sum + s.positive_count, 0);
                return total > 0 ? ((positive / total) * 100).toFixed(1) : '0';
              })()}%
            </div>
          </div>
          <div className="stat-card">
            <div className="stat-label">{t('feedback.agentsRated')}</div>
            <div className="stat-value">{summaries.length}</div>
          </div>
        </div>
      )}

      {showSubmit && (
        <div className="card" style={{ marginBottom: '1.5rem' }}>
          <div className="card-header">
            <h3>{t('feedback.newFeedback')}</h3>
          </div>
          <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            <input
              className="input"
              placeholder={t('feedback.agentId')}
              value={newFeedback.agent_id}
              onChange={(e) => setNewFeedback({ ...newFeedback, agent_id: e.target.value })}
            />
            <input
              className="input"
              placeholder={t('feedback.spanId')}
              value={newFeedback.span_id}
              onChange={(e) => setNewFeedback({ ...newFeedback, span_id: e.target.value })}
            />
            <select
              className="input"
              value={newFeedback.rating}
              onChange={(e) => setNewFeedback({ ...newFeedback, rating: Number(e.target.value) })}
            >
              <option value={1}>{t('feedback.thumbsUp')}</option>
              <option value={-1}>{t('feedback.thumbsDown')}</option>
            </select>
            <textarea
              className="input"
              placeholder={t('feedback.commentPlaceholder')}
              value={newFeedback.comment}
              onChange={(e) => setNewFeedback({ ...newFeedback, comment: e.target.value })}
              rows={3}
              style={{ resize: 'vertical', fontFamily: 'inherit' }}
            />
            <button className="btn btn-primary" onClick={handleSubmit}>
              {t('feedback.submit')}
            </button>
          </div>
        </div>
      )}

      {isLoading && feedbacks.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>{t('feedback.loading')}</span>
        </div>
      )}

      <div className="grid grid-2">
        <div className="card">
          <div className="card-header">
            <h3>{t('feedback.recentFeedback', { count: feedbacks.length })}</h3>
          </div>
          <div className="card-body">
            {!isLoading && feedbacks.length === 0 ? (
              <p className="text-muted">{t('feedback.noFeedback')}</p>
            ) : (
              <div className="table-container">
                <table>
                  <thead>
                    <tr>
                      <th>{t('feedback.agent')}</th>
                      <th>{t('feedback.rating')}</th>
                      <th>{t('feedback.comment')}</th>
                      <th>{t('feedback.time')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {feedbacks.map((fb) => (
                      <tr key={fb.id}>
                        <td><code>{fb.agent_id}</code></td>
                        <td>
                          <span className={`badge badge-${fb.rating > 0 ? 'success' : 'danger'}`}>
                            {fb.rating > 0 ? t('feedback.positive') : t('feedback.negative')}
                          </span>
                        </td>
                        <td>{fb.comment || '-'}</td>
                        <td className="text-muted">{new Date(fb.created_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3>{t('feedback.agentSummary', { count: summaries.length })}</h3>
          </div>
          <div className="card-body">
            {summaries.length === 0 ? (
              <p className="text-muted">{t('feedback.noSummary')}</p>
            ) : (
              <div className="table-container">
                <table>
                  <thead>
                    <tr>
                      <th>{t('feedback.agent')}</th>
                      <th>{t('feedback.total')}</th>
                      <th>{t('feedback.positive')}</th>
                      <th>{t('feedback.negative')}</th>
                      <th>{t('feedback.rate')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {summaries.map((s) => (
                      <tr key={s.agent_id}>
                        <td><code>{s.agent_id}</code></td>
                        <td>{s.total_feedback}</td>
                        <td className="text-success">{s.positive_count}</td>
                        <td className="text-danger">{s.negative_count}</td>
                        <td>{(s.average_rating * 100).toFixed(0)}%</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default FeedbackPage;
