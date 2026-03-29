import { create } from 'zustand';
import type { CatalogSource, LineageGraph, GlossaryTerm, SearchResult, CatalogStats, ImpactAnalysis, ToolUsage } from '../types/catalog';
import apiClient from '../utils/apiClient';

interface CatalogState {
  sources: CatalogSource[];
  lineage: LineageGraph | null;
  tools: ToolUsage[];
  glossary: GlossaryTerm[];
  searchResults: SearchResult[];
  stats: CatalogStats | null;
  selectedSource: CatalogSource | null;
  impact: ImpactAnalysis | null;
  loading: boolean;
  error: string | null;
  fetchSources: (params?: { type?: string; domain?: string; classification?: string; search?: string }) => Promise<void>;
  fetchLineage: () => Promise<void>;
  fetchTools: () => Promise<void>;
  fetchGlossary: () => Promise<void>;
  fetchStats: () => Promise<void>;
  fetchSource: (id: string) => Promise<void>;
  fetchImpact: (id: string) => Promise<void>;
  searchCatalog: (query: string) => Promise<void>;
  createGlossaryTerm: (term: Partial<GlossaryTerm>) => Promise<void>;
  deleteGlossaryTerm: (id: string) => Promise<void>;
}

export const useCatalogStore = create<CatalogState>((set, get) => ({
  sources: [],
  lineage: null,
  tools: [],
  glossary: [],
  searchResults: [],
  stats: null,
  selectedSource: null,
  impact: null,
  loading: false,
  error: null,
  fetchSources: async (params) => {
    set({ loading: true, error: null });
    try {
      const query = new URLSearchParams();
      if (params?.type) query.set('type', params.type);
      if (params?.domain) query.set('domain', params.domain);
      if (params?.classification) query.set('classification', params.classification);
      if (params?.search) query.set('search', params.search);
      const qs = query.toString();
      const path = qs ? `/catalog/sources?${qs}` : '/catalog/sources';
      const response = await apiClient.get<CatalogSource[]>(path);
      set({ sources: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
  fetchLineage: async () => {
    set({ loading: true, error: null });
    try {
      const response = await apiClient.get<LineageGraph>('/catalog/graph');
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
  fetchGlossary: async () => {
    try {
      const response = await apiClient.get<GlossaryTerm[]>('/catalog/glossary');
      set({ glossary: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchStats: async () => {
    try {
      const response = await apiClient.get<CatalogStats>('/catalog/stats');
      set({ stats: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchSource: async (id: string) => {
    try {
      const response = await apiClient.get<CatalogSource>(`/catalog/sources/${id}`);
      set({ selectedSource: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  fetchImpact: async (id: string) => {
    try {
      const response = await apiClient.get<ImpactAnalysis>(`/catalog/sources/${id}/impact`);
      set({ impact: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  searchCatalog: async (query: string) => {
    try {
      const response = await apiClient.get<SearchResult[]>(`/catalog/search?q=${encodeURIComponent(query)}`);
      set({ searchResults: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  createGlossaryTerm: async (term) => {
    try {
      await apiClient.post('/catalog/glossary', term);
      await get().fetchGlossary();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
  deleteGlossaryTerm: async (id: string) => {
    try {
      await apiClient.del(`/catalog/glossary/${id}`);
      await get().fetchGlossary();
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
