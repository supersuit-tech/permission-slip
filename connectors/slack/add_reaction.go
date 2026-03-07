package slack

import (
	"context"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// addReactionAction implements connectors.Action for slack.add_reaction.
// It adds an emoji reaction to a message via POST /reactions.add.
type addReactionAction struct {
	conn *SlackConnector
}

// addReactionParams is the user-facing parameter schema.
type addReactionParams struct {
	Channel   string `json:"channel"`
	Timestamp string `json:"timestamp"`
	Name      string `json:"name"`
}

func (p *addReactionParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	if p.Timestamp == "" {
		return &connectors.ValidationError{Message: "missing required parameter: timestamp"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	// Strip surrounding colons — users often pass `:thumbsup:` instead of `thumbsup`.
	p.Name = strings.Trim(p.Name, ":")
	return nil
}

// addReactionRequest is the Slack API request body for reactions.add.
type addReactionRequest struct {
	Channel   string `json:"channel"`
	Timestamp string `json:"timestamp"`
	Name      string `json:"name"`
}

type addReactionResponse struct {
	slackResponse
}

// Execute adds an emoji reaction to a message and returns confirmation.
func (a *addReactionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addReactionParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := addReactionRequest{
		Channel:   params.Channel,
		Timestamp: params.Timestamp,
		Name:      params.Name,
	}

	var resp addReactionResponse
	if err := a.conn.doPost(ctx, "reactions.add", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	return connectors.JSONResult(map[string]string{
		"channel":   params.Channel,
		"timestamp": params.Timestamp,
		"reaction":  params.Name,
	})
}
