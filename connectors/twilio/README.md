# Twilio Connector

The Twilio connector integrates Permission Slip with the [Twilio REST API](https://www.twilio.com/docs/usage/api). It uses plain `net/http` with HTTP Basic Auth and form-encoded POST requests — no third-party Twilio SDK.

## Connector ID

`twilio`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `account_sid` | Yes | Twilio Account SID — starts with `AC`, 34 characters (e.g., `ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`). Found on [Twilio Console](https://console.twilio.com/). |
| `auth_token` | Yes | Twilio Auth Token — found alongside the Account SID on the console. See [Twilio docs](https://www.twilio.com/docs/iam/api-keys) for key management. |

The credential `auth_type` in the database is `basic`. Credentials are stored encrypted in Supabase Vault and decrypted only at execution time.

### Authentication

All API requests use [HTTP Basic Auth](https://www.twilio.com/docs/iam/api/account#authenticate-with-http) with the Account SID as the username and Auth Token as the password.

## Actions

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `twilio.send_sms` | Send SMS | medium | Send an SMS or MMS message |
| `twilio.send_whatsapp` | Send WhatsApp Message | medium | Send a WhatsApp message (auto-prefixes `whatsapp:` on numbers) |
| `twilio.initiate_call` | Initiate Call | high | Initiate an outbound voice call with TwiML instructions |
| `twilio.get_message` | Get Message Status | low | Check the delivery status of a sent message |
| `twilio.get_call` | Get Call Status | low | Check the status and details of a call |
| `twilio.lookup_phone` | Phone Number Lookup | low | Look up information about a phone number via Lookup API v2 |

### Phone Number Validation

All phone numbers (`to`, `from`, `phone_number`) must be in [E.164 format](https://www.twilio.com/docs/glossary/what-e164): a `+` followed by 1–15 digits, starting with a non-zero country code (e.g., `+15551234567`). Invalid formats return a `ValidationError` with a descriptive message.

### SMS Body Length

SMS message bodies are capped at 1600 characters. Messages exceeding this limit are rejected with a `ValidationError` before hitting the Twilio API.

### SID Validation

- Message SIDs must start with `SM` (SMS) or `MM` (MMS)
- Call SIDs must start with `CA`

Invalid prefixes are rejected with a `ValidationError`.

### WhatsApp Number Prefixing

The `twilio.send_whatsapp` action automatically prepends `whatsapp:` to both `To` and `From` numbers when calling the Twilio API. Callers provide plain E.164 numbers — the connector handles the prefix.

## API Endpoints

| Action | Method | Endpoint |
|--------|--------|----------|
| send_sms | POST | `/2010-04-01/Accounts/{sid}/Messages.json` |
| send_whatsapp | POST | `/2010-04-01/Accounts/{sid}/Messages.json` |
| initiate_call | POST | `/2010-04-01/Accounts/{sid}/Calls.json` |
| get_message | GET | `/2010-04-01/Accounts/{sid}/Messages/{MessageSid}.json` |
| get_call | GET | `/2010-04-01/Accounts/{sid}/Calls/{CallSid}.json` |
| lookup_phone | GET | `https://lookups.twilio.com/v2/PhoneNumbers/{number}` |

Write operations (send_sms, send_whatsapp, initiate_call) use `application/x-www-form-urlencoded` request bodies. Read operations (get_message, get_call, lookup_phone) use query-less GET requests with the resource identifier in the URL path.

## Error Handling

The connector maps Twilio API responses to typed connector errors:

| Twilio Status | Connector Error | HTTP Response |
|---------------|-----------------|---------------|
| 400 | `ValidationError` | 400 Bad Request |
| 401 | `AuthError` | 502 Bad Gateway |
| 403 | `AuthError` | 502 Bad Gateway |
| 404 | `ExternalError` | 502 Bad Gateway |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |

Twilio error responses include a numeric error code and a `more_info` URL. The connector extracts these into the error message (e.g., `[20003] Authentication Error (see https://...)`). Raw response bodies are truncated to 512 characters in error messages to prevent leaking large payloads.

Rate limit responses include the `Retry-After` header value when available.

### Response Size Limit

All responses are capped at 1 MiB (`io.LimitReader`) to prevent memory exhaustion from unexpectedly large responses.

## File Structure

```
connectors/twilio/
├── twilio.go           # TwilioConnector struct, New(), Manifest(), Actions(), ValidateCredentials(), doForm(), doGet()
├── response.go         # checkResponse() — HTTP status → typed error mapping
├── send_sms.go         # twilio.send_sms action
├── send_whatsapp.go    # twilio.send_whatsapp action
├── initiate_call.go    # twilio.initiate_call action
├── get_message.go      # twilio.get_message action
├── get_call.go         # twilio.get_call action
├── lookup_phone.go     # twilio.lookup_phone action
├── *_test.go           # Tests for each action + connector + response
├── helpers_test.go     # Shared test helpers (validCreds, testAccountSID)
└── README.md           # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Twilio API — no real API calls are made.

```bash
go test ./connectors/twilio/... -v
```
