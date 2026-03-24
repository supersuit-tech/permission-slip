package ticketmaster

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// suggestAction implements connectors.Action for ticketmaster.suggest.
type suggestAction struct {
	conn *TicketmasterConnector
}

type suggestParams struct {
	Keyword string `json:"keyword"`
	Source  string `json:"source"`
	Locale  string `json:"locale"`
}

func (p *suggestParams) validate() error {
	if trimString(p.Keyword) == "" {
		return &connectors.ValidationError{Message: "missing required parameter: keyword"}
	}
	return nil
}

func (a *suggestAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params suggestParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("keyword", trimString(params.Keyword))
	appendNonEmpty(q, "source", trimString(params.Source))
	appendNonEmpty(q, "locale", trimString(params.Locale))

	var out json.RawMessage
	if err := a.conn.doGET(ctx, req.Credentials, "suggest.json", q, &out); err != nil {
		return nil, err
	}
	return &connectors.ActionResult{Data: out}, nil
}
