package pii

import "regexp"

// SaudiPatterns returns PII patterns specific to Saudi Arabia.
func SaudiPatterns() []*Pattern {
	return []*Pattern{
		{
			Name:    "saudi_national_id",
			Regex:   regexp.MustCompile(`\b[12]\d{9}\b`),
			Replace: "[SAUDI_ID_REDACTED]",
		},
		{
			Name:    "saudi_iban",
			Regex:   regexp.MustCompile(`\bSA\d{2}\d{2}\d{18}\b`),
			Replace: "[SAUDI_IBAN_REDACTED]",
		},
		{
			Name:    "saudi_phone",
			Regex:   regexp.MustCompile(`\+?966\d{8,9}\b`),
			Replace: "[SAUDI_PHONE_REDACTED]",
		},
	}
}

// UAEPatterns returns PII patterns specific to the UAE.
func UAEPatterns() []*Pattern {
	return []*Pattern{
		{
			Name:    "emirates_id",
			Regex:   regexp.MustCompile(`\b784-\d{4}-\d{7}-\d\b`),
			Replace: "[EMIRATES_ID_REDACTED]",
		},
		{
			Name:    "uae_iban",
			Regex:   regexp.MustCompile(`\bAE\d{2}\d{3}\d{16}\b`),
			Replace: "[UAE_IBAN_REDACTED]",
		},
		{
			Name:    "uae_phone",
			Regex:   regexp.MustCompile(`\+?971\d{8,9}\b`),
			Replace: "[UAE_PHONE_REDACTED]",
		},
	}
}

// QatarPatterns returns PII patterns specific to Qatar.
func QatarPatterns() []*Pattern {
	return []*Pattern{
		{
			Name:    "qatar_id",
			Regex:   regexp.MustCompile(`\b\d{11}\b`),
			Replace: "[QATAR_ID_REDACTED]",
		},
		{
			Name:    "qatar_iban",
			Regex:   regexp.MustCompile(`\bQA\d{2}[A-Z]{4}\d{21}\b`),
			Replace: "[QATAR_IBAN_REDACTED]",
		},
		{
			Name:    "qatar_phone",
			Regex:   regexp.MustCompile(`\+?974\d{8}\b`),
			Replace: "[QATAR_PHONE_REDACTED]",
		},
	}
}

// GCCPhonePatterns returns phone patterns for all GCC countries.
func GCCPhonePatterns() []*Pattern {
	return []*Pattern{
		{
			Name:    "gcc_phone",
			Regex:   regexp.MustCompile(`\+?(?:966|971|974)\d{8,9}\b`),
			Replace: "[GCC_PHONE_REDACTED]",
		},
	}
}

// ArabicNamePattern returns a pattern for detecting Arabic script names.
func ArabicNamePattern() *Pattern {
	return &Pattern{
		Name:    "arabic_name",
		Regex:   regexp.MustCompile(`[\x{0600}-\x{06FF}]{2,}(?:\s+[\x{0600}-\x{06FF}]{2,})+`),
		Replace: "[ARABIC_NAME_REDACTED]",
	}
}
