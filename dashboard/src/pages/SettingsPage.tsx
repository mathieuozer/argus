function SettingsPage() {
  return (
    <div>
      <h2 style={{ marginBottom: '16px' }}>Settings</h2>
      <div
        style={{
          padding: '20px',
          backgroundColor: 'var(--color-surface)',
          borderRadius: '8px',
          border: '1px solid var(--color-border)',
        }}
      >
        <h3 style={{ marginBottom: '12px', fontSize: '16px' }}>Tenant Configuration</h3>
        <p style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}>
          Tenant settings, policy editor, and RBAC configuration will be available here.
        </p>
      </div>
    </div>
  );
}

export default SettingsPage;
