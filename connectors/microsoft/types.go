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

// graphDriveItem represents a file or folder in OneDrive in Graph API format.
type graphDriveItem struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Size             int64           `json:"size"`
	WebURL           string          `json:"webUrl,omitempty"`
	CreatedDateTime  string          `json:"createdDateTime,omitempty"`
	ModifiedDateTime string          `json:"lastModifiedDateTime,omitempty"`
	Folder           *graphFolder    `json:"folder,omitempty"`
	File             *graphFileFacet `json:"file,omitempty"`
}

// graphFolder represents the folder facet of a OneDrive item.
type graphFolder struct {
	ChildCount int `json:"childCount"`
}

// graphFileFacet represents the file facet of a OneDrive item.
type graphFileFacet struct {
	MimeType string `json:"mimeType"`
}

// graphDriveItemsResponse is the Microsoft Graph API response for listing drive items.
type graphDriveItemsResponse struct {
	Value []graphDriveItem `json:"value"`
}
