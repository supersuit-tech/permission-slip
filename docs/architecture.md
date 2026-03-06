# Permission Slip — Architecture Diagrams

## System Context

```mermaid
graph TB
    Agent["AI Agent<br/><i>Ed25519 key pair</i>"]
    PS["Permission Slip<br/><i>Middle-man service</i>"]
    User["User / Approver<br/><i>Human-in-the-loop</i>"]
    Gmail["Gmail API"]
    Stripe["Stripe API"]
    Other["Other APIs..."]

    Agent -- "1. Submit structured action<br/>(signed request)" --> PS
    PS -- "2. Push notification<br/>(approval request)" --> User
    User -- "3. Approve / Deny" --> PS
    PS -- "4. Return result<br/>(or denial)" --> Agent
    PS -- "Execute with<br/>stored credentials" --> Gmail
    PS -- "Execute with<br/>stored credentials" --> Stripe
    PS -- "Execute with<br/>stored credentials" --> Other

    style PS fill:#4A90D9,color:#fff,stroke:#2A5F9E
    style Agent fill:#7B68EE,color:#fff,stroke:#5B48CE
    style User fill:#50C878,color:#fff,stroke:#30A858
    style Gmail fill:#EA4335,color:#fff,stroke:#CA2315
    style Stripe fill:#635BFF,color:#fff,stroke:#4339DF
    style Other fill:#999,color:#fff,stroke:#777
```

## Internal Components

> The Connector Engine and individual connectors shown below are implemented as Go interfaces compiled into the binary. See [ADR-009](adr/009-connector-execution-architecture.md) for the execution architecture.

```mermaid
graph TB
    subgraph Agents
        A1["Agent A"]
        A2["Agent B"]
    end

    subgraph PS ["Permission Slip Service"]
        GW["API Gateway<br/><i>TLS, signature verification,<br/>rate limiting</i>"]
        AR["Agent Registry<br/><i>Public keys, metadata,<br/>agent-user links</i>"]
        AE["Action Engine<br/><i>Action types, schemas,<br/>parameter validation</i>"]
        AC["Action Config Engine<br/><i>User-defined configurations,<br/>wildcard matching,<br/>credential binding</i>"]
        APR["Approval Engine<br/><i>Pending requests, TTL,<br/>confirmation codes</i>"]
        TE["Token Engine<br/><i>JWT issuance, single-use<br/>tracking (jti), params_hash</i>"]
        SA["Standing Approval Engine<br/><i>Pre-authorized grants,<br/>constraint matching,<br/>execution counting</i>"]
        NS["Notification Service<br/><i>Push notifications,<br/>webhooks</i>"]
        CV["Credential Vault<br/><i>Encrypted at rest,<br/>OAuth tokens, API keys</i>"]
        OR["OAuth Provider Registry<br/><i>Built-in + manifest providers,<br/>BYOA client credentials</i>"]
        CE["Connector Engine"]

        subgraph Connectors
            GC["Gmail<br/>Connector"]
            SC["Stripe<br/>Connector"]
            OC["..."]
        end
    end

    subgraph External ["External Services"]
        Gmail["Gmail API"]
        Stripe["Stripe API"]
    end

    User["User / Approver"]

    A1 & A2 --> GW
    GW --> AR
    GW --> AE
    AE --> AC
    AC --> APR
    APR --> NS
    NS --> User
    User --> APR
    APR --> TE
    GW --> SA
    SA --> CE
    TE --> CE
    CE --> GC & SC & OC
    GC --> CV
    SC --> CV
    OR --> CV
    CE --> OR
    GC -- "API call with<br/>user credentials" --> Gmail
    SC -- "API call with<br/>user credentials" --> Stripe

    style PS fill:#E8F0FE,stroke:#4A90D9,color:#000
    style GW fill:#4A90D9,color:#fff,stroke:#2A5F9E
    style CV fill:#D93025,color:#fff,stroke:#B9200F
    style OR fill:#E37400,color:#fff,stroke:#C35400
    style TE fill:#F9AB00,color:#000,stroke:#D98B00
    style AR fill:#7B68EE,color:#fff,stroke:#5B48CE
    style AE fill:#34A853,color:#fff,stroke:#1A8833
    style AC fill:#1B8A5A,color:#fff,stroke:#0D6B3F
    style APR fill:#4285F4,color:#fff,stroke:#2265D4
    style NS fill:#FF6D01,color:#fff,stroke:#DF4D00
    style SA fill:#0F9D58,color:#fff,stroke:#0B7A43
    style CE fill:#9AA0A6,color:#fff,stroke:#7A8086
```

