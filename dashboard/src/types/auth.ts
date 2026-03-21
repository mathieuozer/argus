export interface AuthUser {
  id: string;
  username: string;
  email: string;
  tenantId: string;
  tenantName: string;
  role: 'admin' | 'operator' | 'viewer';
}

export interface LoginRequest {
  username: string;
  password: string;
  tenantId: string;
}

export interface LoginResponse {
  token: string;
  refreshToken: string;
  user: AuthUser;
  expiresAt: string;
}

export interface AuthState {
  user: AuthUser | null;
  token: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
}
