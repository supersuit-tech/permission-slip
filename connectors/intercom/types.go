package intercom

// intercomTicket represents an Intercom ticket (conversation with ticket type).
type intercomTicket struct {
	Type             string       `json:"type,omitempty"`
	ID               string       `json:"id,omitempty"`
	TicketID         string       `json:"ticket_id,omitempty"`
	Title            string       `json:"title,omitempty"`
	State            string       `json:"state,omitempty"`
	TicketType       *ticketType  `json:"ticket_type,omitempty"`
	TicketParts      *ticketParts `json:"ticket_parts,omitempty"`
	TicketAttributes []ticketAttr `json:"ticket_attributes,omitempty"`
	Contacts         *contactList `json:"contacts,omitempty"`
	Assignee         *assignee    `json:"assignee,omitempty"`
	Tags             *tagList     `json:"tags,omitempty"`
}

type ticketType struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type ticketParts struct {
	Parts []ticketPart `json:"ticket_parts,omitempty"`
}

type ticketPart struct {
	Type   string `json:"part_type,omitempty"`
	Body   string `json:"body,omitempty"`
	Author author `json:"author,omitempty"`
}

type author struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

type ticketAttr struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type contactList struct {
	Type     string    `json:"type,omitempty"`
	Contacts []contact `json:"contacts,omitempty"`
}

type contact struct {
	ID    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
}

type assignee struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

type tagList struct {
	Type string `json:"type,omitempty"`
	Tags []tag  `json:"tags,omitempty"`
}

type tag struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// searchTicketsResponse represents the Intercom search API response.
type searchTicketsResponse struct {
	Type       string           `json:"type"`
	TotalCount int              `json:"total_count"`
	Data       []intercomTicket `json:"data"`
}

// tagsListResponse represents the response from listing all tags.
type tagsListResponse struct {
	Type string `json:"type"`
	Data []tag  `json:"data"`
}

// isValidIntercomID checks that an Intercom string ID is safe to embed in a
// URL path — it must be non-empty and not contain path separators or query
// characters that could cause path traversal or injection.
func isValidIntercomID(id string) bool {
	if id == "" {
		return false
	}
	for _, c := range id {
		if c == '/' || c == '?' || c == '#' || c == '\\' {
			return false
		}
	}
	return true
}
