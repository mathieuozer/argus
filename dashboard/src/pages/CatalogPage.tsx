import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useCatalogStore } from '../stores/catalogStore';
import SourceTable from '../components/catalog/SourceTable';
import LineageGraph from '../components/catalog/LineageGraph';
import SourceDetail from '../components/catalog/SourceDetail';
import ToolUsageChart from '../components/catalog/ToolUsageChart';
import type { CatalogSource } from '../types/catalog';

function CatalogPage() {
  const { t } = useTranslation();
  const { sources, lineage, tools, loading, error, fetchSources, fetchLineage, fetchTools } = useCatalogStore();
  const [selectedSource, setSelectedSource] = useState<CatalogSource | null>(null);
  const [activeTab, setActiveTab] = useState<'sources' | 'lineage' | 'tools'>('sources');

  useEffect(() => {
    fetchSources();
    fetchLineage();
    fetchTools();
  }, [fetchSources, fetchLineage, fetchTools]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('catalog.title')}</h2>
          <p>{t('catalog.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <button className="btn" onClick={() => { fetchSources(); fetchLineage(); fetchTools(); }} disabled={loading}>{t('common.refresh')}</button>
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}

      <div className="tab-bar">
        <button className={`tab ${activeTab === 'sources' ? 'active' : ''}`} onClick={() => setActiveTab('sources')}>
          {t('catalog.sources', { count: sources.length })}
        </button>
        <button className={`tab ${activeTab === 'lineage' ? 'active' : ''}`} onClick={() => setActiveTab('lineage')}>
          {t('catalog.dataLineage')}
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

      {activeTab === 'tools' && (
        <ToolUsageChart data={tools} />
      )}
    </div>
  );
}

export default CatalogPage;
