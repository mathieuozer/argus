package pii

import (
	"testing"
)

func TestPatternsForProfile(t *testing.T) {
	tests := []struct {
		name         string
		profile      string
		minPatterns  int
		checkPattern string
	}{
		{"default profile", "", 1, ""},
		{"Saudi profile", "gcc-sa", 4, "saudi_national_id"},
		{"UAE profile", "gcc-ae", 4, "emirates_id"},
		{"Qatar profile", "gcc-qa", 4, "qatar_id"},
		{"Turkey profile", "gov-tr", 4, "tc_kimlik"},
		{"EU GDPR profile", "eu-gdpr", 1, ""},
		{"FedRAMP profile", "fedramp-moderate", 1, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			patterns := PatternsForProfile(tc.profile)
			if len(patterns) < tc.minPatterns {
				t.Errorf("expected at least %d patterns for profile %q, got %d", tc.minPatterns, tc.profile, len(patterns))
			}
			if tc.checkPattern != "" {
				found := false
				for _, p := range patterns {
					if p.Name == tc.checkPattern {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected pattern %q in profile %q", tc.checkPattern, tc.profile)
				}
			}
		})
	}
}

func TestSaudiPatterns(t *testing.T) {
	patterns := SaudiPatterns()

	tests := []struct {
		name    string
		pattern string
		input   string
		match   bool
	}{
		{"saudi id matches", "saudi_national_id", "1234567890", true},
		{"saudi id starts with 2", "saudi_national_id", "2987654321", true},
		{"saudi id starts with 3 no match", "saudi_national_id", "3234567890", false},
		{"saudi iban matches", "saudi_iban", "SA0380000000608010167519", true},
		{"saudi phone matches", "saudi_phone", "+966512345678", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var p *Pattern
			for _, pat := range patterns {
				if pat.Name == tc.pattern {
					p = pat
					break
				}
			}
			if p == nil {
				t.Fatalf("pattern %q not found", tc.pattern)
			}
			matched := p.Regex.MatchString(tc.input)
			if matched != tc.match {
				t.Errorf("pattern %q on %q: got %v, want %v", tc.pattern, tc.input, matched, tc.match)
			}
		})
	}
}

func TestUAEPatterns(t *testing.T) {
	patterns := UAEPatterns()

	tests := []struct {
		name    string
		pattern string
		input   string
		match   bool
	}{
		{"emirates id matches", "emirates_id", "784-1234-5678901-2", true},
		{"emirates id wrong format", "emirates_id", "123-4567-8901234-5", false},
		{"uae iban matches", "uae_iban", "AE070331234567890123456", true},
		{"uae phone matches", "uae_phone", "+971501234567", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var p *Pattern
			for _, pat := range patterns {
				if pat.Name == tc.pattern {
					p = pat
					break
				}
			}
			if p == nil {
				t.Fatalf("pattern %q not found", tc.pattern)
			}
			matched := p.Regex.MatchString(tc.input)
			if matched != tc.match {
				t.Errorf("pattern %q on %q: got %v, want %v", tc.pattern, tc.input, matched, tc.match)
			}
		})
	}
}

func TestTurkeyPatterns(t *testing.T) {
	patterns := TurkeyPatterns()

	tests := []struct {
		name    string
		pattern string
		input   string
		match   bool
	}{
		{"tc kimlik matches", "tc_kimlik", "12345678901", true},
		{"tc kimlik starts with 0 no match", "tc_kimlik", "02345678901", false},
		{"turkish iban matches", "turkish_iban", "TR330006100519786457841326", true},
		{"turkish phone matches", "turkish_phone", "+905321234567", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var p *Pattern
			for _, pat := range patterns {
				if pat.Name == tc.pattern {
					p = pat
					break
				}
			}
			if p == nil {
				t.Fatalf("pattern %q not found", tc.pattern)
			}
			matched := p.Regex.MatchString(tc.input)
			if matched != tc.match {
				t.Errorf("pattern %q on %q: got %v, want %v", tc.pattern, tc.input, matched, tc.match)
			}
		})
	}
}

func TestArabicNamePattern(t *testing.T) {
	p := ArabicNamePattern()
	tests := []struct {
		input string
		match bool
	}{
		{"\u0645\u062d\u0645\u062f \u0623\u062d\u0645\u062f", true},
		{"hello world", false},
		{"\u0639\u0628\u062f \u0627\u0644\u0644\u0647", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			matched := p.Regex.MatchString(tc.input)
			if matched != tc.match {
				t.Errorf("arabic name on %q: got %v, want %v", tc.input, matched, tc.match)
			}
		})
	}
}

func TestAllGCCPatterns(t *testing.T) {
	patterns := AllGCCPatterns()
	if len(patterns) < 9 {
		t.Errorf("expected at least 9 GCC patterns, got %d", len(patterns))
	}
}
