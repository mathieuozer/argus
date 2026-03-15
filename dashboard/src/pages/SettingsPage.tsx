import { useState } from 'react';

const complianceProfiles = [
  {
    id: 'gov-tr',
    label: 'Turkey Government (gov-tr)',
    description: 'KVKK compliance, Tier 3 default classification, storage restricted to TR regions, 5-year audit retention. Air-gap capable.',
  },
  {
    id: 'eu-gdpr',
    label: 'EU GDPR (eu-gdpr)',
    description: 'GDPR patterns for PII scrubbing, EU-only storage regions, configurable audit retention with right-to-erasure support via cryptographic key destruction.',
  },
  {
    id: 'fedramp-moderate',
    label: 'US FedRAMP Moderate (fedramp-moderate)',
    description: 'US Gov region storage, FIPS 140-2 crypto modules, continuous monitoring reports, POA&M tracking for findings.',
  },
];

const isolationTiers = [
  {
    id: 'A',
    label: 'Tier A - Shared',
    description: 'Logical isolation via Row-Level Security. Suitable for SMB and low-sensitivity workloads.',
  },
  {
    id: 'B',
    label: 'Tier B - Dedicated Namespace',
    description: 'Dedicated DB schema and NATS namespace. For regulated enterprise deployments.',
  },
  {
    id: 'C',
    label: 'Tier C - Dedicated Deployment',
    description: 'Separate cluster, separate DB instance. For government, defense, and classified workloads. Fully air-gap capable.',
  },
];

function SettingsPage() {
  const [selectedProfile, setSelectedProfile] = useState('eu-gdpr');
  const [piiScrubEnabled, setPiiScrubEnabled] = useState(true);
  const [selectedTier, setSelectedTier] = useState('A');

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>Settings</h2>
          <p>Configure tenant policies, compliance, and platform settings</p>
        </div>
      </div>

      {/* Tenant Configuration */}
      <div className="section">
        <div className="card mb-6">
          <div className="card-header">
            <div>
              <div className="card-title">Tenant Configuration</div>
              <div className="card-description">
                Current tenant identity and isolation settings
              </div>
            </div>
          </div>
          <div className="detail-row">
            <span className="detail-label">Tenant ID</span>
            <span className="detail-value text-mono">default</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Environment</span>
            <span className="detail-value">
              <span className="badge badge-info">development</span>
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Tenant Enforcement</span>
            <span className="detail-value">
              <span className="badge badge-success">strict</span>
            </span>
          </div>

          <div style={{ marginTop: 'var(--space-5)' }}>
            <label className="input-label">Isolation Tier</label>
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
                    <div className="font-medium">{tier.label}</div>
                    <div className="text-sm text-muted" style={{ marginTop: '2px' }}>
                      {tier.description}
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
              <div className="card-title">RBAC Policies</div>
              <div className="card-description">
                Role-based access control managed by the control-plane via OPA policy engine
              </div>
            </div>
          </div>
          <div className="detail-row">
            <span className="detail-label">Policy Engine</span>
            <span className="detail-value">Open Policy Agent (OPA)</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Authentication</span>
            <span className="detail-value">Bearer JWT with tenant claim</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Cross-Tenant Access</span>
            <span className="detail-value">
              <span className="badge badge-danger">denied</span>
            </span>
          </div>
          <div style={{ marginTop: 'var(--space-4)' }}>
            <p className="text-sm text-muted">
              RBAC policies are defined in the control-plane and enforced at every API endpoint.
              Every new endpoint must include a test asserting that cross-tenant access returns 403.
            </p>
          </div>
        </div>
      </div>

      {/* PII Scrubbing */}
      <div className="section">
        <div className="card mb-6">
          <div className="card-header">
            <div>
              <div className="card-title">PII Scrubbing</div>
              <div className="card-description">
                Automatic detection and removal of personally identifiable information from telemetry data
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
              Enable PII scrubbing on telemetry ingestion
            </label>
          </div>
          {piiScrubEnabled && (
            <div className="animate-fade-in">
              <div className="detail-row">
                <span className="detail-label">Method</span>
                <span className="detail-value">Regex + NER (Named Entity Recognition)</span>
              </div>
              <div className="detail-row">
                <span className="detail-label">Patterns</span>
                <span className="detail-value">
                  <div className="tag-list">
                    <span className="tag">email</span>
                    <span className="tag">IP address</span>
                    <span className="tag">national ID</span>
                    <span className="tag">IBAN</span>
                    <span className="tag">phone</span>
                    <span className="tag">name</span>
                  </div>
                </span>
              </div>
              <div className="detail-row">
                <span className="detail-label">Applied At</span>
                <span className="detail-value">Sidecar collection layer (before storage)</span>
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
              <div className="card-title">Compliance Profile</div>
              <div className="card-description">
                Select the regulatory framework that applies to this tenant
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
                  {profile.label}
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
                    {profile.label}
                  </div>
                  <div className="text-sm text-muted">
                    {profile.description}
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
