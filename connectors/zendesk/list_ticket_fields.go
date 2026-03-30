package zendesk

import (
	"context"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listTicketFieldsAction implements connectors.Action for zendesk.list_ticket_fields.
// It fetches the list of ticket fields (system and custom) via GET /ticket_fields.json.
type listTicketFieldsAction struct {
	conn *ZendeskConnector
}

type ticketFieldOption struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Value   string `json:"raw_name"`
	Default bool   `json:"default"`
}

type ticketField struct {
	ID                 int64               `json:"id"`
	Type               string              `json:"type"`
	Title              string              `json:"title"`
	Description        string              `json:"description,omitempty"`
	Active             bool                `json:"active"`
	Required           bool                `json:"required"`
	CustomFieldOptions []ticketFieldOption `json:"custom_field_options,omitempty"`
}

type ticketFieldsResponse struct {
	TicketFields []ticketField `json:"ticket_fields"`
	Count        int           `json:"count"`
}

func (a *listTicketFieldsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var resp ticketFieldsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/ticket_fields.json", nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
