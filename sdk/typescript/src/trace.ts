import { ArgusClient, Span } from './client';

let defaultClient: ArgusClient | null = null;

export function init(endpoint: string = 'localhost:8080', tenantId: string = ''): void {
  defaultClient = new ArgusClient({ endpoint, tenantId });
}

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
              span.setAttribute('error', err.name);
              span.end();
              throw err;
            });
        }
        span.end();
        return result;
      } catch (err: any) {
        span.setAttribute('error', err.name);
        span.end();
        throw err;
      }
    };

    return descriptor;
  };
}