## Background Jobs

The server runs periodic background jobs when a database connection is configured. Both jobs are started on server boot, run immediately once, then repeat on a configurable interval. They respect context cancellation for graceful shutdown.

| Job | Default Interval | Description |
|-----|-----------------|-------------|
| **Audit log purge** | 1 hour (`AUDIT_PURGE_INTERVAL`) | Deletes expired audit events to prevent unbounded table growth. |
| **OAuth token refresh** | 10 minutes (`OAUTH_REFRESH_INTERVAL`) | Proactively refreshes OAuth access tokens expiring within 15 minutes. Tokens that fail to refresh (revoked, expired refresh token) are marked `needs_reauth`, prompting the user to re-authorize. |

## Agent Registration Flow

> Registration is **user-initiated** via invite codes. See [ADR-005](adr/005-user-initiated-registration.md) for the rationale.

### Agent-facing flow (confirmation code)

The standard flow: agent registers via invite URL, user shares a confirmation code out-of-band, agent submits the code to complete registration.

```mermaid
sequenceDiagram
    participant User as User / Approver
    participant PS as Permission Slip
    participant Agent

    Note over User: User decides to<br/>add an agent

    User->>+PS: Dashboard: "Add Agent"
    Note over PS: Generate invite code<br/>(e.g. PS-R7K3-X9M4)<br/>Store SHA-256 hash
    PS-->>-User: Display invite code:<br/>PS-R7K3-X9M4

    Note over User: User communicates<br/>invite code to agent<br/>(copy-paste, config, etc.)

    Note over Agent: Generate Ed25519<br/>key pair

    Agent->>+PS: POST /invite/PS-R7K3-X9M4<br/>{request_id, public_key}
    Note over PS: Verify signature matches<br/>public_key in body,<br/>validate invite code<br/>(constant-time comparison),<br/>consume invite (single-use)
    PS-->>-Agent: {agent_id, expires_at, verification_required: true}

    Note over PS: Generate confirmation<br/>code (e.g. XK7-M9P)
    PS->>User: Dashboard update:<br/>Display code: XK7-M9P

    Note over User: User communicates<br/>confirmation code to agent

    Agent->>+PS: POST /v1/agents/{agent_id}/verify<br/>{confirmation_code: "XK7-M9P"}
    Note over PS: Verify code<br/>(constant-time comparison)
    PS-->>-Agent: {status: "approved", registered_at}

    PS->>User: Dashboard update:<br/>"Registration complete"

    Note over Agent: Registration complete.<br/>Agent can now request actions.
```

### Dashboard-facing flow (direct registration)

Alternative flow: user completes registration directly from the dashboard UI without exchanging a confirmation code with the agent. Uses session auth (Supabase JWT).

- **Endpoint**: `POST /v1/agents/{agent_id}/register` (session-authenticated)
- Transitions agent from `pending` → `registered`, sets `registered_at`
- Emits `agent.registered` audit event
- Returns 409 if agent is already registered or deactivated

## Action Approval & Execution Flow

```mermaid
sequenceDiagram
    participant Agent
    participant PS as Permission Slip
    participant User as User / Approver
    participant Ext as External Service<br/>(e.g. Gmail)

    Agent->>+PS: POST /v1/approvals/request<br/>{configuration_id: "ac_...",<br/>parameters: {body: "..."}}
    Note over PS: Validate configuration_id,<br/>verify agent signature,<br/>check agent is registered,<br/>match wildcard parameters
    PS-->>-Agent: {approval_id, status: "pending",<br/>expires_at}

    PS->>User: Push notification:<br/>"Send email to bob@example.com?"

    alt User approves
        User->>PS: Approve
        Note over PS: Generate confirmation<br/>code (e.g. RK3-P7M)
        PS->>User: Display code: RK3-P7M
        Note over User: Communicate code<br/>to agent

        Agent->>+PS: POST /v1/approvals/{id}/verify<br/>{confirmation_code: "RK3-P7M"}
        Note over PS: Verify code, issue<br/>single-use JWT token
        PS-->>-Agent: {status: "approved",<br/>token: {access_token, expires_at}}

        Agent->>+PS: POST /v1/actions/execute<br/>Authorization: Bearer <token><br/>{to, subject, body}
        Note over PS: Verify JWT signature,<br/>check jti (single-use),<br/>verify params_hash
        PS->>+Ext: Gmail API call<br/>(using stored credentials)
        Ext-->>-PS: {message_id, status: "sent"}
        PS-->>-Agent: {message_id, status: "sent"}

    else User denies
        User->>PS: Deny
        Note over Agent: Agent polls or is notified
        Agent->>+PS: POST /v1/approvals/{id}/verify<br/>{confirmation_code: "..."}
        PS-->>-Agent: 403 {error: "approval_denied"}

    else Request expires (5 min TTL)
        Note over PS: TTL elapsed,<br/>no user action
        Agent->>+PS: POST /v1/approvals/{id}/verify<br/>{confirmation_code: "..."}
        PS-->>-Agent: 410 {error: "approval_expired"}
    end
```

