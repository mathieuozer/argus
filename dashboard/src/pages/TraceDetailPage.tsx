import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTraceStore } from '../stores/traceStore';
import TraceTimeline from '../components/traces/TraceTimeline';
import SpanDetail from '../components/traces/SpanDetail';
import type { TraceSpan } from '../types/trace';

function TraceDetailPage() {
  const { traceId } = useParams<{ traceId: string }>();
  const navigate = useNavigate();
  const { selectedTrace, loading, error, fetchTraceDetail } = useTraceStore();
  const [selectedSpan, setSelectedSpan] = useState<TraceSpan | null>(null);

  useEffect(() => {
    if (traceId) {
      fetchTraceDetail(traceId);
    }
  }, [traceId, fetchTraceDetail]);

  if (loading) {
    return (
      <div className="loading-container">
        <div className="loading-spinner" />
        <span>Loading trace...</span>
      </div>
    );
  }

  if (error) {
    return <div className="error-banner">{error}</div>;
  }

  if (!selectedTrace) {
    return <div className="empty-state"><h3>Trace not found</h3></div>;
  }

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <button className="btn btn-secondary" onClick={() => navigate('/traces')}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <polyline points="15 18 9 12 15 6" />
            </svg>
            Back
          </button>
          <div>
            <h2>Trace {selectedTrace.trace_id.slice(0, 12)}...</h2>
            <div className="page-header-meta">
              <span>{selectedTrace.total_spans} spans</span>
              <span>{selectedTrace.total_duration_ms}ms total</span>
              {selectedTrace.has_errors && <span className="badge badge-error">Has Errors</span>}
            </div>
          </div>
        </div>
      </div>

      <div className="trace-detail-layout">
        <div className="trace-detail-main">
          {selectedTrace.root_span && (
            <TraceTimeline
              rootSpan={selectedTrace.root_span}
              totalDuration={selectedTrace.total_duration_ms}
              selectedSpanId={selectedSpan?.span_id || null}
              onSelectSpan={setSelectedSpan}
            />
          )}
        </div>

        {selectedSpan && (
          <div className="trace-detail-sidebar">
            <SpanDetail span={selectedSpan} />
          </div>
        )}
      </div>
    </div>
  );
}

export default TraceDetailPage;
