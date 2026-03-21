package guardrails

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/argus-platform/argus/pkg/tenancy"
)

func TestCreateAndListRules(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	rule := Rule{
		Name:    "Block Injection",
		Type:    "prompt_injection",
		Action:  "block",
		Enabled: true,
	}

	body, _ := json.Marshal(rule)
	req := httptest.NewRequest("POST", "/api/v1/guardrails/rules", bytes.NewReader(body))
	ctx := tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.CreateRule(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/v1/guardrails/rules", nil)
	ctx = tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	handler.ListRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCrossTenantIsolation(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	repo.rules["r-1"] = &Rule{
		ID:       "r-1",
		TenantID: "tenant-a",
		Name:     "Rule A",
	}

	req := httptest.NewRequest("GET", "/api/v1/guardrails/rules", nil)
	ctx := tenancy.WithTenant(req.Context(), "tenant-b")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ListRules(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected 0 rules for tenant-b, got %d", len(data))
	}
}

func TestGetStats(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	req := httptest.NewRequest("GET", "/api/v1/guardrails/stats", nil)
	ctx := tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.GetStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