## Email MVP — Component View

Focused on the components involved in the Email MVP user stories.

```mermaid
graph LR
    subgraph Agent Side
        SDK["Agent SDK<br/><i>@permissionslip/sdk</i>"]
    end

    subgraph Permission Slip
        API["API Layer"]
        Reg["Agent Registry"]
        Act["Action Engine"]
        ACfg["Action Config Engine<br/><i>User-defined configurations,<br/>wildcard matching</i>"]

        subgraph "Email Actions"
            ER["email.read<br/><i>subject filter, sender filter,<br/>max results, date range</i>"]
            ES["email.send<br/><i>recipient whitelist,<br/>max attachment size,<br/>body length limit</i>"]
        end

        Appr["Approval Engine"]
        Tok["Token Engine"]
        Vault["Credential Vault<br/><i>Gmail OAuth tokens</i>"]
        Gmail["Gmail Connector"]
        Notif["Notification Service"]
    end

    subgraph User Side
        Web["Web Interface"]
        Phone["Push Notifications"]
    end

    subgraph Google
        GmailAPI["Gmail API"]
        OAuth["Google OAuth 2.0"]
    end

    SDK --> API
    API --> Reg
    API --> Act
    Act --> ACfg
    ACfg --> ER & ES
    ER & ES --> Appr
    Appr --> Tok
    Appr --> Notif
    Notif --> Phone
    Web --> Appr
    Web --> Reg
    Web --> Act
    Web --> ACfg
    Web --> Vault
    Tok --> Gmail
    Gmail --> Vault
    Gmail --> GmailAPI
    OAuth --> Vault

    style Vault fill:#D93025,color:#fff
    style Appr fill:#4285F4,color:#fff
    style ACfg fill:#1B8A5A,color:#fff
    style Tok fill:#F9AB00,color:#000
    style Gmail fill:#EA4335,color:#fff
```

## Security Layers

```mermaid
graph TB
    Request["Incoming Agent Request"]

    Request --> L1
    L1["1. Action Validation<br/><i>Pre-defined types only,<br/>schema-validated parameters,<br/>configuration_id required</i>"]
    L1 --> L2
    L2["2. TLS 1.2+<br/><i>Encrypted in transit</i>"]
    L2 --> L3
    L3["3. Signature Verification<br/><i>Ed25519 / ECDSA P-256<br/>signed every request</i>"]
    L3 --> L4
    L4["4. Replay Protection<br/><i>Timestamp window (5 min),<br/>request_id dedup</i>"]
    L4 --> L5
    L5["5. Human Approval<br/><i>One-off: push notification +<br/>confirmation code<br/>Standing: pre-authorized grant</i>"]
    L5 --> L6
    L6["6. Execution Authorization<br/><i>One-off: single-use JWT<br/>(jti tracking, params_hash)<br/>Standing: constraint match +<br/>execution count check</i>"]
    L6 --> L7
    L7["7. Credential Isolation<br/><i>Vault-only access,<br/>never exposed to agent</i>"]
    L7 --> Exec["Action Executed"]

    style L1 fill:#34A853,color:#fff
    style L2 fill:#34A853,color:#fff
    style L3 fill:#4285F4,color:#fff
    style L4 fill:#4285F4,color:#fff
    style L5 fill:#F9AB00,color:#000
    style L6 fill:#F9AB00,color:#000
    style L7 fill:#D93025,color:#fff
    style Exec fill:#4A90D9,color:#fff
```

