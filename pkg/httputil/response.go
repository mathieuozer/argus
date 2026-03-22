package httputil

import (
	"encoding/json"
	"net/http"
	"reflect"
)

// WriteJSON writes a successful JSON API response with the standard envelope.
// If tenantID is provided, it is included in the meta field.
func WriteJSON(w http.ResponseWriter, status int, data interface{}, tenantID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	meta := map[string]string{}
	if tenantID != "" {
		meta["tenant_id"] = tenantID
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": EnsureNotNil(data),
		"meta": meta,
	})
}

// WriteError writes an error JSON response with the standard error envelope.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// EnsureNotNil converts nil values and nil slices to an empty slice
// so that JSON encoding produces [] instead of null.
func EnsureNotNil(v interface{}) interface{} {
	if v == nil {
		return []interface{}{}
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		return []interface{}{}
	}
	return v
}
