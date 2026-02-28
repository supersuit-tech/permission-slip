# API Scaffolding

The `api/` package provides the shared foundation for all backend HTTP handlers: structured error responses, JSON request/response helpers, dependency injection, and per-request trace IDs.

## File layout

| File | Purpose |
|---|---|
| `router.go` | `NewRouter` — wires up all domain routes and applies middleware |
| `session.go` | `RequireSession` / `RequireProfile` middleware, `UserID` / `Profile` context helpers |
| `signature.go` | `RequireAgentSignature` middleware — Ed25519 signature verification for agent-authenticated endpoints |
| `agent_auth.go` | Agent authentication helpers (`AuthenticatedAgent` context helper, `parsePathAgentID`) |
| `error_codes.go` | `ErrorCode` type and all error code constants from the API spec |
| `errors.go` | `Error`/`ErrorResponse` types, `RespondError`, and convenience constructors (`BadRequest`, `NotFound`, etc.) |
| `middleware.go` | `TraceIDMiddleware` and `TraceID` context helper |
| `json.go` | `DecodeJSON`, `DecodeJSONOrReject`, `RespondJSON`, and request body limits |
| `rand.go` | `generateRandomBytes`, `generatePrefixedID`, `generateRandomCode`, `hashCodeHex` — shared crypto-random and hashing utilities |
| `invitecode.go` | `generateInviteCode`, `buildInviteURL` |
| `confirmationcode.go` | Confirmation code generation and hashing for agents/approvals |
| `profiles.go` | Profile domain routes (`GET /profile`) |
| `onboarding.go` | Onboarding domain routes (`POST /onboarding`) |
| `registration_invites.go` | Registration invite domain routes (`POST /registration-invites`) |
| `registration.go` | Agent-facing registration routes (`POST /invite/{code}`, `POST /agents/{id}/verify`) |
| `agents.go` | Agent domain routes (list, get, update, deactivate, register, `GET /agents/me`) |
| `agent_connectors.go` | Agent connector routes (list, enable, disable) |
| `approvals.go` | Approval domain routes (list, approve, deny) |
| `capabilities.go` | Agent capabilities endpoint (`GET /agents/{id}/capabilities`) |
| `connectors.go` | Connector catalog routes (`GET /connectors`, `GET /connectors/{id}`) |
| `credentials.go` | Credential management routes (list, store, delete) |
| `standing_approvals.go` | Standing approval domain routes (list, create, revoke, execute) |
| `audit_events.go` | Audit event domain routes (list with filtering and pagination) |

## Error Responses

