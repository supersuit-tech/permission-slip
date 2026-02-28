# Manual Testing: Agent Registration Flow

Step-by-step guide to manually test the full invite -> register -> verify -> request approval -> execute flow by pretending to be an agent from another machine (or terminal).

## Prerequisites

- A running Permission Slip server (local or staging)
- `ssh-keygen` (comes with OpenSSH)
- `python3` (for the signing helper script)
- `curl`
- A logged-in user account on the Permission Slip dashboard

**Verify prerequisites are in your PATH** before proceeding:

```bash
command -v python3 && command -v curl && command -v ssh-keygen && echo "All prerequisites found"
```

If any command is missing, you'll see no output for that tool. Install what's needed:

- **macOS**: `python3` and `ssh-keygen` come with Xcode Command Line Tools (`xcode-select --install`). `curl` ships with macOS. If you use Homebrew: `brew install python3 curl`.
- **Linux (Debian/Ubuntu)**: `sudo apt install python3 curl openssh-client`

Set your host and API base URLs:

```bash
# Local development
export PS_HOST="http://localhost:8080"
export PS_API="${PS_HOST}/api/v1"

# Or staging
# export PS_HOST="https://staging.app.permissionslip.dev"
# export PS_API="${PS_HOST}/api/v1"
```

> **Note**: Most API endpoints live under `/api/v1`. The invite endpoint (`POST /invite/{code}`) is the exception — it's served at the host root.

---

## Step 1: Generate an Ed25519 Key Pair

Permission Slip uses Ed25519 keys in OpenSSH format. Generate one:

```bash
ssh-keygen -t ed25519 -f ~/.ssh/ps_test_agent -N "" -C "test-agent"
```

This creates:
- `~/.ssh/ps_test_agent` — private key
- `~/.ssh/ps_test_agent.pub` — public key

