# ADR-009: Connector Execution Architecture

**Status:** Accepted
**Date:** 2026-02-25
**Authors:** SuperSuit team

## Context

Permission Slip has a mature **metadata layer** for connectors: database tables define connectors, their actions, risk levels, parameter schemas, and required credentials. Agents discover available actions via `GET /v1/connectors`, and users enable connectors per-agent via the dashboard.

What's missing is the **execution layer** — the code that actually performs actions against external services when an agent's request is approved. Today, `handleExecuteStandingApproval` records the execution in the database and emits an audit event, but no HTTP call, API interaction, or side effect occurs.

This ADR defines the architecture for connector execution: how connector implementations are structured, how they receive credentials, how actions are dispatched, and how the system scales as connectors grow to support hundreds of actions each.

---

## Decisions

### 1. In-Process Go Implementations

Connectors are implemented as Go packages compiled into the Permission Slip binary. Each connector package lives under a top-level `connectors/` directory and is registered at startup.

**Why this over alternatives:**

| Option | Trade-off |
|---|---|
| **A: In-process Go (chosen)** | Simplest deployment (single binary), compile-time type safety, no IPC overhead, consistent with existing architecture |
| B: Out-of-process (sidecar/subprocess) | Adds deployment complexity, IPC serialization, health monitoring — overhead not justified for initial connector set |
| C: Webhook-based (external HTTP) | Shifts execution outside our trust boundary, introduces latency and reliability concerns, harder to credential-inject safely |

The single-binary model is a core architectural property of Permission Slip. In-process connectors preserve it.

**Trade-off acknowledged:** Adding a new connector requires recompiling. This is acceptable — connectors are a security boundary, and new connectors should go through code review.

---

### 2. Two-Level Interface: Connector + Action

Rather than a monolithic `Execute` method on the connector, the architecture uses two interfaces that separate connector-level concerns (identity, client setup, credential validation) from action-level concerns (parameter handling, API calls):

```go
// connectors/connector.go

// Action executes a single action type against an external service.
type Action interface {
    Execute(ctx context.Context, req ActionRequest) (*ActionResult, error)
}

// Connector represents an integration with an external service.
// It owns shared configuration (HTTP clients, base URLs, auth helpers)
// and registers the actions it supports.
type Connector interface {
    // ID returns the connector identifier (e.g., "github", "slack").
    // Must match the connectors.id value in the database.
    ID() string

    // Actions returns a map of action_type -> Action handler.
    // Keys must match connector_actions.action_type in the database.
    Actions() map[string]Action

    // ValidateCredentials checks that the provided credentials are
    // sufficient and well-formed for this connector (e.g., API key
    // format, required scopes). Called before first execution.
    ValidateCredentials(ctx context.Context, creds map[string]string) error
}

// ActionRequest is passed to every Action.Execute call.
type ActionRequest struct {
    ActionType  string                 // e.g., "github.create_issue"
    Parameters  json.RawMessage        // validated against schema before reaching here
    Credentials map[string]string      // decrypted at execution time; see Decision 4
}

// ActionResult is returned from a successful execution.
type ActionResult struct {
    Data json.RawMessage // service-specific response payload
}
```

**Why two levels:**
- A GitHub connector with 200 actions shouldn't be a 10,000-line file with a giant switch statement.
- Each action is independently testable — mock the credentials, call `Execute`, assert the result.
- Shared connector logic (HTTP client, auth headers, rate limiting, error mapping) is defined once and injected into actions.

---

### 3. File Organization: One Action Per File

Each connector is a Go package. Within the package, shared code lives in one file and each action lives in its own file:

```
connectors/
├── connector.go              # Connector and Action interfaces, Registry, shared types
├── github/
│   ├── github.go             # GitHubConnector struct, shared client, auth helpers
│   ├── create_issue.go       # github.create_issue action
│   ├── merge_pr.go           # github.merge_pr action
│   ├── list_repos.go         # github.list_repos action
│   ├── create_issue_test.go
│   ├── merge_pr_test.go
│   └── ...
├── slack/
│   ├── slack.go              # SlackConnector struct, shared client
│   ├── send_message.go       # slack.send_message action
│   ├── create_channel.go     # slack.create_channel action
│   └── ...
└── registry.go               # Default registry, startup registration
```

**How actions share common code:**

The connector struct holds shared state (HTTP client, base URL, common helpers). Actions receive a pointer to the connector, so they can call shared methods without duplication:

