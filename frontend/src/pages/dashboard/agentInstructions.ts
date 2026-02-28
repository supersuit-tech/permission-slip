/**
 * Generates the three sets of agent-facing instructions that users copy-paste
 * to their AI agent during the registration flow.
 *
 * Each generator returns a plain-text block that is self-contained: the agent
 * should be able to follow the instructions without any prior context about
 * Permission Slip.
 */

// ---------------------------------------------------------------------------
// Signing helper scripts (embedded verbatim in the invite instructions)
// ---------------------------------------------------------------------------

const SIGN_REQUEST_PY = `#!/usr/bin/env python3
"""Signs HTTP requests for the Permission Slip agent protocol."""
import sys, hashlib, time, base64
from pathlib import Path
from urllib.parse import parse_qsl, quote

try:
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
    from cryptography.hazmat.primitives.serialization import load_ssh_private_key
except ImportError:
    print("Install cryptography: pip3 install cryptography", file=sys.stderr)
    sys.exit(1)

def canonicalize_query(raw):
    if not raw:
        return ""
    pairs = sorted(parse_qsl(raw, keep_blank_values=True))
    return "&".join(quote(k, safe="") + "=" + quote(v, safe="") for k, v in pairs)

def main():
    if len(sys.argv) < 4:
        print(f"Usage: {sys.argv[0]} <method> <path> <body_json> <private_key_file> [agent_id]", file=sys.stderr)
        sys.exit(1)

    method, full_path, body_json = sys.argv[1], sys.argv[2], sys.argv[3]
    key_file = sys.argv[4] if len(sys.argv) > 4 else str(Path.home() / ".ssh" / "permission_slip_agent")
    agent_id = sys.argv[5] if len(sys.argv) > 5 else str(2**63 - 1)

    path, query = (full_path.split("?", 1) + [""])[:2]
    query = canonicalize_query(query)
    private_key = load_ssh_private_key(Path(key_file).read_bytes(), password=None)
    timestamp = int(time.time())
    body_hash = hashlib.sha256(body_json.encode()).hexdigest() if body_json else "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    canonical = f"{method.upper()}\\n{path}\\n{query}\\n{timestamp}\\n{body_hash}"
    sig_b64 = base64.urlsafe_b64encode(private_key.sign(canonical.encode())).rstrip(b"=").decode()
    print(f'agent_id="{agent_id}", algorithm="Ed25519", timestamp="{timestamp}", signature="{sig_b64}"')

if __name__ == "__main__":
    main()`;

const PS_CURL_SH = `#!/usr/bin/env bash
set -euo pipefail
method="$1"; path="$2"; body="\${3:-}"; agent_id="\${4:-}"; base_url="\${5:-$PS_API}"
if [ -n "$agent_id" ]; then
    sig=$(python3 ~/.permission-slip/sign_request.py "$method" "$path" "$body" ~/.ssh/permission_slip_agent "$agent_id")
else
    sig=$(python3 ~/.permission-slip/sign_request.py "$method" "$path" "$body" ~/.ssh/permission_slip_agent)
fi
if [ -n "$body" ]; then
    curl -sS -X "$method" -H "Content-Type: application/json" -H "X-Permission-Slip-Signature: $sig" -d "$body" "\${base_url}\${path}" | python3 -m json.tool
else
    curl -sS -X "$method" -H "X-Permission-Slip-Signature: $sig" "\${base_url}\${path}" | python3 -m json.tool
fi`;

// ---------------------------------------------------------------------------
// Step 1 — Invite instructions (sent with the invite URL)
// ---------------------------------------------------------------------------

