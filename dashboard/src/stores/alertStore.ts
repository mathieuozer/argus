import { create } from 'zustand';
import type { PredictiveAlert } from '../types/alert';
import apiClient from '../utils/apiClient';

interface AlertState {
  alerts: PredictiveAlert[];
  loading: boolean;
  error: string | null;
  fetchAlerts: () => Promise<void>;
}

export const useAlertStore = create<AlertState>((set) => ({
  alerts: [],
  loading: false,
  error: null,
  fetchAlerts: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<PredictiveAlert[]>('/alerts');
      set({ alerts: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
}));
