package vault

import (
	"testing"
)

func TestInMemoryClientStore(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		data    map[string][]byte
		wantErr bool
	}{
		{
			name: "store single key",
			path: "secret/argus/ca-key",
			data: map[string][]byte{
				"private_key": []byte("-----BEGIN EC PRIVATE KEY-----\nfake\n-----END EC PRIVATE KEY-----"),
			},
			wantErr: false,
		},
		{
			name: "store multiple keys",
			path: "secret/argus/agent-creds",
			data: map[string][]byte{
				"cert": []byte("cert-data"),
				"key":  []byte("key-data"),
			},
			wantErr: false,
		},
		{
			name:    "store empty data",
			path:    "secret/argus/empty",
			data:    map[string][]byte{},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := NewInMemoryClient()

			err := client.Store(tc.path, tc.data)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestInMemoryClientRead(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*InMemoryClient)
		path    string
		wantErr bool
		wantLen int
	}{
		{
			name: "read existing secret",
			setup: func(c *InMemoryClient) {
				_ = c.Store("secret/test", map[string][]byte{
					"key": []byte("value"),
				})
			},
			path:    "secret/test",
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "read nonexistent secret",
			setup:   func(c *InMemoryClient) {},
			path:    "secret/nonexistent",
			wantErr: true,
		},
		{
			name: "read after overwrite",
			setup: func(c *InMemoryClient) {
				_ = c.Store("secret/test", map[string][]byte{"old": []byte("old-value")})
				_ = c.Store("secret/test", map[string][]byte{"new": []byte("new-value")})
			},
			path:    "secret/test",
			wantErr: false,
			wantLen: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := NewInMemoryClient()
			tc.setup(client)

			data, err := client.Read(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(data) != tc.wantLen {
				t.Errorf("expected %d entries, got %d", tc.wantLen, len(data))
			}
		})
	}

	t.Run("read returns correct values", func(t *testing.T) {
		client := NewInMemoryClient()
		original := map[string][]byte{
			"cert": []byte("cert-pem-data"),
			"key":  []byte("key-pem-data"),
		}
		_ = client.Store("secret/agent", original)

		data, err := client.Read("secret/agent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for k, wantV := range original {
			gotV, ok := data[k]
			if !ok {
				t.Errorf("key %q missing from read result", k)
				continue
			}
			if string(gotV) != string(wantV) {
				t.Errorf("key %q: got %q, want %q", k, string(gotV), string(wantV))
			}
		}
	})
}

func TestInMemoryClientDelete(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*InMemoryClient)
		path    string
		wantErr bool
	}{
		{
			name: "delete existing secret",
			setup: func(c *InMemoryClient) {
				_ = c.Store("secret/test", map[string][]byte{"key": []byte("value")})
			},
			path:    "secret/test",
			wantErr: false,
		},
		{
			name:    "delete nonexistent secret is no-op",
			setup:   func(c *InMemoryClient) {},
			path:    "secret/nonexistent",
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := NewInMemoryClient()
			tc.setup(client)

			err := client.Delete(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the secret is gone
			_, readErr := client.Read(tc.path)
			if readErr == nil {
				t.Error("expected error reading deleted secret, got nil")
			}
		})
	}

	t.Run("store, read, delete, read cycle", func(t *testing.T) {
		client := NewInMemoryClient()

		// Store
		err := client.Store("secret/lifecycle", map[string][]byte{"data": []byte("test")})
		if err != nil {
			t.Fatalf("Store error: %v", err)
		}

		// Read - should succeed
		data, err := client.Read("secret/lifecycle")
		if err != nil {
			t.Fatalf("Read after Store error: %v", err)
		}
		if string(data["data"]) != "test" {
			t.Errorf("expected value %q, got %q", "test", string(data["data"]))
		}

		// Delete
		err = client.Delete("secret/lifecycle")
		if err != nil {
			t.Fatalf("Delete error: %v", err)
		}

		// Read - should fail
		_, err = client.Read("secret/lifecycle")
		if err == nil {
			t.Fatal("expected error reading deleted secret, got nil")
		}
	})

	t.Run("implements Client interface", func(t *testing.T) {
		var _ Client = NewInMemoryClient()
	})
}
