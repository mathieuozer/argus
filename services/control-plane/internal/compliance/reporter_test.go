package compliance

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/argus-platform/argus/pkg/tenancy"
)

func TestGenerateReport(t *testing.T) {
	repo := NewReportRepository()
	handler := NewReportHandler(repo)

	body, _ := json.Marshal(map[string]string{
		"profile_id":   "gcc-sa",
		"period_start": "2025-01-01T00:00:00Z",
		"period_end":   "2025-03-01T00:00:00Z",
	})
	req := httptest.NewRequest("POST", "/api/v1/compliance/reports", bytes.NewReader(body))
	ctx := tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.GenerateReport(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["profile_name"] != "Saudi Arabia (NDMO)" {
		t.Errorf("unexpected profile name: %v", data["profile_name"])
	}
	sections := data["sections"].([]any)
	if len(sections) < 4 {
		t.Errorf("expected at least 4 sections, got %d", len(sections))
	}
}

func TestGenerateReport_InvalidProfile(t *testing.T) {
	repo := NewReportRepository()
	handler := NewReportHandler(repo)

	body, _ := json.Marshal(map[string]string{"profile_id": "nonexistent"})
	req := httptest.NewRequest("POST", "/api/v1/compliance/reports", bytes.NewReader(body))
	ctx := tenancy.WithTenant(req.Context(), "test-tenant")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.GenerateReport(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid profile, got %d", w.Code)
	}
}

func TestGenerateReport_NoTenant(t *testing.T) {
	repo := NewReportRepository()
	handler := NewReportHandler(repo)

	body, _ := json.Marshal(map[string]string{"profile_id": "gcc-sa"})
	req := httptest.NewRequest("POST", "/api/v1/compliance/reports", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.GenerateReport(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing tenant, got %d", w.Code)
	}
}

func TestListReports_TenantIsolation(t *testing.T) {
	repo := NewReportRepository()
	handler := NewReportHandler(repo)

	repo.reports["rpt-1"] = &Report{ID: "rpt-1", TenantID: "tenant-a", Title: "Report A"}

	req := httptest.NewRequest("GET", "/api/v1/compliance/reports", nil)
	ctx := tenancy.WithTenant(req.Context(), "tenant-b")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ListReports(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected 0 reports for tenant-b, got %d", len(data))
	}
}

func TestListReports_ReturnsTenantReports(t *testing.T) {
	repo := NewReportRepository()
	handler := NewReportHandler(repo)

	repo.reports["rpt-1"] = &Report{ID: "rpt-1", TenantID: "tenant-a", Title: "Report A"}
	repo.reports["rpt-2"] = &Report{ID: "rpt-2", TenantID: "tenant-a", Title: "Report B"}
	repo.reports["rpt-3"] = &Report{ID: "rpt-3", TenantID: "tenant-b", Title: "Report C"}

	req := httptest.NewRequest("GET", "/api/v1/compliance/reports", nil)
	ctx := tenancy.WithTenant(req.Context(), "tenant-a")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ListReports(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 2 {
		t.Errorf("expected 2 reports for tenant-a, got %d", len(data))
	}
}

func TestGetReport_NotFound(t *testing.T) {
	repo := NewReportRepository()
	handler := NewReportHandler(repo)

	req := httptest.NewRequest("GET", "/api/v1/compliance/reports/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	ctx := tenancy.WithTenant(req.Context(), "tenant-a")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.GetReport(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetReport_CrossTenantDenied(t *testing.T) {
	repo := NewReportRepository()
	handler := NewReportHandler(repo)

	repo.reports["rpt-1"] = &Report{ID: "rpt-1", TenantID: "tenant-a", Title: "Secret Report"}

	req := httptest.NewRequest("GET", "/api/v1/compliance/reports/rpt-1", nil)
	req.SetPathValue("id", "rpt-1")
	ctx := tenancy.WithTenant(req.Context(), "tenant-b")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.GetReport(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-tenant access, got %d", w.Code)
	}
}

func TestGenerateSections_FedRAMP(t *testing.T) {
	profile := FedRAMPModerateProfile()
	sections := generateSections(profile)

	hasFIPS := false
	hasCM := false
	for _, s := range sections {
		if s.Title == "FIPS 140-2 Compliance" {
			hasFIPS = true
		}
		if s.Title == "Continuous Monitoring" {
			hasCM = true
		}
	}

	if !hasFIPS {
		t.Error("FedRAMP report should include FIPS 140-2 section")
	}
	if !hasCM {
		t.Error("FedRAMP report should include Continuous Monitoring section")
	}
}

func TestGenerateSections_GDPR(t *testing.T) {
	profile := EUGDPRProfile()
	sections := generateSections(profile)

	for _, s := range sections {
		if s.Title == "FIPS 140-2 Compliance" {
			t.Error("GDPR report should not include FIPS section")
		}
	}

	if len(sections) != 4 {
		t.Errorf("GDPR report should have 4 sections (no FIPS, no CM), got %d", len(sections))
	}
}
