// Package coinbaseagentkit implements the Coinbase AgentKit / CDP Server Wallet
// connector using the official CDP Go SDK (JWT API key auth + wallet secret for
// signing). See https://docs.cdp.coinbase.com/agentkit/docs/welcome
package coinbaseagentkit

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	cdp "github.com/coinbase/cdp-sdk/go"
	"github.com/coinbase/cdp-sdk/go/openapi"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	credAPIKeyID     = "api_key_id"
	credAPIKeySecret = "api_key_secret"
	credWalletSecret = "wallet_secret"
	credBasePath     = "base_path" // optional override, e.g. for testing
)

// evmAddrRE matches a checksummed or lower-case 0x-prefixed EVM address.
var evmAddrRE = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// CoinbaseAgentKitConnector executes CDP Wallet API v2 actions (EVM accounts,
// balances, transfers, swaps, faucet).
type CoinbaseAgentKitConnector struct{}

// New creates a CoinbaseAgentKitConnector.
func New() *CoinbaseAgentKitConnector {
	return &CoinbaseAgentKitConnector{}
}

// ID returns "coinbase_agentkit".
func (c *CoinbaseAgentKitConnector) ID() string { return "coinbase_agentkit" }

// Actions returns registered handlers.
func (c *CoinbaseAgentKitConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"coinbase_agentkit.create_evm_account":     &createEvmAccountAction{conn: c},
		"coinbase_agentkit.list_evm_accounts":      &listEvmAccountsAction{conn: c},
		"coinbase_agentkit.send_crypto":            &sendCryptoAction{conn: c},
		"coinbase_agentkit.request_testnet_funds":  &requestTestnetFundsAction{conn: c},
		"coinbase_agentkit.list_token_balances":    &listTokenBalancesAction{conn: c},
		"coinbase_agentkit.get_swap_price":         &getSwapPriceAction{conn: c},
		"coinbase_agentkit.swap_tokens":            &swapTokensAction{conn: c},
	}
}

// ValidateCredentials ensures CDP API key and wallet secret are present.
func (c *CoinbaseAgentKitConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	for _, key := range []string{credAPIKeyID, credAPIKeySecret, credWalletSecret} {
		v, ok := creds.Get(key)
		if !ok || strings.TrimSpace(v) == "" {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("missing required credential: %s — create keys at https://portal.cdp.coinbase.com/", key),
			}
		}
	}
	return nil
}

func newCDPClient(creds connectors.Credentials) (*openapi.ClientWithResponses, error) {
	apiKeyID, _ := creds.Get(credAPIKeyID)
	apiSecret, _ := creds.Get(credAPIKeySecret)
	walletSecret, _ := creds.Get(credWalletSecret)
	basePath, hasBase := creds.Get(credBasePath)

	opts := cdp.ClientOptions{
		APIKeyID:     strings.TrimSpace(apiKeyID),
		APIKeySecret: strings.TrimSpace(apiSecret),
		WalletSecret: strings.TrimSpace(walletSecret),
	}
	if hasBase && strings.TrimSpace(basePath) != "" {
		opts.BasePath = strings.TrimSpace(basePath)
	}

	client, err := cdp.NewClient(opts)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("CDP client initialization failed: %v", err)}
	}
	return client, nil
}

func chainIDForSendNetwork(n openapi.SendEvmTransactionJSONBodyNetwork) (*big.Int, error) {
	switch n {
	case openapi.SendEvmTransactionJSONBodyNetworkBase:
		return big.NewInt(8453), nil
	case openapi.SendEvmTransactionJSONBodyNetworkBaseSepolia:
		return big.NewInt(84532), nil
	case openapi.SendEvmTransactionJSONBodyNetworkEthereum:
		return big.NewInt(1), nil
	case openapi.SendEvmTransactionJSONBodyNetworkEthereumSepolia:
		return big.NewInt(11155111), nil
	case openapi.SendEvmTransactionJSONBodyNetworkPolygon:
		return big.NewInt(137), nil
	case openapi.SendEvmTransactionJSONBodyNetworkArbitrum:
		return big.NewInt(42161), nil
	case openapi.SendEvmTransactionJSONBodyNetworkOptimism:
		return big.NewInt(10), nil
	case openapi.SendEvmTransactionJSONBodyNetworkAvalanche:
		return big.NewInt(43114), nil
	default:
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("unsupported network for chain id mapping: %s", n)}
	}
}

