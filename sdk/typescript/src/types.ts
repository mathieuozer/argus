export interface ArgusConfig {
  endpoint: string;
  tenantId: string;
  agentId: string;
  batchSize: number;
  flushIntervalMs: number;
}

export interface SpanOptions {
  name: string;
  attributes?: Record<string, string>;
}

export interface SpanData {
  spanId: string;
  traceId: string;
  operationName: string;
  startedAt: number;
  durationMs: number;
  attributes: Record<string, string>;
  errorCode?: string;
}

export interface EventData {
  eventType: string;
  payload: Record<string, string>;
  timestamp: number;
}
