package coinbaseagentkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coinbase/cdp-sdk/go/openapi"
	"github.com/supersuit-tech/permission-slip/connectors"
)

type listTokenBalancesParams struct {
	Address   string  `json:"address"`
	Network   string  `json:"network"`
	PageSize  *int    `json:"page_size"`
	PageToken *string `json:"page_token"`
}

type listTokenBalancesAction struct {
	conn *CoinbaseAgentKitConnector
}

func (a *listTokenBalancesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var p listTokenBalancesParams
	if err := json.Unmarshal(req.Parameters, &p); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	addr := strings.TrimSpace(p.Address)
	if !evmAddrRE.MatchString(addr) {
		return nil, &connectors.ValidationError{Message: "address must be a 0x-prefixed 40-hex-character EVM address"}
	}
	net, err := parseListBalancesNetwork(p.Network)
	if err != nil {
		return nil, err
	}

	client, err := newCDPClient(req.Credentials)
	if err != nil {
		return nil, err
	}

	params := &openapi.ListEvmTokenBalancesParams{}
	if p.PageSize != nil && *p.PageSize > 0 {
		ps := openapi.PageSize(*p.PageSize)
		params.PageSize = &ps
	}
	if p.PageToken != nil && strings.TrimSpace(*p.PageToken) != "" {
		pt := openapi.PageToken(strings.TrimSpace(*p.PageToken))
		params.PageToken = &pt
	}

	resp, err := client.ListEvmTokenBalancesWithResponse(ctx, net, addr, params)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("CDP request timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("CDP request failed: %v", err)}
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, mapCDPError(resp.StatusCode(), resp.Body)
	}

	return jsonResult(resp.JSON200)
}
