package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listEmailsAction implements connectors.Action for google.list_emails.
// It lists messages via the Gmail API GET /gmail/v1/users/me/messages
// and fetches metadata for each message.
type listEmailsAction struct {
	conn *GoogleConnector
}

// listEmailsParams is the user-facing parameter schema.
type listEmailsParams struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

func (p *listEmailsParams) normalize() {
	if p.MaxResults <= 0 {
		p.MaxResults = 10
	}
	if p.MaxResults > 100 {
		p.MaxResults = 100
	}
}

// gmailListResponse is the Gmail API response from messages.list.
type gmailListResponse struct {
	Messages []struct {
		ID       string `json:"id"`
		ThreadID string `json:"threadId"`
	} `json:"messages"`
	ResultSizeEstimate int `json:"resultSizeEstimate"`
}

// gmailMessageResponse is the Gmail API response from messages.get
// with format=metadata.
type gmailMessageResponse struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
	Snippet  string `json:"snippet"`
	Payload  struct {
		Headers []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
	} `json:"payload"`
	InternalDate string `json:"internalDate"`
}

// emailSummary is the shape returned to the agent.
type emailSummary struct {
	ID      string `json:"id"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Subject string `json:"subject,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	Date    string `json:"date,omitempty"`
}

// Execute lists recent emails from Gmail and returns their metadata.
func (a *listEmailsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listEmailsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	params.normalize()

	// Step 1: list message IDs.
	q := url.Values{}
	q.Set("maxResults", strconv.Itoa(params.MaxResults))
	if params.Query != "" {
		q.Set("q", params.Query)
	}

	var listResp gmailListResponse
	listURL := a.conn.gmailBaseURL + "/gmail/v1/users/me/messages?" + q.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, listURL, nil, &listResp); err != nil {
		return nil, err
	}

	if len(listResp.Messages) == 0 {
		return connectors.JSONResult(map[string]any{
			"emails":       []emailSummary{},
			"total_estimate": listResp.ResultSizeEstimate,
		})
	}

	// Step 2: fetch metadata for each message.
	summaries := make([]emailSummary, 0, len(listResp.Messages))
	for _, m := range listResp.Messages {
		var msgResp gmailMessageResponse
		msgURL := a.conn.gmailBaseURL + "/gmail/v1/users/me/messages/" + m.ID + "?format=metadata&metadataHeaders=From&metadataHeaders=To&metadataHeaders=Subject&metadataHeaders=Date"
		if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, msgURL, nil, &msgResp); err != nil {
			return nil, err
		}

		summary := emailSummary{
			ID:      msgResp.ID,
			Snippet: msgResp.Snippet,
		}
		for _, h := range msgResp.Payload.Headers {
			switch h.Name {
			case "From":
				summary.From = h.Value
			case "To":
				summary.To = h.Value
			case "Subject":
				summary.Subject = h.Value
			case "Date":
				summary.Date = h.Value
			}
		}
		summaries = append(summaries, summary)
	}

	return connectors.JSONResult(map[string]any{
		"emails":         summaries,
		"total_estimate": listResp.ResultSizeEstimate,
	})
}
