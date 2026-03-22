package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Role represents a user's role in the system.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
	RoleAgent    Role = "agent"
)

// Claims represents the JWT claims payload.
type Claims struct {
	Sub      string `json:"sub"`
	TenantID string `json:"tenant_id"`
	Role     Role   `json:"role"`
	Iat      int64  `json:"iat"`
	Exp      int64  `json:"exp"`
}

type claimsKey struct{}

// ClaimsFromContext retrieves JWT claims from context.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(*Claims)
	return c, ok
}

// WithClaims returns a context with claims attached.
func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, c)
}

// JWTMiddleware validates JWT tokens.
type JWTMiddleware struct {
	secret  []byte
	devMode bool
}

// New creates a new JWT middleware.
func New(secret string) *JWTMiddleware {
	return &JWTMiddleware{
		secret:  []byte(secret),
		devMode: secret == "" || secret == "dev",
	}
}

// Middleware validates the JWT token in the Authorization header.
func (m *JWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		if auth == "" {
			if m.devMode {
				// In dev mode, create synthetic claims
				claims := &Claims{
					Sub:      "dev-user",
					TenantID: r.Header.Get("X-Tenant-ID"),
					Role:     RoleAdmin,
					Iat:      time.Now().Unix(),
					Exp:      time.Now().Add(time.Hour).Unix(),
				}
				ctx := WithClaims(r.Context(), claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authorization header required")
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format")
			return
		}

		claims, err := m.ValidateToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
			return
		}

		if claims.Exp < time.Now().Unix() {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "token expired")
			return
		}

		ctx := WithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GenerateToken creates a signed JWT token.
func (m *JWTMiddleware) GenerateToken(claims *Claims) (string, error) {
	header := base64url([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	payload := base64url(claimsJSON)
	sigInput := header + "." + payload
	sig := base64url(m.sign([]byte(sigInput)))
	return sigInput + "." + sig, nil
}

// ValidateToken validates and parses a JWT token.
func (m *JWTMiddleware) ValidateToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	sigInput := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding")
	}

	expected := m.sign([]byte(sigInput))
	if !hmac.Equal(signature, expected) {
		return nil, fmt.Errorf("invalid signature")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid claims encoding")
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	return &claims, nil
}

func (m *JWTMiddleware) sign(data []byte) []byte {
	h := hmac.New(sha256.New, m.secret)
	h.Write(data)
	return h.Sum(nil)
}

func base64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// RequireRole returns middleware that checks if the user has the required role.
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	roleSet := make(map[Role]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "no claims in context")
				return
			}
			if !roleSet[claims.Role] {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
