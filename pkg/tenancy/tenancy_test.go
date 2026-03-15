package tenancy

import (
	"context"
	"testing"
)

func TestFromContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantID  string
		wantErr bool
	}{
		{
			name:    "valid tenant",
			ctx:     WithTenant(context.Background(), "tenant-1"),
			wantID:  "tenant-1",
			wantErr: false,
		},
		{
			name:    "missing tenant",
			ctx:     context.Background(),
			wantID:  "",
			wantErr: true,
		},
		{
			name:    "empty tenant",
			ctx:     WithTenant(context.Background(), ""),
			wantID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromContext(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromContext() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantID {
				t.Errorf("FromContext() = %v, want %v", got, tt.wantID)
			}
		})
	}
}
