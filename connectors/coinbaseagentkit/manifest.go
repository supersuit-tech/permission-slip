package coinbaseagentkit

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

//go:embed logo.svg
var logoSVG string

// Manifest returns connector metadata for DB auto-seeding.
func (c *CoinbaseAgentKitConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "coinbase_agentkit",
		Name:        "Coinbase AgentKit (CDP Wallets)",
		Description: "Coinbase Developer Platform server wallets and AgentKit-compatible flows: create EVM accounts, check balances, request testnet funds, get swap quotes, send native or ERC-20 transfers, and execute token swaps. Requires a CDP API key, API secret, and wallet secret from https://portal.cdp.coinbase.com/",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			createEvmAccountManifest(),
			listEvmAccountsManifest(),
			sendCryptoManifest(),
			requestTestnetFundsManifest(),
			listTokenBalancesManifest(),
			getSwapPriceManifest(),
			swapTokensManifest(),
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "coinbase_agentkit",
				AuthType:        "api_key",
				InstructionsURL: "https://portal.cdp.coinbase.com/access/api",
				Fields: []connectors.ManifestCredentialField{
					{Key: "api_key_id", Label: "API Key ID", Placeholder: "From CDP Portal → API Keys", Secret: ptrFalse()},
					{Key: "api_key_secret", Label: "API Key Secret", Placeholder: "PEM EC private key or Ed25519 secret from portal"},
					{Key: "wallet_secret", Label: "Wallet Secret", Placeholder: "Base64 PKCS#8 wallet secret from Wallet API product",
						HelpText: "Create keys at https://portal.cdp.coinbase.com/ . The wallet secret is required for signing and sending transactions."},
				},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_coinbase_agentkit_create_wallet",
				ActionType:  "coinbase_agentkit.create_evm_account",
				Name:        "Create agent EVM wallet",
				Description: "Provision a new MPC-secured EVM account for the agent.",
				Parameters:  json.RawMessage(`{"name":"*"}`),
			},
			{
				ID:          "tpl_coinbase_agentkit_balances",
				ActionType:  "coinbase_agentkit.list_token_balances",
				Name:        "Check wallet token balances (read-only)",
				Description: "List token balances for an address on Base, Base Sepolia, or Ethereum.",
				Parameters:  json.RawMessage(`{"address":"*","network":"*"}`),
			},
			{
				ID:          "tpl_coinbase_agentkit_send_usdc_base",
				ActionType:  "coinbase_agentkit.send_crypto",
				Name:        "Send USDC on Base (requires payment method)",
				Description: "Transfer USDC from a CDP-managed wallet. WARNING: spends real funds on mainnet.",
				Parameters:  json.RawMessage(`{"from_address":"*","to_address":"*","network":"base","amount_wei":"*","token_contract":"*"}`),
			},
		},
	}
}

func createEvmAccountManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:      "coinbase_agentkit.create_evm_account",
		Name:            "Create EVM account",
		Description:     "Create a new CDP-managed EVM wallet (MPC-secured). Optionally assign a unique name (2–36 alphanumeric characters and hyphens) for later lookup.",
		RiskLevel:       "low",
		DisplayTemplate: "Create CDP EVM wallet {{name}}",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"additionalProperties": false,
			"properties": {
				"name": {
					"type": "string",
					"description": "Optional unique account name (2–36 chars: letters, digits, hyphens). Shown in CDP Portal and usable with list/get-by-name APIs."
				},
				"account_policy": {
					"type": "string",
					"description": "Optional CDP account-level policy ID from your project (advanced; leave unset unless your org uses custom policies)"
				}
			}
		}`)),
	}
}

func listEvmAccountsManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:      "coinbase_agentkit.list_evm_accounts",
		Name:            "List EVM accounts",
		Description:     "List CDP EVM accounts in the project with optional pagination.",
		RiskLevel:       "low",
		DisplayTemplate: "List CDP EVM accounts",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"additionalProperties": false,
			"properties": {
				"page_size": {
					"type": "integer",
					"minimum": 1,
					"description": "Max accounts per page (omit for API default)"
				},
				"page_token": {
					"type": "string",
					"description": "next_page_token from a previous list response to fetch the next page"
				}
			}
		}`)),
	}
}

func sendCryptoManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:            "coinbase_agentkit.send_crypto",
		Name:                  "Send crypto (native or ERC-20)",
		Description:           "Send native gas token or ERC-20 from a CDP-managed address. Builds an EIP-1559 transaction, signs via CDP, and broadcasts via CDP. WARNING: moves real assets on mainnet networks. amount_wei is in the smallest unit (wei for ETH, base units for tokens).",
		RiskLevel:             "high",
		RequiresPaymentMethod: true,
		DisplayTemplate:       "Send {{amount_wei}} (smallest units) from {{from_address}} to {{to_address}} on {{network}}",
		Preview: &connectors.ActionPreview{
			Layout: "record",
			Fields: map[string]string{
				"title":    "network",
				"subtitle": "to_address",
			},
		},
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["from_address", "to_address", "network", "amount_wei"],
			"additionalProperties": false,
			"properties": {
				"from_address": {
					"type": "string",
					"description": "0x-prefixed sender address (must be a CDP-managed EVM account in this project)"
				},
				"to_address": {
					"type": "string",
					"description": "0x-prefixed recipient address"
				},
				"network": {
					"type": "string",
					"description": "Network id: base, base-sepolia, ethereum, ethereum-sepolia, polygon, arbitrum, optimism, avalanche"
				},
				"amount_wei": {
					"type": "string",
					"description": "Amount in smallest units as a decimal integer string (wei for native ETH; for USDC use 6 decimals, e.g. 1 USDC = 1000000)"
				},
				"token_contract": {
					"type": "string",
					"description": "Optional 0x-prefixed ERC-20 contract; omit to send the chain native gas token only"
				}
			}
		}`)),
	}
}

func requestTestnetFundsManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:      "coinbase_agentkit.request_testnet_funds",
		Name:            "Request testnet funds (faucet)",
		Description:     "Request test tokens from the CDP faucet for development (subject to CDP limits).",
		RiskLevel:       "medium",
		DisplayTemplate: "Faucet {{token}} on {{network}} → {{address}}",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["address", "network", "token"],
			"additionalProperties": false,
			"properties": {
				"address": {
					"type": "string",
					"description": "0x-prefixed EVM address to fund"
				},
				"network": {
					"type": "string",
					"description": "base-sepolia, ethereum-sepolia, or ethereum-hoodi"
				},
				"token": {
					"type": "string",
					"description": "eth, usdc, eurc, or cbbtc"
				}
			}
		}`)),
	}
}

func listTokenBalancesManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:      "coinbase_agentkit.list_token_balances",
		Name:            "List token balances",
		Description:     "List token balances for an address on Base, Base Sepolia, or Ethereum mainnet (CDP data API).",
		RiskLevel:       "low",
		DisplayTemplate: "Token balances for {{address}} on {{network}}",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["address", "network"],
			"additionalProperties": false,
			"properties": {
				"address": {
					"type": "string",
					"description": "0x-prefixed EVM address"
				},
				"network": {
					"type": "string",
					"description": "base, base-sepolia, or ethereum"
				},
				"page_size": {
					"type": "integer",
					"minimum": 1
				},
				"page_token": {
					"type": "string"
				}
			}
		}`)),
	}
}

func getSwapPriceManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:      "coinbase_agentkit.get_swap_price",
		Name:            "Get swap price quote",
		Description:     "Read-only price quote for a token swap via CDP (no transaction). from_amount is in atomic units of from_token. Use 0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE for native per EIP-7528.",
		RiskLevel:       "low",
		DisplayTemplate: "Swap quote on {{network}}: {{from_amount}} ({{from_token}} → {{to_token}})",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["network", "from_token", "to_token", "from_amount", "taker"],
			"additionalProperties": false,
			"properties": {
				"network": {
					"type": "string",
					"description": "base, ethereum, polygon, arbitrum, or optimism"
				},
				"from_token": { "type": "string" },
				"to_token": { "type": "string" },
				"from_amount": {
					"type": "string",
					"description": "Amount in atomic units of from_token"
				},
				"taker": {
					"type": "string",
					"description": "0x-prefixed wallet address that would execute the swap"
				}
			}
		}`)),
	}
}

func swapTokensManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:            "coinbase_agentkit.swap_tokens",
		Name:                  "Execute token swap",
		Description:           "Get a swap quote from CDP, sign with the taker wallet, and broadcast. WARNING: spends tokens and may incur gas — mainnet only for supported swap networks.",
		RiskLevel:             "high",
		RequiresPaymentMethod: true,
		DisplayTemplate:       "Swap on {{network}}: {{from_amount}} from {{from_token}} to {{to_token}} (taker {{taker}})",
		Preview: &connectors.ActionPreview{
			Layout: "record",
			Fields: map[string]string{
				"title":    "network",
				"subtitle": "taker",
			},
		},
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["network", "from_token", "to_token", "from_amount", "taker"],
			"additionalProperties": false,
			"properties": {
				"network": {
					"type": "string",
					"description": "base, ethereum, polygon, arbitrum, or optimism"
				},
				"from_token": { "type": "string" },
				"to_token": { "type": "string" },
				"from_amount": { "type": "string" },
				"taker": {
					"type": "string",
					"description": "0x-prefixed CDP-managed address that holds from_token"
				},
				"slippage_bps": {
					"type": "integer",
					"minimum": 0,
					"description": "Max slippage in basis points (default ~100 if omitted)"
				},
				"gas_price": {
					"type": "string",
					"description": "Optional max fee per gas in wei as decimal string"
				},
				"idempotency_key": {
					"type": "string",
					"description": "Optional idempotency key for the swap quote request"
				}
			}
		}`)),
	}
}

func ptrFalse() *bool {
	b := false
	return &b
}