All API errors use the `ErrorResponse` envelope defined in the Permission Slip spec:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "Missing required field: agent_id",
    "retryable": false,
    "trace_id": "trace_a1b2c3..."
  }
}
```

### ErrorCode

`ErrorCode` is a typed `string` representing machine-readable error codes from the spec (e.g. `ErrInvalidRequest`, `ErrAgentNotFound`). These are the values that appear in the `code` field of every error response. All codes are defined as constants in `api/error_codes.go`.

### Convenience Constructors

Each HTTP status code used by the spec has a constructor. Constructors that map to multiple error codes (400, 401, 403, 404, 409, 410) accept an `ErrorCode` parameter. Constructors with a fixed code (429, 500, 503) do not.

| Constructor | Status | Accepts ErrorCode? | Example |
|---|---|---|---|
| `BadRequest(code, msg)` | 400 | Yes | `BadRequest(ErrInvalidPublicKey, "bad key format")` |
| `Unauthorized(code, msg)` | 401 | Yes | `Unauthorized(ErrInvalidToken, "token expired")` |
| `Forbidden(code, msg)` | 403 | Yes | `Forbidden(ErrAgentNotAuthorized, "not allowed")` |
| `NotFound(code, msg)` | 404 | Yes | `NotFound(ErrApprovalNotFound, "no such approval")` |
| `Conflict(code, msg)` | 409 | Yes | `Conflict(ErrDuplicateRequestID, "request already exists")` |
| `Gone(code, msg)` | 410 | Yes | `Gone(ErrRegistrationExpired, "registration expired")` |
| `TooManyRequests(msg, retryAfter)` | 429 | No (always `rate_limited`) | `TooManyRequests("slow down", 60)` |
| `InternalError(msg)` | 500 | No (always `internal_error`) | `InternalError("unexpected failure")` |
| `ServiceUnavailable(msg)` | 503 | No (always `service_unavailable`) | `ServiceUnavailable("database unreachable")` |

### Writing Error Responses

Use `RespondError` to write an error to the response. It takes the `*http.Request` so it can pull the trace ID from context:

```go
RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "agent not found"))
```

## JSON Helpers

### DecodeJSON

`DecodeJSON(r, &dst)` reads and decodes a JSON request body. It enforces:

- `Content-Type: application/json` (validated with `mime.ParseMediaType` for strict media type matching)
- 1 MB max body size
- No trailing content after the JSON value

Unknown fields in the request body are silently ignored for forward-compatibility — clients may send fields added in newer spec versions.

### DecodeJSONOrReject

`DecodeJSONOrReject(w, r, &dst)` wraps `DecodeJSON` and writes the appropriate error response on failure. Returns `true` on success, `false` if an error was written:

```go
var req CreateAgentRequest
if !DecodeJSONOrReject(w, r, &req) {
    return // error already written
}
```

### RespondJSON

`RespondJSON(w, status, v)` encodes `v` as JSON and writes it with the given status code.

## Trace IDs

Every API request gets a unique trace ID (`trace_` + 32 hex chars) generated by `TraceIDMiddleware`. If the cryptographic random source fails, the middleware logs a warning and falls back to a timestamp-based ID (`trace_t<unix_nano>`) so that observability degrades gracefully rather than silently. The middleware stores the trace ID in the request context and `RespondError` automatically injects it into error responses.

Handlers can access the trace ID directly for logging:

```go
traceID := api.TraceID(r.Context())
log.Printf("[%s] processing request", traceID)
```

If an `ErrorResponse` already has an explicit `TraceID` set, `RespondError` will not override it.

## Dependency Injection

`NewRouter` accepts a `*Deps` struct that holds shared dependencies:

```go
type Deps struct {
    DB                db.DBTX          // nil when running without a database
    Vault             vault.VaultStore // credential secret encryption; nil returns 503 on credential endpoints
    SupabaseJWTSecret string           // HMAC-SHA256 secret for HS256 JWTs (Supabase CLI v1 / test env)
    SupabaseJWKSURL   string           // JWKS endpoint for ES256 JWTs (Supabase CLI v2+)
    JWKSCache         *JWKSCache       // JWKS key cache; initialized once at startup when SupabaseJWKSURL is set
    BaseURL           string           // Public base URL (e.g. "https://app.permissionslip.dev"); used to construct invite URLs
    InviteHMACKey     string           // HMAC key for hashing short codes (invite codes, confirmation codes); if empty, falls back to plain SHA-256
    DevMode           bool             // true when MODE=development; disables rate limiting
}
```

The `Deps` struct is initialized in `main.go`. When `DATABASE_URL` is not set, `deps.DB` is `nil` and the server starts without a database. JWT validation supports both ES256 (via `SupabaseJWKSURL` derived from `SUPABASE_URL`) and HS256 (via `SupabaseJWTSecret` from `SUPABASE_JWT_SECRET`). `BaseURL` is read from `BASE_URL`, `InviteHMACKey` from `INVITE_HMAC_KEY`, and `Vault` from `VAULT_SECRET_KEY`.

## Route Registration

Each domain registers its routes in its own file via a `Register*Routes` function. `NewRouter` calls them all:

```go
// api/router.go
func NewRouter(deps *Deps) http.Handler {
    mux := http.NewServeMux()

    RegisterAgentRoutes(mux, deps)
    RegisterAgentConnectorRoutes(mux, deps)
    RegisterApprovalRoutes(mux, deps)
    RegisterCapabilityRoutes(mux, deps)
    RegisterAuditEventRoutes(mux, deps)
    RegisterConnectorRoutes(mux, deps)
    RegisterCredentialRoutes(mux, deps)
    RegisterOnboardingRoutes(mux, deps)
    RegisterProfileRoutes(mux, deps)
    RegisterRegistrationInviteRoutes(mux, deps)
    RegisterRegistrationRoutes(mux, deps)
    RegisterStandingApprovalRoutes(mux, deps)

    return TraceIDMiddleware(mux)
}
```

Domain files define the registration function and handler closures:

```go
// api/agents.go
func RegisterAgentRoutes(mux *http.ServeMux, deps *Deps) {
    requireProfile := RequireProfile(deps)
    mux.Handle("GET /agents", requireProfile(handleListAgents(deps)))
    mux.Handle("GET /agents/{agent_id}", requireProfile(handleGetAgent(deps)))
    // ...
}
```

To add a new domain, create a new file (e.g., `api/webhooks.go`) with a `Register*Routes` function and add the call in `NewRouter`. This pattern keeps the router small and prevents merge conflicts when multiple features are developed in parallel.

## Session Authentication

`RequireSession` is middleware that validates Supabase session JWTs from the `Authorization: Bearer <token>` header. It supports two signing algorithms:

- **ES256** (preferred, Supabase CLI v2+): Validates via JWKS endpoint using `JWKSCache`
- **HS256** (legacy, Supabase CLI v1 / test env): Validates with `SupabaseJWTSecret`

Both algorithms verify:
- `aud` claim equals `"authenticated"` (the `SupabaseAudAuthenticated` constant)
- Token is not expired
- `sub` claim (user UUID) is present

On success, the user ID is stored in the request context. Retrieve it with `UserID(ctx)`.

### RequireProfile

`RequireProfile` chains `RequireSession` and then looks up the user's profile in the database. If the profile exists, it is stored in the request context for retrieval via `Profile(ctx)`. This eliminates the need for individual handlers to repeat the auth check, DB lookup, and nil check.

Use `RequireProfile` when the handler needs the profile object or when you want to guarantee the user has a profile row. Use `RequireSession` alone for routes that only need the user ID.

```go
func RegisterProfileRoutes(mux *http.ServeMux, deps *Deps) {
    requireProfile := RequireProfile(deps)
    mux.Handle("GET /profile", requireProfile(handleGetProfile()))
}

