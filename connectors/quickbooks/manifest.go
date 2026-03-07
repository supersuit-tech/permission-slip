package quickbooks

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest.
func (c *QuickBooksConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "quickbooks",
		Name:        "QuickBooks Online",
		Description: "QuickBooks Online integration for accounting, invoicing, and financial reporting",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "quickbooks.create_invoice",
				Name:        "Create Invoice",
				Description: "Create and send an invoice with line items, due date, and customer reference",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["customer_id", "line_items"],
					"additionalProperties": false,
					"properties": {
						"customer_id": {
							"type": "string",
							"description": "QuickBooks customer ID (e.g. \"42\")"
						},
						"due_date": {
							"type": "string",
							"description": "Invoice due date in YYYY-MM-DD format (e.g. \"2025-12-31\")"
						},
						"line_items": {
							"type": "array",
							"minItems": 1,
							"description": "Invoice line items",
							"items": {
								"type": "object",
								"additionalProperties": false,
								"properties": {
									"description": {
										"type": "string",
										"description": "Line item description shown on the invoice"
									},
									"amount": {
										"type": "number",
										"minimum": 0.01,
										"description": "Unit price in dollars (e.g. 150.00)"
									},
									"quantity": {
										"type": "number",
										"minimum": 1,
										"default": 1,
										"description": "Quantity (defaults to 1)"
									}
								}
							}
						},
						"email_to": {
							"type": "string",
							"format": "email",
							"description": "Email address to send the invoice to"
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.record_payment",
				Name:        "Record Payment",
				Description: "Record a payment against an invoice — this records a financial transaction",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["customer_id", "amount"],
					"additionalProperties": false,
					"properties": {
						"customer_id": {
							"type": "string",
							"description": "QuickBooks customer ID"
						},
						"amount": {
							"type": "number",
							"minimum": 0.01,
							"description": "Payment amount in dollars (e.g. 500.00)"
						},
						"invoice_id": {
							"type": "string",
							"description": "QuickBooks invoice ID to link this payment to (optional)"
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.create_expense",
				Name:        "Create Expense",
				Description: "Create an expense entry (purchase) — this records a financial transaction",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "lines"],
					"additionalProperties": false,
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Bank or credit card account ID to record the expense against"
						},
						"payment_type": {
							"type": "string",
							"enum": ["Cash", "Check", "CreditCard"],
							"default": "Cash",
							"description": "Payment method used for the expense"
						},
						"lines": {
							"type": "array",
							"minItems": 1,
							"description": "Expense line items",
							"items": {
								"type": "object",
								"additionalProperties": false,
								"properties": {
									"description": {
										"type": "string",
										"description": "Line item description"
									},
									"amount": {
										"type": "number",
										"minimum": 0.01,
										"description": "Line item amount in dollars"
									},
									"account_id": {
										"type": "string",
										"description": "Expense category account ID"
									}
								}
							}
						},
						"vendor_id": {
							"type": "string",
							"description": "Vendor ID for the expense"
						},
						"txn_date": {
							"type": "string",
							"description": "Transaction date in YYYY-MM-DD format"
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.get_profit_loss",
				Name:        "Get Profit & Loss",
				Description: "Retrieve the Profit & Loss (income statement) report — read-only",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"additionalProperties": false,
					"properties": {
						"start_date": {
							"type": "string",
							"description": "Report start date in YYYY-MM-DD format"
						},
						"end_date": {
							"type": "string",
							"description": "Report end date in YYYY-MM-DD format"
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.get_balance_sheet",
				Name:        "Get Balance Sheet",
				Description: "Retrieve the Balance Sheet report — read-only",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"additionalProperties": false,
					"properties": {
						"start_date": {
							"type": "string",
							"description": "Report start date in YYYY-MM-DD format"
						},
						"end_date": {
							"type": "string",
							"description": "Report end date in YYYY-MM-DD format"
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.reconcile_transaction",
				Name:        "Reconcile Transaction",
				Description: "Create a bank deposit to reconcile a transaction — affects bank reconciliation state",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "amount"],
					"additionalProperties": false,
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Bank account ID to deposit into"
						},
						"amount": {
							"type": "number",
							"minimum": 0.01,
							"description": "Deposit amount in dollars"
						},
						"txn_date": {
							"type": "string",
							"description": "Transaction date in YYYY-MM-DD format"
						},
						"description": {
							"type": "string",
							"description": "Description or memo for the deposit"
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.create_customer",
				Name:        "Create Customer",
				Description: "Create a new customer record in QuickBooks",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["display_name"],
					"additionalProperties": false,
					"properties": {
						"display_name": {
							"type": "string",
							"description": "Customer display name (must be unique in QuickBooks)"
						},
						"given_name": {
							"type": "string",
							"description": "Customer first name"
						},
						"family_name": {
							"type": "string",
							"description": "Customer last name"
						},
						"email": {
							"type": "string",
							"format": "email",
							"description": "Customer email address"
						},
						"phone": {
							"type": "string",
							"description": "Customer phone number"
						},
						"company_name": {
							"type": "string",
							"description": "Customer company name"
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.list_accounts",
				Name:        "List Accounts",
				Description: "List accounts from the chart of accounts — read-only",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"additionalProperties": false,
					"properties": {
						"account_type": {
							"type": "string",
							"description": "Filter by account type (e.g. \"Bank\", \"Expense\", \"Income\", \"Accounts Receivable\")"
						},
						"max_results": {
							"type": "integer",
							"default": 100,
							"minimum": 1,
							"maximum": 1000,
							"description": "Maximum number of accounts to return (default 100, max 1000)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "quickbooks",
				AuthType:        "oauth2",
				OAuthProvider:   "quickbooks",
				OAuthScopes:     []string{"com.intuit.quickbooks.accounting"},
				InstructionsURL: "https://developer.intuit.com/app/developer/qbo/docs/develop/authentication-and-authorization",
			},
		},
		OAuthProviders: []connectors.ManifestOAuthProvider{
			{
				ID:           "quickbooks",
				AuthorizeURL: "https://appcenter.intuit.com/connect/oauth2",
				TokenURL:     "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer",
				Scopes:       []string{"com.intuit.quickbooks.accounting"},
			},
		},
		Templates: quickbooksTemplates(),
	}
}

// quickbooksTemplates returns configuration templates for common QuickBooks use cases.
func quickbooksTemplates() []connectors.ManifestTemplate {
	return []connectors.ManifestTemplate{
		// --- Read-only ---
		{
			ID:          "tpl_quickbooks_get_profit_loss",
			ActionType:  "quickbooks.get_profit_loss",
			Name:        "View Profit & Loss report",
			Description: "Agent can retrieve the Profit & Loss report for any date range. Read-only, no financial risk.",
			Parameters:  json.RawMessage(`{"start_date":"*","end_date":"*"}`),
		},
		{
			ID:          "tpl_quickbooks_get_balance_sheet",
			ActionType:  "quickbooks.get_balance_sheet",
			Name:        "View Balance Sheet report",
			Description: "Agent can retrieve the Balance Sheet report for any date range. Read-only, no financial risk.",
			Parameters:  json.RawMessage(`{"start_date":"*","end_date":"*"}`),
		},
		{
			ID:          "tpl_quickbooks_list_accounts",
			ActionType:  "quickbooks.list_accounts",
			Name:        "List chart of accounts",
			Description: "Agent can list accounts from the chart of accounts. Read-only.",
			Parameters:  json.RawMessage(`{"account_type":"*","max_results":"*"}`),
		},
		// --- Write (medium risk) ---
		{
			ID:          "tpl_quickbooks_create_invoices",
			ActionType:  "quickbooks.create_invoice",
			Name:        "Create invoices",
			Description: "Agent can create invoices for any customer with any line items.",
			Parameters:  json.RawMessage(`{"customer_id":"*","due_date":"*","line_items":"*","email_to":"*"}`),
		},
		{
			ID:          "tpl_quickbooks_create_customers",
			ActionType:  "quickbooks.create_customer",
			Name:        "Create customers",
			Description: "Agent can create new customer records in QuickBooks.",
			Parameters:  json.RawMessage(`{"display_name":"*","given_name":"*","family_name":"*","email":"*","phone":"*","company_name":"*"}`),
		},
		// --- Write (high risk) ---
		{
			ID:          "tpl_quickbooks_record_payments",
			ActionType:  "quickbooks.record_payment",
			Name:        "Record payments",
			Description: "Agent can record payments for any customer. High risk — records financial transactions.",
			Parameters:  json.RawMessage(`{"customer_id":"*","amount":"*","invoice_id":"*"}`),
		},
		{
			ID:          "tpl_quickbooks_create_expenses",
			ActionType:  "quickbooks.create_expense",
			Name:        "Create expenses",
			Description: "Agent can create expense entries. High risk — records financial transactions.",
			Parameters:  json.RawMessage(`{"account_id":"*","payment_type":"*","lines":"*","vendor_id":"*","txn_date":"*"}`),
		},
		{
			ID:          "tpl_quickbooks_reconcile",
			ActionType:  "quickbooks.reconcile_transaction",
			Name:        "Reconcile transactions",
			Description: "Agent can create bank deposits to reconcile transactions. High risk — affects bank reconciliation.",
			Parameters:  json.RawMessage(`{"account_id":"*","amount":"*","txn_date":"*","description":"*"}`),
		},
	}
}
