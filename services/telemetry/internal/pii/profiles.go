package pii

// PatternsForProfile returns PII patterns for a given compliance profile.
func PatternsForProfile(profile string) []*Pattern {
	base := DefaultPatterns()
	switch profile {
	case "gcc-sa":
		return append(base, SaudiPatterns()...)
	case "gcc-ae":
		return append(base, UAEPatterns()...)
	case "gcc-qa":
		return append(base, QatarPatterns()...)
	case "gov-tr":
		return append(base, TurkeyPatterns()...)
	case "eu-gdpr":
		return base // base already includes EU patterns
	case "fedramp-moderate":
		return base // base patterns cover US requirements
	default:
		return base
	}
}

// AllGCCPatterns returns PII patterns for all GCC countries combined.
func AllGCCPatterns() []*Pattern {
	var patterns []*Pattern
	patterns = append(patterns, SaudiPatterns()...)
	patterns = append(patterns, UAEPatterns()...)
	patterns = append(patterns, QatarPatterns()...)
	return patterns
}
