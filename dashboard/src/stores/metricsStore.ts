import { create } from 'zustand';
import type { DashboardMetrics } from '../types/metrics';
import apiClient from '../utils/apiClient';

interface MetricsState {
  metrics: DashboardMetrics | null;
  loading: boolean;
  error: string | null;
  fetchMetrics: () => Promise<void>;
}

export const useMetricsStore = create<MetricsState>((set) => ({
  metrics: null,
  loading: false,
  error: null,
  fetchMetrics: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<DashboardMetrics>('/metrics');
      set({ metrics: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
}));
