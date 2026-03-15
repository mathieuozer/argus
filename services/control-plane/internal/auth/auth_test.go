package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	m := New("test-secret-key")

	claims := &Claims{
		Sub:      "user-1",
		TenantID: "tenant-1",
		Role:     RoleAdmin,
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	token, err := m.GenerateToken(claims)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	parsed, err := m.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if parsed.Sub != claims.Sub {
		t.Errorf("Sub = %q, want %q", parsed.Sub, claims.Sub)
	}
	if parsed.TenantID != claims.TenantID {
		t.Errorf("TenantID = %q, want %q", parsed.TenantID, claims.TenantID)
	}
	if parsed.Role != claims.Role {
		t.Errorf("Role = %q, want %q", parsed.Role, claims.Role)
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	m1 := New("secret-1")
	m2 := New("secret-2")

	claims := &Claims{
		Sub:      "user-1",
		TenantID: "tenant-1",
		Role:     RoleViewer,
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	token, _ := m1.GenerateToken(claims)
	_, err := m2.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error validating with wrong secret")
	}
}

func TestValidateTokenInvalidFormat(t *testing.T) {
	m := New("secret")

	tests := []struct {
		name  string
		token string
	}{
		{name: "empty", token: ""},
		{name: "one part", token: "abc"},
		{name: "two parts", token: "abc.def"},
		{name: "four parts", token: "a.b.c.d"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := m.ValidateToken(tc.token)
			if err == nil {
				t.Error("expected error for invalid token format")
			}
		})
	}
}

func TestMiddlewareDevMode(t *testing.T) {
	m := New("dev") // dev mode

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Error("expected claims in context")
			return
		}
		if claims.Role != RoleAdmin {
			t.Errorf("expected admin role in dev mode, got %q", claims.Role)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMiddlewareWithValidToken(t *testing.T) {
	m := New("production-secret")

	claims := &Claims{
		Sub:      "user-1",
		TenantID: "tenant-1",
		Role:     RoleOperator,
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}
	token, _ := m.GenerateToken(claims)

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Error("expected claims in context")
			return
		}
		if c.Role != RoleOperator {
			t.Errorf("Role = %q, want %q", c.Role, RoleOperator)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMiddlewareExpiredToken(t *testing.T) {
	m := New("secret")

	claims := &Claims{
		Sub:      "user-1",
		TenantID: "tenant-1",
		Role:     RoleViewer,
		Iat:      time.Now().Add(-2 * time.Hour).Unix(),
		Exp:      time.Now().Add(-1 * time.Hour).Unix(), // expired
	}
	token, _ := m.GenerateToken(claims)

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for expired token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name       string
		claimRole  Role
		required   []Role
		wantStatus int
	}{
		{
			name:       "allowed role",
			claimRole:  RoleAdmin,
			required:   []Role{RoleAdmin, RoleOperator},
			wantStatus: http.StatusOK,
		},
		{
			name:       "denied role",
			claimRole:  RoleViewer,
			required:   []Role{RoleAdmin},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := RequireRole(tc.required...)(inner)

			claims := &Claims{Role: tc.claimRole}
			ctx := WithClaims(context.Background(), claims)
			req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

func TestClaimsContext(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		claims := &Claims{Sub: "user-1", TenantID: "t1", Role: RoleAdmin}
		ctx := WithClaims(context.Background(), claims)
		got, ok := ClaimsFromContext(ctx)
		if !ok {
			t.Fatal("expected claims in context")
		}
		if got.Sub != "user-1" {
			t.Errorf("Sub = %q, want %q", got.Sub, "user-1")
		}
	})

	t.Run("empty context", func(t *testing.T) {
		_, ok := ClaimsFromContext(context.Background())
		if ok {
			t.Error("expected no claims in empty context")
		}
	})
}
