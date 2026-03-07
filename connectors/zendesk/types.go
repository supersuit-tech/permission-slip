package zendesk

// ticketResponse wraps a single ticket in the Zendesk API response format.
type ticketResponse struct {
	Ticket ticket `json:"ticket"`
}

// ticket represents a Zendesk Support ticket.
type ticket struct {
	ID          int64    `json:"id"`
	Subject     string   `json:"subject,omitempty"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Type        string   `json:"type,omitempty"`
	AssigneeID  *int64   `json:"assignee_id,omitempty"`
	GroupID     *int64   `json:"group_id,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	RequesterID *int64   `json:"requester_id,omitempty"`
}

// ticketComment represents a comment on a Zendesk ticket.
type ticketComment struct {
	Body   string `json:"body"`
	Public bool   `json:"public"`
}

// searchResponse represents the Zendesk search API response.
type searchResponse struct {
	Results []ticket `json:"results"`
	Count   int      `json:"count"`
}

// jobStatusResponse represents the Zendesk job status response for merge.
type jobStatusResponse struct {
	JobStatus jobStatus `json:"job_status"`
}

type jobStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	URL    string `json:"url"`
}

// isValidZendeskID checks that an ID is positive (valid Zendesk ticket ID).
func isValidZendeskID(id int64) bool {
	return id > 0
}