## Data Model (Email MVP)

```mermaid
erDiagram
    USER {
        string user_id PK
        string username
        string email
    }

    REGISTRATION_INVITE {
        string id PK
        string user_id FK
        string invite_code_hash "SHA-256"
        string status "active | consumed | expired"
        int verification_attempts
        timestamp expires_at
        timestamp created_at
    }

    AGENT {
        bigint agent_id PK "auto-incrementing"
        string public_key
        string name
        string version
        timestamp registered_at
    }

    CREDENTIAL {
        string credential_id PK
        string user_id FK
        string service "gmail"
        blob vault_secret_id "Vault reference"
    }

    CONNECTOR {
        string id PK "gmail, stripe, etc."
        string name
        string description
    }

    AGENT_CONNECTOR {
        bigint agent_id FK
        string approver_id FK
        string connector_id FK
        timestamp enabled_at
    }

    APPROVAL_REQUEST {
        string approval_id PK
        bigint agent_id FK
        string user_id FK
        string action_type
        json parameters
        string status "pending | approved | denied | expired | cancelled"
        string confirmation_code
        int failed_attempts
        timestamp expires_at
        timestamp created_at
    }

    TOKEN {
        string jti PK
        string approval_id FK
        string scope "action type"
        string params_hash "SHA-256 of JCS params"
        boolean consumed
        timestamp expires_at
    }

    ACTION_CONFIGURATION {
        string id PK "ac_ prefix"
        bigint agent_id FK
        string user_id FK
        string connector_id FK
        string action_type FK "composite with connector_id"
        string credential_id FK "nullable, SET NULL"
        json parameters "fixed values or wildcard *"
        string status "active | disabled"
        string name
        string description "nullable"
        timestamp created_at
        timestamp updated_at
    }

    STANDING_APPROVAL {
        string standing_approval_id PK
        bigint agent_id FK
        string user_id FK
        string action_type "email.read | email.send"
        string action_version "1"
        json constraints "same schema as ACTION_CONFIG"
        string status "active | expired | revoked | exhausted"
        int max_executions "null = unlimited"
        int execution_count "current count"
        timestamp starts_at
        timestamp expires_at "null = no expiration"
        timestamp created_at
        timestamp revoked_at "null if not revoked"
    }

    STANDING_APPROVAL_EXECUTION {
        bigint id PK
        string standing_approval_id FK
        json parameters "optional execution params"
        timestamp executed_at
    }

    AUDIT_EVENT {
        bigint id PK
        string user_id FK
        bigint agent_id FK
        string event_type "approval.approved | approval.denied | approval.cancelled | agent.registered | agent.deactivated | standing_approval.executed"
        string outcome
        string source_id "approval_id or agent_id"
        string source_type "approval | agent | standing_approval"
        json agent_meta "point-in-time agent metadata snapshot"
        json action "action context (optional)"
        timestamp created_at
    }

    USER ||--o{ REGISTRATION_INVITE : "creates invite"
    REGISTRATION_INVITE ||--o| AGENT : "authorizes registration"
    USER ||--o{ AGENT : registers
    USER ||--o{ CREDENTIAL : "stores (user-scoped)"
    AGENT ||--o{ AGENT_CONNECTOR : "enabled for"
    AGENT_CONNECTOR }o--|| CONNECTOR : "references"
    AGENT ||--o{ ACTION_CONFIGURATION : "configured for"
    USER ||--o{ ACTION_CONFIGURATION : creates
    CONNECTOR ||--o{ ACTION_CONFIGURATION : "provides action"
    CREDENTIAL ||--o{ ACTION_CONFIGURATION : "bound to (nullable)"
    AGENT ||--o{ APPROVAL_REQUEST : submits
    USER ||--o{ APPROVAL_REQUEST : reviews
    APPROVAL_REQUEST ||--o| TOKEN : "issues (on approve)"
    USER ||--o{ STANDING_APPROVAL : creates
    AGENT ||--o{ STANDING_APPROVAL : "authorized by"
    STANDING_APPROVAL ||--o{ STANDING_APPROVAL_EXECUTION : "tracks executions"
    STANDING_APPROVAL ||--o{ AUDIT_EVENT : "generates (per execution)"
    USER ||--o{ AUDIT_EVENT : owns
    AGENT ||--o{ AUDIT_EVENT : generates
