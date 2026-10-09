package nahan

import (
	"context"
	"net"
	"strings"

	"github.com/matinsenpai/senpaiscanner/internal/result"
)

// Classifier assigns countries to endpoints based on available metadata.
// Priority: Colo (primary) → ASN hints (fallback) → UNKNOWN
type Classifier struct {
	Profiles     map[string]CountryProfile
	ASNClient    ASNClient         // optional, for IP -> ASN/org lookup
	ASNClientCtx ASNClientWithContext // optional, for IP -> ASN/org lookup (with context)
}

// ASNClient is an interface for ASN/org lookups.
type ASNClient interface {
	ASN(ip net.IP) (int, string, error) // asn, org, error
}

// ASNClientWithContext is an interface for ASN lookups with context support.
type ASNClientWithContext interface {
	ASN(ctx context.Context, ip net.IP) (int, string, error)
}

// ClassificationResult holds the country assignment result.
type ClassificationResult struct {
	Country     string
	Confidence  float64
	Method      string // "colo", "asn", "unknown"
	ASN         int
	ISP         string
}

// Classify assigns a country to a result based on available metadata.
func (c *Classifier) Classify(r *result.Result) ClassificationResult {
	return c.ClassifyWithContext(context.Background(), r)
}

// ClassifyWithContext assigns a country to a result with context support.
// Priority: 1. Colo (0.7) → 2. ASN hints (0.5) → 3. UNKNOWN
func (c *Classifier) ClassifyWithContext(ctx context.Context, r *result.Result) ClassificationResult {
	ip := r.IP

	// 1. Colo-based classification (primary for Nahan)
	// Cloudflare colo indicates the PoP serving the target region.
	// CAI/ALEX/HRG → Egypt, LOS/ABV/PHC → Nigeria
	if r.Colo != "" {
		for code, p := range c.Profiles {
			for _, preferred := range p.PreferredColos {
				if r.Colo == preferred {
					return ClassificationResult{
						Country:    code,
						Confidence: 0.7,
						Method:     "colo",
					}
				}
			}
		}
	}

	// 2. ASN/org-based classification (fallback)
	if c.ASNClientCtx != nil {
		if asn, org, err := c.ASNClientCtx.ASN(ctx, ip); err == nil {
			for code := range c.Profiles {
				if containsCountryHint(org, code) {
					return ClassificationResult{
						Country:    code,
						Confidence: 0.5,
						Method:     "asn",
						ASN:        asn,
						ISP:        org,
					}
				}
			}
			return ClassificationResult{
				Country:    "UNKNOWN",
				Confidence: 0.3,
				Method:     "asn",
				ASN:        asn,
				ISP:        org,
			}
		}
	} else if c.ASNClient != nil {
		if asn, org, err := c.ASNClient.ASN(ip); err == nil {
			for code := range c.Profiles {
				if containsCountryHint(org, code) {
					return ClassificationResult{
						Country:    code,
						Confidence: 0.5,
						Method:     "asn",
						ASN:        asn,
						ISP:        org,
					}
				}
			}
			return ClassificationResult{
				Country:    "UNKNOWN",
				Confidence: 0.3,
				Method:     "asn",
				ASN:        asn,
				ISP:        org,
			}
		}
	}

	// 3. Unknown
	return ClassificationResult{
		Country:    "UNKNOWN",
		Confidence: 0.0,
		Method:     "unknown",
	}
}

// containsCountryHint checks if org name contains hints for the country.
func containsCountryHint(org, countryCode string) bool {
	orgLower := strings.ToLower(org)
	hints := map[string][]string{
		"EG": {"egypt", "misr", "telecom egypt", "etisalat egypt", "vodafone egypt", "orange egypt"},
		"NG": {"nigeria", "mtn nigeria", "airtel nigeria", "glo nigeria", "9mobile nigeria", "nitel", "ipnx"},
	}
	if list, ok := hints[countryCode]; ok {
		for _, hint := range list {
			if strings.Contains(orgLower, hint) {
				return true
			}
		}
	}
	return false
}