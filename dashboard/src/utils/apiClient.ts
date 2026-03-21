import type { ApiResponse, ApiError } from '../types/api';

const API_BASE = '/api/v1';

class ApiClient {
  private tenantId: string = 'default';

  setTenantId(tenantId: string) {
    this.tenantId = tenantId;
  }

  private async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const url = `${API_BASE}${path}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Tenant-ID': this.tenantId,
      ...((options.headers as Record<string, string>) || {}),
    };

    const response = await fetch(url, { ...options, headers });

    if (!response.ok) {
      const error: ApiError = await response.json();
      throw new Error(error.error?.message || 'Request failed');
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