func parseSendNetwork(s string) (openapi.SendEvmTransactionJSONBodyNetwork, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	n := openapi.SendEvmTransactionJSONBodyNetwork(s)
	switch n {
	case openapi.SendEvmTransactionJSONBodyNetworkArbitrum,
		openapi.SendEvmTransactionJSONBodyNetworkAvalanche,
		openapi.SendEvmTransactionJSONBodyNetworkBase,
		openapi.SendEvmTransactionJSONBodyNetworkBaseSepolia,
		openapi.SendEvmTransactionJSONBodyNetworkEthereum,
		openapi.SendEvmTransactionJSONBodyNetworkEthereumSepolia,
		openapi.SendEvmTransactionJSONBodyNetworkOptimism,
		openapi.SendEvmTransactionJSONBodyNetworkPolygon:
		return n, nil
	default:
		return "", &connectors.ValidationError{Message: fmt.Sprintf("invalid network %q — use base, base-sepolia, ethereum, ethereum-sepolia, polygon, arbitrum, optimism, or avalanche", s)}
	}
}

func parseFaucetNetwork(s string) (openapi.RequestEvmFaucetJSONBodyNetwork, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	n := openapi.RequestEvmFaucetJSONBodyNetwork(s)
	switch n {
	case openapi.RequestEvmFaucetJSONBodyNetworkBaseSepolia,
		openapi.RequestEvmFaucetJSONBodyNetworkEthereumHoodi,
		openapi.RequestEvmFaucetJSONBodyNetworkEthereumSepolia:
		return n, nil
	default:
		return "", &connectors.ValidationError{Message: fmt.Sprintf("invalid faucet network %q — use base-sepolia, ethereum-sepolia, or ethereum-hoodi", s)}
	}
}

func parseFaucetToken(s string) (openapi.RequestEvmFaucetJSONBodyToken, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	n := openapi.RequestEvmFaucetJSONBodyToken(s)
	switch n {
	case openapi.RequestEvmFaucetJSONBodyTokenCbbtc,
		openapi.RequestEvmFaucetJSONBodyTokenEth,
		openapi.RequestEvmFaucetJSONBodyTokenEurc,
		openapi.RequestEvmFaucetJSONBodyTokenUsdc:
		return n, nil
	default:
		return "", &connectors.ValidationError{Message: fmt.Sprintf("invalid faucet token %q — use eth, usdc, eurc, or cbbtc", s)}
	}
}

func parseListBalancesNetwork(s string) (openapi.ListEvmTokenBalancesNetwork, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	n := openapi.ListEvmTokenBalancesNetwork(s)
	switch n {
	case openapi.ListEvmTokenBalancesNetworkBase,
		openapi.ListEvmTokenBalancesNetworkBaseSepolia,
		openapi.ListEvmTokenBalancesNetworkEthereum:
		return n, nil
	default:
		return "", &connectors.ValidationError{Message: fmt.Sprintf("invalid balance query network %q — use base, base-sepolia, or ethereum", s)}
	}
}

func parseSwapNetwork(s string) (openapi.EvmSwapsNetwork, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	n := openapi.EvmSwapsNetwork(s)
	switch n {
	case openapi.EvmSwapsNetworkArbitrum,
		openapi.EvmSwapsNetworkBase,
		openapi.EvmSwapsNetworkEthereum,
		openapi.EvmSwapsNetworkOptimism,
		openapi.EvmSwapsNetworkPolygon:
		return n, nil
	default:
		return "", &connectors.ValidationError{Message: fmt.Sprintf("invalid swap network %q — use base, ethereum, polygon, arbitrum, or optimism", s)}
	}
}

func mapCDPError(status int, body []byte) error {
	msg := strings.TrimSpace(string(body))
	if len(msg) > 4096 {
		msg = msg[:4096] + "…"
	}
	if msg == "" {
		msg = "(empty response body)"
	}
	switch {
	case status == 401 || status == 403:
		return &connectors.AuthError{Message: fmt.Sprintf("Coinbase CDP auth error (%d): %s", status, msg)}
	case status == 429:
		return &connectors.RateLimitError{Message: "Coinbase CDP rate limit exceeded"}
	case status >= 500:
		return &connectors.ExternalError{StatusCode: status, Message: fmt.Sprintf("Coinbase CDP server error (%d): %s", status, msg)}
	default:
		return &connectors.ExternalError{StatusCode: status, Message: fmt.Sprintf("Coinbase CDP API error (%d): %s", status, msg)}
	}
}

func jsonResult(v any) (*connectors.ActionResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("encoding result: %v", err)}
	}
	return &connectors.ActionResult{Data: b}, nil
}
