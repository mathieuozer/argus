import { create } from 'zustand';
import type { Agent } from '../types/agent';
import apiClient from '../utils/apiClient';

interface AgentState {
  agents: Agent[];
  loading: boolean;
  error: string | null;
  fetchAgents: () => Promise<void>;
}

export const useAgentStore = create<AgentState>((set) => ({
  agents: [],
  loading: false,
  error: null,
  fetchAgents: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<Agent[]>('/agents');
      set({ agents: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
}));
