import { create } from 'zustand';
import type { Feedback, FeedbackSummary } from '../types/feedback';
import apiClient from '../utils/apiClient';

interface FeedbackState {
  feedbacks: Feedback[];
  summaries: FeedbackSummary[];
  isLoading: boolean;
  error: string | null;
  fetchFeedbacks: (agentId?: string) => Promise<void>;
  submitFeedback: (feedback: Partial<Feedback>) => Promise<void>;
  fetchSummary: () => Promise<void>;
}

export const useFeedbackStore = create<FeedbackState>((set) => ({
  feedbacks: [],
  summaries: [],
  isLoading: false,
  error: null,

  fetchFeedbacks: async (agentId?: string) => {
    set({ isLoading: true, error: null });
    try {
      const url = agentId ? `/feedback?agent_id=${agentId}` : '/feedback';
      const response = await apiClient.get<Feedback[]>(url);
      set({ feedbacks: response.data, isLoading: false });
    } catch (err) {
      set({ error: (err as Error).message, isLoading: false });
    }
  },

  submitFeedback: async (feedback) => {
    try {
      await apiClient.post('/feedback', feedback);
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },

  fetchSummary: async () => {
    try {
      const response = await apiClient.get<FeedbackSummary[]>('/feedback/summary');
      set({ summaries: response.data });
    } catch (err) {
      set({ error: (err as Error).message });
    }
  },
}));
