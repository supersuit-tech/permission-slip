package microsoft

// Shared Microsoft Graph API types used across multiple actions.
//
// These types mirror the JSON structures expected by the Microsoft Graph v1.0 API.
// See: https://learn.microsoft.com/en-us/graph/api/overview
//
// Types are defined here rather than in individual action files to avoid
// duplication — for example, graphMailAddress is used by both list_emails
// and create_calendar_event.

// graphEmailBody represents the body of an email or event in Graph API format.
type graphEmailBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// graphMailAddress represents an email address in Graph API format.
type graphMailAddress struct {
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}

// graphDateTimeZone represents a date/time with timezone in Graph API format.
type graphDateTimeZone struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

// graphEventLocation represents an event location in Graph API format.
type graphEventLocation struct {
	DisplayName string `json:"displayName"`
}

// graphAttendee represents an event attendee in Graph API format.
type graphAttendee struct {
	EmailAddress struct {
		Address string `json:"address"`
	} `json:"emailAddress"`
	Type string `json:"type"`
}
