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

// intercomContact represents an Intercom contact (user or lead).
type intercomContact struct {
	Type             string         `json:"type,omitempty"`
	ID               string         `json:"id,omitempty"`
	Role             string         `json:"role,omitempty"`
	Email            string         `json:"email,omitempty"`
	Phone            string         `json:"phone,omitempty"`
	Name             string         `json:"name,omitempty"`
	CreatedAt        int64          `json:"created_at,omitempty"`
	UpdatedAt        int64          `json:"updated_at,omitempty"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
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

// contactsSearchResponse represents the Intercom contacts search API response.
type contactsSearchResponse struct {
	Type       string           `json:"type"`
	TotalCount int              `json:"total_count"`
	Data       []intercomContact `json:"data"`
}

// intercomConversation represents a single Intercom conversation summary.
type intercomConversation struct {
	Type         string `json:"type"`
	ID           string `json:"id"`
	Title        string `json:"title"`
	State        string `json:"state"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
	WaitingSince int64  `json:"waiting_since"`
	SnoozedUntil int64  `json:"snoozed_until"`
}

// conversationsResponse represents the Intercom conversations list API response.
type conversationsResponse struct {
	Type          string                 `json:"type"`
	TotalCount    int                    `json:"total_count"`
	Conversations []intercomConversation `json:"conversations"`
}

// outboundMessageResponse represents the Intercom outbound message API response.
type outboundMessageResponse struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	MessageType string `json:"message_type"`
	Body        string `json:"body"`
}

// intercomArticle represents a help center article in Intercom.
type intercomArticle struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	State     string `json:"state"`
	URL       string `json:"url"`
	AuthorID  string `json:"author_id"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
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
