package calendly

// Shared Calendly API response types used across multiple action files.

// calendlyLocation represents the location object in Calendly event responses.
type calendlyLocation struct {
	Type     string `json:"type"`
	Location string `json:"location"`
	JoinURL  string `json:"join_url"`
}

// calendlyGuest represents an event guest/invitee in Calendly event responses.
type calendlyGuest struct {
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
