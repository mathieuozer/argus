import { create } from 'zustand';
import type { Task } from '../types/task';
import apiClient from '../utils/apiClient';

interface TaskState {
  tasks: Task[];
  loading: boolean;
  error: string | null;
  fetchTasks: () => Promise<void>;
  submitTask: (agentId: string, inputHash: string) => Promise<void>;
}

export const useTaskStore = create<TaskState>((set) => ({
  tasks: [],
  loading: false,
  error: null,
  fetchTasks: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<Task[]>('/tasks');
      set({ tasks: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  submitTask: async (agentId: string, inputHash: string) => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.post<Task>('/tasks', {
        agent_id: agentId,
        input_hash: inputHash,
      });
      set((state) => ({
        tasks: [...state.tasks, response.data],
        loading: false,
      }));
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
}));
