package pii

import (
	"testing"
)

func TestScrub(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "scrubs email address",
			input: "Contact user at john.doe@example.com for details",
			want:  "Contact user at [EMAIL_REDACTED] for details",
		},
		{
			name:  "scrubs multiple emails",
			input: "Send to alice@test.org and bob@corp.co.uk",
			want:  "Send to [EMAIL_REDACTED] and [EMAIL_REDACTED]",
		},
		{
			name:  "scrubs IP address",
			input: "Request from 192.168.1.100 was blocked",
			want:  "Request from [IP_REDACTED] was blocked",
		},
		{
			name:  "scrubs credit card with dashes",
			input: "Card: 4111-1111-1111-1111 is on file",
			want:  "Card: [CC_REDACTED] is on file",
		},
		{
			name:  "scrubs credit card without dashes",
			input: "Card: 4111111111111111 is on file",
			want:  "Card: [CC_REDACTED] is on file",
		},
		{
			name:  "no PII leaves text unchanged",
			input: "Normal log message with no sensitive data",
			want:  "Normal log message with no sensitive data",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "scrubs email and IP together",
			input: "User john@example.com from 10.0.0.1 logged in",
			want:  "User [EMAIL_REDACTED] from [IP_REDACTED] logged in",
		},
	}

	scrubber := New()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scrubber.Scrub(tc.input)
			if got != tc.want {
				t.Errorf("Scrub(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestScrubMap(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]string
		want  map[string]string
	}{
		{
			name: "scrubs PII from all map values",
			input: map[string]string{
				"email":   "user@example.com",
				"message": "Hello from 192.168.1.1",
				"safe":    "no PII here",
			},
			want: map[string]string{
				"email":   "[EMAIL_REDACTED]",
				"message": "Hello from [IP_REDACTED]",
				"safe":    "no PII here",
			},
		},
		{
			name:  "empty map",
			input: map[string]string{},
			want:  map[string]string{},
		},
	}

	scrubber := New()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scrubber.ScrubMap(tc.input)

			if len(got) != len(tc.want) {
				t.Fatalf("ScrubMap returned %d entries, want %d", len(got), len(tc.want))
			}

			for key, wantVal := range tc.want {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("key %q missing from result", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("key %q: got %q, want %q", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestContainsPII(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "detects email",
			input: "contact alice@example.com",
			want:  true,
		},
		{
			name:  "detects IP address",
			input: "from host 10.0.0.1",
			want:  true,
		},
		{
			name:  "detects credit card",
			input: "card 4111 1111 1111 1111",
			want:  true,
		},
		{
			name:  "no PII returns false",
			input: "regular text with no sensitive data",
			want:  false,
		},
		{
			name:  "empty string returns false",
			input: "",
			want:  false,
		},
	}

	scrubber := New()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scrubber.ContainsPII(tc.input)
			if got != tc.want {
				t.Errorf("ContainsPII(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestRedact(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "whitespace only", input: "   "},
		{name: "with email", input: "test@example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Redact(tc.input)
			if tc.input == "" || tc.input == "   " {
				if got != tc.input {
					t.Errorf("Redact(%q) = %q, want %q", tc.input, got, tc.input)
				}
			} else {
				scrubber := New()
				want := scrubber.Scrub(tc.input)
				if got != want {
					t.Errorf("Redact(%q) = %q, want %q", tc.input, got, want)
				}
			}
		})
	}
}
