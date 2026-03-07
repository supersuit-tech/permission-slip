package x

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendDMAction implements connectors.Action for x.send_dm.
// It sends a DM via POST /2/dm_conversations/with/{recipient_id}/messages.
type sendDMAction struct {
	conn *XConnector
}

// sendDMParams are the parameters parsed from ActionRequest.Parameters.
type sendDMParams struct {
	RecipientID string `json:"recipient_id"`
	Text        string `json:"text"`
}

func (p *sendDMParams) validate() error {
	if p.RecipientID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: recipient_id"}
	}
	if p.Text == "" {
		return &connectors.ValidationError{Message: "missing required parameter: text"}
	}
	if len(p.Text) > 10000 {
		return &connectors.ValidationError{Message: "text exceeds 10,000 character limit for direct messages"}
	}
	return nil
}

// Execute sends a direct message and returns the message metadata.
func (a *sendDMAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendDMParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]string{"text": params.Text}

	var xResp struct {
		Data struct {
			DMConversationID string `json:"dm_conversation_id"`
			DMEventID        string `json:"dm_event_id"`
		} `json:"data"`
	}

	path := "/dm_conversations/with/" + url.PathEscape(params.RecipientID) + "/messages"
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &xResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(xResp.Data)
}
