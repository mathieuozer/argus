export interface ArgusConfig {
  endpoint: string;
  tenantId: string;
}

export interface SpanOptions {
  name: string;
  attributes?: Record<string, string>;
}

export interface SpanData {
  spanId: string;
  name: string;
  startedAt: number;
  durationMs: number;
  attributes: Record<string, string>;
}
