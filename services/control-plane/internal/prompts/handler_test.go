package prompts

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/argus-platform/argus/pkg/tenancy"
)

func TestCreateAndListPrompts(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	body, _ := json.Marshal(Prompt{Name: "System Prompt", Description: "Main system prompt", AgentID: "agent-1"})
	req := httptest.NewRequest("POST", "/api/v1/prompts", bytes.NewReader(body))
	ctx := tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.CreatePrompt(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/v1/prompts", nil)
	ctx = tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	handler.ListPrompts(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCrossTenantIsolation(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)
	repo.prompts["p-1"] = &Prompt{ID: "p-1", TenantID: "tenant-a", Name: "Prompt A"}

	req := httptest.NewRequest("GET", "/api/v1/prompts/p-1", nil)
	req.SetPathValue("id", "p-1")
	ctx := tenancy.WithTenant(req.Context(), "tenant-b")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.GetPrompt(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-tenant, got %d", w.Code)
	}
}
