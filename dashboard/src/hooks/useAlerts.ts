import { useEffect } from 'react';
import { useAlertStore } from '../stores/alertStore';

export function useAlerts() {
  const store = useAlertStore();

  useEffect(() => {
    store.fetchAlerts();
  }, [store.fetchAlerts]);

  return store;
}
