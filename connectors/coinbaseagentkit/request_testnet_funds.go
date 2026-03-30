package coinbaseagentkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coinbase/cdp-sdk/go/openapi"
	"github.com/supersuit-tech/permission-slip/connectors"
)

type requestTestnetFundsParams struct {
	Address string `json:"address"`
	Network string `json:"network"`
	Token   string `json:"token"`
}

type requestTestnetFundsAction struct {
	conn *CoinbaseAgentKitConnector
}

func (a *requestTestnetFundsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var p requestTestnetFundsParams
	if err := json.Unmarshal(req.Parameters, &p); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	addr := strings.TrimSpace(p.Address)
	if !evmAddrRE.MatchString(addr) {
		return nil, &connectors.ValidationError{Message: "address must be a 0x-prefixed 40-hex-character EVM address"}
	}
	net, err := parseFaucetNetwork(p.Network)
	if err != nil {
		return nil, err
	}
	tok, err := parseFaucetToken(p.Token)
	if err != nil {
		return nil, err
	}

	client, err := newCDPClient(req.Credentials)
	if err != nil {
		return nil, err
	}

	resp, err := client.RequestEvmFaucetWithResponse(ctx, openapi.RequestEvmFaucetJSONRequestBody{
		Address: addr,
		Network: net,
		Token:   tok,
	})
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("CDP faucet request timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("CDP faucet request failed: %v", err)}
	}

	if resp.StatusCode() != 200 {
		return nil, mapCDPError(resp.StatusCode(), resp.Body)
	}

	return jsonResult(map[string]any{
		"address": addr,
		"network": string(net),
		"token":   string(tok),
		"status":  "requested",
	})
}
