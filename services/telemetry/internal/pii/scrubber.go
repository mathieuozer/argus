package pii

import (
	"regexp"
	"strings"
)

// Scrubber removes PII from telemetry data.
type Scrubber struct {
	patterns []*Pattern
}

// Pattern defines a PII detection pattern.
type Pattern struct {
	Name    string
	Regex   *regexp.Regexp
	Replace string
}

// DefaultPatterns returns the default PII patterns.
// Order matters: more specific patterns must come first to prevent
// partial matches by broader patterns (e.g., credit card before phone).
func DefaultPatterns() []*Pattern {
	return []*Pattern{
		{Name: "email", Regex: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`), Replace: "[EMAIL_REDACTED]"},
		{Name: "credit_card", Regex: regexp.MustCompile(`\b\d{4}[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{4}\b`), Replace: "[CC_REDACTED]"},
		{Name: "iban", Regex: regexp.MustCompile(`\b[A-Z]{2}\d{2}[\s]?\d{4}[\s]?\d{4}[\s]?\d{4}[\s]?\d{4}[\s]?\d{0,2}\b`), Replace: "[IBAN_REDACTED]"},
		{Name: "ip_address", Regex: regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), Replace: "[IP_REDACTED]"},
		{Name: "phone", Regex: regexp.MustCompile(`(?:\+?\d{1,3}[\s\-])?\(?\d{2,4}\)?[\s\-]\d{3,4}[\s\-]\d{3,4}`), Replace: "[PHONE_REDACTED]"},
	}
}

// New creates a new PII scrubber with default patterns.
func New() *Scrubber {
	return &Scrubber{patterns: DefaultPatterns()}
}

// NewWithPatterns creates a new PII scrubber with custom patterns.
func NewWithPatterns(patterns []*Pattern) *Scrubber {
	return &Scrubber{patterns: patterns}
}

// Scrub removes PII from the given text.
func (s *Scrubber) Scrub(text string) string {
	result := text
	for _, p := range s.patterns {
		result = p.Regex.ReplaceAllString(result, p.Replace)
	}
	return result
}

// ScrubMap removes PII from all values in a map.
func (s *Scrubber) ScrubMap(data map[string]string) map[string]string {
	result := make(map[string]string, len(data))
	for k, v := range data {
		result[k] = s.Scrub(v)
	}
	return result
}

// ContainsPII checks if text contains any PII patterns.
func (s *Scrubber) ContainsPII(text string) bool {
	for _, p := range s.patterns {
		if p.Regex.MatchString(text) {
			return true
		}
	}
	return false
}

// Redact is a convenience function that fully redacts text if it contains PII.
func Redact(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	return New().Scrub(text)
}
