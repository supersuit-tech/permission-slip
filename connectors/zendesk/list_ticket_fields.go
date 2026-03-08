package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listTicketFieldsAction implements connectors.Action for zendesk.list_ticket_fields.
// It fetches the list of ticket fields (system and custom) via GET /ticket_fields.json.
type listTicketFieldsAction struct {
	conn *ZendeskConnector
}

type ticketField struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Active      bool   `json:"active"`
	Required    bool   `json:"required"`
	Custom      bool   `json:"custom_field_options,omitempty"`
}

type ticketFieldsResponse struct {
	TicketFields []ticketField `json:"ticket_fields"`
	Count        int           `json:"count"`
}

func (a *listTicketFieldsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	// No parameters needed for this action — it lists all ticket fields.
	var ignored map[string]any
	if len(req.Parameters) > 0 && string(req.Parameters) != "null" {
		if err := json.Unmarshal(req.Parameters, &ignored); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
	}

	var resp ticketFieldsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/ticket_fields.json", nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
