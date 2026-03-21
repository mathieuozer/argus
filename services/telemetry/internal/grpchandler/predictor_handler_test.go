package grpchandler

import (
	"context"
	"testing"
	"time"

	telemetryv1 "github.com/argus-platform/argus/gen/go/telemetry"
	"github.com/argus-platform/argus/services/telemetry/internal/predictor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPredict(t *testing.T) {
	tests := []struct {
		name     string
		req      *telemetryv1.PredictRequest
		wantCode codes.Code
	}{
		{
			name: "valid prediction with low risk",
			req: &telemetryv1.PredictRequest{
				TenantId: "tenant-1",
				AgentId:  "agent-1",
				Features: &telemetryv1.PredictionFeatures{
					LatencyP99Ratio: 1.5,
					TokenVelocity:   10.0,
					RetryRate:       0.01,
					ErrorRateDelta:  0.0,
					ContextFillPct:  0.3,
					ToolCallDepth:   2.0,
					ConsecutiveSlow: 0.0,
					CostAcceleration: 1.0,
				},
			},
		},
		{
			name: "valid prediction with high risk",
			req: &telemetryv1.PredictRequest{
				TenantId: "tenant-1",
				AgentId:  "agent-1",
				Features: &telemetryv1.PredictionFeatures{
					LatencyP99Ratio: 5.0,
					TokenVelocity:   100.0,
					RetryRate:       0.8,
					ErrorRateDelta:  0.5,
					ContextFillPct:  0.95,
					ToolCallDepth:   10.0,
					ConsecutiveSlow: 15.0,
					CostAcceleration: 5.0,
				},
			},
		},
		{
			name: "missing tenant_id",
			req: &telemetryv1.PredictRequest{
				AgentId:  "agent-1",
				Features: &telemetryv1.PredictionFeatures{},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing features",
			req: &telemetryv1.PredictRequest{
				TenantId: "tenant-1",
				AgentId:  "agent-1",
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := predictor.NewClient("")
			store := NewInMemoryAlertStore()
			h := NewPredictorHandler(client, store)

			resp, err := h.Predict(context.Background(), tc.req)
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
			if resp.FailureProbability < 0 || resp.FailureProbability > 1 {
				t.Errorf("failure probability out of range: %f", resp.FailureProbability)
			}
		})
	}
}

func TestListAlerts(t *testing.T) {
	tests := []struct {
		name     string
		setup    []*StoredAlert
		req      *telemetryv1.ListAlertsRequest
		wantLen  int
		wantCode codes.Code
	}{
		{
			name: "list all alerts for tenant",
			setup: []*StoredAlert{
				{ID: "alert-1", TenantID: "tenant-1", AgentID: "agent-1", Status: "open", CreatedAt: time.Now()},
				{ID: "alert-2", TenantID: "tenant-1", AgentID: "agent-2", Status: "open", CreatedAt: time.Now()},
				{ID: "alert-3", TenantID: "tenant-2", AgentID: "agent-1", Status: "open", CreatedAt: time.Now()},
			},
			req:     &telemetryv1.ListAlertsRequest{TenantId: "tenant-1"},
			wantLen: 2,
		},
		{
			name: "filter by agent",
			setup: []*StoredAlert{
				{ID: "alert-1", TenantID: "tenant-1", AgentID: "agent-1", Status: "open", CreatedAt: time.Now()},
				{ID: "alert-2", TenantID: "tenant-1", AgentID: "agent-2", Status: "open", CreatedAt: time.Now()},
			},
			req: &telemetryv1.ListAlertsRequest{
				TenantId: "tenant-1",
				AgentId:  strPtr("agent-1"),
			},
			wantLen: 1,
		},
		{
			name: "filter by status",
			setup: []*StoredAlert{
				{ID: "alert-1", TenantID: "tenant-1", AgentID: "agent-1", Status: "open", CreatedAt: time.Now()},
				{ID: "alert-2", TenantID: "tenant-1", AgentID: "agent-1", Status: "acknowledged", CreatedAt: time.Now()},
			},
			req: &telemetryv1.ListAlertsRequest{
				TenantId: "tenant-1",
				Status:   alertStatusPtr(telemetryv1.AlertStatus_ALERT_STATUS_OPEN),
			},
			wantLen: 1,
		},
		{
			name:     "missing tenant_id",
			req:      &telemetryv1.ListAlertsRequest{},
			wantCode: codes.InvalidArgument,
		},
		{
			name:    "empty result",
			req:     &telemetryv1.ListAlertsRequest{TenantId: "tenant-1"},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := predictor.NewClient("")
			store := NewInMemoryAlertStore()
			for _, a := range tc.setup {
				store.AddAlert(a)
			}
			h := NewPredictorHandler(client, store)

			resp, err := h.ListAlerts(context.Background(), tc.req)
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
			if len(resp.Alerts) != tc.wantLen {
				t.Errorf("expected %d alerts, got %d", tc.wantLen, len(resp.Alerts))
			}
		})
	}
}

