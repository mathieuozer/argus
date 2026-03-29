import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useCatalogStore } from '../stores/catalogStore';
import SourceTable from '../components/catalog/SourceTable';
import LineageGraph from '../components/catalog/LineageGraph';
import SourceDetail from '../components/catalog/SourceDetail';
import ToolUsageChart from '../components/catalog/ToolUsageChart';
import type { CatalogSource } from '../types/catalog';

type Tab = 'sources' | 'lineage' | 'glossary' | 'search' | 'tools';

function CatalogPage() {
  const { t } = useTranslation();
  const {
    sources, lineage, tools, glossary, searchResults, stats,
    loading, error, fetchSources, fetchLineage, fetchTools,
    fetchGlossary, fetchStats, searchCatalog,
  } = useCatalogStore();
  const [selectedSource, setSelectedSource] = useState<CatalogSource | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('sources');
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    fetchSources();
    fetchLineage();
    fetchTools();
    fetchGlossary();
    fetchStats();
  }, [fetchSources, fetchLineage, fetchTools, fetchGlossary, fetchStats]);

  const handleSearch = () => {
    if (searchQuery.trim()) {
      searchCatalog(searchQuery.trim());
      setActiveTab('search');
    }
  };

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('catalog.title')}</h2>
          <p>{t('catalog.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <div style={{ display: 'flex', gap: 'var(--spacing-sm)' }}>
            <input
              type="text"
              className="input"
              placeholder={t('catalog.searchPlaceholder')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              style={{ width: 250 }}
            />
            <button className="btn btn-primary" onClick={handleSearch}>{t('common.search')}</button>
          </div>
          <button className="btn" onClick={() => { fetchSources(); fetchLineage(); fetchStats(); }} disabled={loading}>{t('common.refresh')}</button>
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {/* Stats overview */}
      {stats && (
        <div className="grid grid-auto" style={{ marginBottom: 'var(--spacing-lg)' }}>
          <div className="card">
            <div className="stat-label">{t('catalog.totalSources')}</div>
            <div className="stat-value">{stats.total_sources}</div>
          </div>
          <div className="card">
            <div className="stat-label">{t('catalog.avgQuality')}</div>
            <div className="stat-value">{stats.avg_quality_score.toFixed(1)}%</div>
          </div>
          <div className="card">
            <div className="stat-label">{t('catalog.piiSources')}</div>
            <div className="stat-value">{stats.pii_sources}</div>
          </div>
          <div className="card">
            <div className="stat-label">{t('catalog.lineageEdges')}</div>
            <div className="stat-value">{stats.total_lineage_edges}</div>
          </div>
          <div className="card">
            <div className="stat-label">{t('catalog.glossaryTerms')}</div>
            <div className="stat-value">{stats.total_glossary_terms}</div>
          </div>
        </div>
      )}

      <div className="tab-bar">
        <button className={`tab ${activeTab === 'sources' ? 'active' : ''}`} onClick={() => setActiveTab('sources')}>
          {t('catalog.sources', { count: sources.length })}
        </button>
        <button className={`tab ${activeTab === 'lineage' ? 'active' : ''}`} onClick={() => setActiveTab('lineage')}>
          {t('catalog.dataLineage')}
        </button>
        <button className={`tab ${activeTab === 'glossary' ? 'active' : ''}`} onClick={() => setActiveTab('glossary')}>
          {t('catalog.glossary')} ({glossary.length})
        </button>
        <button className={`tab ${activeTab === 'search' ? 'active' : ''}`} onClick={() => setActiveTab('search')}>
          {t('catalog.searchResults')} {searchResults.length > 0 && `(${searchResults.length})`}
        </button>
        <button className={`tab ${activeTab === 'tools' ? 'active' : ''}`} onClick={() => setActiveTab('tools')}>
          {t('catalog.toolUsage')}
        </button>
      </div>

      {loading && sources.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>{t('catalog.discoveringSources')}</span>
        </div>
      )}

      {activeTab === 'sources' && (
        <div className="catalog-layout">
          <div className="catalog-main">
            <SourceTable sources={sources} onSelect={setSelectedSource} />
          </div>
          {selectedSource && (
            <div className="catalog-sidebar">
              <SourceDetail source={selectedSource} onClose={() => setSelectedSource(null)} />
            </div>
          )}
        </div>
      )}

      {activeTab === 'lineage' && lineage && (
        <div className="card">
          <LineageGraph data={lineage} />
        </div>
      )}

      {activeTab === 'glossary' && (
        <div>
          <h3>{t('catalog.businessGlossary')}</h3>
          {glossary.length === 0 ? (
            <p style={{ opacity: 0.7 }}>{t('catalog.noGlossaryTerms')}</p>
          ) : (
            <div className="grid grid-auto">
              {glossary.map((term) => (
                <div key={term.id} className="card">
                  <h4>{term.term}</h4>
                  <p style={{ fontSize: '0.9em', opacity: 0.85 }}>{term.definition}</p>
                  <div style={{ marginTop: 'var(--spacing-sm)', display: 'flex', gap: 'var(--spacing-xs)', flexWrap: 'wrap' }}>
                    <span className="badge badge-info">{term.domain}</span>
                    {term.related_terms?.map((rt) => (
                      <span key={rt} className="badge">{rt}</span>
                    ))}
                  </div>
                  <div style={{ marginTop: 'var(--spacing-xs)', fontSize: '0.85em', opacity: 0.6 }}>
                    Owner: {term.owner} | {term.linked_assets?.length || 0} linked assets
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {activeTab === 'search' && (
        <div>
          <h3>{t('catalog.searchResults')} ({searchResults.length})</h3>
          {searchResults.length === 0 ? (
            <p style={{ opacity: 0.7 }}>{t('catalog.noSearchResults')}</p>
          ) : (
            <table className="table">
              <thead>
                <tr>
                  <th>{t('common.name')}</th>
                  <th>{t('common.type')}</th>
                  <th>{t('catalog.domain')}</th>
                  <th>{t('catalog.classification')}</th>
                  <th>{t('catalog.relevance')}</th>
                  <th>{t('catalog.matchField')}</th>
                </tr>
              </thead>
              <tbody>
                {searchResults.map((r) => (
                  <tr key={r.id}>
                    <td><strong>{r.name}</strong><br /><span style={{ opacity: 0.7, fontSize: '0.85em' }}>{r.description}</span></td>
                    <td><span className="badge">{r.type}</span></td>
                    <td>{r.domain}</td>
                    <td><span className={`badge ${r.classification === 'restricted' ? 'badge-error' : r.classification === 'confidential' ? 'badge-warning' : 'badge-info'}`}>{r.classification}</span></td>
                    <td>{(r.relevance * 100).toFixed(0)}%</td>
                    <td><span className="badge">{r.match_field}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {activeTab === 'tools' && (
        <ToolUsageChart data={tools} />
      )}
    </div>
  );
}

export default CatalogPage;
