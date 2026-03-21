import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useEvalStore } from '../stores/evalStore';
import type { TestSuite } from '../types/eval';

function EvalsPage() {
  const { t } = useTranslation();
  const { suites, runs, isLoading, error, fetchSuites, fetchRuns, runEval, createSuite } = useEvalStore();
  const [showCreate, setShowCreate] = useState(false);
  const [newSuite, setNewSuite] = useState({ name: '', description: '', agentId: '' });

  useEffect(() => {
    fetchSuites();
    fetchRuns();
  }, [fetchSuites, fetchRuns]);

  const handleCreate = async () => {
    await createSuite({
      ...newSuite,
      testCases: [
        { id: 'tc-1', name: 'Default Test', input: 'Hello', expectedOutput: 'Hi', criteria: {}, maxLatencyMs: 5000 },
      ],
    });
    setShowCreate(false);
    setNewSuite({ name: '', description: '', agentId: '' });
  };

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('evals.title')}</h2>
          <p>{t('evals.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <button className="btn btn-primary" onClick={() => setShowCreate(!showCreate)}>
            {t('evals.createSuite')}
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

      {showCreate && (
        <div className="card" style={{ marginBottom: '1.5rem' }}>
          <div className="card-header">
            <h3>{t('evals.newSuite')}</h3>
          </div>
          <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            <input
              className="input"
              placeholder={t('evals.suiteName')}
              value={newSuite.name}
              onChange={(e) => setNewSuite({ ...newSuite, name: e.target.value })}
            />
            <input
              className="input"
              placeholder={t('evals.suiteDescription')}
              value={newSuite.description}
              onChange={(e) => setNewSuite({ ...newSuite, description: e.target.value })}
            />
            <input
              className="input"
              placeholder={t('evals.agentId')}
              value={newSuite.agentId}
              onChange={(e) => setNewSuite({ ...newSuite, agentId: e.target.value })}
            />
            <button className="btn btn-primary" onClick={handleCreate}>
              {t('evals.create')}
            </button>
          </div>
        </div>
      )}

      {isLoading && suites.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>{t('evals.loading')}</span>
        </div>
      )}

      <div className="grid grid-2">
        <div className="card">
          <div className="card-header">
            <h3>{t('evals.testSuites', { count: suites.length })}</h3>
          </div>
          <div className="card-body">
            {!isLoading && suites.length === 0 ? (
              <p className="text-muted">{t('evals.noSuites')}</p>
            ) : (
              <div className="table-container">
                <table>
                  <thead>
                    <tr>
                      <th>{t('common.name')}</th>
                      <th>{t('evals.agent')}</th>
                      <th>{t('evals.cases')}</th>
                      <th>{t('common.actions')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {suites.map((suite: TestSuite) => (
                      <tr key={suite.id}>
                        <td>{suite.name}</td>
                        <td>
                          <code>{suite.agentId}</code>
                        </td>
                        <td>{suite.testCases?.length || 0}</td>
                        <td>
                          <button className="btn btn-sm" onClick={() => runEval(suite.id)}>
                            {t('evals.run')}
                          </button>
                        </td>
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
            <h3>{t('evals.recentRuns', { count: runs.length })}</h3>
          </div>
          <div className="card-body">
            {runs.length === 0 ? (
              <p className="text-muted">{t('evals.noRuns')}</p>
            ) : (
              <div className="table-container">
                <table>
                  <thead>
                    <tr>
                      <th>{t('evals.suite')}</th>
                      <th>{t('evals.score')}</th>
                      <th>{t('evals.passFail')}</th>
                      <th>{t('common.status')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {runs.map((run) => (
                      <tr key={run.id}>
                        <td>{run.suiteName}</td>
                        <td>{(run.score * 100).toFixed(0)}%</td>
                        <td>
                          <span className="text-success">{run.passedCases}</span>
                          {' / '}
                          <span className="text-danger">{run.failedCases}</span>
                        </td>
                        <td>
                          <span
                            className={`badge badge-${run.status === 'completed' ? 'success' : run.status === 'failed' ? 'danger' : 'warning'}`}
                          >
                            {run.status}
                          </span>
                        </td>
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

export default EvalsPage;
