package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getMetricsAction implements connectors.Action for datadog.get_metrics.
type getMetricsAction struct {
	conn *DatadogConnector
}

type getMetricsParams struct {
	Query string `json:"query"`
	From  int64  `json:"from"`
	To    int64  `json:"to"`
}

func (p *getMetricsParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query"}
	}
	if p.From == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: from"}
	}
	if p.To == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: to"}
	}
	if p.From >= p.To {
		return &connectors.ValidationError{Message: "parameter 'from' must be before 'to'"}
	}
	return nil
}

// Execute queries Datadog time series metrics.
func (a *getMetricsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getMetricsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("query", params.Query)
	q.Set("from", strconv.FormatInt(params.From, 10))
	q.Set("to", strconv.FormatInt(params.To, 10))

	var respBody json.RawMessage
	path := "/api/v1/query?" + q.Encode()
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
