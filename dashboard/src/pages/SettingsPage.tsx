import { useState } from 'react';
import { useTranslation } from 'react-i18next';

function SettingsPage() {
  const { t } = useTranslation();

  const complianceProfiles = [
    {
      id: 'gov-tr',
      labelKey: 'settings.profiles.govTr',
      descriptionKey: 'settings.profiles.govTrDescription',
    },
    {
      id: 'eu-gdpr',
      labelKey: 'settings.profiles.euGdpr',
      descriptionKey: 'settings.profiles.euGdprDescription',
    },
    {
      id: 'fedramp-moderate',
      labelKey: 'settings.profiles.fedRamp',
      descriptionKey: 'settings.profiles.fedRampDescription',
    },
  ];

  const isolationTiers = [
    {
      id: 'A',
      labelKey: 'settings.tierALabel',
      descriptionKey: 'settings.tierADescription',
    },
    {
      id: 'B',
      labelKey: 'settings.tierBLabel',
      descriptionKey: 'settings.tierBDescription',
    },
    {
      id: 'C',
      labelKey: 'settings.tierCLabel',
      descriptionKey: 'settings.tierCDescription',
    },
  ];
  const [selectedProfile, setSelectedProfile] = useState('eu-gdpr');
  const [piiScrubEnabled, setPiiScrubEnabled] = useState(true);
  const [selectedTier, setSelectedTier] = useState('A');

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{t('settings.title')}</h2>
          <p>{t('settings.subtitle')}</p>
        </div>
      </div>

      {/* Tenant Configuration */}
      <div className="section">
        <div className="card mb-6">
          <div className="card-header">
            <div>
              <div className="card-title">{t('settings.tenantConfig')}</div>
              <div className="card-description">
                {t('settings.tenantConfigDescription')}
              </div>
            </div>
          </div>
          <div className="detail-row">
            <span className="detail-label">{t('settings.tenantId')}</span>
            <span className="detail-value text-mono">default</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">{t('settings.environment')}</span>
            <span className="detail-value">
              <span className="badge badge-info">development</span>
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">{t('settings.tenantEnforcement')}</span>
            <span className="detail-value">
              <span className="badge badge-success">strict</span>
            </span>
          </div>

          <div style={{ marginTop: 'var(--space-5)' }}>
            <label className="input-label">{t('settings.isolationTier')}</label>
            <div className="grid" style={{ gap: 'var(--space-3)', marginTop: 'var(--space-2)' }}>
              {isolationTiers.map((tier) => (
                <label
                  key={tier.id}
                  className="card card-clickable"
                  style={{
                    padding: 'var(--space-3) var(--space-4)',
                    borderColor: selectedTier === tier.id ? 'var(--color-primary)' : undefined,
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'flex-start',
                    gap: 'var(--space-3)',
                  }}
                >
                  <input
                    type="radio"
                    name="isolation-tier"
                    value={tier.id}
                    checked={selectedTier === tier.id}
                    onChange={(e) => setSelectedTier(e.target.value)}
                    style={{ marginTop: '4px', accentColor: 'var(--color-primary)' }}
                  />
                  <div>
                    <div className="font-medium">{t(tier.labelKey)}</div>
                    <div className="text-sm text-muted" style={{ marginTop: '2px' }}>
                      {t(tier.descriptionKey)}
                    </div>
                  </div>
                </label>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* RBAC Policies */}
      <div className="section">
        <div className="card mb-6">
          <div className="card-header">
            <div>
              <div className="card-title">{t('settings.rbacPolicies')}</div>
              <div className="card-description">
                {t('settings.rbacDescription')}
              </div>
            </div>
          </div>
          <div className="detail-row">
            <span className="detail-label">{t('settings.policyEngine')}</span>
            <span className="detail-value">{t('settings.policyEngineValue')}</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">{t('settings.authentication')}</span>
            <span className="detail-value">{t('settings.authenticationValue')}</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">{t('settings.crossTenantAccess')}</span>
            <span className="detail-value">
              <span className="badge badge-danger">{t('settings.crossTenantDenied')}</span>
            </span>
          </div>
          <div style={{ marginTop: 'var(--space-4)' }}>
            <p className="text-sm text-muted">
              {t('settings.rbacNote')}
            </p>
          </div>
        </div>
      </div>

      {/* PII Scrubbing */}
      <div className="section">
        <div className="card mb-6">
          <div className="card-header">
            <div>
              <div className="card-title">{t('settings.piiScrubbing')}</div>
              <div className="card-description">
                {t('settings.piiDescription')}
              </div>
            </div>
          </div>
          <div style={{ marginBottom: 'var(--space-4)' }}>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={piiScrubEnabled}
                onChange={(e) => setPiiScrubEnabled(e.target.checked)}
              />
              {t('settings.enablePii')}
            </label>
          </div>
          {piiScrubEnabled && (
            <div className="animate-fade-in">
              <div className="detail-row">
                <span className="detail-label">{t('settings.piiMethod')}</span>
                <span className="detail-value">{t('settings.piiMethodValue')}</span>
              </div>
              <div className="detail-row">
                <span className="detail-label">{t('settings.piiPatterns')}</span>
                <span className="detail-value">
                  <div className="tag-list">
                    <span className="tag">{t('settings.piiPatternLabels.email')}</span>
                    <span className="tag">{t('settings.piiPatternLabels.ipAddress')}</span>
                    <span className="tag">{t('settings.piiPatternLabels.nationalId')}</span>
                    <span className="tag">{t('settings.piiPatternLabels.iban')}</span>
                    <span className="tag">{t('settings.piiPatternLabels.phone')}</span>
                    <span className="tag">{t('settings.piiPatternLabels.name')}</span>
                  </div>
                </span>
              </div>
              <div className="detail-row">
                <span className="detail-label">{t('settings.piiAppliedAt')}</span>
                <span className="detail-value">{t('settings.piiAppliedAtValue')}</span>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Compliance Profile */}
      <div className="section">
        <div className="card">
          <div className="card-header">
            <div>
              <div className="card-title">{t('settings.complianceProfile')}</div>
              <div className="card-description">
                {t('settings.complianceDescription')}
              </div>
            </div>
          </div>
          <div className="form-group">
            <select
              className="select"
              value={selectedProfile}
              onChange={(e) => setSelectedProfile(e.target.value)}
            >
              {complianceProfiles.map((profile) => (
                <option key={profile.id} value={profile.id}>
                  {t(profile.labelKey)}
                </option>
              ))}
            </select>
          </div>
          {complianceProfiles
            .filter((p) => p.id === selectedProfile)
            .map((profile) => (
              <div key={profile.id} className="animate-fade-in" style={{ marginTop: 'var(--space-3)' }}>
                <div
                  style={{
                    padding: 'var(--space-3) var(--space-4)',
                    backgroundColor: 'var(--color-primary-muted)',
                    borderRadius: 'var(--radius-md)',
                    border: '1px solid rgba(99, 102, 241, 0.2)',
                  }}
                >
                  <div className="font-medium text-primary" style={{ marginBottom: 'var(--space-1)' }}>
                    {t(profile.labelKey)}
                  </div>
                  <div className="text-sm text-muted">
                    {t(profile.descriptionKey)}
                  </div>
                </div>
              </div>
            ))}
        </div>
      </div>
    </div>
  );
}

export default SettingsPage;
