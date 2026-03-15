import { useEffect } from 'react';
import { useAgentStore } from '../stores/agentStore';

export function useAgents() {
  const store = useAgentStore();

  useEffect(() => {
    store.fetchAgents();
  }, [store.fetchAgents]);

  return store;
}
