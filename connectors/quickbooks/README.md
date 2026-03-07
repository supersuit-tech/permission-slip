# QuickBooks Online Connector

The QuickBooks Online connector integrates Permission Slip with the [QuickBooks Online REST API](https://developer.intuit.com/app/developer/qbo/docs/api/accounting/all-entities/account). It uses plain `net/http` — no third-party SDK.

## Connector ID

`quickbooks`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | An OAuth 2.0 Bearer token obtained via the QuickBooks OAuth flow. |
| `realm_id` | Yes | The QuickBooks company ID (also called "realm ID"), returned during the OAuth callback. |

The credential `auth_type` in the database is `oauth2`. The connector defines a custom OAuth provider (`quickbooks`) with Intuit's authorize and token URLs.

**Setup:** [QuickBooks Authentication & Authorization](https://developer.intuit.com/app/developer/qbo/docs/develop/authentication-and-authorization)

**OAuth scopes:** `com.intuit.quickbooks.accounting`

**Sandbox:** Intuit provides a [sandbox environment](https://developer.intuit.com/app/developer/qbo/docs/develop/sandboxes) for testing. The `realm_id` for your sandbox company is separate from production.

## Actions

| Action | Risk | Required Params | Description |
|--------|------|-----------------|-------------|
| `quickbooks.list_accounts` | low | *(none)* | List accounts from the chart of accounts. Optional: `account_type`, `max_results` |
| `quickbooks.get_profit_loss` | low | *(none)* | Retrieve the Profit & Loss (income statement) report. Optional: `start_date`, `end_date` |
| `quickbooks.get_balance_sheet` | low | *(none)* | Retrieve the Balance Sheet report. Optional: `start_date`, `end_date` |
| `quickbooks.create_customer` | medium | `display_name` | Create a new customer record. Optional: `given_name`, `family_name`, `email`, `phone`, `company_name` |
| `quickbooks.create_invoice` | medium | `customer_id`, `line_items` | Create an invoice with line items. Optional: `due_date`, `email_to` |
| `quickbooks.record_payment` | high | `customer_id`, `amount` | Record a customer payment. Optional: `invoice_id` |
| `quickbooks.create_expense` | high | `account_id`, `lines` | Create an expense (purchase) against an account. Optional: `payment_type`, `vendor_id`, `txn_date` |
| `quickbooks.reconcile_transaction` | high | `account_id`, `amount` | Create a bank deposit for reconciliation. Optional: `txn_date`, `description` |

## Validation Limits

- Invoice line items: max 250
- Expense line items: max 250
- List accounts max_results: 1–1000 (default 100)
- Account type filter: validated against a 15-type allowlist (see `validAccountTypes` in `list_accounts.go`)
- Amounts: must be > 0 (minimum 0.01)
- Dates: YYYY-MM-DD format

## Configuration Templates

Templates represent progressive permission levels:

| Template | Action | Risk | Use Case |
|----------|--------|------|----------|
| View Profit & Loss | `get_profit_loss` | low | Read-only financial reporting |
| View Balance Sheet | `get_balance_sheet` | low | Read-only financial reporting |
| List chart of accounts | `list_accounts` | low | Read-only account browsing |
| Create invoices | `create_invoice` | medium | Billing automation |
| Create customers | `create_customer` | medium | CRM integration |
| Record payments | `record_payment` | high | Accounts receivable |
| Create expenses | `create_expense` | high | Expense tracking |
| Reconcile transactions | `reconcile_transaction` | high | Bank reconciliation |

## Error Handling

The connector maps QuickBooks API errors to typed connector errors:

| HTTP Status | Error Type | Meaning |
|-------------|-----------|---------|
| 400 | `ValidationError` | Bad request parameters |
| 401 | `AuthError` | Invalid or expired token |
| 403 | `AuthError` | Insufficient permissions |
| 404 | `ValidationError` | Resource not found |
| 429 | `RateLimitError` | Rate limit exceeded (respects `Retry-After` header) |
| 5xx | `ExternalError` | QuickBooks service error |

QuickBooks returns structured `Fault` error envelopes; the connector extracts the first error's `Message`, `Detail`, and `code` fields for informative error messages.

## Adding a New Action

1. Create `action_name.go` with a struct embedding `conn *QuickBooksConnector` and implementing `connectors.Action`.
2. Register the action in `Actions()` in `quickbooks.go`.
3. Add the action schema to `manifest.go` (including risk level and JSON Schema).
4. Add a template in `quickbooksTemplates()` if relevant.
5. Write tests using `httptest.NewServer()` (see existing `*_test.go` files for patterns).
6. Run `go test ./connectors/quickbooks/...` to verify.
