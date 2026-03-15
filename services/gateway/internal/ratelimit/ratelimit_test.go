package ratelimit

import (
	"testing"
	"time"
)

func TestAllow(t *testing.T) {
	t.Run("within limit", func(t *testing.T) {
		tests := []struct {
			name        string
			maxRequests int
			window      time.Duration
			key         string
			numRequests int
			wantAllAll  bool
		}{
			{
				name:        "single request within limit",
				maxRequests: 5,
				window:      1 * time.Minute,
				key:         "192.168.1.1",
				numRequests: 1,
				wantAllAll:  true,
			},
			{
				name:        "exactly at limit",
				maxRequests: 3,
				window:      1 * time.Minute,
				key:         "10.0.0.1",
				numRequests: 3,
				wantAllAll:  true,
			},
			{
				name:        "multiple keys do not interfere",
				maxRequests: 2,
				window:      1 * time.Minute,
				key:         "key-a",
				numRequests: 2,
				wantAllAll:  true,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				limiter := New(tc.maxRequests, tc.window)

				for i := 0; i < tc.numRequests; i++ {
					allowed := limiter.Allow(tc.key)
					if tc.wantAllAll && !allowed {
						t.Errorf("request %d: expected allowed, got denied", i+1)
					}
				}
			})
		}
	})

	t.Run("exceeds limit", func(t *testing.T) {
		tests := []struct {
			name        string
			maxRequests int
			window      time.Duration
			key         string
		}{
			{
				name:        "exceeds by one",
				maxRequests: 3,
				window:      1 * time.Minute,
				key:         "192.168.1.1",
			},
			{
				name:        "single request limit",
				maxRequests: 1,
				window:      1 * time.Minute,
				key:         "10.0.0.1",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				limiter := New(tc.maxRequests, tc.window)

				// Use up all allowed requests
				for i := 0; i < tc.maxRequests; i++ {
					allowed := limiter.Allow(tc.key)
					if !allowed {
						t.Fatalf("request %d: expected allowed before limit, got denied", i+1)
					}
				}

				// The next request should be denied
				allowed := limiter.Allow(tc.key)
				if allowed {
					t.Error("expected request to be denied after exceeding limit")
				}
			})
		}
	})

	t.Run("window reset allows new requests", func(t *testing.T) {
		// Use a very short window so it resets quickly
		limiter := New(1, 10*time.Millisecond)
		key := "reset-test"

		// First request should be allowed
		if !limiter.Allow(key) {
			t.Fatal("first request should be allowed")
		}

		// Second request should be denied
		if limiter.Allow(key) {
			t.Fatal("second request should be denied before window reset")
		}

		// Wait for the window to expire
		time.Sleep(20 * time.Millisecond)

		// After reset, should be allowed again
		if !limiter.Allow(key) {
			t.Error("request after window reset should be allowed")
		}
	})

	t.Run("different keys are independent", func(t *testing.T) {
		limiter := New(1, 1*time.Minute)

		if !limiter.Allow("key-a") {
			t.Fatal("first request for key-a should be allowed")
		}

		// key-a is now at limit, but key-b should still be allowed
		if !limiter.Allow("key-b") {
			t.Error("first request for key-b should be allowed even when key-a is at limit")
		}

		// key-a should be denied
		if limiter.Allow("key-a") {
			t.Error("second request for key-a should be denied")
		}
	})

	t.Run("first request for new key always allowed", func(t *testing.T) {
		limiter := New(100, 1*time.Minute)

		keys := []string{"192.168.0.1", "10.0.0.1", "172.16.0.1", "user-a", "user-b"}
		for _, key := range keys {
			if !limiter.Allow(key) {
				t.Errorf("first request for key %q should be allowed", key)
			}
		}
	})
}
