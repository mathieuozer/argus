import { create } from 'zustand';
import type { GuardrailRule, GuardrailViolation, GuardrailStats } from '../types/guardrail';
import apiClient from '../utils/apiClient';

interface GuardrailState {
  rules: GuardrailRule[];
  violations: GuardrailViolation[];
  stats: GuardrailStats | null;
  isLoading: boolean;
  error: string | null;
  fetchRules: () => Promise<void>;
  fetchViolations: () => Promise<void>;
  fetchStats: () => Promise<void>;
  createRule: (rule: Partial<GuardrailRule>) => Promise<void>;
}

export const useGuardrailStore = create<GuardrailState>((set) => ({
  rules: [],
  violations: [],
  stats: null,
  isLoading: false,
  error: null,

  fetchRules: async () => {
    set({ isLoading: true });
    try {
      const response = await apiClient.get<GuardrailRule[]>('/guardrails/rules');
      set({ rules: response.data, isLoading: false });
    } catch {
      set({ error: 'Failed to fetch rules', isLoading: false });
    }
  },

  fetchViolations: async () => {
    try {
      const response = await apiClient.get<GuardrailViolation[]>('/guardrails/violations');
      set({ violations: response.data });
    } catch {
      set({ error: 'Failed to fetch violations' });
    }
  },

  fetchStats: async () => {
    try {
      const response = await apiClient.get<GuardrailStats>('/guardrails/stats');
      set({ stats: response.data });
    } catch {
      set({ error: 'Failed to fetch stats' });
    }
  },

  createRule: async (rule) => {
    try {
      await apiClient.post('/guardrails/rules', rule);
      const response = await apiClient.get<GuardrailRule[]>('/guardrails/rules');
      set({ rules: response.data });
    } catch {
      set({ error: 'Failed to create rule' });
    }
  },
}));
