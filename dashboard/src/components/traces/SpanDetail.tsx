import { useTranslation } from 'react-i18next';
import type { TraceSpan } from '../../types/trace';

interface SpanDetailProps {
  span: TraceSpan;
}

function SpanDetail({ span }: SpanDetailProps) {
  const { t } = useTranslation();

  return (
    <div className="span-detail">
      <div className="span-detail-header">
        <h4>{span.operation_name}</h4>
        {span.error_code && (
          <span className="badge badge-error">{span.error_code}</span>
        )}
      </div>

      <div className="span-detail-section">
        <h5>{t('traceDetail.timing')}</h5>
        <div className="span-detail-grid">
          <span className="span-detail-label">{t('traceDetail.spanId')}</span>
          <span className="span-detail-value mono">{span.span_id}</span>
          <span className="span-detail-label">{t('traceDetail.duration')}</span>
          <span className="span-detail-value">{span.duration_ms}ms</span>
          <span className="span-detail-label">{t('traceDetail.started')}</span>
          <span className="span-detail-value">{new Date(span.started_at).toLocaleString()}</span>
          {span.agent_id && (
            <>
              <span className="span-detail-label">{t('traceDetail.agent')}</span>
              <span className="span-detail-value">{span.agent_id}</span>
            </>
          )}
        </div>
      </div>

      {Object.keys(span.attributes).length > 0 && (
        <div className="span-detail-section">
          <h5>{t('traceDetail.attributes')}</h5>
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
          <h5>{t('traceDetail.children')}</h5>
          <span className="span-detail-value">{t('traceDetail.childSpans', { count: span.children.length })}</span>
        </div>
      )}
    </div>
  );
}

export default SpanDetail;
