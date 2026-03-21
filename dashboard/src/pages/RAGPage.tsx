import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useRAGStore } from '../stores/ragStore';

function RAGPage() {
  const { t } = useTranslation();
  const { retrievals, sources, quality, isLoading, error, fetchRetrievals, fetchSources, fetchQuality } = useRAGStore();

  useEffect(() => {
    fetchRetrievals();
    fetchSources();
    fetchQuality();
  }, [fetchRetrievals, fetchSources, fetchQuality]);

  return (
    <div className="page">
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('rag.title')}</h2>
          <p>{t('rag.subtitle')}</p>
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

      {quality.length > 0 && (
        <div className="grid grid-3" style={{ marginBottom: '1.5rem' }}>
          <div className="stat-card">
            <div className="stat-label">{t('rag.avgRelevance')}</div>
            <div className="stat-value">{(quality[0].avg_relevance * 100).toFixed(1)}%</div>
          </div>
          <div className="stat-card">
            <div className="stat-label">{t('rag.avgLatency')}</div>
            <div className="stat-value">{quality[0].avg_latency_ms.toFixed(0)}ms</div>
          </div>
          <div className="stat-card">
            <div className="stat-label">{t('rag.totalQueries')}</div>
            <div className="stat-value">{quality[0].total_queries}</div>
          </div>
        </div>
      )}

      <div className="grid grid-2">
        <div className="card">
          <div className="card-header">
            <h3>{t('rag.recentRetrievals', { count: retrievals.length })}</h3>
          </div>
          <div className="card-body">
            {isLoading ? (
              <div className="loading-container">
                <div className="loading-spinner" />
                <span>{t('rag.loading')}</span>
              </div>
            ) : retrievals.length === 0 ? (
              <p className="text-muted">{t('rag.noRetrievals')}</p>
            ) : (
              <div className="table-container">
                <table>
                  <thead>
                    <tr>
                      <th>{t('rag.query')}</th>
                      <th>{t('rag.agent')}</th>
                      <th>{t('rag.chunks')}</th>
                      <th>{t('rag.relevance')}</th>
                      <th>{t('rag.latency')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {retrievals.map((r) => (
                      <tr key={r.id}>
                        <td>{r.query}</td>
                        <td><code>{r.agent_id}</code></td>
                        <td>{r.num_chunks}</td>
                        <td>{(r.avg_relevance * 100).toFixed(0)}%</td>
                        <td>{r.latency_ms}ms</td>
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
            <h3>{t('rag.sources', { count: sources.length })}</h3>
          </div>
          <div className="card-body">
            {sources.length === 0 ? (
              <p className="text-muted">{t('rag.noSources')}</p>
            ) : (
              <div className="table-container">
                <table>
                  <thead>
                    <tr>
                      <th>{t('common.name')}</th>
                      <th>{t('common.type')}</th>
                      <th>{t('rag.chunks')}</th>
                      <th>{t('rag.usage')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sources.map((s) => (
                      <tr key={s.id}>
                        <td>{s.name}</td>
                        <td><span className="badge">{s.type}</span></td>
                        <td>{s.total_chunks}</td>
                        <td>{s.usage_count}</td>
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

export default RAGPage;
