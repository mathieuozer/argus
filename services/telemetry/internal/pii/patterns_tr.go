package pii

import "regexp"

// TurkeyPatterns returns PII patterns specific to Turkey.
func TurkeyPatterns() []*Pattern {
	return []*Pattern{
		{
			Name:    "tc_kimlik",
			Regex:   regexp.MustCompile(`\b[1-9]\d{10}\b`),
			Replace: "[TC_KIMLIK_REDACTED]",
		},
		{
			Name:    "turkish_iban",
			Regex:   regexp.MustCompile(`\bTR\d{2}\d{5}[A-Z0-9]{17}\b`),
			Replace: "[TR_IBAN_REDACTED]",
		},
		{
			Name:    "turkish_phone",
			Regex:   regexp.MustCompile(`\+?90\d{10}\b`),
			Replace: "[TR_PHONE_REDACTED]",
		},
		{
			Name:    "turkish_tax_id",
			Regex:   regexp.MustCompile(`\b\d{10}\b`),
			Replace: "[TR_TAX_ID_REDACTED]",
		},
	}
}
