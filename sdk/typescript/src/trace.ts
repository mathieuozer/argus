import { ArgusClient, Span } from './client';

let defaultClient: ArgusClient | null = null;

/**
 * Initialize the default Argus client for decorator usage.
 */
export function init(config: {
  endpoint?: string;
  tenantId?: string;
  agentId?: string;
} = {}): ArgusClient {
  defaultClient = new ArgusClient(config);
  return defaultClient;
}

/**
 * Get the default client.
 */
export function getClient(): ArgusClient | null {
  return defaultClient;
}

/**
 * Close the default client.
 */
export async function shutdown(): Promise<void> {
  if (defaultClient) {
    await defaultClient.close();
    defaultClient = null;
  }
}

/**
 * Method decorator that traces function execution.
 *
 * Usage:
 *   class MyService {
 *     @trace()
 *     async doWork() { ... }
 *
 *     @trace('custom-name')
 *     doSync() { ... }
 *   }
 */
export function trace(name?: string) {
  return function (
    target: any,
    propertyKey: string,
    descriptor: PropertyDescriptor,
  ): PropertyDescriptor {
    const originalMethod = descriptor.value;
    const spanName = name ?? `${target.constructor.name}.${propertyKey}`;

    descriptor.value = function (...args: any[]) {
      if (!defaultClient) {
        return originalMethod.apply(this, args);
      }

      const span = defaultClient.startSpan(spanName);
      try {
        const result = originalMethod.apply(this, args);
        if (result instanceof Promise) {
          return result
            .then((res: any) => {
              span.end();
              return res;
            })
            .catch((err: Error) => {
              span.setError(err);
              span.end();
              throw err;
            });
        }
        span.end();
        return result;
      } catch (err: any) {
        span.setError(err);
        span.end();
        throw err;
      }
    };

    return descriptor;
  };
}

/**
 * Wraps an async function with tracing (for use without decorators).
 *
 * Usage:
 *   const tracedFn = traced('my-operation', async () => { ... });
 */
export function traced<T>(name: string, fn: () => Promise<T>): Promise<T> {
  if (!defaultClient) {
    return fn();
  }

  const span = defaultClient.startSpan(name);
  return fn()
    .then((result) => {
      span.end();
      return result;
    })
    .catch((err: Error) => {
      span.setError(err);
      span.end();
      throw err;
    });
}
