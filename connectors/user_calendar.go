package connectors

// UserCalendar is a calendar the signed-in user can access, returned for UI
// dropdowns when configuring action parameters (e.g. calendar_id).
type UserCalendar struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsPrimary   bool   `json:"is_primary"`
}
