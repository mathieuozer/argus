package rag

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/argus-platform/argus/pkg/tenancy"
)

func TestListRetrievals(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	repo.retrievals = append(repo.retrievals, &Retrieval{
		ID: "ret-1", TenantID: "test-tenant", AgentID: "agent-1",
		Query: "test query", NumChunks: 5, AvgRelevance: 0.85,
		LatencyMs: 150, CreatedAt: time.Now(),
	})

	req := httptest.NewRequest("GET", "/api/v1/rag/retrievals", nil)
	ctx := tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ListRetrievals(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 retrieval, got %d", len(data))
	}
}

func TestCrossTenantIsolation(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	repo.retrievals = append(repo.retrievals, &Retrieval{
		ID: "ret-1", TenantID: "tenant-a", AgentID: "agent-1",
	})

	req := httptest.NewRequest("GET", "/api/v1/rag/retrievals", nil)
	ctx := tenancy.WithTenant(req.Context(), "tenant-b")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ListRetrievals(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected 0 retrievals for tenant-b, got %d", len(data))
	}
}
