---
name: permission-slip-approvals
description: Non-blocking approval flow for Permission Slip — request, optional push wake or detached watcher, heartbeat sweep backstop, end your turn, continue when woken.
---

# Permission Slip Approvals (OpenClaw)

Use this skill when you need human approval for a Permission Slip action and must **not** block the OpenClaw session while waiting.

## The pattern

1. **Request** — run `permission-slip request ...`. If the response is `pending`, read `wait_hint`, `wait_command`, and `push_wake_configured` from the JSON output.
2. **Push wake (when configured)** — if `push_wake_configured` is true, the Permission Slip server POSTs to your OpenClaw gateway hooks on resolution. You may **skip** spawning a watcher and end your turn immediately.
3. **Watch in background (fallback)** — when no webhook is configured (or as extra safety), run `wait_command` as a **detached background process**. Do **not** poll `permission-slip status` in a loop inside your turn.
4. **Heartbeat sweep (every beat)** — run `permission-slip pending` on every OpenClaw heartbeat. Act on any items in `resolved`; this catches push wakes that were missed.
5. **End your turn** — the main session stays responsive while the human reviews on the dashboard.
6. **On wake** — when OpenClaw delivers a system event like `Permission Slip appr_x resolved: approved — continue the task`, run `permission-slip status appr_x` to fetch `execution_result` and continue the task.

If `request` returns `executed: true` or `status: approved`, the action already ran — skip watching.

## Verified metadata constraints (`$meta`)

For connectors that resolve envelope metadata (Proton Mail message-targeted actions), prefer **`$meta` constraints** over plain parameter pins on `from` / `to` — agents can spoof param values, but `$meta` is checked against server-fetched headers.

**Proton Mail examples:**

```bash
# Auto-approve read_email only for Amazon auto-confirm messages
permission-slip request --action protonmail.read_email \
  --standing-constraints '{"message_id":"*","folder":"*","$meta":{"from":{"$pattern":"auto-confirm@amazon.com"}}}' \
  --params '{"message_id":42,"folder":"INBOX"}'

# Recipient rule: at least one To address must match (per message)
# {"message_id":"*","$meta":{"to":{"$pattern":"*@mycorp.com"}}}
```

- Use `$meta.from` (or legacy `sender`) for sender rules — not a top-level `from` param on `read_email`.
- `$meta.bcc` rarely matches received mail (IMAP omits Bcc on inbox messages); do not rely on it for inbox automation.
- For `protonmail.send_email`, constrain outbound `to` / `cc` / `bcc` as normal params; array patterns require **every** recipient to match.

See [Proton Mail connector docs](../../docs/connectors/protonmail.md#standing-approval-constraints-meta) for the full action table.

## Commands

```bash
# Submit an action (auto-executes when a standing approval matches)
permission-slip request --action email.send --params '{"to":"user@example.com","subject":"Hi"}'

# Heartbeat backstop — run on every OpenClaw heartbeat:
permission-slip pending

# When pending without push_wake_configured — spawn watcher in background:
permission-slip watch appr_xxxxxxxx --session-key <your session key>

# After wake — fetch the outcome:
permission-slip status appr_xxxxxxxx

# One-time setup for push wakes (private tailnet URL + hooks token):
permission-slip webhook set --url http://<host>:18789/hooks --token <token>
# Grok Bot (public Cursor webhook — do not use the OpenClaw private-URL path):
permission-slip webhook set --provider grokbot \
  --url https://api2.cursor.sh/automations/webhook/<id> \
  --token <authorization-header-value>
permission-slip webhook status --test
```

## Multiple pending approvals

Run **one** `permission-slip watch <id>` per pending approval when using the watcher fallback. At personal-use scale, N small background processes is fine. With push wakes configured, heartbeat `pending` alone is usually enough.

## Recovery

- **Gateway restarted while watching** — the watcher process may be orphaned. Heartbeat `pending` or `permission-slip status <id>` recovers.
- **Push wake missed** — heartbeat sweep picks up resolved approvals within one beat interval.
- **Accidentally polling `status` in a loop** — use `pending` on heartbeat or the watcher pattern instead.

## OpenClaw notify details

```bash
permission-slip watch appr_x --session-key agent:main:imessage
```

Default notify uses `openclaw system event` when `openclaw` is on PATH. With `--session-key`, the default template uses `--mode next-heartbeat --session-key {session_key}` for a reliable targeted wake (not `--mode now`, which can return ok without resuming an idle session).

## Related docs

- [OpenClaw integration docs](../../docs/integrations/openclaw.md)
- [Self-hosted OpenClaw push wake setup](../../docs/deployment-self-hosted.md#openclaw-push-wakes)
