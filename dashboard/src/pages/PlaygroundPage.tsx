import { useState } from 'react';
import { useTranslation } from 'react-i18next';

function PlaygroundPage() {
  const { t } = useTranslation();
  const [agentId, setAgentId] = useState('');
  const [prompt, setPrompt] = useState('');
  const [response, setResponse] = useState('');
  const [isRunning, setIsRunning] = useState(false);

  const handleRun = async () => {
    setIsRunning(true);
    setResponse('');
    // Simulate response
    setTimeout(() => {
      setResponse(`[Agent: ${agentId || 'default'}] Response to: "${prompt}"\n\nThis is a simulated response from the playground. In production, this would submit a task via POST /api/v1/tasks with playground: true and poll for completion.`);
      setIsRunning(false);
    }, 1500);
  };

  return (
    <div className="page">
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('playground.title')}</h2>
          <p>{t('playground.subtitle')}</p>
        </div>
      </div>

      <div className="grid grid-2">
        <div className="card">
          <div className="card-header">
            <h3>{t('playground.input')}</h3>
          </div>
          <div className="card-body" style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            <input
              className="input"
              placeholder={t('playground.agentIdPlaceholder')}
              value={agentId}
              onChange={(e) => setAgentId(e.target.value)}
            />
            <textarea
              className="input"
              placeholder={t('playground.promptPlaceholder')}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              rows={8}
              style={{ resize: 'vertical', fontFamily: 'inherit' }}
            />
            <button className="btn btn-primary" onClick={handleRun} disabled={isRunning || !prompt}>
              {isRunning ? t('playground.running') : t('playground.run')}
            </button>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3>{t('playground.response')}</h3>
          </div>
          <div className="card-body">
            {response ? (
              <pre style={{ whiteSpace: 'pre-wrap', fontFamily: 'var(--font-mono)', fontSize: '0.8125rem', color: 'var(--color-text)' }}>
                {response}
              </pre>
            ) : (
              <p className="text-muted">
                {isRunning ? t('playground.processing') : t('playground.responseWillAppear')}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default PlaygroundPage;
