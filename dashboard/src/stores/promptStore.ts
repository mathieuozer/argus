import { create } from 'zustand';
import type { Prompt, PromptVersion } from '../types/prompt';
import apiClient from '../utils/apiClient';

interface PromptState {
  prompts: Prompt[];
  versions: PromptVersion[];
  isLoading: boolean;
  error: string | null;
  fetchPrompts: () => Promise<void>;
  createPrompt: (prompt: Partial<Prompt>) => Promise<void>;
  fetchVersions: (promptId: string) => Promise<void>;
  createVersion: (promptId: string, version: Partial<PromptVersion>) => Promise<void>;
}

export const usePromptStore = create<PromptState>((set) => ({
  prompts: [],
  versions: [],
  isLoading: false,
  error: null,

  fetchPrompts: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await apiClient.get<Prompt[]>('/prompts');
      set({ prompts: response.data, isLoading: false });
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
    }
  },

  createPrompt: async (prompt) => {
    try {
      await apiClient.post('/prompts', prompt);
      const response = await apiClient.get<Prompt[]>('/prompts');
      set({ prompts: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },

  fetchVersions: async (promptId) => {
    try {
      const response = await apiClient.get<PromptVersion[]>(`/prompts/${promptId}/versions`);
      set({ versions: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },

  createVersion: async (promptId, version) => {
    try {
      await apiClient.post(`/prompts/${promptId}/versions`, version);
      const response = await apiClient.get<PromptVersion[]>(`/prompts/${promptId}/versions`);
      set({ versions: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
