import { create } from 'zustand';
import type { AuditEntry, AuditFilters } from '../types/audit';
import apiClient from '../utils/apiClient';

interface AuditState {
  entries: AuditEntry[];
  loading: boolean;
  error: string | null;
  fetchEntries: (filters?: AuditFilters) => Promise<void>;
}

export const useAuditStore = create<AuditState>((set) => ({
  entries: [],
  loading: false,
  error: null,
  fetchEntries: async (filters?: AuditFilters) => {
    set({ loading: true, error: null });
    try {
      const params = new URLSearchParams();
      if (filters?.actor) params.set('actor', filters.actor);
      if (filters?.action) params.set('action', filters.action);
      if (filters?.from) params.set('from', filters.from);
      if (filters?.to) params.set('to', filters.to);
      if (filters?.limit) params.set('limit', String(filters.limit));
      const query = params.toString();
      const path = query ? `/audit?${query}` : '/audit';
      const response = await apiClient.get<AuditEntry[]>(path);
      set({ entries: response.data, loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },
}));
