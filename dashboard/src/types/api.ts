export interface ApiResponse<T> {
  data: T;
  meta: {
    tenant_id: string;
    request_id?: string;
  };
}

export interface ApiError {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
}
