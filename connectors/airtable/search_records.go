package airtable

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchRecordsAction implements connectors.Action for airtable.search_records.
// It uses the list records endpoint with filterByFormula for searching.
type searchRecordsAction struct {
	conn *AirtableConnector
}

type searchRecordsParams struct {
	BaseID     string      `json:"base_id"`
	Table      string      `json:"table"`
	Formula    string      `json:"formula"`
	Fields     []string    `json:"fields,omitempty"`
	MaxRecords int         `json:"max_records,omitempty"`
	Sort       []sortField `json:"sort,omitempty"`
}

func (p *searchRecordsParams) validate() error {
	if err := validateBaseID(p.BaseID); err != nil {
		return err
	}
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if p.Formula == "" {
		return &connectors.ValidationError{Message: "missing required parameter: formula"}
	}
	return nil
}

// Execute searches records using an Airtable formula filter.
func (a *searchRecordsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchRecordsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/%s/%s", a.conn.baseURL, params.BaseID, url.PathEscape(params.Table))

	q := url.Values{}
	q.Set("filterByFormula", params.Formula)

	maxRecords := params.MaxRecords
	if maxRecords == 0 {
		maxRecords = 100
	}
	q.Set("maxRecords", strconv.Itoa(maxRecords))

	for _, f := range params.Fields {
		q.Add("fields[]", f)
	}
	for i, s := range params.Sort {
		q.Set(fmt.Sprintf("sort[%d][field]", i), s.Field)
		if s.Direction != "" {
			q.Set(fmt.Sprintf("sort[%d][direction]", i), s.Direction)
		}
	}

	reqURL += "?" + q.Encode()

	var resp listRecordsResponse
	if err := a.conn.doRequest(ctx, "GET", reqURL, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(toListRecordsResult(resp))
}
