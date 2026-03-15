import { useEffect } from 'react';
import { useTaskStore } from '../stores/taskStore';

export function useTasks() {
  const store = useTaskStore();

  useEffect(() => {
    store.fetchTasks();
  }, [store.fetchTasks]);

  return store;
}