export function generateInviteInstructions(
  inviteCode: string,
  origin: string,
): string {
  return `# Permission Slip — Agent Registration

Permission Slip is a protocol that lets AI agents request human approval before
performing sensitive actions (sending emails, managing calendars, etc.). You have
been invited to register as an agent on a Permission Slip server.

You may be registered with multiple Permission Slip servers, or by multiple users
on the same server. Each registration gives you a different agent_id, but you
reuse the same signing key for all of them.

Follow every step below in order.

## Prerequisites

You need python3, curl, and ssh-keygen. Install the cryptography library:

    pip3 install cryptography

## Step 1: Set up your signing key

If you already have a Permission Slip key pair at ~/.ssh/permission_slip_agent,
skip to Step 2. Otherwise generate one — you will reuse this key for all
Permission Slip registrations across all servers and users:

    ssh-keygen -t ed25519 -f ~/.ssh/permission_slip_agent -N "" -C "permission-slip"

## Step 2: Create the signing helpers

    mkdir -p ~/.permission-slip

Save the signing script:

    cat > ~/.permission-slip/sign_request.py << 'PYEOF'
${SIGN_REQUEST_PY}
PYEOF

Save the curl helper:

    cat > ~/.permission-slip/ps_curl.sh << 'SHEOF'
${PS_CURL_SH}
SHEOF

## Step 3: Register with this invite

Run these commands. The invite code is single-use and expires in 15 minutes.
After you register, you will have ~5 minutes to verify with a confirmation code
(the expires_at in the response reflects this verification deadline).

    export PS_HOST="${origin}"
    export PS_API="\${PS_HOST}/api/v1"

    PUB_KEY=$(cut -d' ' -f1,2 < ~/.ssh/permission_slip_agent.pub)
    REQUEST_ID=$(python3 -c "import uuid; print(uuid.uuid4())")

    BODY='{"request_id":"'"$REQUEST_ID"'","public_key":"'"$PUB_KEY"'","metadata":{"name":"YOUR_AGENT_NAME","version":"1.0.0"}}'

    cd ~/.permission-slip && bash ps_curl.sh POST "/invite/${inviteCode}" "$BODY" "" "$PS_HOST"

IMPORTANT: Replace YOUR_AGENT_NAME with your actual name (e.g. "OpenClaw", "Claude").

Expected response:

    {
        "agent_id": <number>,
        "expires_at": "<timestamp>",
        "verification_required": true
    }

## What to do next

1. Save the agent_id from the response — you will need it to finish registration.
2. Tell the user your agent_id and that you have completed step 1.
3. Wait for the user to send you a confirmation code and further instructions.

Do NOT proceed until you receive the confirmation code from the user.

NOTE: After registration is complete, the user will send you instructions for
saving this registration to ~/.permission-slip/registrations.json so you can
look it up later. You do not need to do this now.`;
}

// ---------------------------------------------------------------------------
// Step 2 — Verification instructions (sent with the confirmation code)
// ---------------------------------------------------------------------------

export function generateVerificationInstructions(
  agentId: number,
  confirmationCode: string,
  origin: string,
): string {
  return `# Permission Slip — Complete Your Registration

You are almost done registering with Permission Slip. Use the confirmation code
below to verify your identity.

    Confirmation code: ${confirmationCode}
    Your agent ID:     ${agentId}

Run these commands (they use the signing helpers from the previous step):

    export PS_HOST="${origin}"
    export PS_API="\${PS_HOST}/api/v1"

    AGENT_ID=${agentId}
    CONFIRM_CODE="${confirmationCode}"
    REQUEST_ID=$(python3 -c "import uuid; print(uuid.uuid4())")

    BODY='{"request_id":"'"$REQUEST_ID"'","confirmation_code":"'"$CONFIRM_CODE"'"}'

    cd ~/.permission-slip && bash ps_curl.sh POST "/agents/$AGENT_ID/verify" "$BODY" "$AGENT_ID"

Expected response:

    {
        "status": "registered",
        "registered_at": "<timestamp>"
    }

Once you see "registered", tell the user that registration is complete and wait
for further instructions.`;
}

// ---------------------------------------------------------------------------
// Step 3 — Post-registration instructions (the agent should memorize these)
// ---------------------------------------------------------------------------

export function generatePostRegistrationInstructions(
  agentId: number,
  origin: string,
): string {
  return `# Permission Slip — You Are Registered

You are now registered with a Permission Slip server. Save this registration so
you can look it up when interacting with this server in the future.

## Save this registration

Add this entry to ~/.permission-slip/registrations.json. If the file does not
exist, create it. If it already exists, append to the "registrations" array.

    {
      "registrations": [
        {
          "server": "${origin}",
          "api_base": "${origin}/api/v1",
          "agent_id": ${agentId},
          "registered_at": "${new Date().toISOString()}"
        }
      ]
    }

You may have multiple registrations — different servers, or different users on
the same server. Each registration has its own agent_id. When making requests,
look up the correct agent_id for the target server from this file.

## Your signing key and helpers

These are shared across ALL registrations (do not create per-server copies):

    Signing key:     ~/.ssh/permission_slip_agent
    Signing helpers: ~/.permission-slip/

## Discovering available actions

Permission Slip connects to third-party services (email, calendar, etc.) through
connectors. To see what this server supports:

    # List all connectors
    curl -s ${origin}/api/v1/connectors | python3 -m json.tool

    # Get details for a specific connector (e.g. gmail)
    curl -s ${origin}/api/v1/connectors/gmail | python3 -m json.tool

These endpoints are public and do not require authentication.

## How approvals work

When you need to perform a sensitive action on behalf of a user:

1. You request approval by calling the Permission Slip API with the action details.
2. The user reviews the request on their dashboard and approves or denies it.
3. If approved, the user shares a confirmation code with you.
4. You submit the confirmation code to receive an execution token.
5. You use the token to execute the action.

Every request must include a cryptographic signature using your signing key and
the correct agent_id for the target server.

## Remember

- Always sign API requests with your key at ~/.ssh/permission_slip_agent.
- Use the correct agent_id for the server you are talking to (check registrations.json).
- Check available connectors before requesting an action to ensure it is supported.
- The user must approve every sensitive action before you can execute it.`;
}
