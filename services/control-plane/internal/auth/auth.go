package auth

import (
	"net/http"
	"strings"
)

// JWTMiddleware is a stub JWT authentication middleware.
type JWTMiddleware struct {
	secret string
}

// New creates a new JWT middleware.
func New(secret string) *JWTMiddleware {
	return &JWTMiddleware{secret: secret}
}

// Middleware validates the JWT token in the Authorization header.
func (m *JWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			// In development, allow unauthenticated requests
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"invalid authorization header"}}`, http.StatusUnauthorized)
			return
		}

		// Stub: in production, validate the JWT token
		_ = token
		next.ServeHTTP(w, r)
	})
}
