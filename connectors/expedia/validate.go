package expedia

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// dateLayout is the expected date format for Expedia Rapid API parameters.
const dateLayout = "2006-01-02"

// validateDate checks that a date string is a valid YYYY-MM-DD date.
// Returns a ValidationError with a helpful message if invalid.
func validateDate(field, value string) error {
	if _, err := time.Parse(dateLayout, value); err != nil {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid %s %q: must be a valid date in YYYY-MM-DD format", field, value),
		}
	}
	return nil
}

// validateDateRange checks that checkin is before checkout.
func validateDateRange(checkin, checkout string) error {
	ci, err := time.Parse(dateLayout, checkin)
	if err != nil {
		return nil // individual date validation will catch this
	}
	co, err := time.Parse(dateLayout, checkout)
	if err != nil {
		return nil // individual date validation will catch this
	}
	if !ci.Before(co) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("checkout %q must be after checkin %q", checkout, checkin),
		}
	}
	return nil
}

// occupancyPattern validates the Expedia occupancy format.
// Valid examples: "2" (2 adults), "2-0,4" (2 adults + 1 child age 4),
// "2-0,4,7" (2 adults + 2 children ages 4 and 7).
var occupancyPattern = regexp.MustCompile(`^\d+(-\d+(,\d+)*)?$`)

// validateOccupancy checks that the occupancy string matches Expedia's format.
func validateOccupancy(value string) error {
	if !occupancyPattern.MatchString(value) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid occupancy %q: use format like \"2\" (2 adults) or \"2-0,4\" (2 adults + 1 child age 4)", value),
		}
	}
	return nil
}

// validateEmail performs a basic check that the email contains an @ with text on both sides.
func validateEmail(value string) error {
	at := strings.Index(value, "@")
	if at < 1 || at >= len(value)-1 {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid email %q: must contain '@' with text on both sides", value),
		}
	}
	return nil
}
