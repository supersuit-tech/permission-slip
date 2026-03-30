// This file contains shared helpers for Stripe list (GET) actions.
// All four list endpoints (customers, invoices, charges, subscriptions)
// use the same Stripe cursor-based pagination model and the same list
// response envelope, so these are centralized here rather than duplicated.
package stripe

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// defaultListLimit is the number of items returned when the caller does not
// specify a limit. Stripe accepts 1–100; we default to 10 to keep payloads
// small and discourage unbounded fetches.
const (
	defaultListLimit = 10
	maxListLimit     = 100
)

// stripeListResponse is the common envelope returned by Stripe list endpoints
// (GET /v1/customers, /v1/invoices, /v1/charges, /v1/subscriptions, etc.).
// Each endpoint wraps its items in a "data" array alongside pagination fields.
type stripeListResponse struct {
	Data    json.RawMessage `json:"data"`
	HasMore bool            `json:"has_more"`
	Object  string          `json:"object"`
}

// validateListLimit checks that limit is within the allowed range [0, maxListLimit].
// 0 is treated as "use default" by callers — it is not an error.
func validateListLimit(limit int) error {
	if limit < 0 || limit > maxListLimit {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("limit must be between 0 and %d (0 uses default of %d)", maxListLimit, defaultListLimit),
		}
	}
	return nil
}

// resolveLimit returns the effective limit to use in the Stripe API request.
// If the caller did not specify a limit (0), the default is used.
func resolveLimit(limit int) int {
	if limit == 0 {
		return defaultListLimit
	}
	return limit
}
