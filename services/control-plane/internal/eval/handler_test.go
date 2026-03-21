package eval

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/argus-platform/argus/pkg/tenancy"
)

func TestCreateAndListSuites(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	suite := TestSuite{
		Name:        "Basic Tests",
		Description: "Basic evaluation tests",
		AgentID:     "test-agent",
		TestCases: []TestCase{
			{
				ID:             "tc-1",
				Name:           "Simple query",
				Input:          "What is 2+2?",
				ExpectedOutput: "4",
			},
		},
	}

	body, _ := json.Marshal(suite)
	req := httptest.NewRequest("POST", "/api/v1/evals/suites", bytes.NewReader(body))
	ctx := tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.CreateSuite(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List suites
	req = httptest.NewRequest("GET", "/api/v1/evals/suites", nil)
	ctx = tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	handler.ListSuites(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRunEval(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	suite := &TestSuite{
		ID:       "suite-1",
		TenantID: "test-tenant",
		Name:     "Test Suite",
		AgentID:  "test-agent",
		TestCases: []TestCase{
			{ID: "tc-1", Name: "Test 1", Input: "Hello"},
			{ID: "tc-2", Name: "Test 2", Input: "World"},
		},
	}
	repo.suites[suite.ID] = suite

	req := httptest.NewRequest("POST", "/api/v1/evals/suites/suite-1/run", nil)
	req.SetPathValue("id", "suite-1")
	ctx := tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.RunEval(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["status"] != "completed" {
		t.Errorf("expected status completed, got %v", data["status"])
	}
}

func TestCrossTenantIsolation(t *testing.T) {
	repo := NewRepository()
	handler := NewHandler(repo)

	suite := &TestSuite{
		ID:       "suite-1",
		TenantID: "tenant-a",
		Name:     "Suite A",
	}
	repo.suites[suite.ID] = suite

	// Try to access from tenant-b
	req := httptest.NewRequest("GET", "/api/v1/evals/suites/suite-1", nil)
	req.SetPathValue("id", "suite-1")
	ctx := tenancy.WithTenant(req.Context(), "tenant-b")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.GetSuite(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-tenant access, got %d", w.Code)
	}
}
