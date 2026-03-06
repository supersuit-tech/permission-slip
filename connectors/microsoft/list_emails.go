package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listEmailsAction implements connectors.Action for microsoft.list_emails.
// It lists emails from a mail folder via GET /me/mailFolders/{folder}/messages.
type listEmailsAction struct {
	conn *MicrosoftConnector
}

// listEmailsParams is the user-facing parameter schema.
type listEmailsParams struct {
	Folder string `json:"folder"`
	Top    int    `json:"top"`
}

func (p *listEmailsParams) defaults() {
	if p.Folder == "" {
		p.Folder = "inbox"
	}
	if p.Top <= 0 {
		p.Top = 10
	}
	if p.Top > 50 {
		p.Top = 50
	}
}

// graphMessagesResponse is the Microsoft Graph API response for listing messages.
type graphMessagesResponse struct {
	Value []graphMessage `json:"value"`
}

type graphMessage struct {
	ID             string            `json:"id"`
	Subject        string            `json:"subject"`
	From           *graphMailAddress  `json:"from,omitempty"`
	ToRecipients   []graphMailAddress `json:"toRecipients,omitempty"`
	ReceivedAt     string            `json:"receivedDateTime,omitempty"`
	IsRead         bool              `json:"isRead"`
	BodyPreview    string            `json:"bodyPreview,omitempty"`
	HasAttachments bool              `json:"hasAttachments"`
}

type graphMailAddress struct {
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}

// emailSummary is the simplified response returned to the caller.
type emailSummary struct {
	ID             string   `json:"id"`
	Subject        string   `json:"subject"`
	From           string   `json:"from"`
	To             []string `json:"to"`
	ReceivedAt     string   `json:"received_at"`
	IsRead         bool     `json:"is_read"`
	Preview        string   `json:"preview"`
	HasAttachments bool     `json:"has_attachments"`
}

// Execute lists emails from the specified mail folder.
func (a *listEmailsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listEmailsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	params.defaults()

	path := fmt.Sprintf("/me/mailFolders/%s/messages?$top=%d&$orderby=receivedDateTime%%20desc&$select=id,subject,from,toRecipients,receivedDateTime,isRead,bodyPreview,hasAttachments", params.Folder, params.Top)

	var resp graphMessagesResponse
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	summaries := make([]emailSummary, len(resp.Value))
	for i, msg := range resp.Value {
		var from string
		if msg.From != nil {
			from = msg.From.EmailAddress.Address
		}
		to := make([]string, len(msg.ToRecipients))
		for j, r := range msg.ToRecipients {
			to[j] = r.EmailAddress.Address
		}
		summaries[i] = emailSummary{
			ID:             msg.ID,
			Subject:        msg.Subject,
			From:           from,
			To:             to,
			ReceivedAt:     msg.ReceivedAt,
			IsRead:         msg.IsRead,
			Preview:        msg.BodyPreview,
			HasAttachments: msg.HasAttachments,
		}
	}

	return connectors.JSONResult(summaries)
}
