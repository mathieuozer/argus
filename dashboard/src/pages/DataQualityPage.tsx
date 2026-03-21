import { useEffect, useState } from 'react';
import { useDataQualityStore } from '../stores/dataQualityStore';
import QualityScoreCard from '../components/dataquality/QualityScoreCard';
import ViolationList from '../components/dataquality/ViolationList';
import DriftChart from '../components/dataquality/DriftChart';
import RuleEditor from '../components/dataquality/RuleEditor';

function DataQualityPage() {
  const { scores, rules, violations, drift, loading, error, fetchScores, fetchRules, fetchViolations, fetchDrift, createRule, deleteRule } = useDataQualityStore();
  const [showEditor, setShowEditor] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);

  useEffect(() => {
    fetchScores();
    fetchRules();
    fetchViolations();
  }, [fetchScores, fetchRules, fetchViolations]);

  useEffect(() => {
    if (selectedAgent) {
      fetchDrift(selectedAgent);
    }
  }, [selectedAgent, fetchDrift]);

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <h2>Data Quality</h2>
          <p>Validate agent outputs, track quality scores, detect data drift</p>
        </div>
        <div className="page-header-actions">
          <button className="btn btn-primary" onClick={() => setShowEditor(true)}>Add Rule</button>
          <button className="btn" onClick={() => { fetchScores(); fetchRules(); fetchViolations(); }} disabled={loading}>Refresh</button>
        </div>
      </div>

      {error && <div className="error-banner">{error}</div>}

      {showEditor && (
        <div className="card" style={{ marginBottom: 'var(--spacing-lg)' }}>
          <RuleEditor
            onSave={(rule) => { createRule(rule); setShowEditor(false); }}
            onCancel={() => setShowEditor(false)}
          />
        </div>
      )}

      {scores.length > 0 && (
        <div style={{ marginBottom: 'var(--spacing-lg)' }}>
          <h3>Quality Scores</h3>
          <div className="grid grid-auto">
            {scores.map((score) => (
              <div key={score.agent_id} onClick={() => setSelectedAgent(score.agent_id)} style={{ cursor: 'pointer' }}>
                <QualityScoreCard score={score} />
              </div>
            ))}
          </div>
        </div>
      )}

      {selectedAgent && drift.length > 0 && (
        <div style={{ marginBottom: 'var(--spacing-lg)' }}>
          <h3>Drift Analysis: {selectedAgent}</h3>
          <DriftChart data={drift} />
        </div>
      )}

      {rules.length > 0 && (
        <div style={{ marginBottom: 'var(--spacing-lg)' }}>
          <h3>Validation Rules ({rules.length})</h3>
          <table className="table">
            <thead>
              <tr>
                <th>Agent</th>
                <th>Name</th>
                <th>Type</th>
                <th>Target</th>
                <th>Severity</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={rule.id}>
                  <td className="mono">{rule.agent_id}</td>
                  <td>{rule.name}</td>
                  <td><span className="badge">{rule.type}</span></td>
                  <td>{rule.target}</td>
                  <td><span className={`badge ${rule.severity === 'critical' ? 'badge-error' : rule.severity === 'warning' ? 'badge-warning' : 'badge-info'}`}>{rule.severity}</span></td>
                  <td>{rule.enabled ? <span className="badge badge-success">Active</span> : <span className="badge">Disabled</span>}</td>
                  <td>
                    <button className="btn btn-sm btn-secondary" onClick={() => deleteRule(rule.id)}>Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div>
        <h3>Recent Violations</h3>
        <ViolationList violations={violations} />
      </div>

      {loading && scores.length === 0 && (
        <div className="loading-container">
          <div className="loading-spinner" />
          <span>Loading data quality metrics...</span>
        </div>
      )}
    </div>
  );
}

export default DataQualityPage;
