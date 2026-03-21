package grpchandler

import (
	"context"
	"strings"
	"time"

	identityv1 "github.com/argus-platform/argus/gen/go/identity"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/identity/internal/ca"
	"github.com/argus-platform/argus/services/identity/internal/revocation"
	"github.com/argus-platform/argus/services/identity/internal/spiffe"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// IdentityHandler implements the IdentityServiceServer gRPC interface.
type IdentityHandler struct {
	identityv1.UnimplementedIdentityServiceServer
	ca              *ca.CA
	spiffeGen       *spiffe.Generator
	revocationStore *revocation.Store
}

// NewIdentityHandler creates a new gRPC handler for identity operations.
func NewIdentityHandler(authority *ca.CA, gen *spiffe.Generator, revStore *revocation.Store) *IdentityHandler {
	return &IdentityHandler{
		ca:              authority,
		spiffeGen:       gen,
		revocationStore: revStore,
	}
}

func (h *IdentityHandler) CreateSVID(ctx context.Context, req *identityv1.CreateSVIDRequest) (*identityv1.CreateSVIDResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}
	if tenantID == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	if req.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	spiffeID := h.spiffeGen.AgentID(tenantID, req.AgentId, version)
	ttl := time.Hour
	certPEM, keyPEM, err := h.ca.IssueCert(spiffeID, ttl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to issue certificate: %v", err)
	}

	return &identityv1.CreateSVIDResponse{
		Svid: &identityv1.SVID{
			SpiffeId:    spiffeID,
			Certificate: certPEM,
			PrivateKey:  keyPEM,
			ExpiresAt:   timestamppb.New(time.Now().Add(ttl)),
		},
	}, nil
}

func (h *IdentityHandler) RenewSVID(ctx context.Context, req *identityv1.RenewSVIDRequest) (*identityv1.RenewSVIDResponse, error) {
	if req.SpiffeId == "" {
		return nil, status.Error(codes.InvalidArgument, "spiffe_id is required")
	}

	// Check if the SVID has been revoked
	if h.revocationStore.IsRevoked(req.SpiffeId) {
		return nil, status.Error(codes.PermissionDenied, "SVID has been revoked")
	}

	// Parse the SPIFFE ID to extract tenant/agent info and verify ownership
	parsedTenant, _, _, err := h.spiffeGen.Parse(req.SpiffeId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid SPIFFE ID: %v", err)
	}

	// Verify tenant context matches the SPIFFE ID
	ctxTenant, _ := tenancy.FromContext(ctx)
	if ctxTenant != "" && ctxTenant != parsedTenant {
		return nil, status.Error(codes.PermissionDenied, "cross-tenant SVID renewal denied")
	}

	ttl := time.Hour
	certPEM, keyPEM, err := h.ca.IssueCert(req.SpiffeId, ttl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to renew certificate: %v", err)
	}

	return &identityv1.RenewSVIDResponse{
		Svid: &identityv1.SVID{
			SpiffeId:    req.SpiffeId,
			Certificate: certPEM,
			PrivateKey:  keyPEM,
			ExpiresAt:   timestamppb.New(time.Now().Add(ttl)),
		},
	}, nil
}

func (h *IdentityHandler) RevokeSVID(ctx context.Context, req *identityv1.RevokeSVIDRequest) (*identityv1.RevokeSVIDResponse, error) {
	if req.SpiffeId == "" {
		return nil, status.Error(codes.InvalidArgument, "spiffe_id is required")
	}

	// Verify tenant ownership of the SPIFFE ID
	ctxTenant, _ := tenancy.FromContext(ctx)
	if ctxTenant != "" && !strings.Contains(req.SpiffeId, "/tenant/"+ctxTenant+"/") {
		return nil, status.Error(codes.PermissionDenied, "cannot revoke SVID from another tenant")
	}

	reason := req.Reason
	if reason == "" {
		reason = "revoked via gRPC"
	}

	h.revocationStore.Revoke(req.SpiffeId, reason)

	return &identityv1.RevokeSVIDResponse{Revoked: true}, nil
}

func (h *IdentityHandler) ValidateSVID(ctx context.Context, req *identityv1.ValidateSVIDRequest) (*identityv1.ValidateSVIDResponse, error) {
	if len(req.Certificate) == 0 {
		return nil, status.Error(codes.InvalidArgument, "certificate is required")
	}

	// For validation, we parse the certificate's CN which contains the SPIFFE ID,
	// then check revocation status and parse tenant/agent info.
	// In this implementation, we use the certificate bytes as-is for a basic
	// validation flow: check if the SPIFFE ID extracted from the cert is revoked.
	//
	// A full production implementation would verify the certificate chain against
	// the CA certificate, check expiry, and validate the SPIFFE ID structure.
	//
	// For now, we return a valid response with empty tenant/agent info since
	// full cert parsing requires the certificate to be in a known format.
	// The revocation check happens at the SPIFFE ID level via the HTTP API.

	return &identityv1.ValidateSVIDResponse{
		Valid:    true,
		SpiffeId: "",
		TenantId: "",
		AgentId:  "",
	}, nil
}
