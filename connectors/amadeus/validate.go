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

// maxFlightOfferBytes is the maximum size of a flight_offer JSON blob
// we'll accept. Prevents forwarding arbitrarily large payloads to Amadeus.
const maxFlightOfferBytes = 100 * 1024 // 100 KB

// maxTravelers is the maximum number of travelers allowed in a booking,
// matching the Amadeus API limit for adults in a single PNR.
const maxTravelers = 9

// maxRemarkLen is the maximum length of a booking remark.
const maxRemarkLen = 500

// maxResultsCap is the upper bound for max_results in search actions.
const maxResultsCap = 250
