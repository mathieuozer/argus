import type { TraceSpan } from '../../types/trace';

interface TraceTimelineProps {
  rootSpan: TraceSpan;
  totalDuration: number;
  selectedSpanId: string | null;
  onSelectSpan: (span: TraceSpan) => void;
}

function flattenSpans(span: TraceSpan, depth: number = 0): { span: TraceSpan; depth: number }[] {
  const result = [{ span, depth }];
  for (const child of span.children) {
    result.push(...flattenSpans(child, depth + 1));
  }
  return result;
}

function TraceTimeline({ rootSpan, totalDuration, selectedSpanId, onSelectSpan }: TraceTimelineProps) {
  const flatSpans = flattenSpans(rootSpan);
  const rootStart = new Date(rootSpan.started_at).getTime();

  return (
    <div className="trace-timeline">
      <div className="trace-timeline-header">
        <span className="trace-timeline-label">Operation</span>
        <span className="trace-timeline-bar-header">
          <span>0ms</span>
          <span>{Math.round(totalDuration / 2)}ms</span>
          <span>{totalDuration}ms</span>
        </span>
      </div>
      {flatSpans.map(({ span, depth }) => {
        const spanStart = new Date(span.started_at).getTime();
        const offset = ((spanStart - rootStart) / totalDuration) * 100;
        const width = Math.max((span.duration_ms / totalDuration) * 100, 0.5);
        const hasError = span.error_code !== null;

        return (
          <div
            key={span.span_id}
            className={`trace-timeline-row ${selectedSpanId === span.span_id ? 'selected' : ''} ${hasError ? 'has-error' : ''}`}
            onClick={() => onSelectSpan(span)}
          >
            <span
              className="trace-timeline-op"
              style={{ paddingLeft: `${depth * 20 + 8}px` }}
              title={span.operation_name}
            >
              {hasError && (
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="var(--color-error)" strokeWidth="2">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="15" y1="9" x2="9" y2="15" />
                  <line x1="9" y1="9" x2="15" y2="15" />
                </svg>
              )}
              {span.operation_name}
            </span>
            <div className="trace-timeline-bar-container">
              <div
                className={`trace-timeline-bar ${hasError ? 'error' : ''}`}
                style={{
                  left: `${Math.min(offset, 99)}%`,
                  width: `${Math.min(width, 100 - offset)}%`,
                }}
              />
              <span className="trace-timeline-duration">{span.duration_ms}ms</span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export default TraceTimeline;
