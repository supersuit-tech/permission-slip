package notify

import "strings"

// SMSRegion identifies a destination region for SMS pricing.
type SMSRegion string

const (
	SMSRegionUSCA          SMSRegion = "us_ca"
	SMSRegionUKEU          SMSRegion = "uk_eu"
	SMSRegionInternational SMSRegion = "international"
)

// SMSRegionForPhone returns the billing region for a phone number based on
// its international dialing prefix. Numbers should be in E.164 format
// (e.g. "+15551234567"). This is a best-effort mapping; unrecognised prefixes
// default to "international".
func SMSRegionForPhone(phone string) SMSRegion {
	if !strings.HasPrefix(phone, "+") {
		return SMSRegionInternational
	}

	// US/Canada: +1
	if strings.HasPrefix(phone, "+1") {
		return SMSRegionUSCA
	}

	// UK: +44
	if strings.HasPrefix(phone, "+44") {
		return SMSRegionUKEU
	}

	// EU country codes (common ones)
	euPrefixes := []string{
		"+33", // France
		"+49", // Germany
		"+39", // Italy
		"+34", // Spain
		"+31", // Netherlands
		"+32", // Belgium
		"+43", // Austria
		"+41", // Switzerland
		"+46", // Sweden
		"+47", // Norway
		"+45", // Denmark
		"+358", // Finland
		"+353", // Ireland
		"+351", // Portugal
		"+48", // Poland
		"+420", // Czech Republic
		"+36", // Hungary
		"+40", // Romania
		"+30", // Greece
	}

	for _, prefix := range euPrefixes {
		if strings.HasPrefix(phone, prefix) {
			return SMSRegionUKEU
		}
	}

	return SMSRegionInternational
}