```go
// connectors/github/github.go
package github

type GitHubConnector struct {
    httpClient *http.Client
    baseURL    string
}

// newRequest creates an authenticated GitHub API request.
// Shared by all actions — avoids duplicating auth header setup.
func (c *GitHubConnector) newRequest(ctx context.Context, method, path string, creds map[string]string, body io.Reader) (*http.Request, error) {
    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+creds["api_key"])
    req.Header.Set("Accept", "application/vnd.github+json")
    return req, nil
}

func New() *GitHubConnector {
    return &GitHubConnector{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        baseURL:    "https://api.github.com",
    }
}

func (c *GitHubConnector) ID() string { return "github" }

func (c *GitHubConnector) Actions() map[string]connectors.Action {
    return map[string]connectors.Action{
        "github.create_issue": &createIssueAction{conn: c},
        "github.merge_pr":     &mergePRAction{conn: c},
    }
}
```

```go
// connectors/github/create_issue.go
package github

type createIssueAction struct {
    conn *GitHubConnector  // shared client, helpers, auth
}

func (a *createIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
    var params struct {
        Owner string `json:"owner"`
        Repo  string `json:"repo"`
        Title string `json:"title"`
        Body  string `json:"body"`
    }
    if err := json.Unmarshal(req.Parameters, &params); err != nil {
        return nil, fmt.Errorf("invalid parameters: %w", err)
    }

    path := fmt.Sprintf("/repos/%s/%s/issues", params.Owner, params.Repo)
    httpReq, err := a.conn.newRequest(ctx, "POST", path, req.Credentials, /* body */)
    // ... execute request using a.conn.httpClient ...
}
```

**Why this scales:**
- Adding a new action = adding one file + one line in `Actions()`. No other files change.
- A connector with 1,000 actions has 1,000 small, focused files — each independently readable, reviewable, and testable.
- Go's package system provides natural scoping — action types don't collide across connectors.
- Shared helpers (auth, error mapping, pagination) are methods on the connector struct, available to all actions without import gymnastics.

---

### 4. Credential Injection at Execution Time

Credentials are decrypted from the vault **only at execution time** and passed to the action via `ActionRequest.Credentials`. They are never stored in memory longer than the execution duration.

**Flow:**

```
Agent request
    │
    ▼
Validate signature + match standing approval (or one-off token)
    │
    ▼
Look up user's stored credential for this connector's required service
    │
    ▼
Decrypt from Supabase Vault (VaultStore.ReadSecret)
    │
    ▼
Pass decrypted credentials to Action.Execute via ActionRequest
    │
    ▼
Action uses credentials to call external service
    │
    ▼
Credentials go out of scope — not cached, not stored
```

**Why not inject credentials into the connector at startup:**
- Credentials are user-scoped (different users have different API keys for the same service).
- An agent may act on behalf of different users over its lifetime.
- Decrypting only at execution time minimizes the window where plaintext credentials exist in memory.

---

### 5. Connector Registry

A registry maps connector IDs to their implementations. It's populated at startup and injected into the API handler via `Deps`:

```go
// connectors/registry.go

type Registry struct {
    connectors map[string]Connector
}

func NewRegistry() *Registry {
    return &Registry{connectors: make(map[string]Connector)}
}

func (r *Registry) Register(c Connector) {
    r.connectors[c.ID()] = c
}

func (r *Registry) Get(id string) (Connector, bool) {
    c, ok := r.connectors[id]
    return c, ok
}

func (r *Registry) GetAction(actionType string) (Action, bool) {
    // actionType format is "connector.action", e.g. "github.create_issue"
    // Extract connector ID from prefix
    parts := strings.SplitN(actionType, ".", 2)
    if len(parts) != 2 {
        return nil, false
    }
    conn, ok := r.connectors[parts[0]]
    if !ok {
        return nil, false
    }
    action, ok := conn.Actions()[actionType]
    return action, ok
}
```

**Startup wiring** (in `main.go`):

```go
registry := connectors.NewRegistry()
registry.Register(github.New())
registry.Register(slack.New())

deps := &api.Deps{
    // ...existing fields...
    Connectors: registry,
}
```

The registry is added to `api.Deps` alongside the existing `DB`, `Vault`, etc. fields — same dependency injection pattern used throughout the codebase (see ADR-008).

---

### 6. Custom Connector Support (Future-Compatible, Not Initially Implemented)

The interface-based design means external connector implementations are architecturally possible without any changes to the core system. Anyone can implement the `Connector` and `Action` interfaces.

**What this means concretely:**
- The `Connector` interface is the contract. Any Go package that implements it is a valid connector.
- The `Registry` accepts any `Connector` implementation — it doesn't distinguish "built-in" from "custom."
- A future plugin loading mechanism (Go `plugin.Open`, gRPC sidecar, WASM, etc.) would just be a new way to get `Connector` implementations into the registry.

**What we're not building now:**
- No dynamic loading at runtime.
- No connector marketplace or distribution mechanism.
- No configuration UI for custom connectors.
- All connectors ship compiled into the binary.

