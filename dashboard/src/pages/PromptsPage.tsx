import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { usePromptStore } from '../stores/promptStore';

function PromptsPage() {
  const { t } = useTranslation();
  const { prompts, isLoading, error, fetchPrompts, createPrompt } = usePromptStore();
  const [showCreate, setShowCreate] = useState(false);
  const [newPrompt, setNewPrompt] = useState({ name: '', description: '', agent_id: '' });

  useEffect(() => { fetchPrompts(); }, [fetchPrompts]);

  const handleCreate = async () => {
    await createPrompt(newPrompt);
    setShowCreate(false);
    setNewPrompt({ name: '', description: '', agent_id: '' });
  };

  return (
    <div className="page">
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('prompts.title')}</h2>
          <p>{t('prompts.subtitle')}</p>
        </div>
        <div className="page-header-actions">
          <button className="btn btn-primary" onClick={() => setShowCreate(!showCreate)}>
            {t('prompts.newPrompt')}
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
            <h3>{t('prompts.createPrompt')}</h3>
          </div>
          <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            <input
              className="input"
              placeholder={t('prompts.promptName')}
              value={newPrompt.name}
              onChange={(e) => setNewPrompt({ ...newPrompt, name: e.target.value })}
            />
            <input
              className="input"
              placeholder={t('prompts.description')}
              value={newPrompt.description}
              onChange={(e) => setNewPrompt({ ...newPrompt, description: e.target.value })}
            />
            <input
              className="input"
              placeholder={t('prompts.agentId')}
              value={newPrompt.agent_id}
              onChange={(e) => setNewPrompt({ ...newPrompt, agent_id: e.target.value })}
            />
            <button className="btn btn-primary" onClick={handleCreate}>
              {t('prompts.create')}
            </button>
          </div>
        </div>
      )}

      {isLoading && prompts.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>{t('prompts.loading')}</span>
        </div>
      )}

      <div className="card">
        <div className="card-header">
          <h3>{t('prompts.promptList', { count: prompts.length })}</h3>
        </div>
        <div className="card-body">
          {!isLoading && prompts.length === 0 ? (
            <p className="text-muted">{t('prompts.noPrompts')}</p>
          ) : (
            <div className="table-container">
              <table>
                <thead>
                  <tr>
                    <th>{t('common.name')}</th>
                    <th>{t('prompts.agent')}</th>
                    <th>{t('prompts.activeVersion')}</th>
                    <th>{t('prompts.updated')}</th>
                  </tr>
                </thead>
                <tbody>
                  {prompts.map((p) => (
                    <tr key={p.id}>
                      <td>{p.name}</td>
                      <td><code>{p.agent_id}</code></td>
                      <td>v{p.active_version}</td>
                      <td className="text-muted">{new Date(p.updated_at).toLocaleDateString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default PromptsPage;
