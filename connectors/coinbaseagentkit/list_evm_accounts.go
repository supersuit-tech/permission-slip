package coinbaseagentkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coinbase/cdp-sdk/go/openapi"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listEvmAccountsParams struct {
	PageSize  *int    `json:"page_size"`
	PageToken *string `json:"page_token"`
}

type listEvmAccountsAction struct {
	conn *CoinbaseAgentKitConnector
}

func (a *listEvmAccountsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var p listEvmAccountsParams
	if err := json.Unmarshal(req.Parameters, &p); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	client, err := newCDPClient(req.Credentials)
	if err != nil {
		return nil, err
	}

	params := &openapi.ListEvmAccountsParams{}
	if p.PageSize != nil && *p.PageSize > 0 {
		ps := openapi.PageSize(*p.PageSize)
		params.PageSize = &ps
	}
	if p.PageToken != nil && strings.TrimSpace(*p.PageToken) != "" {
		pt := openapi.PageToken(strings.TrimSpace(*p.PageToken))
		params.PageToken = &pt
	}

	resp, err := client.ListEvmAccountsWithResponse(ctx, params)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("CDP request timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("CDP request failed: %v", err)}
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, mapCDPError(resp.StatusCode(), resp.Body)
	}

	out := make([]map[string]any, 0, len(resp.JSON200.Accounts))
	for _, acc := range resp.JSON200.Accounts {
		m := map[string]any{"address": acc.Address}
		if acc.Name != nil {
			m["name"] = *acc.Name
		}
		if acc.CreatedAt != nil {
			m["created_at"] = acc.CreatedAt
		}
		out = append(out, m)
	}

	result := map[string]any{"accounts": out}
	if resp.JSON200.NextPageToken != nil {
		result["next_page_token"] = *resp.JSON200.NextPageToken
	}
	return jsonResult(result)
}
