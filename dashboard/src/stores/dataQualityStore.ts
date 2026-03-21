import { create } from 'zustand';
import type { DQRule, DQScore, DQViolation, DriftPoint } from '../types/dataquality';
import apiClient from '../utils/apiClient';

interface DataQualityState {
  rules: DQRule[];
  scores: DQScore[];
  violations: DQViolation[];
  drift: DriftPoint[];
  loading: boolean;
  error: string | null;
  fetchRules: () => Promise<void>;
  fetchScores: (agentId?: string) => Promise<void>;
  fetchViolations: () => Promise<void>;
  fetchDrift: (agentId: string) => Promise<void>;
  createRule: (rule: Omit<DQRule, 'id' | 'created_at' | 'updated_at'>) => Promise<void>;
  deleteRule: (id: string) => Promise<void>;
}

export const useDataQualityStore = create<DataQualityState>((set, get) => ({
  rules: [],
  scores: [],
  violations: [],
  drift: [],
  loading: false,
  error: null,
  fetchRules: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<DQRule[]>('/data-quality/rules');
      set({ rules: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchScores: async (agentId?: string) => {
    set({ loading: true, error: null });
    try {
      const path = agentId ? `/data-quality/scores/${agentId}` : '/data-quality/scores';
      const response = await apiClient.get<DQScore[]>(path);
      set({ scores: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchViolations: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<DQViolation[]>('/data-quality/violations');
      set({ violations: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchDrift: async (agentId: string) => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<DriftPoint[]>(`/data-quality/drift/${agentId}`);
      set({ drift: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  createRule: async (rule) => {
    try {
      await apiClient.post('/data-quality/rules', rule);
      await get().fetchRules();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  deleteRule: async (id: string) => {
    try {
      await apiClient.del(`/data-quality/rules/${id}`);
      await get().fetchRules();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