Read the public key (you'll need it later):

```bash
cat ~/.ssh/ps_test_agent.pub
# Output: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... test-agent
```

**Important**: Only the first two fields matter (`ssh-ed25519 AAAA...`). The trailing comment (`test-agent`) is ignored by Permission Slip but won't cause errors if included.

---

## Step 2: Set Up the Signing Helper

Create a working directory for the test and `cd` into it. **All remaining commands in this guide should be run from this directory.**

```bash
mkdir -p ~/ps-test && cd ~/ps-test
```

Every request to Permission Slip must include a cryptographic signature in the `X-Permission-Slip-Signature` header. Save this Python script as `sign_request.py` in your working directory:

```python
#!/usr/bin/env python3
"""
Signs HTTP requests for the Permission Slip agent protocol.

Usage:
  python3 sign_request.py <method> <path> <body_json> <private_key_file> [agent_id]

Prints the X-Permission-Slip-Signature header value to stdout.
"""
import sys
import json
import hashlib
import time
import base64
from pathlib import Path

try:
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
    from cryptography.hazmat.primitives.serialization import (
        load_ssh_private_key,
        Encoding,
        PublicFormat,
    )
except ImportError:
    print("Install cryptography: pip3 install cryptography", file=sys.stderr)
    sys.exit(1)


def load_private_key(path: str) -> Ed25519PrivateKey:
    key_bytes = Path(path).read_bytes()
    return load_ssh_private_key(key_bytes, password=None)


def hash_body(body: str) -> str:
    if not body:
        return "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    return hashlib.sha256(body.encode()).hexdigest()


def build_canonical(method: str, path: str, query: str, timestamp: int, body_hash: str) -> str:
    return f"{method.upper()}\n{path}\n{query}\n{timestamp}\n{body_hash}"


def sign(private_key: Ed25519PrivateKey, message: bytes) -> bytes:
    return private_key.sign(message)


def main():
    if len(sys.argv) < 4:
        print(f"Usage: {sys.argv[0]} <method> <path> <body_json> <private_key_file> [agent_id]", file=sys.stderr)
        sys.exit(1)

    method = sys.argv[1]
    full_path = sys.argv[2]
    body_json = sys.argv[3]
    key_file = sys.argv[4] if len(sys.argv) > 4 else str(Path.home() / ".ssh" / "ps_test_agent")
    agent_id = sys.argv[5] if len(sys.argv) > 5 else str(2**63 - 1)  # max int64 placeholder

    # Split path and query
    if "?" in full_path:
        path, query = full_path.split("?", 1)
    else:
        path, query = full_path, ""

    private_key = load_private_key(key_file)
    timestamp = int(time.time())
    body_hash = hash_body(body_json)
    canonical = build_canonical(method, path, query, timestamp, body_hash)
    sig_bytes = sign(private_key, canonical.encode())
    sig_b64 = base64.urlsafe_b64encode(sig_bytes).rstrip(b"=").decode()

    header = f'agent_id="{agent_id}", algorithm="Ed25519", timestamp="{timestamp}", signature="{sig_b64}"'
    print(header)


if __name__ == "__main__":
    main()
```

Install the dependency:

```bash
pip3 install cryptography
```

Test it:

```bash
python3 sign_request.py POST /invite/PS-TEST '{"test": true}' ~/.ssh/ps_test_agent
# Should print: agent_id="9223372036854775807", algorithm="Ed25519", timestamp="...", signature="..."
```

### Helper script for curl

Save this as `ps_curl.sh` in your working directory (next to `sign_request.py`):

> **Usage**: `bash ps_curl.sh <METHOD> <PATH> <BODY> [AGENT_ID] [BASE_URL]`
> - `PATH` is the router path used for signing (e.g., `/invite/PS-ABCD-1234`)
> - `BASE_URL` defaults to `$PS_API`; override for non-API routes (e.g., pass `$PS_HOST` for the invite endpoint)

```bash
#!/usr/bin/env bash
# ps_curl.sh — helper for making signed requests to Permission Slip

set -euo pipefail

method="$1"
path="$2"
body="${3:-}"
agent_id="${4:-}"
base_url="${5:-$PS_API}"

if [ -n "$agent_id" ]; then
    sig=$(python3 sign_request.py "$method" "$path" "$body" ~/.ssh/ps_test_agent "$agent_id")
else
    sig=$(python3 sign_request.py "$method" "$path" "$body" ~/.ssh/ps_test_agent)
fi

if [ -n "$body" ]; then
    curl -sS -X "$method" \
        -H "Content-Type: application/json" \
        -H "X-Permission-Slip-Signature: $sig" \
        -d "$body" \
        "${base_url}${path}" | python3 -m json.tool
else
    curl -sS -X "$method" \
        -H "X-Permission-Slip-Signature: $sig" \
        "${base_url}${path}" | python3 -m json.tool
fi
```

Test it:

```bash
bash ps_curl.sh POST /invite/PS-TEST '{"test": true}' "" "$PS_HOST"
# Should print a JSON error response (invalid invite) — that's fine, it means the script works
```

---

## Step 3: Create an Invite (Dashboard Side)

On the Permission Slip dashboard, click **"Add Agent"** to create a registration invite. This calls `POST /api/v1/registration-invites` internally and gives you an invite URL like:

```
https://app.permissionslip.dev/invite/PS-ABCD-1234
```

The invite code is the last path segment: `PS-ABCD-1234`

**Note**: Invites expire (default 15 minutes) and are single-use.

---

## Step 4: Register with the Invite Code (Agent Side)

Read your public key and build the registration request, replacing the invite code with your actual value:

```bash
PUB_KEY=$(cut -d' ' -f1,2 < ~/.ssh/ps_test_agent.pub)
REQUEST_ID=$(python3 -c "import uuid; print(uuid.uuid4())")

BODY=$(cat <<EOF
{
  "request_id": "$REQUEST_ID",
  "public_key": "$PUB_KEY",
  "metadata": {
    "name": "Test Agent",
    "version": "1.0.0"
  }
}
EOF
)

INVITE_CODE="PS-ABCD-1234" && bash ps_curl.sh POST "/invite/$INVITE_CODE" "$BODY" "" "$PS_HOST"
```

**Expected response** (200):

```json
{
    "agent_id": 42,
    "expires_at": "2026-02-23T12:25:00Z",
    "verification_required": true
}
```

Note the `agent_id` from the response — you'll need it in step 6.

**What happened**: The server created a pending agent registration. The dashboard now shows a **confirmation code** (e.g., `XK7-M9P`) that you need to complete registration.

---

## Step 5: Get the Confirmation Code (Dashboard Side)

Go back to the Permission Slip dashboard. You should see the pending agent with a 6-character confirmation code displayed (format: `XXX-XXX`, e.g., `XK7-M9P`).

Copy this code — you'll use it in the next step.

**Timing**: You have ~5 minutes before the registration expires.

---

## Step 6: Verify Registration (Agent Side)

Submit the confirmation code to complete registration:

```bash
AGENT_ID=42             # replace with agent_id from step 4 response
CONFIRM_CODE="XK7-M9P"  # replace with actual code from dashboard
REQUEST_ID=$(python3 -c "import uuid; print(uuid.uuid4())")

BODY=$(cat <<EOF
{
  "request_id": "$REQUEST_ID",
  "confirmation_code": "$CONFIRM_CODE"
}
EOF
)

bash ps_curl.sh POST "/agents/$AGENT_ID/verify" "$BODY" "$AGENT_ID"
```

**Expected response** (200):

```json
{
    "status": "registered",
    "registered_at": "2026-02-23T12:20:15Z"
}
```

The agent is now registered. The dashboard should show the agent's status as **"registered"**.

---

## Step 7: Discover Capabilities (Agent Side)

Now that you're registered, discover what you can do:

```bash
# List all available connectors (public, no auth needed)
curl -s "${PS_API}/connectors" | python3 -m json.tool

# Get details for a specific connector (e.g., gmail)
curl -s "${PS_API}/connectors/gmail" | python3 -m json.tool
```

The connector endpoints show what services and actions Permission Slip supports. Use the connector detail endpoint to see full parameter schemas for each action.

You can also check your agent-specific capabilities (requires signature authentication):

```bash
bash ps_curl.sh GET "/agents/$AGENT_ID/capabilities" "" "$AGENT_ID"
```

This shows which connectors are enabled for you, whether credentials are set up, and any standing approvals you have.

---

## Step 8: Request Approval for an Action (Agent Side) — *Planned*

> **Note**: Steps 8-11 use agent-facing approval and execution endpoints that are not yet implemented. The current backend only exposes dashboard-facing approval routes. The following describes the planned flow.

Pick an action from the connector catalog and request approval:

```bash
REQUEST_ID=$(python3 -c "import uuid; print(uuid.uuid4())")

BODY=$(cat <<EOF
{
  "agent_id": $AGENT_ID,
  "request_id": "$REQUEST_ID",
  "approver": "your_username",
  "action": {
    "type": "email.send",
    "version": "1",
    "parameters": {
      "to": ["test@example.com"],
      "subject": "Test from Permission Slip agent",
      "body": "This is a test email sent via the Permission Slip protocol."
    }
  },
  "context": {
    "description": "Testing the approval flow",
    "risk_level": "low"
  }
}
EOF
)

bash ps_curl.sh POST "/approvals/request" "$BODY" "$AGENT_ID"
```

**Expected response** (200):

```json
{
    "approval_id": "appr_xyz789",
    "approval_url": "https://app.permissionslip.dev/...",
    "status": "pending",
    "expires_at": "2026-02-23T12:30:00Z",
    "verification_required": true
}
```

Note the `approval_id` from the response — you'll need it in step 10.

---

## Step 9: Approve the Request (Dashboard Side)

The dashboard shows the pending approval with full action details. Click **"Approve"**.

After approving, the dashboard shows a **confirmation code** (e.g., `RK3-P7M`). Copy it — you'll use it in the next step.

---

## Step 10: Verify Approval and Get Token (Agent Side)

Submit the approval confirmation code to get an execution token:

```bash
APPROVAL_ID="appr_xyz789"  # replace with approval_id from step 8 response
APPROVAL_CODE="RK3-P7M"    # replace with code from dashboard
REQUEST_ID=$(python3 -c "import uuid; print(uuid.uuid4())")

BODY=$(cat <<EOF
{
  "request_id": "$REQUEST_ID",
  "confirmation_code": "$APPROVAL_CODE"
}
EOF
)

bash ps_curl.sh POST "/approvals/$APPROVAL_ID/verify" "$BODY" "$AGENT_ID"
```

**Expected response** (200):

```json
{
    "status": "approved",
    "approved_at": "2026-02-23T12:25:45Z",
    "token": {
        "access_token": "eyJhbGciOi...",
        "expires_at": "2026-02-23T12:30:45Z",
        "scope": "email.send",
        "scope_version": "1"
    }
}
```

Note the `access_token` from the response — you'll need it in step 11.

---

## Step 11: Execute the Action (Agent Side)

Use the token to execute the approved action:

```bash
ACTION_TOKEN="eyJhbGciOi..."  # replace with access_token from step 10 response
BODY=$(cat <<EOF
{
  "token": "$ACTION_TOKEN",
  "action_id": "email.send",
  "parameters": {
    "to": ["test@example.com"],
    "subject": "Test from Permission Slip agent",
    "body": "This is a test email sent via the Permission Slip protocol."
  }
}
EOF
)

bash ps_curl.sh POST "/actions/execute" "$BODY" "$AGENT_ID"
```

**Expected response** (200):

```json
{
    "status": "success",
    "action_id": "email.send",
    "executed_at": "2026-02-23T12:26:00Z",
    "result": {
        "message_id": "msg_abc123"
    }
}
```

**Important**: The token is single-use. If you need to execute the same action again, you must request a new approval.

---

## Troubleshooting

| Error | Cause | Fix |
|---|---|---|
| `command not found: python3` or `command not found: curl` | Prerequisites not installed or not in PATH | Run `command -v python3 && command -v curl` to verify. See [Prerequisites](#prerequisites) for install instructions |
| `invalid_signature` | Signature doesn't match | Check your key file path, ensure body matches exactly what was signed |
| `timestamp_expired` | Clock skew > 5 minutes | Sync your system clock (`ntpdate` or similar) |
| `invite_expired` | Invite TTL elapsed | Create a new invite from the dashboard |
| `invite_locked` | 5 failed attempts | Create a new invite from the dashboard |
| `registration_expired` | Didn't verify within 5 min | Re-register with a new invite |
| `invalid_code` | Wrong confirmation code | Check the dashboard for the correct code, watch `attempts_remaining` |
| `agent_id_mismatch` | Agent ID in path != header | Make sure you're passing the correct `$AGENT_ID` |
| `agent_not_authorized` | Agent not registered with approver | Check the approver username matches your account |
| `Expecting value: line 1 column 1 (char 0)` | `python3 -m json.tool` received empty or non-JSON input | The server likely isn't reachable. Run `curl -v "${PS_API}/connectors"` to test connectivity. Check that `$PS_API` is set and the server is running. See [Debugging empty responses](#debugging-empty-responses) below |

### Debugging Empty Responses

If you see `Expecting value: line 1 column 1 (char 0)`, it means `python3 -m json.tool` received no JSON to parse. The `ps_curl.sh` helper uses `curl -sS` which hides the progress bar but still shows connection errors. If you're using an older version of the script with `curl -s` (fully silent), connection failures are hidden entirely.

**Debug steps:**

1. **Verify the server is reachable:**
   ```bash
   curl -v "${PS_API}/connectors"
   ```
   If you see `Connection refused`, the server isn't running or `$PS_API` points to the wrong address.

2. **Check your environment variables are set:**
   ```bash
   echo "PS_API=$PS_API  AGENT_ID=$AGENT_ID  CONFIRM_CODE=$CONFIRM_CODE"
   ```

3. **Run the curl command manually with verbose output** to see the full HTTP exchange:
   ```bash
   REQUEST_ID=$(python3 -c "import uuid; print(uuid.uuid4())")
   BODY="{\"request_id\": \"$REQUEST_ID\", \"confirmation_code\": \"$CONFIRM_CODE\"}"
   sig=$(python3 sign_request.py POST "/agents/$AGENT_ID/verify" "$BODY" ~/.ssh/ps_test_agent "$AGENT_ID")
   curl -v -X POST \
       -H "Content-Type: application/json" \
       -H "X-Permission-Slip-Signature: $sig" \
       -d "$BODY" \
       "${PS_API}/agents/$AGENT_ID/verify"
   ```
   This will show connection details, HTTP status, headers, and the raw response body.

### Debugging Signatures

If you're getting `invalid_signature` errors, verify your canonical request matches what the server expects. The `<PATH>` must be the **router path** (what the handler sees after prefix stripping), not the full URL path. For API routes under `/api/v1`, this means using paths like `/agents/42/verify`, not `/api/v1/agents/42/verify`. For the invite route (served at the host root), the path is `/invite/{code}`.

```
<METHOD>\n<PATH>\n<QUERY>\n<TIMESTAMP>\n<BODY_SHA256_HEX>
```

Example for registration:

```
POST
/invite/PS-ABCD-1234

1708617600
a1b2c3d4e5f6...
```

Note the empty line for the query string (no query params). The body hash is the lowercase hex SHA-256 of the raw JSON body bytes.
