package notify

import (
	"encoding/json"
	"fmt"
)

// CardExpiringInfo holds the card details extracted from the Approval.Context
// JSON for card-expiring notifications. Used by email, SMS, and push templates.
type CardExpiringInfo struct {
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	Label    string `json:"label,omitempty"` // user-assigned label (e.g. "Work Card")
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
	Expired  bool   `json:"expired"` // true if the card is already expired
}

// DisplayBrand returns the human-readable brand name for card-expiring templates.
func (c CardExpiringInfo) DisplayBrand() string {
	return FormatBrand(c.Brand)
}

// CardIdentifier returns a concise identifier like "Visa ending in 4242" or
// "Work Card (Visa ending in 4242)" when a label is set. Used in subject lines
// and notification bodies so users can identify which card needs attention.
func (c CardExpiringInfo) CardIdentifier() string {
	base := fmt.Sprintf("%s ending in %s", c.DisplayBrand(), c.Last4)
	if c.Label != "" {
		return fmt.Sprintf("%s (%s)", c.Label, base)
	}
	return base
}

// extractCardExpiringInfo pulls card details from the context JSONB.
func extractCardExpiringInfo(raw json.RawMessage) CardExpiringInfo {
	var info CardExpiringInfo
	if len(raw) > 0 {
		json.Unmarshal(raw, &info) //nolint:errcheck // best-effort
	}
	return info
}

// formatCardExpiry returns "MM/YY" for display in notifications.
func formatCardExpiry(month, year int) string {
	return fmt.Sprintf("%02d/%02d", month, year%100)
}
