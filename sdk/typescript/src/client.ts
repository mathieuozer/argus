import { ArgusConfig, SpanOptions } from './types';

export class ArgusClient {
  private endpoint: string;
  private tenantId: string;

  constructor(config: Partial<ArgusConfig> = {}) {
    this.endpoint = config.endpoint ?? 'localhost:8080';
    this.tenantId = config.tenantId ?? '';
  }

  startSpan(name: string): Span {
    return new Span(name, this);
  }

  async emitEvent(eventType: string, payload?: Record<string, string>): Promise<void> {
    // Stub: in production, send event to Argus platform
  }

  async close(): Promise<void> {
    // Stub: flush pending data
  }
}

export class Span {
  private name: string;
  private startedAt: number;
  private attributes: Record<string, string> = {};
  private client: ArgusClient;

  constructor(name: string, client: ArgusClient) {
    this.name = name;
    this.startedAt = Date.now();
    this.client = client;
  }

  setAttribute(key: string, value: string): void {
    this.attributes[key] = value;
  }

  end(): void {
    // Stub: in production, send span to Argus platform
  }
}
