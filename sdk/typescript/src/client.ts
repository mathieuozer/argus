import { ArgusConfig, SpanData, EventData } from './types';

let idCounter = 0;
function generateId(): string {
  return `${Date.now()}-${++idCounter}`;
}

export class ArgusClient {
  private endpoint: string;
  private tenantId: string;
  private agentId: string;
  private batchSize: number;
  private flushIntervalMs: number;

  private spans: SpanData[] = [];
  private events: EventData[] = [];
  private timer: ReturnType<typeof setInterval> | null = null;
  private closed = false;

  constructor(config: Partial<ArgusConfig> = {}) {
    this.endpoint = config.endpoint ?? 'http://localhost:8080';
    this.tenantId = config.tenantId ?? '';
    this.agentId = config.agentId ?? '';
    this.batchSize = config.batchSize ?? 100;
    this.flushIntervalMs = config.flushIntervalMs ?? 5000;

    this.timer = setInterval(() => this.flush(), this.flushIntervalMs);
  }

  startSpan(name: string, traceId?: string): Span {
    return new Span(name, this, traceId ?? generateId());
  }

  emitEvent(eventType: string, payload: Record<string, string> = {}): void {
    this.events.push({
      eventType,
      payload,
      timestamp: Date.now(),
    });
    if (this.events.length >= this.batchSize) {
      this.flush();
    }
  }

  /** @internal */
  _addSpan(data: SpanData): void {
    this.spans.push(data);
    if (this.spans.length >= this.batchSize) {
      this.flush();
    }
  }

  get pendingSpans(): number {
    return this.spans.length;
  }

  get pendingEvents(): number {
    return this.events.length;
  }

  async flush(): Promise<void> {
    const spans = this.spans.splice(0);
    const events = this.events.splice(0);

    const promises: Promise<void>[] = [];

    if (spans.length > 0) {
      promises.push(this.send('/api/v1/telemetry/spans', { spans }));
    }
    if (events.length > 0) {
      promises.push(this.send('/api/v1/telemetry/events', { events }));
    }

    await Promise.allSettled(promises);
  }

  private async send(path: string, data: Record<string, unknown>): Promise<void> {
    data.tenant_id = this.tenantId;
    data.agent_id = this.agentId;

    try {
      const response = await fetch(this.endpoint + path, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': this.tenantId,
        },
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        // Silently drop - don't block the agent
      }
    } catch {
      // Silently drop network errors
    }
  }

  async close(): Promise<void> {
    if (this.closed) return;
    this.closed = true;

    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }

    await this.flush();
  }
}

export class Span {
  readonly spanId: string;
  readonly traceId: string;
  readonly name: string;
  private startedAt: number;
  private attributes: Record<string, string> = {};
  private errorCode?: string;
  private client: ArgusClient;

  constructor(name: string, client: ArgusClient, traceId: string) {
    this.spanId = generateId();
    this.traceId = traceId;
    this.name = name;
    this.startedAt = Date.now();
    this.client = client;
  }

  setAttribute(key: string, value: string): this {
    this.attributes[key] = value;
    return this;
  }

  setError(error: Error): this {
    this.errorCode = error.name;
    this.attributes['error'] = error.message;
    return this;
  }

  child(name: string): Span {
    return new Span(name, this.client, this.traceId);
  }

  end(): void {
    const durationMs = Date.now() - this.startedAt;
    this.client._addSpan({
      spanId: this.spanId,
      traceId: this.traceId,
      operationName: this.name,
      startedAt: this.startedAt,
      durationMs,
      attributes: { ...this.attributes },
      errorCode: this.errorCode,
    });
  }
}
