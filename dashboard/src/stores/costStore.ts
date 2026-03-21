import { create } from 'zustand';
import type { CostBreakdown, CostTrend, CostBudget, CostAnomaly } from '../types/cost';
import apiClient from '../utils/apiClient';

interface CostState {
  breakdown: CostBreakdown[];
  trends: CostTrend[];
  budgets: CostBudget[];
  anomalies: CostAnomaly[];
  loading: boolean;
  error: string | null;
  fetchBreakdown: (groupBy?: string) => Promise<void>;
  fetchTrends: (agentId?: string) => Promise<void>;
  fetchBudgets: () => Promise<void>;
  fetchAnomalies: () => Promise<void>;
  createBudget: (budget: Omit<CostBudget, 'id' | 'spent_usd' | 'created_at'>) => Promise<void>;
}

export const useCostStore = create<CostState>((set, get) => ({
  breakdown: [],
  trends: [],
  budgets: [],
  anomalies: [],
  loading: false,
  error: null,
  fetchBreakdown: async (groupBy = 'agent') => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<CostBreakdown[]>(`/costs/breakdown?group_by=${groupBy}`);
      set({ breakdown: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchTrends: async (agentId?: string) => {
    set({ loading: true, error: null });
    try {
      const path = agentId ? `/costs/trends?agent_id=${agentId}` : '/costs/trends';
      const response = await apiClient.get<CostTrend[]>(path);
      set({ trends: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchBudgets: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<CostBudget[]>('/costs/budgets');
      set({ budgets: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchAnomalies: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<CostAnomaly[]>('/costs/anomalies');
      set({ anomalies: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  createBudget: async (budget) => {
    try {
      await apiClient.post('/costs/budgets', budget);
      await get().fetchBudgets();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
