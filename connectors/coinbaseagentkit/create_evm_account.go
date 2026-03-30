package coinbaseagentkit

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/coinbase/cdp-sdk/go/openapi"
	"github.com/supersuit-tech/permission-slip/connectors"
)

var evmAccountNameRE = regexp.MustCompile(`^[a-zA-Z0-9-]{2,36}$`)

type createEvmAccountParams struct {
	Name           *string `json:"name"`
	AccountPolicy  *string `json:"account_policy"`
}

type createEvmAccountAction struct {
	conn *CoinbaseAgentKitConnector
}

func (a *createEvmAccountAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var p createEvmAccountParams
	if err := json.Unmarshal(req.Parameters, &p); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	client, err := newCDPClient(req.Credentials)
	if err != nil {
		return nil, err
	}

	body := openapi.CreateEvmAccountJSONRequestBody{}
	if p.Name != nil {
		n := strings.TrimSpace(*p.Name)
		if n != "" {
			if !evmAccountNameRE.MatchString(n) {
				return nil, &connectors.ValidationError{Message: "name must be 2–36 characters: letters, digits, and hyphens only"}
			}
			body.Name = &n
		}
	}
	if p.AccountPolicy != nil {
		ap := strings.TrimSpace(*p.AccountPolicy)
		if ap != "" {
			body.AccountPolicy = &ap
		}
	}

	resp, err := client.CreateEvmAccountWithResponse(ctx, nil, body)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("CDP request timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("CDP request failed: %v", err)}
	}

	if resp.StatusCode() != 201 || resp.JSON201 == nil {
		return nil, mapCDPError(resp.StatusCode(), resp.Body)
	}

	return jsonResult(map[string]any{
		"address":    resp.JSON201.Address,
		"name":       resp.JSON201.Name,
		"created_at": resp.JSON201.CreatedAt,
	})
}
