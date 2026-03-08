package plaid

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest describing all
// supported actions, required credentials, and pre-built templates.
//go:embed logo.svg
var logoSVG string

func (c *PlaidConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "plaid",
		Name:        "Plaid",
		Description: "Plaid integration for banking data, account balances, transactions, and identity verification",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "plaid.create_link_token",
				Name:        "Create Link Token",
				Description: "Create a link token to initiate the bank connection flow",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["user_id", "products"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "A unique identifier for the user connecting their bank account"
						},
						"products": {
							"type": "array",
							"items": {"type": "string", "enum": ["auth", "transactions", "identity", "balance"]},
							"description": "Plaid products to enable (e.g. auth, transactions, identity, balance)"
						},
						"country_codes": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Country codes to enable (defaults to [\"US\"])"
						},
						"language": {
							"type": "string",
							"description": "Language for the Link flow (defaults to \"en\")"
						}
					}
				}`)),
			},
			{
				ActionType:  "plaid.get_balances",
				Name:        "Get Account Balances",
				Description: "Get real-time or cached account balances for a connected bank account",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["access_token"],
					"properties": {
						"access_token": {
							"type": "string",
							"description": "The access token for the connected bank account (Item)"
						},
						"account_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Optional list of account IDs to filter results"
						}
					}
				}`)),
			},
			{
				ActionType:  "plaid.list_transactions",
				Name:        "List Transactions",
				Description: "List transactions for a connected bank account with date range and category filters",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["access_token", "start_date", "end_date"],
					"properties": {
						"access_token": {
							"type": "string",
							"description": "The access token for the connected bank account (Item)"
						},
						"start_date": {
							"type": "string",
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
							"description": "Start date in YYYY-MM-DD format"
						},
						"end_date": {
							"type": "string",
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
							"description": "End date in YYYY-MM-DD format"
						},
						"account_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Optional list of account IDs to filter results"
						},
						"count": {
							"type": "integer",
							"minimum": 1,
							"maximum": 500,
							"description": "Maximum number of transactions to return (default 100, max 500)"
						},
						"offset": {
							"type": "integer",
							"minimum": 0,
							"description": "Number of transactions to skip (for pagination)"
						}
					}
				}`)),
			},
			{
				ActionType:  "plaid.get_accounts",
				Name:        "Get Accounts",
				Description: "Get account details including name, type, and mask for a connected bank",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["access_token"],
					"properties": {
						"access_token": {
							"type": "string",
							"description": "The access token for the connected bank account (Item)"
						},
						"account_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Optional list of account IDs to filter results"
						}
					}
				}`)),
			},
			{
				ActionType:  "plaid.get_identity",
				Name:        "Get Identity",
				Description: "Get account holder identity information (name, address, phone, email)",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["access_token"],
					"properties": {
						"access_token": {
							"type": "string",
							"description": "The access token for the connected bank account (Item)"
						},
						"account_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Optional list of account IDs to filter results"
						}
					}
				}`)),
			},
			{
				ActionType:  "plaid.get_institution",
				Name:        "Get Institution",
				Description: "Get details about a financial institution (name, logo, supported products)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["institution_id"],
					"properties": {
						"institution_id": {
							"type": "string",
							"description": "The Plaid institution ID (e.g. ins_1)"
						},
						"country_codes": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Country codes to search (defaults to [\"US\"])"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "plaid", AuthType: "custom", InstructionsURL: "https://plaid.com/docs/quickstart/"},
		},
		Templates: plaidTemplates(),
	}
}

// plaidTemplates returns the pre-built action configuration templates.
func plaidTemplates() []connectors.ManifestTemplate {
	return []connectors.ManifestTemplate{
		{
			ID:          "tpl_plaid_create_link_token",
			ActionType:  "plaid.create_link_token",
			Name:        "Create bank connection link",
			Description: "Agent can create link tokens to initiate bank connections.",
			Parameters:  json.RawMessage(`{"user_id":"*","products":"*"}`),
		},
		{
			ID:          "tpl_plaid_get_balances",
			ActionType:  "plaid.get_balances",
			Name:        "Read account balances",
			Description: "Agent can read real-time account balances.",
			Parameters:  json.RawMessage(`{"access_token":"*"}`),
		},
		{
			ID:          "tpl_plaid_list_transactions",
			ActionType:  "plaid.list_transactions",
			Name:        "List transactions",
			Description: "Agent can list transactions with date range filters.",
			Parameters:  json.RawMessage(`{"access_token":"*","start_date":"*","end_date":"*"}`),
		},
		{
			ID:          "tpl_plaid_get_accounts",
			ActionType:  "plaid.get_accounts",
			Name:        "Read account details",
			Description: "Agent can read account details (name, type, mask).",
			Parameters:  json.RawMessage(`{"access_token":"*"}`),
		},
		{
			ID:          "tpl_plaid_get_identity",
			ActionType:  "plaid.get_identity",
			Name:        "Read identity information",
			Description: "Agent can read account holder identity (PII — name, address, phone).",
			Parameters:  json.RawMessage(`{"access_token":"*"}`),
		},
		{
			ID:          "tpl_plaid_get_institution",
			ActionType:  "plaid.get_institution",
			Name:        "Look up institutions",
			Description: "Agent can look up financial institution details.",
			Parameters:  json.RawMessage(`{"institution_id":"*"}`),
		},
	}
}
