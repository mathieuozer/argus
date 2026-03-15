import { useEffect } from 'react';
import { useMetricsStore } from '../stores/metricsStore';

export function useMetrics() {
  const store = useMetricsStore();

  useEffect(() => {
    store.fetchMetrics();
  }, [store.fetchMetrics]);

  return store;
}
