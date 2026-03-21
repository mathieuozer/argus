import { create } from 'zustand';
import type { SLO, ErrorBudgetPoint } from '../types/slo';
import apiClient from '../utils/apiClient';

interface SLOState {
  slos: SLO[];
  errorBudget: ErrorBudgetPoint[];
  loading: boolean;
  error: string | null;
  fetchSLOs: () => Promise<void>;
  fetchErrorBudget: (sloId: string) => Promise<void>;
  createSLO: (slo: Omit<SLO, 'id' | 'current' | 'budget_remaining' | 'status' | 'created_at'>) => Promise<void>;
  deleteSLO: (id: string) => Promise<void>;
}

export const useSLOStore = create<SLOState>((set, get) => ({
  slos: [],
  errorBudget: [],
  loading: false,
  error: null,
  fetchSLOs: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<SLO[]>('/slos');
      set({ slos: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchErrorBudget: async (sloId: string) => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<ErrorBudgetPoint[]>(`/slos/${sloId}/budget`);
      set({ errorBudget: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  createSLO: async (slo) => {
    try {
      await apiClient.post('/slos', slo);
      await get().fetchSLOs();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  deleteSLO: async (id: string) => {
    try {
      await apiClient.del(`/slos/${id}`);
      await get().fetchSLOs();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
