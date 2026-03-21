package compliance

import (
	"testing"
)

func TestAllProfiles(t *testing.T) {
	profiles := AllProfiles()
	expected := []string{"gcc-sa", "gcc-ae", "gcc-qa", "gov-tr", "eu-gdpr", "fedramp-moderate"}
	for _, id := range expected {
		if _, ok := profiles[id]; !ok {
			t.Errorf("missing profile: %s", id)
		}
	}
}

func TestGetProfile(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"gcc-sa", true},
		{"gcc-ae", true},
		{"gcc-qa", true},
		{"gov-tr", true},
		{"eu-gdpr", true},
		{"fedramp-moderate", true},
		{"nonexistent", false},
	}

	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			p := GetProfile(tc.id)
			if (p != nil) != tc.expected {
				t.Errorf("GetProfile(%q): got %v, want exists=%v", tc.id, p, tc.expected)
			}
		})
	}
}

func TestGCCProfiles(t *testing.T) {
	profiles := []struct {
		name    string
		profile *Profile
		region  string
	}{
		{"Saudi", GCCSaudiProfile(), "sa-riyadh-1"},
		{"UAE", GCCUAEProfile(), "ae-abudhabi-1"},
		{"Qatar", GCCQatarProfile(), "qa-doha-1"},
	}

	for _, tc := range profiles {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.profile.PIIScrubEnabled {
				t.Error("GCC profiles must have PII scrub enabled")
			}
			if tc.profile.DefaultDataTier != 3 {
				t.Errorf("GCC profiles should default to Tier 3, got %d", tc.profile.DefaultDataTier)
			}
			if !tc.profile.AirGapCapable {
				t.Error("GCC profiles must be air-gap capable")
			}
			if tc.profile.DefaultIsolation != TierC {
				t.Errorf("GCC profiles should default to Tier C isolation, got %s", tc.profile.DefaultIsolation)
			}
			found := false
			for _, r := range tc.profile.StorageRegions {
				if r == tc.region {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected region %q in storage regions", tc.region)
			}
		})
	}
}

func TestGovTurkeyProfile(t *testing.T) {
	p := GovTurkeyProfile()
	if !p.RightToErasure {
		t.Error("Turkey (KVKK) should support right to erasure")
	}
	if p.DefaultDataTier != 3 {
		t.Errorf("expected tier 3, got %d", p.DefaultDataTier)
	}
}

func TestFedRAMPProfile(t *testing.T) {
	p := FedRAMPModerateProfile()
	if !p.FIPSRequired {
		t.Error("FedRAMP should require FIPS")
	}
	if !p.ContinuousMonitor {
		t.Error("FedRAMP should require continuous monitoring")
	}
}