**Why design for it anyway:**
- The interface costs nothing — it's the same abstraction we'd use for built-in connectors.
- It avoids painting ourselves into a corner (e.g., if we hard-coded connector logic into the API handlers, extracting it later would be painful).
- Third-party developers who fork Permission Slip can add connectors by implementing the interface and calling `registry.Register()`.

---

## Execution Flow (End to End)

Bringing it together, here's how an approved action executes:

```
Agent                          Permission Slip                        External Service
  │                                  │                                      │
  │  POST /v1/actions/execute        │                                      │
  │  (or standing approval execute)  │                                      │
  │─────────────────────────────────>│                                      │
  │                                  │                                      │
  │                   1. Verify authorization (token or standing approval)  │
  │                   2. Look up action_type in connector registry          │
  │                   3. Find user's credential for the connector's service │
  │                   4. Decrypt credential from Vault                      │
  │                   5. Call action.Execute(ctx, ActionRequest{            │
  │                        ActionType: "github.create_issue",              │
  │                        Parameters: {...},                              │
  │                        Credentials: {"api_key": "ghp_..."},            │
  │                      })                                                │
  │                                  │───────────────────────────────────>  │
  │                                  │  HTTP request with user's API key    │
  │                                  │<───────────────────────────────────  │
  │                   6. Record execution result                           │
  │                   7. Emit audit event                                   │
  │                                  │                                      │
  │<─────────────────────────────────│                                      │
  │  {result: {data: {...}}}         │                                      │
```

---

## Error Handling

Actions return Go errors. The execution layer maps them to appropriate HTTP responses:

| Error Type | HTTP Status | Example |
|---|---|---|
| Parameter validation failure | 400 Bad Request | Missing required field, wrong type |
| Credential validation failure | 400 Bad Request | API key format invalid, missing scope |
| External service auth failure | 502 Bad Gateway | API key revoked, expired token |
| External service error | 502 Bad Gateway | GitHub API returned 500 |
| External service rate limit | 429 Too Many Requests | Connector should surface retry-after |
| Timeout | 504 Gateway Timeout | External service didn't respond in time |
| No connector registered | 400 Bad Request | action_type prefix doesn't match any connector |
| No action found | 400 Bad Request | action_type exists in DB but not in code (data/code mismatch) |

Actions should return typed errors (e.g., `*connectors.ExternalError`, `*connectors.AuthError`) so the execution layer can map them precisely. Untyped errors default to 500.

---

## Rationale

- **Single binary preserved.** In-process execution maintains the zero-dependency deployment model. No sidecars, no plugin directories, no runtime dependencies beyond Postgres.
- **Two-level interface prevents monoliths.** Splitting Connector (shared setup) from Action (per-operation logic) means each action is a small, focused unit — critical when connectors grow to hundreds of actions.
- **One file per action scales linearly.** Adding a new action is a single file + one line of registration. No existing files modified. Code review scope is minimal.
- **Interface-first design keeps options open.** The same interface that structures built-in connectors also enables future custom connector support. No refactoring needed.
- **Credential isolation maintained.** Credentials are decrypted only at execution time, passed in, and discarded. The architecture doesn't introduce any new credential storage or caching.

## Alternatives Considered

- **Monolithic connector Execute method.** A single `Execute(actionType, params, creds)` on the connector, with a switch statement dispatching to actions. Works for small connectors, but becomes unmanageable at scale. A connector with 200 actions would be a single massive file.
- **Action types as standalone functions.** Each action as a top-level function instead of a struct implementing an interface. Loses the ability to share connector state (HTTP client, helpers) without passing it as extra parameters to every function.
- **Database-driven execution (stored procedures).** Executing actions via PL/pgSQL functions that make HTTP calls using `pg_net`. Keeps everything in Postgres, but makes debugging, testing, and error handling significantly harder. External HTTP calls from the database also complicate connection pooling and timeout management.
- **Configuration-based connector definitions (no code).** Define connectors entirely via YAML/JSON config (URL templates, header mappings, response transforms). Works for simple REST APIs but breaks down for services with complex auth flows, pagination, error handling, or multi-step operations.

## Consequences

- A new top-level `connectors/` directory is added to the project.
- `api.Deps` gains a `Connectors` field of type `*connectors.Registry`.
- The execution endpoints (`/v1/actions/execute` and standing approval execute) gain a code path that calls `registry.GetAction(actionType)` and invokes the action with decrypted credentials.
- Each connector needs a corresponding entry in the database (`connectors` + `connector_actions` tables). If a connector is registered in code but not in the database (or vice versa), the system should log a warning at startup.
- Connector implementations need their own test suites. Unit tests mock the HTTP client; integration tests (optional, not in CI) hit real APIs with test credentials.
- The `connectors/` package has no dependency on `api/` or `db/` — it only depends on standard library and external service SDKs. The API layer depends on `connectors/`, not the reverse.
