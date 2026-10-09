package nahan

// CountryProfile defines a country target for Nahan mode.
type CountryProfile struct {
	Code           string   // ISO country code, e.g., "EG", "NG"
	Name           string   // Human-readable name
	PreferredColos []string // Cloudflare colo codes preferred for this country
	MinConfidence  float64  // Minimum confidence to assign this country
}

// DefaultProfiles returns the built-in country profiles for Nahan.
func DefaultProfiles() map[string]CountryProfile {
	return map[string]CountryProfile{
		"EG": {
			Code:           "EG",
			Name:           "Egypt",
			PreferredColos: []string{"CAI", "ALEX", "HRG", "SSH", "LXR", "ASW"},
			MinConfidence:  0.6,
		},
		"NG": {
			Code:           "NG",
			Name:           "Nigeria",
			PreferredColos: []string{"LOS", "ABV", "PHC", "KAN", "ENU", "IBA"},
			MinConfidence:  0.6,
		},
	}
}

// GetProfile returns a profile by code, or nil if not found.
func GetProfile(code string, profiles map[string]CountryProfile) *CountryProfile {
	if p, ok := profiles[code]; ok {
		return &p
	}
	return nil
}

// AllCodes returns all profile codes.
func AllCodes(profiles map[string]CountryProfile) []string {
	codes := make([]string, 0, len(profiles))
	for code := range profiles {
		codes = append(codes, code)
	}
	return codes
}