package coinbaseagentkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coinbase/cdp-sdk/go/openapi"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type getSwapPriceParams struct {
	Network    string `json:"network"`
	FromToken  string `json:"from_token"`
	ToToken    string `json:"to_token"`
	FromAmount string `json:"from_amount"`
	Taker      string `json:"taker"`
}

type getSwapPriceAction struct {
	conn *CoinbaseAgentKitConnector
}

func (a *getSwapPriceAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var p getSwapPriceParams
	if err := json.Unmarshal(req.Parameters, &p); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	net, err := parseSwapNetwork(p.Network)
	if err != nil {
		return nil, err
	}
	fromTok := strings.TrimSpace(p.FromToken)
	toTok := strings.TrimSpace(p.ToToken)
	if !evmAddrRE.MatchString(fromTok) || !evmAddrRE.MatchString(toTok) {
		return nil, &connectors.ValidationError{Message: "from_token and to_token must be 0x-prefixed 40-hex-character contract addresses (use 0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE for native per EIP-7528)"}
	}
	fromAmt := strings.TrimSpace(p.FromAmount)
	if fromAmt == "" {
		return nil, &connectors.ValidationError{Message: "from_amount is required (atomic units as a decimal string)"}
	}
	taker := strings.TrimSpace(p.Taker)
	if !evmAddrRE.MatchString(taker) {
		return nil, &connectors.ValidationError{Message: "taker must be the 0x-prefixed wallet address that will execute the swap"}
	}

	client, err := newCDPClient(req.Credentials)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetEvmSwapPriceWithResponse(ctx, &openapi.GetEvmSwapPriceParams{
		Network:    net,
		FromToken:  openapi.FromToken(fromTok),
		ToToken:    openapi.ToToken(toTok),
		FromAmount: openapi.FromAmount(fromAmt),
		Taker:      openapi.Taker(taker),
	})
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
