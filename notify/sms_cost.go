package notify

// sms_cost.go — Destination-based SMS billing region lookup.
// Region constants here align with stripe.SMSRates keys.
//
// NOTE: This lookup is not yet wired into the billing flow. The billing
// endpoint (api/billing.go) currently charges all SMS at the "us_ca" rate.
// A future iteration should pass SMSRegionForPhone(recipient.Phone) to
// CreateSMSInvoiceItem instead of the hardcoded "us_ca" region.

import "strings"

// SMSRegion identifies a destination region for SMS pricing.
// Values match the keys in stripe.SMSRates.
type SMSRegion string

const (
	SMSRegionUSCA          SMSRegion = "us_ca"
	SMSRegionUKEU          SMSRegion = "uk_eu"
	SMSRegionInternational SMSRegion = "international"
)

// euPrefixes lists E.164 dialing codes for EU/EEA countries that map to the
// UK/EU billing region. Package-level to avoid per-call allocation. Longer
// prefixes (3+ digits) are listed first so prefix matching is unambiguous.
var euPrefixes = []string{
	"+420", // Czech Republic
	"+358", // Finland
	"+353", // Ireland
	"+351", // Portugal
	"+33",  // France
	"+49",  // Germany
	"+39",  // Italy
	"+34",  // Spain
	"+31",  // Netherlands
	"+32",  // Belgium
	"+43",  // Austria
	"+41",  // Switzerland
	"+46",  // Sweden
	"+47",  // Norway
	"+45",  // Denmark
	"+48",  // Poland
	"+36",  // Hungary
	"+40",  // Romania
	"+30",  // Greece
}

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

	for _, prefix := range euPrefixes {
		if strings.HasPrefix(phone, prefix) {
			return SMSRegionUKEU
		}
	}

	return SMSRegionInternational
}
