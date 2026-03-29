import { create } from 'zustand';
import type { DQRule, DQScore, DQViolation, DriftPoint, DataContract, DataProfile, QualityTrend, QualityIncident, Anomaly, DQSummary } from '../types/dataquality';
import apiClient from '../utils/apiClient';

interface DataQualityState {
  rules: DQRule[];
  scores: DQScore[];
  violations: DQViolation[];
  drift: DriftPoint[];
  contracts: DataContract[];
  profiles: DataProfile[];
  trend: QualityTrend | null;
  incidents: QualityIncident[];
  anomalies: Anomaly[];
  summary: DQSummary | null;
  loading: boolean;
  error: string | null;
  fetchRules: () => Promise<void>;
  fetchScores: (agentId?: string) => Promise<void>;
  fetchViolations: () => Promise<void>;
  fetchDrift: (agentId: string) => Promise<void>;
  fetchContracts: () => Promise<void>;
  fetchProfiles: (agentId?: string) => Promise<void>;
  fetchTrend: (agentId: string) => Promise<void>;
  fetchIncidents: () => Promise<void>;
  fetchAnomalies: (agentId?: string) => Promise<void>;
  fetchSummary: () => Promise<void>;
  createRule: (rule: Partial<DQRule>) => Promise<void>;
  deleteRule: (id: string) => Promise<void>;
  createContract: (contract: Partial<DataContract>) => Promise<void>;
  deleteContract: (id: string) => Promise<void>;
}

export const useDataQualityStore = create<DataQualityState>((set, get) => ({
  rules: [],
  scores: [],
  violations: [],
  drift: [],
  contracts: [],
  profiles: [],
  trend: null,
  incidents: [],
  anomalies: [],
  summary: null,
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
  fetchContracts: async () => {
    try {
      const response = await apiClient.get<DataContract[]>('/data-quality/contracts');
      set({ contracts: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchProfiles: async (agentId?: string) => {
    try {
      const path = agentId ? `/data-quality/profiles/${agentId}` : '/data-quality/profiles';
      const response = await apiClient.get<DataProfile[]>(path);
      set({ profiles: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchTrend: async (agentId: string) => {
    try {
      const response = await apiClient.get<QualityTrend>(`/data-quality/trends/${agentId}`);
      set({ trend: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchIncidents: async () => {
    try {
      const response = await apiClient.get<QualityIncident[]>('/data-quality/incidents');
      set({ incidents: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchAnomalies: async (agentId?: string) => {
    try {
      const path = agentId ? `/data-quality/anomalies?agent_id=${agentId}` : '/data-quality/anomalies';
      const response = await apiClient.get<Anomaly[]>(path);
      set({ anomalies: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchSummary: async () => {
    try {
      const response = await apiClient.get<DQSummary>('/data-quality/summary');
      set({ summary: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
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
  createContract: async (contract) => {
    try {
      await apiClient.post('/data-quality/contracts', contract);
      await get().fetchContracts();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  deleteContract: async (id: string) => {
    try {
      await apiClient.del(`/data-quality/contracts/${id}`);
      await get().fetchContracts();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
