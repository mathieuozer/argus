package grpchandler

import (
	"context"
	"testing"

	identityv1 "github.com/argus-platform/argus/gen/go/identity"
	"github.com/argus-platform/argus/services/identity/internal/ca"
	"github.com/argus-platform/argus/services/identity/internal/revocation"
	"github.com/argus-platform/argus/services/identity/internal/spiffe"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestHandler(t *testing.T) *IdentityHandler {
	t.Helper()
	authority, err := ca.NewDevCA()
	if err != nil {
		t.Fatalf("failed to create CA: %v", err)
	}
	gen := spiffe.NewGenerator("argus.local")
	revStore := revocation.NewStore()
	return NewIdentityHandler(authority, gen, revStore)
}

func TestCreateSVID(t *testing.T) {
	tests := []struct {
		name     string
		req      *identityv1.CreateSVIDRequest
		wantCode codes.Code
	}{
		{
			name: "create SVID successfully",
			req: &identityv1.CreateSVIDRequest{
				TenantId: "tenant-1",
				AgentId:  "agent-1",
				Version:  "1.0.0",
			},
		},
		{
			name: "create SVID with default version",
			req: &identityv1.CreateSVIDRequest{
				TenantId: "tenant-1",
				AgentId:  "agent-1",
			},
		},
		{
			name: "missing tenant_id",
			req: &identityv1.CreateSVIDRequest{
				AgentId: "agent-1",
				Version: "1.0.0",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing agent_id",
			req: &identityv1.CreateSVIDRequest{
				TenantId: "tenant-1",
				Version:  "1.0.0",
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t)

			resp, err := h.CreateSVID(context.Background(), tc.req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Svid == nil {
				t.Fatal("expected SVID in response, got nil")
			}
			if resp.Svid.SpiffeId == "" {
				t.Error("expected non-empty SPIFFE ID")
			}
			if len(resp.Svid.Certificate) == 0 {
				t.Error("expected non-empty certificate")
			}
			if len(resp.Svid.PrivateKey) == 0 {
				t.Error("expected non-empty private key")
			}
			if resp.Svid.ExpiresAt == nil {
				t.Error("expected non-nil ExpiresAt")
			}
		})
	}
}

func TestRenewSVID(t *testing.T) {
	tests := []struct {
		name       string
		setupRevoke string
		req        *identityv1.RenewSVIDRequest
		wantCode   codes.Code
	}{
		{
			name: "renew SVID successfully",
			req: &identityv1.RenewSVIDRequest{
				SpiffeId: "spiffe://argus.local/tenant/tenant-1/agent/agent-1/v1.0.0",
			},
		},
		{
			name: "missing spiffe_id",
			req:  &identityv1.RenewSVIDRequest{},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid spiffe_id format",
			req: &identityv1.RenewSVIDRequest{
				SpiffeId: "invalid-id",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:        "revoked SVID",
			setupRevoke: "spiffe://argus.local/tenant/tenant-1/agent/agent-1/v1.0.0",
			req: &identityv1.RenewSVIDRequest{
				SpiffeId: "spiffe://argus.local/tenant/tenant-1/agent/agent-1/v1.0.0",
			},
			wantCode: codes.PermissionDenied,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t)
			if tc.setupRevoke != "" {
				h.revocationStore.Revoke(tc.setupRevoke, "test revocation")
			}

			resp, err := h.RenewSVID(context.Background(), tc.req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Svid == nil {
				t.Fatal("expected SVID in response, got nil")
			}
			if resp.Svid.SpiffeId != tc.req.SpiffeId {
				t.Errorf("expected SPIFFE ID %q, got %q", tc.req.SpiffeId, resp.Svid.SpiffeId)
			}
			if len(resp.Svid.Certificate) == 0 {
				t.Error("expected non-empty certificate")
			}
		})
	}
}

func TestRevokeSVID(t *testing.T) {
	tests := []struct {
		name     string
		req      *identityv1.RevokeSVIDRequest
		wantCode codes.Code
	}{
		{
			name: "revoke SVID successfully",
			req: &identityv1.RevokeSVIDRequest{
				SpiffeId: "spiffe://argus.local/tenant/tenant-1/agent/agent-1/v1.0.0",
				Reason:   "compromised key",
			},
		},
		{
			name: "revoke with default reason",
			req: &identityv1.RevokeSVIDRequest{
				SpiffeId: "spiffe://argus.local/tenant/tenant-1/agent/agent-1/v1.0.0",
			},
		},
		{
			name:     "missing spiffe_id",
			req:      &identityv1.RevokeSVIDRequest{},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t)

			resp, err := h.RevokeSVID(context.Background(), tc.req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !resp.Revoked {
				t.Error("expected Revoked to be true")
			}

			// Verify the SVID is now in the revocation list
			if !h.revocationStore.IsRevoked(tc.req.SpiffeId) {
				t.Error("SVID should be in revocation list after revoking")
			}
		})
	}
}

func TestValidateSVID(t *testing.T) {
	tests := []struct {
		name     string
		req      *identityv1.ValidateSVIDRequest
		wantCode codes.Code
	}{
		{
			name: "validate certificate",
			req: &identityv1.ValidateSVIDRequest{
				Certificate: []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"),
			},
		},
		{
			name:     "missing certificate",
			req:      &identityv1.ValidateSVIDRequest{},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t)

			resp, err := h.ValidateSVID(context.Background(), tc.req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected code %v, got %v", tc.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !resp.Valid {
				t.Error("expected Valid to be true")
			}
		})
	}
}

func TestCreateThenRevokeThenRenew(t *testing.T) {
	t.Run("create, revoke, and attempt renew", func(t *testing.T) {
		h := newTestHandler(t)

		// Create an SVID
		createResp, err := h.CreateSVID(context.Background(), &identityv1.CreateSVIDRequest{
			TenantId: "tenant-1",
			AgentId:  "agent-1",
			Version:  "1.0.0",
		})
		if err != nil {
			t.Fatalf("CreateSVID failed: %v", err)
		}

		spiffeID := createResp.Svid.SpiffeId

		// Revoke the SVID
		_, err = h.RevokeSVID(context.Background(), &identityv1.RevokeSVIDRequest{
			SpiffeId: spiffeID,
			Reason:   "test revocation",
		})
		if err != nil {
			t.Fatalf("RevokeSVID failed: %v", err)
		}

		// Try to renew — should fail
		_, err = h.RenewSVID(context.Background(), &identityv1.RenewSVIDRequest{
			SpiffeId: spiffeID,
		})
		if err == nil {
			t.Fatal("expected RenewSVID to fail for revoked SVID")
		}
		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got: %v", err)
		}
		if st.Code() != codes.PermissionDenied {
			t.Errorf("expected PermissionDenied, got %v", st.Code())
		}
	})
}
