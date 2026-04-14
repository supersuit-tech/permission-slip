package slack

import (
	"context"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// removeReactionAction implements connectors.Action for slack.remove_reaction.
// It removes an emoji reaction via POST /reactions.remove.
type removeReactionAction struct {
	conn *SlackConnector
}

type removeReactionParams struct {
	Channel   string `json:"channel"`
	Timestamp string `json:"timestamp"`
	Name      string `json:"name"`
}

func (p *removeReactionParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	if err := validateMessageTS(p.Timestamp); err != nil {
		return err
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	p.Name = strings.Trim(p.Name, ":")
	return nil
}

type removeReactionRequest struct {
	Channel   string `json:"channel"`
	Timestamp string `json:"timestamp"`
	Name      string `json:"name"`
}

type removeReactionResponse struct {
	slackResponse
}

// Execute removes an emoji reaction from a message.
func (a *removeReactionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params removeReactionParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := removeReactionRequest{
		Channel:   params.Channel,
		Timestamp: params.Timestamp,
		Name:      params.Name,
	}

	var resp removeReactionResponse
	if err := a.conn.doPost(ctx, "reactions.remove", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, resp.asError()
	}

	return connectors.JSONResult(map[string]string{
		"channel":   params.Channel,
		"timestamp": params.Timestamp,
		"reaction":  params.Name,
	})
}
