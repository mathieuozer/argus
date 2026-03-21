import { create } from 'zustand';
import type { CatalogSource, LineageGraph, ToolUsage } from '../types/catalog';
import apiClient from '../utils/apiClient';

interface CatalogState {
  sources: CatalogSource[];
  lineage: LineageGraph | null;
  tools: ToolUsage[];
  loading: boolean;
  error: string | null;
  fetchSources: (agentId?: string) => Promise<void>;
  fetchLineage: () => Promise<void>;
  fetchTools: () => Promise<void>;
}

export const useCatalogStore = create<CatalogState>((set) => ({
  sources: [],
  lineage: null,
  tools: [],
  loading: false,
  error: null,
  fetchSources: async (agentId?: string) => {
    set({ loading: true, error: null });
    try {
      const path = agentId ? `/catalog/sources/${agentId}` : '/catalog/sources';
      const response = await apiClient.get<CatalogSource[]>(path);
      set({ sources: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchLineage: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<LineageGraph>('/catalog/lineage/graph');
      set({ lineage: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchTools: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<ToolUsage[]>('/catalog/tools');
      set({ tools: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
}));
