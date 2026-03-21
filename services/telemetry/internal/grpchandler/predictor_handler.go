package grpchandler

import (
	"context"
	"fmt"
	"time"

	telemetryv1 "github.com/argus-platform/argus/gen/go/telemetry"
	"github.com/argus-platform/argus/pkg/tenancy"
	"github.com/argus-platform/argus/services/telemetry/internal/predictor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AlertStore defines the interface for alert persistence used by the predictor handler.
type AlertStore interface {
	ListAlerts(tenantID string, agentID string, statusFilter string) ([]*StoredAlert, error)
	AcknowledgeAlert(tenantID, alertID string) (*StoredAlert, error)
}

// StoredAlert represents an alert stored in-memory or in a database.
type StoredAlert struct {
	ID              string
	TenantID        string
	AgentID         string
	Probability     float64
	EstimatedTTFSec int32
	PrecursorType   string
	Evidence        []string
	Status          string
	CreatedAt       time.Time
}

// InMemoryAlertStore provides in-memory alert storage for dev/testing.
type InMemoryAlertStore struct {
	alerts []*StoredAlert
}

// NewInMemoryAlertStore creates a new in-memory alert store.
func NewInMemoryAlertStore() *InMemoryAlertStore {
	return &InMemoryAlertStore{
		alerts: make([]*StoredAlert, 0),
	}
}

// AddAlert adds an alert to the store.
func (s *InMemoryAlertStore) AddAlert(alert *StoredAlert) {
	s.alerts = append(s.alerts, alert)
}

// ListAlerts returns alerts matching the given filters.
func (s *InMemoryAlertStore) ListAlerts(tenantID string, agentID string, statusFilter string) ([]*StoredAlert, error) {
	var result []*StoredAlert
	for _, a := range s.alerts {
		if a.TenantID != tenantID {
			continue
		}
		if agentID != "" && a.AgentID != agentID {
			continue
		}
		if statusFilter != "" && a.Status != statusFilter {
			continue
		}
		result = append(result, a)
	}
	return result, nil
}

// AcknowledgeAlert sets an alert's status to acknowledged.
func (s *InMemoryAlertStore) AcknowledgeAlert(tenantID, alertID string) (*StoredAlert, error) {
	for _, a := range s.alerts {
		if a.ID == alertID && a.TenantID == tenantID {
			a.Status = "acknowledged"
			return a, nil
		}
	}
	return nil, fmt.Errorf("alert %s not found for tenant %s", alertID, tenantID)
}

// PredictorHandler implements the PredictorServiceServer gRPC interface.
type PredictorHandler struct {
	telemetryv1.UnimplementedPredictorServiceServer
	predictor  *predictor.Client
	alertStore AlertStore
}

// NewPredictorHandler creates a new gRPC handler for predictor operations.
func NewPredictorHandler(p *predictor.Client, store AlertStore) *PredictorHandler {
	return &PredictorHandler{predictor: p, alertStore: store}
}

func (h *PredictorHandler) Predict(ctx context.Context, req *telemetryv1.PredictRequest) (*telemetryv1.PredictResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}
	if tenantID == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	if req.Features == nil {
		return nil, status.Error(codes.InvalidArgument, "features are required")
	}

	features := protoToFeatures(req.Features)
	prediction, err := h.predictor.PredictAndEvaluate(tenantID, req.AgentId, features)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	shouldAlert := prediction.FailureProbability >= 0.5

	return &telemetryv1.PredictResponse{
		FailureProbability: prediction.FailureProbability,
		TtfSeconds:         int32(prediction.TTFSeconds),
		PrecursorType:      prediction.PrecursorType,
		ShouldAlert:        shouldAlert,
	}, nil
}

func (h *PredictorHandler) ListAlerts(ctx context.Context, req *telemetryv1.ListAlertsRequest) (*telemetryv1.ListAlertsResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}
	if tenantID == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	agentID := ""
	if req.AgentId != nil {
		agentID = *req.AgentId
	}

	statusFilter := ""
	if req.Status != nil {
		statusFilter = protoAlertStatusToString(*req.Status)
	}

	alerts, err := h.alertStore.ListAlerts(tenantID, agentID, statusFilter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoAlerts := make([]*telemetryv1.PredictiveAlert, len(alerts))
	for i, a := range alerts {
		protoAlerts[i] = storedAlertToProto(a)
	}

	return &telemetryv1.ListAlertsResponse{
		Alerts: protoAlerts,
	}, nil
}

func (h *PredictorHandler) AcknowledgeAlert(ctx context.Context, req *telemetryv1.AcknowledgeAlertRequest) (*telemetryv1.AcknowledgeAlertResponse, error) {
	tenantID := req.TenantId
	if tenantID == "" {
		tenantID, _ = tenancy.FromContext(ctx)
	}
	if tenantID == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id is required")
	}

	if req.AlertId == "" {
		return nil, status.Error(codes.InvalidArgument, "alert_id is required")
	}

	alert, err := h.alertStore.AcknowledgeAlert(tenantID, req.AlertId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &telemetryv1.AcknowledgeAlertResponse{
		Alert: storedAlertToProto(alert),
	}, nil
}

func protoToFeatures(f *telemetryv1.PredictionFeatures) *predictor.Features {
	return &predictor.Features{
		LatencyP99Ratio:  f.LatencyP99Ratio,
		TokenVelocity:    f.TokenVelocity,
		RetryRate:        f.RetryRate,
		ErrorRateDelta:   f.ErrorRateDelta,
		ContextFillPct:   f.ContextFillPct,
		ToolCallDepth:    f.ToolCallDepth,
		ConsecutiveSlow:  f.ConsecutiveSlow,
		CostAcceleration: f.CostAcceleration,
	}
}

func storedAlertToProto(a *StoredAlert) *telemetryv1.PredictiveAlert {
	return &telemetryv1.PredictiveAlert{
		Id:                  a.ID,
		TenantId:            a.TenantID,
		AgentId:             a.AgentID,
		FailureProbability:  a.Probability,
		EstimatedTtfSeconds: a.EstimatedTTFSec,
		PrecursorType:       a.PrecursorType,
		Evidence:            a.Evidence,
		Status:              stringToProtoAlertStatus(a.Status),
		CreatedAt:           timestamppb.New(a.CreatedAt),
	}
}

func stringToProtoAlertStatus(s string) telemetryv1.AlertStatus {
	switch s {
	case "open":
		return telemetryv1.AlertStatus_ALERT_STATUS_OPEN
	case "acknowledged":
		return telemetryv1.AlertStatus_ALERT_STATUS_ACKNOWLEDGED
	case "resolved":
		return telemetryv1.AlertStatus_ALERT_STATUS_RESOLVED
	case "false_positive":
		return telemetryv1.AlertStatus_ALERT_STATUS_FALSE_POSITIVE
	default:
		return telemetryv1.AlertStatus_ALERT_STATUS_UNSPECIFIED
	}
}

func protoAlertStatusToString(s telemetryv1.AlertStatus) string {
	switch s {
	case telemetryv1.AlertStatus_ALERT_STATUS_OPEN:
		return "open"
	case telemetryv1.AlertStatus_ALERT_STATUS_ACKNOWLEDGED:
		return "acknowledged"
	case telemetryv1.AlertStatus_ALERT_STATUS_RESOLVED:
		return "resolved"
	case telemetryv1.AlertStatus_ALERT_STATUS_FALSE_POSITIVE:
		return "false_positive"
	default:
		return ""
	}
}
