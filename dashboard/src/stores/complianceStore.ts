import { create } from 'zustand';
import type { ComplianceReport } from '../types/compliance';
import apiClient from '../utils/apiClient';

interface ComplianceState {
  reports: ComplianceReport[];
  selectedReport: ComplianceReport | null;
  isLoading: boolean;
  error: string | null;
  fetchReports: () => Promise<void>;
  generateReport: (profileId: string, periodStart: string, periodEnd: string) => Promise<void>;
  fetchReport: (id: string) => Promise<void>;
}

export const useComplianceStore = create<ComplianceState>((set) => ({
  reports: [],
  selectedReport: null,
  isLoading: false,
  error: null,

  fetchReports: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await apiClient.get<ComplianceReport[]>('/compliance/reports');
      set({ reports: response.data, isLoading: false });
    } catch {
      set({ error: 'Failed to fetch reports', isLoading: false });
    }
  },

  generateReport: async (profileId, periodStart, periodEnd) => {
    set({ isLoading: true, error: null });
    try {
      await apiClient.post('/compliance/reports', {
        profile_id: profileId,
        period_start: periodStart,
        period_end: periodEnd,
      });
      const response = await apiClient.get<ComplianceReport[]>('/compliance/reports');
      set({ reports: response.data, isLoading: false });
    } catch {
      set({ error: 'Failed to generate report', isLoading: false });
    }
  },

  fetchReport: async (id) => {
    try {
      const response = await apiClient.get<ComplianceReport>(`/compliance/reports/${id}`);
      set({ selectedReport: response.data });
    } catch {
      set({ error: 'Failed to fetch report' });
    }
  },
}));
