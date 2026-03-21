import { create } from 'zustand';
import type { AuthUser, LoginRequest, AuthState } from '../types/auth';

interface AuthStore extends AuthState {
  login: (request: LoginRequest) => Promise<void>;
  logout: () => void;
  refreshAuth: () => Promise<void>;
  clearError: () => void;
}

export const useAuthStore = create<AuthStore>((set, get) => ({
  user: JSON.parse(localStorage.getItem('argus_user') || 'null') as AuthUser | null,
  token: localStorage.getItem('argus_token'),
  refreshToken: localStorage.getItem('argus_refresh_token'),
  isAuthenticated: !!localStorage.getItem('argus_token'),
  isLoading: false,
  error: null,

  login: async (request: LoginRequest) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: { message: 'Login failed' } }));
        throw new Error(err.error?.message || 'Login failed');
      }

      const data = await response.json();
      const { token, refreshToken, user } = data.data;

      localStorage.setItem('argus_token', token);
      localStorage.setItem('argus_refresh_token', refreshToken);
      localStorage.setItem('argus_user', JSON.stringify(user));

      set({
        user,
        token,
        refreshToken,
        isAuthenticated: true,
        isLoading: false,
        error: null,
      });
    } catch (error) {
      set({
        isLoading: false,
        error: error instanceof Error ? error.message : 'Login failed',
      });
      throw error;
    }
  },

  logout: () => {
    localStorage.removeItem('argus_token');
    localStorage.removeItem('argus_refresh_token');
    localStorage.removeItem('argus_user');
    set({
      user: null,
      token: null,
      refreshToken: null,
      isAuthenticated: false,
      error: null,
    });
  },

  refreshAuth: async () => {
    const { refreshToken } = get();
    if (!refreshToken) {
      get().logout();
      return;
    }

    try {
      const response = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken }),
      });

      if (!response.ok) {
        get().logout();
        return;
      }

      const data = await response.json();
      const { token, refreshToken: newRefresh, user } = data.data;

      localStorage.setItem('argus_token', token);
      localStorage.setItem('argus_refresh_token', newRefresh);
      localStorage.setItem('argus_user', JSON.stringify(user));

      set({ user, token, refreshToken: newRefresh });
    } catch {
      get().logout();
    }
  },

  clearError: () => set({ error: null }),
}));