func TestAcknowledgeAlert(t *testing.T) {
	tests := []struct {
		name     string
		setup    []*StoredAlert
		req      *telemetryv1.AcknowledgeAlertRequest
		wantCode codes.Code
	}{
		{
			name: "acknowledge existing alert",
			setup: []*StoredAlert{
				{ID: "alert-1", TenantID: "tenant-1", AgentID: "agent-1", Status: "open", CreatedAt: time.Now()},
			},
			req: &telemetryv1.AcknowledgeAlertRequest{TenantId: "tenant-1", AlertId: "alert-1"},
		},
		{
			name: "alert not found",
			req:  &telemetryv1.AcknowledgeAlertRequest{TenantId: "tenant-1", AlertId: "nonexistent"},
			wantCode: codes.NotFound,
		},
		{
			name:     "missing tenant_id",
			req:      &telemetryv1.AcknowledgeAlertRequest{AlertId: "alert-1"},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing alert_id",
			req:  &telemetryv1.AcknowledgeAlertRequest{TenantId: "tenant-1"},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := predictor.NewClient("")
			store := NewInMemoryAlertStore()
			for _, a := range tc.setup {
				store.AddAlert(a)
			}
			h := NewPredictorHandler(client, store)

			resp, err := h.AcknowledgeAlert(context.Background(), tc.req)
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
			if resp.Alert == nil {
				t.Fatal("expected alert in response, got nil")
			}
			if resp.Alert.Status != telemetryv1.AlertStatus_ALERT_STATUS_ACKNOWLEDGED {
				t.Errorf("expected acknowledged status, got %v", resp.Alert.Status)
			}
		})
	}
}

func TestAlertStatusConversion(t *testing.T) {
	tests := []struct {
		str   string
		proto telemetryv1.AlertStatus
	}{
		{"open", telemetryv1.AlertStatus_ALERT_STATUS_OPEN},
		{"acknowledged", telemetryv1.AlertStatus_ALERT_STATUS_ACKNOWLEDGED},
		{"resolved", telemetryv1.AlertStatus_ALERT_STATUS_RESOLVED},
		{"false_positive", telemetryv1.AlertStatus_ALERT_STATUS_FALSE_POSITIVE},
		{"unknown", telemetryv1.AlertStatus_ALERT_STATUS_UNSPECIFIED},
	}

	for _, tc := range tests {
		t.Run(tc.str, func(t *testing.T) {
			result := stringToProtoAlertStatus(tc.str)
			if result != tc.proto {
				t.Errorf("stringToProtoAlertStatus(%q) = %v, want %v", tc.str, result, tc.proto)
			}

			roundTrip := protoAlertStatusToString(tc.proto)
			if tc.str != "unknown" && roundTrip != tc.str {
				t.Errorf("round-trip failed: %q -> %v -> %q", tc.str, tc.proto, roundTrip)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func alertStatusPtr(s telemetryv1.AlertStatus) *telemetryv1.AlertStatus {
	return &s
}
