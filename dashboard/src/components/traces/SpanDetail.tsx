import type { TraceSpan } from '../../types/trace';

interface SpanDetailProps {
  span: TraceSpan;
}

function SpanDetail({ span }: SpanDetailProps) {
  return (
    <div className="span-detail">
      <div className="span-detail-header">
        <h4>{span.operation_name}</h4>
        {span.error_code && (
          <span className="badge badge-error">{span.error_code}</span>
        )}
      </div>

      <div className="span-detail-section">
        <h5>Timing</h5>
        <div className="span-detail-grid">
          <span className="span-detail-label">Span ID</span>
          <span className="span-detail-value mono">{span.span_id}</span>
          <span className="span-detail-label">Duration</span>
          <span className="span-detail-value">{span.duration_ms}ms</span>
          <span className="span-detail-label">Started</span>
          <span className="span-detail-value">{new Date(span.started_at).toLocaleString()}</span>
          {span.agent_id && (
            <>
              <span className="span-detail-label">Agent</span>
              <span className="span-detail-value">{span.agent_id}</span>
            </>
          )}
        </div>
      </div>

      {Object.keys(span.attributes).length > 0 && (
        <div className="span-detail-section">
          <h5>Attributes</h5>
          <div className="span-detail-grid">
            {Object.entries(span.attributes).map(([key, value]) => (
              <div key={key} className="span-detail-attr">
                <span className="span-detail-label">{key}</span>
                <span className="span-detail-value mono">{value}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {span.children.length > 0 && (
        <div className="span-detail-section">
          <h5>Children</h5>
          <span className="span-detail-value">{span.children.length} child span(s)</span>
        </div>
      )}
    </div>
  );
}

export default SpanDetail;
