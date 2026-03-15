package errors

import (
	"fmt"
	"testing"
)

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		wantCode Code
		wantMsg  string
	}{
		{
			name:     "simple error",
			err:      New(CodeAgentNotFound, "agent xyz not found"),
			wantCode: CodeAgentNotFound,
			wantMsg:  "AGENT_NOT_FOUND: agent xyz not found",
		},
		{
			name:     "wrapped error",
			err:      Wrap(CodeInternal, "db failure", fmt.Errorf("connection refused")),
			wantCode: CodeInternal,
			wantMsg:  "INTERNAL_ERROR: db failure: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("got %q, want %q", tt.err.Error(), tt.wantMsg)
			}
			if !Is(tt.err, tt.wantCode) {
				t.Errorf("Is(%v, %v) = false, want true", tt.err, tt.wantCode)
			}
		})
	}
}
