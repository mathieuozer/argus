import type { ApiResponse, ApiError } from '../types/api';
import { useAuthStore } from '../stores/authStore';

const API_BASE = '/api/v1';

class ApiClient {
  private tenantId: string = 'default';

  setTenantId(tenantId: string) {
    this.tenantId = tenantId;
  }

  private async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const url = `${API_BASE}${path}`;
    const token = useAuthStore.getState().token;

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Tenant-ID': this.tenantId,
      ...((options.headers as Record<string, string>) || {}),
    };

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(url, { ...options, headers });

    if (response.status === 401) {
      useAuthStore.getState().logout();
      window.location.href = '/login';
      throw new Error('Session expired');
    }

    if (!response.ok) {
      try {
        const error: ApiError = await response.json();
        throw new Error(error.error?.message || 'Request failed');
      } catch (parseErr) {
        if (parseErr instanceof SyntaxError) {
          throw new Error(`Request failed with status ${response.status}`);
        }
        throw parseErr;
      }
    }

    return response.json();
  }

  async get<T>(path: string): Promise<ApiResponse<T>> {
    return this.request<ApiResponse<T>>(path, { method: 'GET' });
  }

  async post<T>(path: string, body: unknown): Promise<ApiResponse<T>> {
    return this.request<ApiResponse<T>>(path, {
      method: 'POST',
      body: JSON.stringify(body),
    });
  }

  async put<T>(path: string, body: unknown): Promise<ApiResponse<T>> {
    return this.request<ApiResponse<T>>(path, {
      method: 'PUT',
      body: JSON.stringify(body),
    });
  }

  async patch<T>(path: string, body: unknown): Promise<ApiResponse<T>> {
    return this.request<ApiResponse<T>>(path, {
      method: 'PATCH',
      body: JSON.stringify(body),
    });
  }

  async del<T>(path: string): Promise<ApiResponse<T>> {
    return this.request<ApiResponse<T>>(path, { method: 'DELETE' });
  }
}

export const apiClient = new ApiClient();
export default apiClient;
