package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		data     interface{}
		tenantID string
		wantData interface{}
		wantMeta map[string]string
	}{
		{
			name:     "with tenant ID",
			status:   http.StatusOK,
			data:     map[string]string{"key": "value"},
			tenantID: "tenant-1",
			wantData: map[string]interface{}{"key": "value"},
			wantMeta: map[string]string{"tenant_id": "tenant-1"},
		},
		{
			name:     "without tenant ID",
			status:   http.StatusOK,
			data:     []string{"a", "b"},
			tenantID: "",
			wantData: []interface{}{"a", "b"},
			wantMeta: map[string]string{},
		},
		{
			name:     "nil data becomes empty array",
			status:   http.StatusOK,
			data:     nil,
			tenantID: "t1",
			wantData: []interface{}{},
			wantMeta: map[string]string{"tenant_id": "t1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSON(w, tc.status, tc.data, tc.tenantID)

			if w.Code != tc.status {
				t.Errorf("status = %d, want %d", w.Code, tc.status)
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if _, ok := resp["data"]; !ok {
				t.Error("response missing 'data' field")
			}
			if _, ok := resp["meta"]; !ok {
				t.Error("response missing 'meta' field")
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errField, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing 'error' field")
	}
	if errField["code"] != "NOT_FOUND" {
		t.Errorf("error code = %q, want NOT_FOUND", errField["code"])
	}
	if errField["message"] != "resource not found" {
		t.Errorf("error message = %q, want 'resource not found'", errField["message"])
	}
}

func TestEnsureNotNil(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want bool // true = should be empty slice
	}{
		{"nil interface", nil, true},
		{"nil string slice", ([]string)(nil), true},
		{"empty string slice", []string{}, false},
		{"non-nil slice", []string{"a"}, false},
		{"map value", map[string]int{"a": 1}, false},
		{"string value", "hello", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := EnsureNotNil(tc.in)
			if tc.want {
				arr, ok := result.([]interface{})
				if !ok || len(arr) != 0 {
					t.Errorf("expected empty []interface{}, got %T %v", result, result)
				}
			} else {
				if result == nil {
					t.Error("expected non-nil result")
				}
			}
		})
	}
}
