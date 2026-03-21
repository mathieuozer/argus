package compliance

import "time"

// IsolationTier represents the infrastructure isolation level.
type IsolationTier string

const (
	TierA IsolationTier = "A" // Shared (logical isolation via RLS)
	TierB IsolationTier = "B" // Dedicated namespace
	TierC IsolationTier = "C" // Dedicated deployment
)

// Profile defines a compliance profile with all its requirements.
type Profile struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	StorageRegions    []string      `json:"storage_regions"`
	PIIScrubEnabled   bool          `json:"pii_scrub_enabled"`
	PIIProfile        string        `json:"pii_profile"`
	DefaultDataTier   int           `json:"default_data_tier"`
	AuditRetention    time.Duration `json:"audit_retention"`
	AirGapCapable     bool          `json:"air_gap_capable"`
	DefaultIsolation  IsolationTier `json:"default_isolation"`
	RequiredCerts     []string      `json:"required_certs"`
	RightToErasure    bool          `json:"right_to_erasure"`
	FIPSRequired      bool          `json:"fips_required"`
	ContinuousMonitor bool          `json:"continuous_monitor"`
}

// AllProfiles returns all available compliance profiles.
func AllProfiles() map[string]*Profile {
	return map[string]*Profile{
		"gcc-sa":           GCCSaudiProfile(),
		"gcc-ae":           GCCUAEProfile(),
		"gcc-qa":           GCCQatarProfile(),
		"gov-tr":           GovTurkeyProfile(),
		"eu-gdpr":          EUGDPRProfile(),
		"fedramp-moderate": FedRAMPModerateProfile(),
	}
}

// GetProfile returns a compliance profile by ID.
func GetProfile(id string) *Profile {
	profiles := AllProfiles()
	return profiles[id]
}

// GCCSaudiProfile returns the Saudi Arabia government compliance profile.
func GCCSaudiProfile() *Profile {
	return &Profile{
		ID:                "gcc-sa",
		Name:              "Saudi Arabia (NDMO)",
		Description:       "Saudi National Data Management Office compliance",
		StorageRegions:    []string{"sa-riyadh-1", "sa-jeddah-1"},
		PIIScrubEnabled:   true,
		PIIProfile:        "gcc-sa",
		DefaultDataTier:   3,
		AuditRetention:    5 * 365 * 24 * time.Hour, // 5 years
		AirGapCapable:     true,
		DefaultIsolation:  TierC,
		RequiredCerts:     []string{"NDMO-classification", "NCA-ECC"},
		RightToErasure:    false,
		FIPSRequired:      false,
		ContinuousMonitor: true,
	}
}

// GCCUAEProfile returns the UAE government compliance profile.
func GCCUAEProfile() *Profile {
	return &Profile{
		ID:                "gcc-ae",
		Name:              "UAE (NESA/IAS)",
		Description:       "UAE National Electronic Security Authority compliance",
		StorageRegions:    []string{"ae-abudhabi-1", "ae-dubai-1"},
		PIIScrubEnabled:   true,
		PIIProfile:        "gcc-ae",
		DefaultDataTier:   3,
		AuditRetention:    5 * 365 * 24 * time.Hour,
		AirGapCapable:     true,
		DefaultIsolation:  TierC,
		RequiredCerts:     []string{"NESA-IAS", "Abu-Dhabi-ADSIC"},
		RightToErasure:    false,
		FIPSRequired:      false,
		ContinuousMonitor: true,
	}
}

// GCCQatarProfile returns the Qatar government compliance profile.
func GCCQatarProfile() *Profile {
	return &Profile{
		ID:                "gcc-qa",
		Name:              "Qatar (NIA)",
		Description:       "Qatar National Information Assurance compliance",
		StorageRegions:    []string{"qa-doha-1"},
		PIIScrubEnabled:   true,
		PIIProfile:        "gcc-qa",
		DefaultDataTier:   3,
		AuditRetention:    7 * 365 * 24 * time.Hour, // 7 years
		AirGapCapable:     true,
		DefaultIsolation:  TierC,
		RequiredCerts:     []string{"NIA-policy", "Q-CERT"},
		RightToErasure:    false,
		FIPSRequired:      false,
		ContinuousMonitor: true,
	}
}

// GovTurkeyProfile returns the Turkey government compliance profile.
func GovTurkeyProfile() *Profile {
	return &Profile{
		ID:                "gov-tr",
		Name:              "Turkey Government (KVKK)",
		Description:       "Turkish Personal Data Protection Law compliance",
		StorageRegions:    []string{"tr-east-1", "tr-west-1"},
		PIIScrubEnabled:   true,
		PIIProfile:        "gov-tr",
		DefaultDataTier:   3,
		AuditRetention:    5 * 365 * 24 * time.Hour,
		AirGapCapable:     true,
		DefaultIsolation:  TierC,
		RequiredCerts:     []string{"TS-ISO-27001", "KVKK-VERB\u0130S"},
		RightToErasure:    true,
		FIPSRequired:      false,
		ContinuousMonitor: false,
	}
}

// EUGDPRProfile returns the EU GDPR compliance profile.
func EUGDPRProfile() *Profile {
	return &Profile{
		ID:                "eu-gdpr",
		Name:              "EU GDPR",
		Description:       "EU General Data Protection Regulation compliance",
		StorageRegions:    []string{"eu-west-1", "eu-central-1", "eu-north-1"},
		PIIScrubEnabled:   true,
		PIIProfile:        "eu-gdpr",
		DefaultDataTier:   2,
		AuditRetention:    3 * 365 * 24 * time.Hour,
		AirGapCapable:     false,
		DefaultIsolation:  TierB,
		RequiredCerts:     []string{"ISO-27001", "SOC2-Type2"},
		RightToErasure:    true,
		FIPSRequired:      false,
		ContinuousMonitor: false,
	}
}

// FedRAMPModerateProfile returns the US FedRAMP Moderate compliance profile.
func FedRAMPModerateProfile() *Profile {
	return &Profile{
		ID:                "fedramp-moderate",
		Name:              "FedRAMP Moderate",
		Description:       "US Federal Risk and Authorization Management Program - Moderate",
		StorageRegions:    []string{"us-gov-west-1", "us-gov-east-1"},
		PIIScrubEnabled:   true,
		PIIProfile:        "fedramp-moderate",
		DefaultDataTier:   2,
		AuditRetention:    3 * 365 * 24 * time.Hour,
		AirGapCapable:     true,
		DefaultIsolation:  TierB,
		RequiredCerts:     []string{"FedRAMP-ATO", "FIPS-140-2"},
		RightToErasure:    false,
		FIPSRequired:      true,
		ContinuousMonitor: true,
	}
}
