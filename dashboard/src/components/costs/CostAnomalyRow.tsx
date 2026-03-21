import type { CostAnomaly } from '../../types/cost';

interface CostAnomalyRowProps {
  anomaly: CostAnomaly;
}

function CostAnomalyRow({ anomaly }: CostAnomalyRowProps) {
  return (
    <tr>
      <td className="mono">{anomaly.agent_id}</td>
      <td>${anomaly.expected_usd.toFixed(2)}</td>
      <td className="text-error">${anomaly.actual_usd.toFixed(2)}</td>
      <td>
        <span className={`badge ${anomaly.ratio > 3 ? 'badge-error' : 'badge-warning'}`}>
          {anomaly.ratio.toFixed(1)}x
        </span>
      </td>
      <td className="text-muted">{new Date(anomaly.detected_at).toLocaleString()}</td>
      <td>
        <span className={`badge ${anomaly.status === 'open' ? 'badge-error' : anomaly.status === 'acknowledged' ? 'badge-warning' : 'badge-success'}`}>
          {anomaly.status}
        </span>
      </td>
    </tr>
  );
}

export default CostAnomalyRow;