func handleGetProfile() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        profile := Profile(r.Context()) // *db.Profile, guaranteed non-nil
        // ...
    }
}
```

Handlers behind `RequireProfile` should read `Profile(r.Context())` to make the middleware dependency explicit. If the middleware were accidentally removed, the nil return would surface the misconfiguration immediately.

Note: Both `RequireSession` and `RequireProfile` wrap an `http.Handler` (not `http.HandlerFunc`), so use `mux.Handle` instead of `mux.HandleFunc` for protected routes.

### Agent Signature Authentication

`RequireAgentSignature` is middleware for agent-facing endpoints (e.g., `GET /agents/me`, `GET /agents/{id}/capabilities`). It verifies an Ed25519 signature in the `X-Permission-Slip-Signature` header. See [docs/agents.md](agents.md#signing-requests) for the signature format.

On success, the authenticated agent is stored in the request context. Retrieve it with `AuthenticatedAgent(ctx)`.

For endpoints where the handler computes the agent ID from the path (e.g., registration), use the lower-level `requireAgentSignature` function directly instead of the middleware.

## API Response Types

Database models in `db/` do not carry json tags — they represent the database schema, not the API contract. Each API handler defines its own response struct with json tags:

```go
// db/profiles.go — no json tags
type Profile struct {
    ID        string
    Username  string
    CreatedAt time.Time
}

// api/profiles.go — json tags for the API response
type profileResponse struct {
    ID        string    `json:"id"`
    Username  string    `json:"username"`
    CreatedAt time.Time `json:"created_at"`
}
```

This decouples the database schema from the API contract, allowing either to change independently.
