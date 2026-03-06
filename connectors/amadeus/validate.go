package amadeus

import (
	"regexp"
	"time"
)

// validDate checks that s is a valid YYYY-MM-DD date string.
func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// iataCodeRe matches a 3-letter uppercase IATA code.
var iataCodeRe = regexp.MustCompile(`^[A-Z]{3}$`)

// validIATACode checks that s is a valid 3-letter IATA code (e.g., "SFO").
func validIATACode(s string) bool {
	return iataCodeRe.MatchString(s)
}

// validCabins is the set of allowed cabin classes for flight search.
var validCabins = map[string]bool{
	"ECONOMY":         true,
	"PREMIUM_ECONOMY": true,
	"BUSINESS":        true,
	"FIRST":           true,
}

// validGenders is the set of allowed gender values for Amadeus traveler records.
var validGenders = map[string]bool{
	"MALE":   true,
	"FEMALE": true,
}

const (
	// maxFlightOfferBytes is the maximum size of a flight_offer JSON blob
	// we'll accept. Prevents forwarding arbitrarily large payloads to Amadeus.
	maxFlightOfferBytes = 100 * 1024 // 100 KB

	// maxTravelers is the maximum number of travelers allowed in a booking,
	// matching the Amadeus API limit for adults in a single PNR.
	maxTravelers = 9

	// maxRemarkLen is the maximum length of a booking remark.
	maxRemarkLen = 500

	// maxIdempotencyKeyLen is the maximum length of an idempotency key.
	// UUIDs are 36 chars; we allow up to 128 for flexibility.
	maxIdempotencyKeyLen = 128

	// maxResultsCap is the upper bound for max_results in search actions.
	maxResultsCap = 250
)
