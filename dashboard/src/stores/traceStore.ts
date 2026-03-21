import { create } from 'zustand';
import type { TraceSummary, TraceDetail } from '../types/trace';
import apiClient from '../utils/apiClient';

interface TraceState {
  traces: TraceSummary[];
  selectedTrace: TraceDetail | null;
  loading: boolean;
  error: string | null;
  fetchTraces: (agentId?: string) => Promise<void>;
  fetchTraceDetail: (traceId: string) => Promise<void>;
}

export const useTraceStore = create<TraceState>((set) => ({
  traces: [],
  selectedTrace: null,
  loading: false,
  error: null,
  fetchTraces: async (agentId?: string) => {
    set({ loading: true, error: null });
    try {
      const path = agentId ? `/traces?agent_id=${agentId}` : '/traces';
      const response = await apiClient.get<TraceSummary[]>(path);
      set({ traces: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchTraceDetail: async (traceId: string) => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<TraceDetail>(`/traces/${traceId}`);
      set({ selectedTrace: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
}));
