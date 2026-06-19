package imessage

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type searchAction struct {
	conn *IMessageConnector
}

type searchParams struct {
	Query string `json:"query"`
	Match string `json:"match"`
	Limit int    `json:"limit"`
}

func (p *searchParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query"}
	}
	if p.Match == "" {
		p.Match = "contains"
	}
	if p.Match != "contains" && p.Match != "exact" {
		return &connectors.ValidationError{Message: "match must be \"contains\" or \"exact\""}
	}
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 200 {
		return &connectors.ValidationError{Message: "limit must be at most 200"}
	}
	return nil
}

func (a *searchAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	actionCtx, cancel := a.conn.actionTimeout(ctx)
	defer cancel()

	lines, err := a.conn.client.runCLI(actionCtx, req.Credentials,
		"search",
		"--query", params.Query,
		"--match", params.Match,
		"--limit", strconv.Itoa(params.Limit),
		"--json",
	)
	if err != nil {
		return nil, err
	}

	messages := make([]message, 0, len(lines))
	for _, line := range lines {
		var m message
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, &connectors.ExternalError{Message: "parse search result: " + err.Error()}
		}
		messages = append(messages, m)
	}
	return connectors.JSONResult(map[string]any{
		"messages": messages,
		"count":    len(messages),
	})
}
