import { create } from 'zustand';
import type { Retrieval, RAGSource, QualityTrend } from '../types/rag';
import apiClient from '../utils/apiClient';

interface RAGState {
  retrievals: Retrieval[];
  sources: RAGSource[];
  quality: QualityTrend[];
  isLoading: boolean;
  error: string | null;
  fetchRetrievals: (agentId?: string) => Promise<void>;
  fetchSources: () => Promise<void>;
  fetchQuality: () => Promise<void>;
}

export const useRAGStore = create<RAGState>((set) => ({
  retrievals: [],
  sources: [],
  quality: [],
  isLoading: false,
  error: null,

  fetchRetrievals: async (agentId?: string) => {
    set({ isLoading: true, error: null });
    try {
      const url = agentId ? `/rag/retrievals?agent_id=${agentId}` : '/rag/retrievals';
      const response = await apiClient.get<Retrieval[]>(url);
      set({ retrievals: response.data, isLoading: false });
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
    }
  },

  fetchSources: async () => {
    try {
      const response = await apiClient.get<RAGSource[]>('/rag/sources');
      set({ sources: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },

  fetchQuality: async () => {
    try {
      const response = await apiClient.get<QualityTrend[]>('/rag/quality');
      set({ quality: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
