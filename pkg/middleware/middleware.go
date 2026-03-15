package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/argus-platform/argus/pkg/tenancy"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	TenantHeader   = "X-Tenant-ID"
	TenantMetadata = "x-tenant-id"
)

// TenantHTTP extracts the tenant ID from the HTTP header and injects it into context.
func TenantHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get(TenantHeader)
		if tenantID == "" {
			http.Error(w, `{"error":{"code":"TENANT_REQUIRED","message":"X-Tenant-ID header is required"}}`, http.StatusBadRequest)
			return
		}
		ctx := tenancy.WithTenant(r.Context(), tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TenantUnaryInterceptor extracts the tenant ID from gRPC metadata.
func TenantUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "missing metadata")
		}

		tenantIDs := md.Get(TenantMetadata)
		if len(tenantIDs) == 0 || tenantIDs[0] == "" {
			return nil, status.Error(codes.InvalidArgument, "x-tenant-id metadata is required")
		}

		ctx = tenancy.WithTenant(ctx, tenantIDs[0])
		return handler(ctx, req)
	}
}

// RequestLogger logs HTTP requests.
func RequestLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("tenant", r.Header.Get(TenantHeader)),
			)
			next.ServeHTTP(w, r)
		})
	}
}

// CORS adds CORS headers for the dashboard.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, "+TenantHeader)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// stripBearerPrefix removes "Bearer " prefix from token strings.
func stripBearerPrefix(token string) string {
	return strings.TrimPrefix(token, "Bearer ")
}
