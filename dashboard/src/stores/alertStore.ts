import { create } from 'zustand';
import type { PredictiveAlert, AlertStatus } from '../types/alert';
import apiClient from '../utils/apiClient';

interface AlertState {
  alerts: PredictiveAlert[];
  loading: boolean;
  error: string | null;
  fetchAlerts: () => Promise<void>;
  updateAlertStatus: (alertId: string, status: AlertStatus) => Promise<void>;
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
  updateAlertStatus: async (alertId: string, status: AlertStatus) => {
    try {
      await apiClient.patch<PredictiveAlert>(`/alerts/${alertId}`, { status });
      set((state) => ({
        alerts: state.alerts.map((a) =>
          a.id === alertId ? { ...a, status } : a
        ),
      }));
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
