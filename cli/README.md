# @permission-slip/cli

Agent-facing CLI for [Permission Slip](https://github.com/supersuit-tech/permission-slip). All commands output compact JSON by default; pass `--pretty` for formatted JSON.

## Setup

Set your server URL once (required — there is no default host):

```bash
export PS_SERVER=https://your-permission-slip.example.com
# or
permission-slip config set default_server https://your-permission-slip.example.com
```

Register and verify:

```bash
permission-slip register --invite-code <code>
permission-slip verify --code <confirmation_code>
```

## Standing approval constraints

Standing approvals store JSON **constraints** that limit when auto-approve applies. The web and mobile apps edit v2 structured constraints; legacy flat maps still evaluate identically via a read adapter (no database backfill required).

### Format constraints for humans

Use `auto-approve format` to turn stored JSON into readable text (legacy flat or v2 structured):

```bash
# JSON output: { "lines": [...], "text": "..." }
permission-slip auto-approve format \
  --constraints '{"recipient":{"$pattern":"*@acme.com"},"$meta":{"from":"boss@acme.com"}}'

# Plain text (one line per constraint)
permission-slip auto-approve format \
  --constraints '{"$version":2,"match":"any","groups":[...]}' \
  --text
```

Example plain-text output:

```
recipient: *@acme.com
Verified sender: boss@acme.com
```

With multiple scenarios (v2), lines are prefixed `Scenario 1:`, `Scenario 2:`, etc. Negated rules show `not` before the value.

### Propose a new standing approval

```bash
permission-slip auto-approve request \
  --action-type email.send \
  --constraints '{"$version":2,"match":"any","groups":[{"match":"all","conditions":[{"field":"recipient","op":"matches","value":{"$pattern":"*@company.com"}}]}]}'
```

The proposal must be approved in the web or mobile app before it takes effect.

See [docs/standing-approval-constraints.md](../docs/standing-approval-constraints.md) for constraint syntax, semantics, and backward-compatibility notes.

## Common commands

| Command | Purpose |
|---------|---------|
| `capabilities` | List connector actions and standing approvals |
| `request` | Request approval for an action |
| `request-bulk` | Bulk approval for N same-type actions |
| `watch` | Poll a pending approval and wake the session on resolve |
| `pending` | Heartbeat sweep for pending/recent approvals |
| `changelog` | Show CLI updates since your last session |

Run `permission-slip --help` for the full command list.

## Development

```bash
cd cli && npm install && npm test && npm run build
```

The CLI depends on `@permission-slip/constraints-format` in `shared/constraints/` (built automatically on `postinstall`).
