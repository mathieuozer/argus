package feedback

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/argus-platform/argus/pkg/tenancy"
)

func TestSubmitAndListFeedback(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	fb := Feedback{AgentID: "agent-1", SpanID: "span-1", Rating: 1, Comment: "Great response"}
	body, _ := json.Marshal(fb)
	req := httptest.NewRequest("POST", "/api/v1/feedback", bytes.NewReader(body))
	ctx := tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.SubmitFeedback(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/v1/feedback", nil)
	ctx = tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	handler.ListFeedback(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCrossTenantIsolation(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	repo.feedback = append(repo.feedback, &Feedback{
		ID: "fb-1", TenantID: "tenant-a", AgentID: "agent-1", Rating: 1,
	})

	req := httptest.NewRequest("GET", "/api/v1/feedback", nil)
	ctx := tenancy.WithTenant(req.Context(), "tenant-b")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ListFeedback(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected 0 feedback for tenant-b, got %d", len(data))
	}
}

func TestGetSummary(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	req := httptest.NewRequest("GET", "/api/v1/feedback/summary", nil)
	ctx := tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.GetSummary(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
