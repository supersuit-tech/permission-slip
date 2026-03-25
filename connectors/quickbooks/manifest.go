package quickbooks

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest.
func (c *QuickBooksConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "quickbooks",
		Name:        "QuickBooks Online",
		Description: "QuickBooks Online integration for accounting, invoicing, and financial reporting",
		LogoSVG:     logoSVG,
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
							"description": "QuickBooks customer ID (e.g. \"42\")",
							"x-ui": {
								"label": "Customer",
								"help_text": "QuickBooks customer ID — use quickbooks.list_customers to find IDs"
							}
						},
						"due_date": {
							"type": "string",
							"format": "date",
							"description": "Invoice due date in YYYY-MM-DD format (e.g. \"2025-12-31\")",
							"x-ui": {
								"widget": "date",
								"label": "Due date",
								"help_text": "Invoice due date"
							}
						},
						"line_items": {
							"type": "array",
							"minItems": 1,
							"maxItems": 250,
							"description": "Invoice line items (max 250)",
							"x-ui": {
								"label": "Line items"
							},
							"items": {
								"type": "object",
								"additionalProperties": false,
								"properties": {
									"description": {
										"type": "string",
										"description": "Line item description shown on the invoice",
										"x-ui": {
											"label": "Description",
											"placeholder": "Description of goods or services"
										}
									},
									"amount": {
										"type": "number",
										"minimum": 0.01,
										"description": "Unit price in dollars (e.g. 150.00)",
										"x-ui": {
											"label": "Amount",
											"placeholder": "100.00",
											"help_text": "Amount in your company's currency (e.g. 100.00 for $100)"
										}
									},
									"quantity": {
										"type": "number",
										"minimum": 1,
										"default": 1,
										"description": "Quantity (defaults to 1)",
										"x-ui": {
											"label": "Quantity",
											"placeholder": "1"
										}
									}
								}
							}
						},
						"email_to": {
							"type": "string",
							"format": "email",
							"description": "Email address to send the invoice to",
							"x-ui": {
								"label": "Send to email",
								"placeholder": "customer@example.com"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.record_payment",
				Name:        "Record Payment",
				Description: "Record a customer payment against an open invoice — WARNING: this records a financial transaction that affects accounts receivable",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["customer_id", "amount"],
					"additionalProperties": false,
					"properties": {
						"customer_id": {
							"type": "string",
							"description": "QuickBooks customer ID",
							"x-ui": {
								"label": "Customer",
								"help_text": "QuickBooks customer ID — use quickbooks.list_customers to find IDs"
							}
						},
						"amount": {
							"type": "number",
							"minimum": 0.01,
							"description": "Payment amount in dollars (e.g. 500.00)",
							"x-ui": {
								"label": "Amount",
								"placeholder": "100.00",
								"help_text": "Amount in your company's currency (e.g. 100.00 for $100)"
							}
						},
						"invoice_id": {
							"type": "string",
							"description": "QuickBooks invoice ID to link this payment to (optional)",
							"x-ui": {
								"label": "Invoice ID",
								"help_text": "QuickBooks invoice ID to link this payment to"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.create_expense",
				Name:        "Create Expense",
				Description: "Create an expense (purchase transaction) against a bank or credit card account — WARNING: this records a financial transaction that affects your bank balance",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "lines"],
					"additionalProperties": false,
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Bank or credit card account ID to record the expense against",
							"x-ui": {
								"label": "Account",
								"help_text": "QuickBooks account ID — use quickbooks.list_accounts to find IDs"
							}
						},
						"payment_type": {
							"type": "string",
							"enum": ["Cash", "Check", "CreditCard"],
							"default": "Cash",
							"description": "Payment method used for the expense",
							"x-ui": {
								"widget": "select",
								"label": "Payment type"
							}
						},
						"lines": {
							"type": "array",
							"minItems": 1,
							"maxItems": 250,
							"description": "Expense line items (max 250)",
							"x-ui": {
								"label": "Line items"
							},
							"items": {
								"type": "object",
								"additionalProperties": false,
								"properties": {
									"description": {
										"type": "string",
										"description": "Line item description",
										"x-ui": {
											"label": "Description",
											"placeholder": "Description of the expense"
										}
									},
									"amount": {
										"type": "number",
										"minimum": 0.01,
										"description": "Line item amount in dollars",
										"x-ui": {
											"label": "Amount",
											"placeholder": "100.00",
											"help_text": "Amount in your company's currency (e.g. 100.00 for $100)"
										}
									},
									"account_id": {
										"type": "string",
										"description": "Expense category account ID",
										"x-ui": {
											"label": "Account",
											"help_text": "QuickBooks account ID — use quickbooks.list_accounts to find IDs"
										}
									}
								}
							}
						},
						"vendor_id": {
							"type": "string",
							"description": "Vendor ID for the expense",
							"x-ui": {
								"label": "Vendor",
								"help_text": "QuickBooks vendor ID"
							}
						},
						"txn_date": {
							"type": "string",
							"format": "date",
							"description": "Transaction date in YYYY-MM-DD format",
							"x-ui": {
								"widget": "date",
								"label": "Transaction date",
								"help_text": "Date the expense was incurred"
							}
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
							"format": "date",
							"description": "Report start date in YYYY-MM-DD format",
							"x-ui": {
								"widget": "date",
								"datetime_range_pair": "end_date",
								"datetime_range_role": "lower",
								"label": "Start date",
								"help_text": "Beginning of the reporting period"
							}
						},
						"end_date": {
							"type": "string",
							"format": "date",
							"description": "Report end date in YYYY-MM-DD format",
							"x-ui": {
								"widget": "date",
								"datetime_range_pair": "start_date",
								"datetime_range_role": "upper",
								"label": "End date",
								"help_text": "End of the reporting period"
							}
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
							"format": "date",
							"description": "Report start date in YYYY-MM-DD format",
							"x-ui": {
								"widget": "date",
								"datetime_range_pair": "end_date",
								"datetime_range_role": "lower",
								"label": "Start date",
								"help_text": "Beginning of the reporting period"
							}
						},
						"end_date": {
							"type": "string",
							"format": "date",
							"description": "Report end date in YYYY-MM-DD format",
							"x-ui": {
								"widget": "date",
								"datetime_range_pair": "start_date",
								"datetime_range_role": "upper",
								"label": "End date",
								"help_text": "End of the reporting period"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.reconcile_transaction",
				Name:        "Reconcile Transaction",
				Description: "Create a bank deposit to reconcile a transaction — WARNING: this creates a deposit entry that affects bank reconciliation and account balances",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "amount"],
					"additionalProperties": false,
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Bank account ID to deposit into",
							"x-ui": {
								"label": "Account",
								"help_text": "QuickBooks account ID — use quickbooks.list_accounts to find IDs"
							}
						},
						"amount": {
							"type": "number",
							"minimum": 0.01,
							"description": "Deposit amount in dollars",
							"x-ui": {
								"label": "Amount",
								"placeholder": "100.00",
								"help_text": "Amount in your company's currency (e.g. 100.00 for $100)"
							}
						},
						"txn_date": {
							"type": "string",
							"format": "date",
							"description": "Transaction date in YYYY-MM-DD format",
							"x-ui": {
								"widget": "date",
								"label": "Transaction date",
								"help_text": "Date of the deposit"
							}
						},
						"description": {
							"type": "string",
							"description": "Description or memo for the deposit",
							"x-ui": {
								"label": "Description",
								"placeholder": "Memo or description for this deposit",
								"widget": "textarea"
							}
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
							"description": "Customer display name (must be unique in QuickBooks)",
							"x-ui": {
								"label": "Display name",
								"help_text": "Must be unique across all customers/vendors"
							}
						},
						"given_name": {
							"type": "string",
							"description": "Customer first name",
							"x-ui": {
								"label": "First name",
								"placeholder": "Jane"
							}
						},
						"family_name": {
							"type": "string",
							"description": "Customer last name",
							"x-ui": {
								"label": "Last name",
								"placeholder": "Doe"
							}
						},
						"email": {
							"type": "string",
							"format": "email",
							"description": "Customer email address",
							"x-ui": {
								"label": "Email",
								"placeholder": "jane@example.com"
							}
						},
						"phone": {
							"type": "string",
							"description": "Customer phone number",
							"x-ui": {
								"label": "Phone",
								"placeholder": "(555) 123-4567"
							}
						},
						"company_name": {
							"type": "string",
							"description": "Customer company name",
							"x-ui": {
								"label": "Company name",
								"placeholder": "Acme Inc."
							}
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
							"enum": ["Bank", "Accounts Receivable", "Other Current Asset", "Fixed Asset", "Other Asset", "Accounts Payable", "Credit Card", "Other Current Liability", "Long Term Liability", "Equity", "Income", "Cost of Goods Sold", "Expense", "Other Income", "Other Expense"],
							"description": "Filter by account type",
							"x-ui": {
								"widget": "select",
								"label": "Account type"
							}
						},
						"max_results": {
							"type": "integer",
							"default": 100,
							"minimum": 1,
							"maximum": 1000,
							"description": "Maximum number of accounts to return (default 100, max 1000)",
							"x-ui": {
								"label": "Max results"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.create_vendor",
				Name:        "Create Vendor",
				Description: "Create a new vendor record in QuickBooks",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["display_name"],
					"additionalProperties": false,
					"properties": {
						"display_name": {
							"type": "string",
							"description": "Vendor display name (must be unique in QuickBooks)",
							"x-ui": {
								"label": "Display name",
								"help_text": "Must be unique across all customers/vendors"
							}
						},
						"given_name": {
							"type": "string",
							"description": "Vendor contact first name",
							"x-ui": {
								"label": "First name",
								"placeholder": "Jane"
							}
						},
						"family_name": {
							"type": "string",
							"description": "Vendor contact last name",
							"x-ui": {
								"label": "Last name",
								"placeholder": "Doe"
							}
						},
						"email": {
							"type": "string",
							"format": "email",
							"description": "Vendor email address",
							"x-ui": {
								"label": "Email",
								"placeholder": "vendor@example.com"
							}
						},
						"phone": {
							"type": "string",
							"description": "Vendor phone number",
							"x-ui": {
								"label": "Phone",
								"placeholder": "(555) 123-4567"
							}
						},
						"company_name": {
							"type": "string",
							"description": "Vendor company name",
							"x-ui": {
								"label": "Company name",
								"placeholder": "Acme Inc."
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.create_bill",
				Name:        "Create Bill",
				Description: "Create a bill (accounts payable) for a vendor. WARNING: creates a financial liability entry.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["vendor_id", "line_items"],
					"additionalProperties": false,
					"properties": {
						"vendor_id": {
							"type": "string",
							"description": "QuickBooks vendor ID",
							"x-ui": {
								"label": "Vendor",
								"help_text": "QuickBooks vendor ID"
							}
						},
						"line_items": {
							"type": "array",
							"minItems": 1,
							"x-ui": {
								"label": "Line items"
							},
							"items": {
								"type": "object",
								"required": ["amount"],
								"additionalProperties": false,
								"properties": {
									"amount": {
										"type": "number",
										"minimum": 0.01,
										"description": "Line item amount in dollars",
										"x-ui": {
											"label": "Amount",
											"placeholder": "100.00",
											"help_text": "Amount in your company's currency (e.g. 100.00 for $100)"
										}
									},
									"description": {
										"type": "string",
										"description": "Line item description",
										"x-ui": {
											"label": "Description",
											"placeholder": "Description of the bill item"
										}
									},
									"account_id": {
										"type": "string",
										"description": "Expense account ID",
										"x-ui": {
											"label": "Account",
											"help_text": "QuickBooks account ID — use quickbooks.list_accounts to find IDs"
										}
									}
								}
							},
							"description": "Bill line items"
						},
						"due_date": {
							"type": "string",
							"format": "date",
							"description": "Payment due date in YYYY-MM-DD format",
							"x-ui": {
								"widget": "date",
								"label": "Due date",
								"help_text": "Payment due date for this bill"
							}
						},
						"txn_date": {
							"type": "string",
							"format": "date",
							"description": "Bill date in YYYY-MM-DD format (defaults to today)",
							"x-ui": {
								"widget": "date",
								"label": "Bill date",
								"help_text": "Date the bill was received (defaults to today)"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.list_invoices",
				Name:        "List Invoices",
				Description: "List invoices with optional filtering by customer or date range — read-only",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"additionalProperties": false,
					"properties": {
						"customer_id": {
							"type": "string",
							"description": "Filter invoices by customer ID",
							"x-ui": {
								"label": "Customer",
								"help_text": "QuickBooks customer ID — use quickbooks.list_customers to find IDs"
							}
						},
						"start_date": {
							"type": "string",
							"format": "date",
							"description": "Filter invoices on or after this date (YYYY-MM-DD)",
							"x-ui": {
								"widget": "date",
								"datetime_range_pair": "end_date",
								"datetime_range_role": "lower",
								"label": "Start date",
								"help_text": "Filter invoices on or after this date"
							}
						},
						"end_date": {
							"type": "string",
							"format": "date",
							"description": "Filter invoices on or before this date (YYYY-MM-DD)",
							"x-ui": {
								"widget": "date",
								"datetime_range_pair": "start_date",
								"datetime_range_role": "upper",
								"label": "End date",
								"help_text": "Filter invoices on or before this date"
							}
						},
						"max_results": {
							"type": "integer",
							"default": 100,
							"minimum": 1,
							"maximum": 1000,
							"description": "Maximum number of invoices to return (default 100, max 1000)",
							"x-ui": {
								"label": "Max results"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.list_customers",
				Name:        "List Customers",
				Description: "List customer records from QuickBooks — read-only",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"additionalProperties": false,
					"properties": {
						"display_name": {
							"type": "string",
							"description": "Filter customers by display name (partial match)",
							"x-ui": {
								"label": "Display name",
								"placeholder": "Search by name"
							}
						},
						"max_results": {
							"type": "integer",
							"default": 100,
							"minimum": 1,
							"maximum": 1000,
							"description": "Maximum number of customers to return (default 100, max 1000)",
							"x-ui": {
								"label": "Max results"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "quickbooks.send_invoice",
				Name:        "Send Invoice",
				Description: "Email an invoice to a customer via QuickBooks",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["invoice_id"],
					"additionalProperties": false,
					"properties": {
						"invoice_id": {
							"type": "string",
							"description": "QuickBooks invoice ID to send",
							"x-ui": {
								"label": "Invoice ID"
							}
						},
						"email_to": {
							"type": "string",
							"format": "email",
							"description": "Override recipient email address. Omit to use the customer's email on file.",
							"x-ui": {
								"label": "Send to email",
								"placeholder": "customer@example.com"
							}
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
