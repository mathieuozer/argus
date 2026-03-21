import { create } from 'zustand';
import type { TestSuite, EvalRun } from '../types/eval';
import apiClient from '../utils/apiClient';

interface EvalState {
  suites: TestSuite[];
  runs: EvalRun[];
  selectedRun: EvalRun | null;
  isLoading: boolean;
  error: string | null;
  fetchSuites: () => Promise<void>;
  fetchRuns: () => Promise<void>;
  createSuite: (suite: Partial<TestSuite>) => Promise<void>;
  runEval: (suiteId: string) => Promise<void>;
  fetchRun: (runId: string) => Promise<void>;
}

export const useEvalStore = create<EvalState>((set) => ({
  suites: [],
  runs: [],
  selectedRun: null,
  isLoading: false,
  error: null,

  fetchSuites: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await apiClient.get<TestSuite[]>('/evals/suites');
      set({ suites: response.data, isLoading: false });
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
    }
  },

  fetchRuns: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await apiClient.get<EvalRun[]>('/evals/runs');
      set({ runs: response.data, isLoading: false });
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
    }
  },

  createSuite: async (suite) => {
    try {
      await apiClient.post('/evals/suites', suite);
      const response = await apiClient.get<TestSuite[]>('/evals/suites');
      set({ suites: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },

  runEval: async (suiteId) => {
    try {
      const response = await apiClient.post<EvalRun>(`/evals/suites/${suiteId}/run`, {});
      set((state) => ({ runs: [response.data, ...state.runs] }));
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },

  fetchRun: async (runId) => {
    try {
      const response = await apiClient.get<EvalRun>(`/evals/runs/${runId}`);
      set({ selectedRun: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
